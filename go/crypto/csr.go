package crypto

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
)

type CSR interface {
	PEM() string
	ASN1() []byte
	OBJ() *x509.CertificateRequest
}

type csr struct {
	pem string
	obj *x509.CertificateRequest
}

func (csr *csr) PEM() string {
	if csr.pem == "" {
		csr.pem = string(pem.EncodeToMemory(&pem.Block{
			Type:  "CERTIFICATE REQUEST",
			Bytes: csr.ASN1()}))
	}
	return csr.pem
}

func (csr *csr) ASN1() []byte {
	return csr.obj.Raw
}

func (csr *csr) OBJ() *x509.CertificateRequest {
	return csr.obj
}

func NewCSR(csrFile string, privateKey PrivateKey, dnsNames ...string) (CSR, error) {
	csrBin, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.SHA256WithRSA,
		Subject:            pkix.Name{CommonName: dnsNames[0]},
		DNSNames:           dnsNames,
	}, privateKey)

	if err != nil {
		return nil, err
	}

	result := &csr{}

	result.obj, err = x509.ParseCertificateRequest(csrBin)
	if err != nil {
		return nil, err
	}

	return result, nil
}
