package pki

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/dueckminor/home-assistant-addons/go/crypto"
)

type CA interface {
	IssueCertificate(csr crypto.CSR) (chain crypto.CertificateChain, err error)
}

type TLSServer interface {
	AddTLSConfig(sni string, tlsConfig *tls.Config)
}

type ServerCertificate interface {
	io.Closer
	SetTLSServer(tlsServer TLSServer)
	GetCertAndKey() (crypto.PrivateKey, crypto.CertificateChain)
}

type serverCertificate struct {
	cancel      func()
	tlsServer   TLSServer
	key         crypto.PrivateKey
	chain       crypto.CertificateChain
	keyFile     string
	certFile    string
	newKeyFile  string
	newCsrFile  string
	newCertFile string
	tlsConfig   *tls.Config
	issuer      CA
	dnsNames    []string
}

func (sc *serverCertificate) Close() error {
	sc.cancel()
	return nil
}

func (sc *serverCertificate) SetTLSServer(tlsServer TLSServer) {
	sc.tlsServer = tlsServer
	sc.tlsServer.AddTLSConfig(sc.dnsNames[0], sc.tlsConfig)
}

func (sc *serverCertificate) updateTLSServer() {
	if nil == sc.tlsServer || nil == sc.tlsConfig {
		return
	}
	for _, sni := range sc.dnsNames {
		sc.tlsServer.AddTLSConfig(sni, sc.tlsConfig)
	}
}

func (sc *serverCertificate) GetCertAndKey() (crypto.PrivateKey, crypto.CertificateChain) {
	return sc.key, sc.chain
}

func (sc *serverCertificate) refreshLoop(ctx context.Context) {
	for {
		err := sc.refreshLoopStep(ctx)
		if err != nil {
			fmt.Println(err)
			select {
			case <-ctx.Done():
				return
			case <-time.Tick(time.Second * 5):
				continue
			}
		}
	}
}

func (sc *serverCertificate) refreshLoopStep(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in refreshLoopStep: %v", r)
		}
	}()
	key, err := crypto.ReadPrivateKey(sc.keyFile)
	if os.IsNotExist(err) {
		sc.createCert()
		return nil
	}
	if err != nil {
		return err
	}

	chain, err := crypto.GetCertificateChain(sc.certFile)
	if err != nil {
		return err
	}

	sc.tlsConfig = &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: chain.ASN1(),
				PrivateKey:  key,
				Leaf:        chain[0].OBJ(),
			},
		},
	}

	sc.updateTLSServer()

	lifetime := chain[0].OBJ().NotAfter.Sub(chain[0].OBJ().NotBefore)
	used := time.Now().UTC().Sub(chain[0].OBJ().NotBefore)
	percentUsed := float64(used) * 100.0 / float64(lifetime)

	fmt.Printf("Certificate (%s) has used %.2f%% of its lifetime\n", sc.dnsNames[0], percentUsed)
	fmt.Println("Valid-NotBefore:", chain[0].OBJ().NotBefore)
	fmt.Println("Valid-NotAfter:", chain[0].OBJ().NotAfter)

	if percentUsed < 60.0 {
		fmt.Println("no need to update")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Hour * 24):
			return nil
		}
	}

	fmt.Println("trying to get a new certificate")
	sc.createCert()
	return nil
}

func (sc *serverCertificate) createCert() (err error) {
	key, err := crypto.GetOrCreatePrivateKeyFile(sc.newKeyFile)
	if err != nil {
		return err
	}

	csr, err := crypto.NewCSR(sc.newCsrFile, key, sc.dnsNames...)
	if err != nil {
		return err
	}
	err = os.WriteFile(sc.newCsrFile, []byte(csr.PEM()), os.ModePerm)
	if err != nil {
		return err
	}

	chain, err := sc.issuer.IssueCertificate(csr)
	if err != nil {
		return err
	}

	err = os.WriteFile(sc.newCertFile, []byte(chain.PEM()), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Rename(sc.newCertFile, sc.certFile)
	if err != nil {
		return err
	}
	err = os.Rename(sc.newKeyFile, sc.keyFile)
	if err != nil {
		return err
	}
	err = os.Remove(sc.newCsrFile)
	if err != nil {
		return err
	}

	return nil
}

func NewServerCertificate(filename string, issuer CA, dnsNames ...string) (sc ServerCertificate) {
	ctx, cancel := context.WithCancel(context.Background())

	result := &serverCertificate{
		cancel:      cancel,
		keyFile:     filename + ".key.pem",
		certFile:    filename + ".cert.pem",
		newKeyFile:  filename + ".new-key.pem",
		newCsrFile:  filename + ".new-csr.pem",
		newCertFile: filename + ".new-cert.pem",
		issuer:      issuer,
		dnsNames:    dnsNames,
	}

	go result.refreshLoop(ctx)

	return result
}
