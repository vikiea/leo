package cryptox

import (
	"crypto/x509"
	"encoding/pem"
)

// CertToPEMString taxes an x509 cert and returns a PEM encoded string version.
// Deprecated: Do not use. use github.com/go-leo/cryptox instead.
func CertToPEMString(cert *x509.Certificate) string {
	block := pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}
	return string(pem.EncodeToMemory(&block))
}
