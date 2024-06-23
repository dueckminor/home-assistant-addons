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
}

type privateKey struct {
	crypto.Signer
}

func GetPrivateKey(keyFile string) (PrivateKey, error) {
	data, err := os.ReadFile(keyFile)
	if err != nil {
		return nil, err
	}
	pemBlock, _ := pem.Decode(data)
	key, err := x509.ParsePKCS8PrivateKey(pemBlock.Bytes)
	if err != nil {
		return nil, err
	}
	return privateKey{
		Signer: key.(crypto.Signer),
	}, nil
}

func CreatePrivateKey(keyFile string) (PrivateKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	encodedKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, err
	}
	pemdata := pem.EncodeToMemory(
		&pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: encodedKey,
		},
	)
	err = os.WriteFile(keyFile, pemdata, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return privateKey{
		Signer: key,
	}, nil
}

func GetOrCreatePrivateKey(keyFile string) (PrivateKey, error) {
	signer, err := GetPrivateKey(keyFile)
	if err == nil || !os.IsNotExist(err) {
		return signer, err
	}
	return CreatePrivateKey(keyFile)
}
