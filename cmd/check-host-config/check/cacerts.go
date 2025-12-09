package check

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/osbuild/images/cmd/check-host-config/mockos"
	"github.com/osbuild/images/internal/buildconfig"
)

type CACertsCheck struct{}

func (c CACertsCheck) Metadata() Metadata {
	return Metadata{
		Name:                   "CA Certs Check",
		ShortName:              "cacerts",
		Timeout:                30 * time.Second,
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}
}

func (c CACertsCheck) Run(ctx context.Context, log Logger, config *buildconfig.BuildConfig) error {
	cacerts := config.Blueprint.Customizations.CACerts
	if cacerts == nil || len(cacerts.PEMCerts) == 0 {
		return Skip("no CA certs to check")
	}

	// Check all CA certs
	checkedCount := 0
	for i, pemCert := range cacerts.PEMCerts {
		if pemCert == "" {
			log.Printf("Skipping empty CA cert at index %d\n", i)
			continue
		}
		checkedCount++

		log.Printf("Parsing CA cert %d\n", i+1)
		block, _ := pem.Decode([]byte(pemCert))
		if block == nil {
			return Fail("failed to decode PEM certificate at index", fmt.Sprintf("%d", i))
		}

		if block.Type != "CERTIFICATE" {
			return Fail("PEM block is not a CERTIFICATE at index", fmt.Sprintf("%d", i), "got:", block.Type)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return Fail("failed to parse certificate at index", fmt.Sprintf("%d", i), "error:", err.Error())
		}

		// Extract serial number (format as hex, lowercase)
		serial := strings.ToLower(cert.SerialNumber.Text(16))
		log.Printf("Extracting serial from CA cert %d: %s\n", i+1, serial)

		// Extract CN from certificate subject
		cn := cert.Subject.CommonName
		if cn == "" {
			// Fallback: try to extract from Subject.String() if CommonName is empty
			// Subject.String() format: "CN=value,OU=...,O=..."
			subjectStr := cert.Subject.String()
			if idx := strings.Index(subjectStr, "CN="); idx != -1 {
				cnPart := subjectStr[idx+3:]
				// CN value might be followed by , or end of string
				if commaIdx := strings.Index(cnPart, ","); commaIdx != -1 {
					cn = cnPart[:commaIdx]
				} else {
					cn = cnPart
				}
				cn = strings.TrimSpace(cn)
			}
		}

		if cn == "" {
			return Fail("failed to extract CN from CA cert subject at index", fmt.Sprintf("%d", i))
		}

		log.Printf("Extracting CN from CA cert %d: %s\n", i+1, cn)

		// Check anchor file
		anchorPath := "/etc/pki/ca-trust/source/anchors/" + serial + ".pem"
		log.Printf("Checking CA cert %d anchor file serial '%s'\n", i+1, serial)
		if !mockos.ExistsContext(ctx, log, anchorPath) {
			return Fail("file missing for cert", fmt.Sprintf("%d", i+1), "at", anchorPath)
		}

		// Check extracted CA cert file
		log.Printf("Checking extracted CA cert %d file named '%s'\n", i+1, cn)
		found, err := mockos.GrepContext(ctx, log, cn, "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem")
		if err != nil {
			return Fail("extracted CA cert not found in the bundle for cert", fmt.Sprintf("%d", i+1), "cn:", cn, "error:", err.Error())
		}
		if !found {
			return Fail("extracted CA cert not found in the bundle for cert", fmt.Sprintf("%d", i+1), "cn:", cn)
		}
	}

	if checkedCount == 0 {
		return Skip("all CA certs are empty")
	}

	return Pass()
}
