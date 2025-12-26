package ftp

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/gonzalop/ftp/server"
)

// Config holds the FTP server configuration
type Config struct {
	DataDir    string
	Port       int
	Username   string
	Password   string
	ServerName string
	Debug      bool
}

// Server wraps the FTP server functionality
type Server struct {
	config    *Config
	ftpServer *server.Server
}

// NewServer creates a new FTP server instance
func NewServer(config *Config) (*Server, error) {
	if config.Username == "" || config.Password == "" {
		return nil, fmt.Errorf("FTP username and password must be provided")
	}

	return &Server{
		config: config,
	}, nil
}

// validateUser validates FTP credentials
func (s *Server) validateUser(user string, pass string, host string) (string, bool, error) {
	authenticated := (user == s.config.Username && pass == s.config.Password)

	if !authenticated {
		log.Printf("Authentication failed for user: %s", user)
		return "", false, nil
	}

	// Return the data directory and grant write access (false = write allowed)
	return s.config.DataDir, false, nil
}

// generateSelfSignedCert creates a self-signed certificate
func (s *Server) generateSelfSignedCert() tls.Certificate {
	// Generate a private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: s.config.ServerName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{s.config.ServerName},
		BasicConstraintsValid: true,
	}

	// Create the certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		log.Fatalf("Failed to create certificate: %v", err)
	}

	// Encode certificate and key to PEM format
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)})

	// Create TLS certificate
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("Failed to create TLS certificate: %v", err)
	}

	return cert
}

// Start starts the FTP server
func (s *Server) Start() error {
	// Ensure the FTP directory exists
	ftpRoot := s.config.DataDir + "/ftp"
	if err := os.MkdirAll(ftpRoot, 0755); err != nil {
		return fmt.Errorf("could not create FTP directory: %v", err)
	}

	// Ensure cam1 subdirectory exists (commonly used by cameras)
	cam1Dir := ftpRoot + "/cam1"
	if err := os.MkdirAll(cam1Dir, 0755); err != nil {
		return fmt.Errorf("could not create cam1 directory: %v", err)
	}

	log.Printf("FTP server starting, root directory: %s", ftpRoot)

	// Create a driver to serve the local directory
	driver, err := server.NewFSDriver(ftpRoot,
		server.WithDisableAnonymous(true),
		server.WithAuthenticator(s.validateUser),
	)
	if err != nil {
		return fmt.Errorf("failed to create FTP driver: %v", err)
	}

	// Create logger for FTP server
	logLevel := slog.LevelInfo
	if s.config.Debug {
		logLevel = slog.LevelDebug
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))

	// Create TLS config for FTPS support
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{s.generateSelfSignedCert()},
		ServerName:         s.config.ServerName,
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}

	// Create the server
	addr := fmt.Sprintf(":%d", s.config.Port)
	srv, err := server.NewServer(addr,
		server.WithDriver(driver),
		server.WithLogger(logger),
		server.WithTLS(tlsConfig),
	)
	if err != nil {
		return fmt.Errorf("failed to create FTP server: %v", err)
	}

	s.ftpServer = srv
	log.Printf("Starting FTPS server on %s", addr)
	return srv.ListenAndServe()
}

// Stop stops the FTP server
func (s *Server) Stop() error {
	if s.ftpServer != nil {
		// The gonzalop/ftp server doesn't have a Stop method,
		// so we can't gracefully stop it here
		log.Printf("FTP server stopping...")
	}
	return nil
}
