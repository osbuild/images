package cert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// ParseCerts parses a PEM-encoded certificate chain formatted as concatenated strings
// and returns a slice of x509.Certificate.
func ParseCerts(cert string) ([]*x509.Certificate, error) {
	result := make([]*x509.Certificate, 0, 1)
	block := []byte(cert)
	var blocks [][]byte
	for {
		var certDERBlock *pem.Block
		certDERBlock, block = pem.Decode(block)
		if certDERBlock == nil {
			break
		}

		if certDERBlock.Type == "CERTIFICATE" {
			blocks = append(blocks, certDERBlock.Bytes)
		}
	}

	for _, block := range blocks {
		cert, err := x509.ParseCertificate(block)
		if err != nil {
			return nil, fmt.Errorf("failed to parse certificate: %w", err)
		}
		result = append(result, cert)
	}

	return result, nil
}
