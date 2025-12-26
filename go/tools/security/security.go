package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/dueckminor/home-assistant-addons/go/ftp"
	"github.com/dueckminor/home-assistant-addons/go/security"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := sync.WaitGroup{}

	// Command line flags
	var (
		dataDir    = flag.String("data", "/data", "the data directory")
		dist       = flag.String("dist", "", "the dist dir for the security (or uri)")
		ftpPort    = flag.Int("ftp-port", 21, "the FTP port")
		httpPort   = flag.Int("http-port", 80, "the HTTP port")
		debug      = flag.Bool("debug", false, "enable debug logging")
		serverName = flag.String("server-name", "security", "the server name for TLS certificate")
	)
	flag.Parse()

	// Get credentials from environment
	ftpUser := os.Getenv("FTP_USERNAME")
	ftpPassword := os.Getenv("FTP_PASSWORD")

	// Set default credentials if not provided (for development)
	if ftpUser == "" {
		ftpUser = "admin"
		log.Printf("Warning: Using default FTP username 'admin'")
	}
	if ftpPassword == "" {
		ftpPassword = "admin123"
		log.Printf("Warning: Using default FTP password")
	}

	security := security.NewSecurity(*httpPort, *dist, *dataDir)
	err := security.Start(ctx, &wg)

	// Create FTP server configuration
	config := &ftp.Config{
		DataDir:    *dataDir,
		Port:       *ftpPort,
		Username:   ftpUser,
		Password:   ftpPassword,
		ServerName: *serverName,
		Debug:      *debug,
	}

	// Create and start FTP server
	server, err := ftp.NewServer(config)
	if err != nil {
		log.Fatalf("Failed to create FTP server: %v", err)
	}

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("FTP server error: %v", err)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	log.Printf("Received signal %v, shutting down...", sig)
	cancel()

	wg.Wait()

	// Stop the server
	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}

	log.Println("Server stopped")
}
