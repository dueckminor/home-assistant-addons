package mqttbridge

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/dueckminor/home-assistant-addons/go/embed/mqtt_bridge_dist"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"github.com/dueckminor/home-assistant-addons/go/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type Server struct {
	httpServer *http.Server
	listener   net.Listener
	mqttConn   mqtt.Conn
	endpoints  *Endpoints
}

func NewServer(adminPort int, distAdmin string) *Server {
	s := &Server{}

	r := gin.Default()

	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", adminPort),
		Handler: r,
	}

	if distAdmin != "" {
		ginutil.ServeFromUri(r, distAdmin)
	} else {
		ginutil.ServeEmbedFS(r, mqtt_bridge_dist.FS, "dist")
	}

	ep := NewEndpoints(s)
	s.endpoints = ep
	api := r.Group("/api")
	ep.setupEndpoints(api)

	AddHandler(func(topic Topic) {
		ep.SendEvent(
			Event{
				Source: "mqtt",
				Time:   topic.Time,
				Topic:  topic.Name,
				Value:  topic.Value,
			},
		)
	})

	// Start the event broadcaster
	ep.StartEventBroadcaster()

	return s
}

func (s *Server) SetMqttConn(mqttConn mqtt.Conn) {
	s.mqttConn = mqttConn
}

func (s *Server) GetEndpoints() *Endpoints {
	return s.endpoints
}

func (s *Server) Listen() (err error) {
	fmt.Println("Listen: ", s.httpServer.Addr)
	s.listener, err = net.Listen("tcp", s.httpServer.Addr)
	return err
}

func (s *Server) Serve(ctx context.Context) {
	fmt.Println("Serve...")
	go func() {
		<-ctx.Done()
		fmt.Println("Serve done...")
		s.httpServer.Shutdown(context.Background())
	}()
	s.httpServer.Serve(s.listener)
}
