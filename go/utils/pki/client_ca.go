package pki

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/utils/crypto"
	cryptorand "github.com/dueckminor/home-assistant-addons/go/utils/crypto/rand"
	"software.sslmate.com/src/go-pkcs12"
)

type IssuedCert struct {
	Name     string    `json:"name"`
	Serial   int64     `json:"serial"`
	IssuedAt time.Time `json:"issued_at"`
}

type ClientCA struct {
	mu         sync.RWMutex
	dataDir    string
	name       string
	key        crypto.PrivateKey
	cert       *x509.Certificate
	pool       *x509.CertPool
	issued     []IssuedCert
	nextSerial atomic.Int64
}

func NewClientCA(dataDir, name string) (*ClientCA, error) {
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, err
	}

	ca := &ClientCA{dataDir: dataDir, name: name}

	keyFile := filepath.Join(dataDir, "ca.key.pem")
	certFile := filepath.Join(dataDir, "ca.cert.pem")

	key, err := crypto.GetOrCreatePrivateKeyFile(keyFile)
	if err != nil {
		return nil, err
	}
	ca.key = key

	certData, err := os.ReadFile(certFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		cert, err := ca.createCACert()
		if err != nil {
			return nil, err
		}
		certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
		if err := os.WriteFile(certFile, certPEM, 0600); err != nil {
			return nil, err
		}
		ca.cert = cert
	} else {
		block, _ := pem.Decode(certData)
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		ca.cert = cert
	}

	ca.pool = x509.NewCertPool()
	ca.pool.AddCert(ca.cert)
	ca.nextSerial.Store(2)

	return ca, nil
}

func (ca *ClientCA) createCACert() (*x509.Certificate, error) {
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "Gateway Client CA (" + ca.name + ")",
		},
		NotBefore:             time.Now().Add(-1 * time.Minute),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}
	certDER, err := x509.CreateCertificate(rand.Reader, template, template, ca.key.Public(), ca.key)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDER)
}

// Pool returns the x509 cert pool containing the CA certificate, for use in TLS ClientCAs.
func (ca *ClientCA) Pool() *x509.CertPool {
	return ca.pool
}

// CACertPEM returns the CA certificate in PEM format.
func (ca *ClientCA) CACertPEM() string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw}))
}

// IssueClientCert generates a new client certificate signed by this CA.
// Returns the PKCS#12 bundle bytes, a one-time password to protect it, and any error.
func (ca *ClientCA) IssueClientCert(name string) (pfxData []byte, password string, err error) {
	clientKey, err := crypto.CreatePrivateKey()
	if err != nil {
		return nil, "", err
	}

	serial := ca.nextSerial.Add(1)
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject:      pkix.Name{CommonName: name},
		NotBefore:    time.Now().Add(-1 * time.Minute),
		NotAfter:     time.Now().Add(2 * 365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, clientKey.Public(), ca.key)
	if err != nil {
		return nil, "", err
	}
	clientCert, err := x509.ParseCertificate(certDER)
	if err != nil {
		return nil, "", err
	}

	password, err = cryptorand.GetString(12)
	if err != nil {
		return nil, "", err
	}

	pfxData, err = pkcs12.Legacy.Encode(clientKey.RSA(), clientCert, []*x509.Certificate{ca.cert}, password)
	if err != nil {
		return nil, "", err
	}

	ca.mu.Lock()
	ca.issued = append(ca.issued, IssuedCert{
		Name:     name,
		Serial:   serial,
		IssuedAt: time.Now(),
	})
	ca.mu.Unlock()

	return pfxData, password, nil
}

// IssuedCerts returns a snapshot of all issued client certificates.
func (ca *ClientCA) IssuedCerts() []IssuedCert {
	ca.mu.RLock()
	defer ca.mu.RUnlock()
	result := make([]IssuedCert, len(ca.issued))
	copy(result, ca.issued)
	return result
}
