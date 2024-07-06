package crypto

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

type PrivateKey interface {
	crypto.Signer
	PEM() string
	RSA() *rsa.PrivateKey
}

type PublicKey interface {
	crypto.PublicKey
	PEM() string
}

type privateKey struct {
	crypto.Signer
	pem string
}

type publicKey struct {
	crypto.PublicKey
	pem string
}

func (p *privateKey) PEM() string {
	return p.pem
}
func (p *privateKey) RSA() *rsa.PrivateKey {
	return p.Signer.(*rsa.PrivateKey)
}
func (p *publicKey) PEM() string {
	return p.pem
}

func (p *privateKey) PublicKey() PublicKey {
	return &publicKey{
		PublicKey: p.Signer.Public(),
	}
}

func (p *privateKey) encode() error {
	encodedKey, err := x509.MarshalPKCS8PrivateKey(p.Signer)
	if err != nil {
		return err
	}
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: encodedKey,
		},
	)
	p.pem = string(pemdata)
	return nil
}

func ParsePrivateKey(blob string) (PrivateKey, error) {
	pemBlock, _ := pem.Decode([]byte(blob))
	key, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return &privateKey{
		Signer: key.(crypto.Signer),
		pem:    string(blob),
	}, nil
}

func CreatePrivateKey() (PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	p := &privateKey{Signer: key}
	p.encode()
	return p, nil
}

func ReadPrivateKey(keyFile string) (PrivateKey, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	return ParsePrivateKey(string(data))
}

func CreatePrivateKeyFile(keyFile string) (PrivateKey, error) {
	p, err := CreatePrivateKey()
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(keyFile, []byte(p.PEM()), os.ModePerm)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func GetOrCreatePrivateKeyFile(keyFile string) (PrivateKey, error) {
	signer, err := ReadPrivateKey(keyFile)
	if err == nil || !os.IsNotExist(err) {
		return signer, err
	}
	return CreatePrivateKeyFile(keyFile)
}
