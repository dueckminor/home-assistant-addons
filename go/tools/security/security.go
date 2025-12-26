package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"net"
	"os"
	"time"

	"github.com/gonzalop/ftp/server"
)

var data string
var ftpPort int
var debug bool

func init() {
	flag.StringVar(&data, "data", "/data", "the data dir")
	flag.BoolVar(&debug, "debug", false, "start in debug mode (without authentication)")
	flag.IntVar(&ftpPort, "ftp-port", 21, "the FTP port")
	flag.Parse()
}

var ftpRootDir = "/data/ftp"
var ftpUser = os.Getenv("FTP_USERNAME")
var ftpPassword = os.Getenv("FTP_PASSWORD")

func validateUser(user string, pass string, host string) (string, bool, error) {
	authenticated := (user == ftpUser && pass == ftpPassword)

	if !authenticated {
		log.Printf("Authentication failed for user: %s", user)
		return "", false, nil
	}

	return ftpRootDir, false, nil
}

func main() {
	// Ensure the FTP directory exists
	if err := os.MkdirAll(ftpRootDir, 0755); err != nil {
		log.Fatalf("Could not create FTP directory: %v", err)
	}

	log.Printf("FTP server starting, root directory: %s", ftpRootDir)

	// Create a driver to serve the local directory
	driver, err := server.NewFSDriver(ftpRootDir,
		server.WithDisableAnonymous(true),
		server.WithAuthenticator(validateUser),
	)
	if err != nil {
		log.Fatal(err)
	}

	// Create logger for FTP server
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create TLS config for FTPS support
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{generateSelfSignedCert()},
		ServerName:         "security",
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
	}

	addr := fmt.Sprintf(":%d", ftpPort)

	// Create and start the server
	srv, err := server.NewServer(addr,
		server.WithDriver(driver),
		server.WithLogger(logger),
		server.WithTLS(tlsConfig),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Starting FTPS server on %s\n", addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

// generateSelfSignedCert creates a self-signed certificate for testing
func generateSelfSignedCert() tls.Certificate {
	// Generate a private key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		log.Fatalf("Failed to generate private key: %v", err)
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "security",
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		// Include both localhost and the actual network IP
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		DNSNames:              []string{"security"},
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
