package mqttbridge

import (
	"context"
	"embed"
	"fmt"
	"net"
	"net/http"

	"github.com/dueckminor/home-assistant-addons/go/ginutil"
	"github.com/dueckminor/home-assistant-addons/go/services/mqtt"
	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFS embed.FS

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
		ginutil.ServeEmbedFS(r, distFS, "dist")
	}

	ep := NewEndpoints(s)
	s.endpoints = ep
	api := r.Group("/api")
	ep.setupEndpoints(api)

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

func (s *Server) SendEvent(event Event) {
	if s.endpoints != nil {
		s.endpoints.SendEvent(event)
	}
}

func (s *Server) SetOnFirstClientConnected(callback OnFirstClientConnectedFunc) {
	if s.endpoints != nil {
		s.endpoints.SetOnFirstClientConnected(callback)
	}
}

func (s *Server) SetOnLastClientDisconnected(callback OnLastClientDisconnectedFunc) {
	if s.endpoints != nil {
		s.endpoints.SetOnLastClientDisconnected(callback)
	}
}

func (s *Server) Listen() (err error) {
	s.listener, err = net.Listen("tcp", s.httpServer.Addr)
	return err
}

func (s *Server) Serve(ctx context.Context) {
	go func() {
		<-ctx.Done()
		s.httpServer.Shutdown(context.Background())
	}()
	s.httpServer.Serve(s.listener)
}
