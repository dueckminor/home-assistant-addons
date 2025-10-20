package mqttbridge

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// Callback function types
type OnFirstClientConnectedFunc func()
type OnLastClientDisconnectedFunc func()

type Endpoints struct {
	server                   *Server
	upgrader                 websocket.Upgrader
	clients                  map[*websocket.Conn]bool
	clientsMu                sync.RWMutex
	eventChan                chan Event
	onFirstClientConnected   OnFirstClientConnectedFunc
	onLastClientDisconnected OnLastClientDisconnectedFunc
}

func NewEndpoints(server *Server) *Endpoints {
	return &Endpoints{
		server: server,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
		clients:   make(map[*websocket.Conn]bool),
		eventChan: make(chan Event, 1024),
	}
}

// SetOnFirstClientConnected sets the callback function that gets called when the first client connects
func (ep *Endpoints) SetOnFirstClientConnected(callback OnFirstClientConnectedFunc) {
	ep.onFirstClientConnected = callback
}

// SetOnLastClientDisconnected sets the callback function that gets called when the last client disconnects
func (ep *Endpoints) SetOnLastClientDisconnected(callback OnLastClientDisconnectedFunc) {
	ep.onLastClientDisconnected = callback
}

type Event struct {
	Source string    `json:"source"`
	Time   time.Time `json:"time"`
	Topic  string    `json:"topic"`
	Value  any       `json:"value"`
}

func (ep *Endpoints) setupEndpoints(r *gin.RouterGroup) {
	r.GET("/events", ep.handleWebSocket)
}

// handleWebSocket upgrades HTTP connection to WebSocket for real-time events
func (ep *Endpoints) handleWebSocket(c *gin.Context) {
	ws, err := ep.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection to WebSocket: %v", err)
		return
	}
	defer ws.Close()

	// Add client to the map
	ep.clientsMu.Lock()
	wasEmpty := len(ep.clients) == 0
	ep.clients[ws] = true
	clientCount := len(ep.clients)
	ep.clientsMu.Unlock()

	// Call onFirstClientConnected callback if this is the first client
	if wasEmpty && ep.onFirstClientConnected != nil {
		ep.onFirstClientConnected()
	}

	log.Printf("WebSocket client connected. Total clients: %d", clientCount)

	// Remove client when connection closes
	defer func() {
		ep.clientsMu.Lock()
		delete(ep.clients, ws)
		clientCount := len(ep.clients)
		willBeEmpty := clientCount == 0
		ep.clientsMu.Unlock()

		// Call onLastClientDisconnected callback if this was the last client
		if willBeEmpty && ep.onLastClientDisconnected != nil {
			ep.onLastClientDisconnected()
		}

		log.Printf("WebSocket client disconnected. Total clients: %d", clientCount)
	}()

	// Set up ping-pong mechanism for keep-alive
	const (
		writeWait  = 10 * time.Second    // Time allowed to write a message to the peer
		pongWait   = 60 * time.Second    // Time allowed to read the next pong message from the peer
		pingPeriod = (pongWait * 9) / 10 // Send pings to peer with this period. Must be less than pongWait
	)

	// Set read deadline and pong handler
	ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	// Start ping ticker
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	// Channel to signal when to stop ping goroutine
	done := make(chan struct{})
	defer close(done)

	// Goroutine to send periodic pings
	go func() {
		for {
			select {
			case <-ticker.C:
				ws.SetWriteDeadline(time.Now().Add(writeWait))
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					log.Printf("Failed to send ping: %v", err)
					return
				}
			case <-done:
				return
			}
		}
	}()

	// Read messages from client (mainly for handling pong responses)
	for {
		messageType, message, err := ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket unexpected close error: %v", err)
			} else {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		// Log received message types for debugging (except pong which is handled automatically)
		if messageType != websocket.PongMessage {
			log.Printf("Received WebSocket message type: %d, length: %d", messageType, len(message))
		}
	}
}

// BroadcastEvent sends an event to all connected WebSocket clients
func (ep *Endpoints) BroadcastEvent(event Event) {
	ep.clientsMu.RLock()
	clientsList := make([]*websocket.Conn, 0, len(ep.clients))
	for client := range ep.clients {
		clientsList = append(clientsList, client)
	}
	ep.clientsMu.RUnlock()

	var clientsToRemove []*websocket.Conn

	for _, client := range clientsList {
		// Set write deadline to prevent hanging
		client.SetWriteDeadline(time.Now().Add(10 * time.Second))
		err := client.WriteJSON(event)
		if err != nil {
			log.Printf("Error sending event to WebSocket client: %v", err)
			clientsToRemove = append(clientsToRemove, client)
		}
	}

	// Remove failed clients
	if len(clientsToRemove) > 0 {
		ep.clientsMu.Lock()
		for _, client := range clientsToRemove {
			client.Close()
			delete(ep.clients, client)
		}
		ep.clientsMu.Unlock()
		log.Printf("Removed %d failed WebSocket clients", len(clientsToRemove))
	}
}

// StartEventBroadcaster starts a goroutine that listens for events and broadcasts them
func (ep *Endpoints) StartEventBroadcaster() {
	go func() {
		for event := range ep.eventChan {
			ep.BroadcastEvent(event)
		}
	}()
}

// SendEvent queues an event to be broadcasted to all WebSocket clients
func (ep *Endpoints) SendEvent(event Event) {
	select {
	case ep.eventChan <- event:
	default:
		log.Println("Event channel full, dropping event")
	}
}
