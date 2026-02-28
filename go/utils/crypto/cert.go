package crypto

import (
	"crypto/x509"
	"encoding/pem"
	"os"
	"strings"
)

type Certificate interface {
	PEM() string
	ASN1() []byte
	OBJ() *x509.Certificate
}

type CertificateChain []Certificate

func (chain CertificateChain) PEM() string {
	var result strings.Builder
	for _, cert := range chain {
		result.WriteString(cert.PEM())
	}
	return result.String()
}

func (chain CertificateChain) ASN1() [][]byte {
	result := make([][]byte, len(chain))
	for i, cert := range chain {
		result[i] = cert.ASN1()
	}
	return result
}

type certificate struct {
	pem string
	obj *x509.Certificate
}

func (cert *certificate) PEM() string {
	if cert.pem == "" {
		cert.pem = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE",
			Bytes: cert.ASN1()}))
	}
	return cert.pem
}

func (cert *certificate) ASN1() []byte {
	return cert.obj.Raw
}

func (cert *certificate) OBJ() *x509.Certificate {
	return cert.obj
}

func NewCertificateFromASN1(asn1 []byte) (Certificate, error) {
	obj, err := x509.ParseCertificate(asn1)
	if err != nil {
		return nil, err
	}
	return &certificate{obj: obj}, nil
}

func NewCertificateFromPEM(data string) (Certificate, error) {
	block, _ := pem.Decode([]byte(data))
	obj, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	return &certificate{pem: data, obj: obj}, nil
}

func GetCertificate(certFile string) (Certificate, error) {
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	return NewCertificateFromPEM(string(data))
}

func GetCertificateChain(certFile string) (CertificateChain, error) {
	result := make(CertificateChain, 0)
	data, err := os.ReadFile(certFile)
	if err != nil {
		return nil, err
	}
	for {
		var block *pem.Block
		block, data = pem.Decode([]byte(data))
		if block == nil {
			return result, nil
		}
		obj, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		result = append(result, &certificate{obj: obj})
	}
}
