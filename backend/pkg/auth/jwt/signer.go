package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"os"
)

// LoadPrivateKey loads an RSA private key. Path takes precedence over base64 PEM.
func LoadPrivateKey(path, pemBase64 string) (*rsa.PrivateKey, error) {
	data, err := loadPEMBytes(path, pemBase64)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block for private key")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		// fallback: PKCS1
		return x509.ParsePKCS1PrivateKey(block.Bytes)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not RSA")
	}
	return rsaKey, nil
}

// LoadPublicKey loads an RSA public key. Path takes precedence over base64 PEM.
func LoadPublicKey(path, pemBase64 string) (*rsa.PublicKey, error) {
	data, err := loadPEMBytes(path, pemBase64)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to decode PEM block for public key")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("public key is not RSA")
	}
	return rsaPub, nil
}

func loadPEMBytes(path, b64 string) ([]byte, error) {
	if path != "" {
		return os.ReadFile(path)
	}
	if b64 != "" {
		return base64.StdEncoding.DecodeString(b64)
	}
	return nil, errors.New("neither key path nor PEM base64 provided")
}
