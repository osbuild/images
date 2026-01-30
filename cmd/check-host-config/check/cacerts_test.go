package check_test

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/osbuild/blueprint/pkg/blueprint"
	check "github.com/osbuild/images/cmd/check-host-config/check"
	"github.com/osbuild/images/internal/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestCert creates a test X509 certificate and returns it as PEM
func generateTestCert(t *testing.T, cn string, serial *big.Int) string {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			CommonName: cn,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	require.NoError(t, err)

	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	return string(certPEM)
}

func TestCACertsCheck(t *testing.T) {
	// Generate a test certificate with a known serial number
	serial := big.NewInt(1234567890)
	cn := "Test CA Certificate"
	pemCert := generateTestCert(t, cn, serial)

	// Calculate expected serial (hex, lowercase)
	expectedSerial := strings.ToLower(serial.Text(16))

	test.MockGlobal(t, &check.Exists, func(name string) bool {
		// Check for anchor file
		if name == "/etc/pki/ca-trust/source/anchors/"+expectedSerial+".pem" {
			return true
		}
		return false
	})

	test.MockGlobal(t, &check.Grep, func(pattern, filename string) (bool, error) {
		// Mock grep to check if CN is in bundle
		if filename == "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem" && pattern == cn {
			return true, nil
		}
		return false, nil
	})

	chk, found := check.FindCheckByName("cacerts")
	require.True(t, found, "CA Certs Check not found")
	config := buildConfig(&blueprint.Customizations{
		CACerts: &blueprint.CACustomization{
			PEMCerts: []string{pemCert},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}

func TestCACertsCheckMultiple(t *testing.T) {
	// Generate two test certificates
	serial1 := big.NewInt(1111111111)
	cn1 := "First CA Certificate"
	pemCert1 := generateTestCert(t, cn1, serial1)
	expectedSerial1 := strings.ToLower(serial1.Text(16))

	serial2 := big.NewInt(2222222222)
	cn2 := "Second CA Certificate"
	pemCert2 := generateTestCert(t, cn2, serial2)
	expectedSerial2 := strings.ToLower(serial2.Text(16))

	test.MockGlobal(t, &check.Exists, func(name string) bool {
		// Check for both anchor files
		if name == "/etc/pki/ca-trust/source/anchors/"+expectedSerial1+".pem" {
			return true
		}
		if name == "/etc/pki/ca-trust/source/anchors/"+expectedSerial2+".pem" {
			return true
		}
		return false
	})

	test.MockGlobal(t, &check.Grep, func(pattern, filename string) (bool, error) {
		// Mock grep to check if CN is in bundle
		if filename == "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem" && (pattern == cn1 || pattern == cn2) {
			return true, nil
		}
		return false, nil
	})

	chk, found := check.FindCheckByName("cacerts")
	require.True(t, found, "CA Certs Check not found")
	config := buildConfig(&blueprint.Customizations{
		CACerts: &blueprint.CACustomization{
			PEMCerts: []string{pemCert1, pemCert2},
		},
	})

	require.NoError(t, chk.Func(chk.Meta, config))
}

func TestCACertsCheckSkip(t *testing.T) {
	chk, found := check.FindCheckByName("cacerts")
	require.True(t, found, "CA Certs Check not found")
	config := buildConfig(&blueprint.Customizations{
		CACerts: &blueprint.CACustomization{
			PEMCerts: []string{},
		},
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.True(t, check.IsSkip(err))
}

func TestCACertsCheckEmptyCert(t *testing.T) {
	chk, found := check.FindCheckByName("cacerts")
	require.True(t, found, "CA Certs Check not found")
	config := buildConfig(&blueprint.Customizations{
		CACerts: &blueprint.CACustomization{
			PEMCerts: []string{""},
		},
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.True(t, check.IsSkip(err))
}

func TestCACertsCheckMissingAnchor(t *testing.T) {
	serial := big.NewInt(9999999999)
	cn := "Missing Anchor Test"
	pemCert := generateTestCert(t, cn, serial)

	test.MockGlobal(t, &check.Exists, func(name string) bool {
		// Anchor file does not exist
		return false
	})

	test.MockGlobal(t, &check.Grep, func(pattern, filename string) (bool, error) {
		// Return false to simulate CN not found
		return false, nil
	})

	chk, found := check.FindCheckByName("cacerts")
	require.True(t, found, "CA Certs Check not found")
	config := buildConfig(&blueprint.Customizations{
		CACerts: &blueprint.CACustomization{
			PEMCerts: []string{pemCert},
		},
	})

	err := chk.Func(chk.Meta, config)
	require.Error(t, err)
	assert.False(t, check.IsSkip(err))
	assert.True(t, check.IsFail(err))
}
