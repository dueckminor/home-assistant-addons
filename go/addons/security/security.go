package security

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/dueckminor/home-assistant-addons/go/embed/security_dist"
	"github.com/dueckminor/home-assistant-addons/go/utils/ginutil"
	"github.com/gin-gonic/gin"
)

type Security interface {
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Wait() error
}

func NewSecurity(httpPort int, dist string, dataDir string) Security {
	return &security{
		httpPort: httpPort,
		dist:     dist,
		dataDir:  dataDir,
	}
}

type security struct {
	httpPort int
	dist     string
	dataDir  string
}

func (s *security) Start(ctx context.Context, wg *sync.WaitGroup) error {
	r := gin.Default()

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.httpPort),
		Handler: r,
	}

	go func() {
		<-ctx.Done()
		httpServer.Shutdown(context.Background())
	}()

	if s.dist != "" {
		ginutil.ServeFromUri(r, s.dist)
	} else {
		ginutil.ServeEmbedFS(r, security_dist.FS, "dist")
	}

	s.setupSecurityEndpoints(r)

	wg.Add(1)

	go func() {
		defer func() {
			wg.Done()
		}()
		fmt.Printf("Config server started on port %d\n", s.httpPort)
		err := httpServer.ListenAndServe()
		if err != nil {
			fmt.Printf("Config server error: %v\n", err)
		}
	}()

	return nil
}

func (s *security) Wait() error {
	return nil
}
