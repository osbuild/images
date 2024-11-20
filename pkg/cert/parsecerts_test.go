package cert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// taken from osbuild:test/data/certs/cert1.pem
const exampleCert = `
-----BEGIN CERTIFICATE-----
MIIDhTCCAm2gAwIBAgIUVya7VJ3O8W8SqwuEa0BZ4HSsXvAwDQYJKoZIhvcNAQEL
BQAwUTELMAkGA1UEBhMCREUxDzANBgNVBAgMBkJlcmxpbjEPMA0GA1UEBwwGQmVy
bGluMQwwCgYDVQQKDANPcmcxEjAQBgNVBAMMCWxvY2FsaG9zdDAgFw0yNDA4MjYx
MDQyNDBaGA8yMTI0MDgwMjEwNDI0MFowUTELMAkGA1UEBhMCREUxDzANBgNVBAgM
BkJlcmxpbjEPMA0GA1UEBwwGQmVybGluMQwwCgYDVQQKDANPcmcxEjAQBgNVBAMM
CWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAJnGjlvN
O3F/Z7Lr/r+6Xp2DosnNwoPHhG2e61KnFzgZfaxbklal5ORpuV/gLIg7lrbpdZe7
WvK+16RanL6fLitis/tYVFyvz1MXqBYYrEoFGvVg9fOiis7hjpdZcpNDH9SngoAN
O0Wvv4T6LQS0cC7ZAFZjvmJ+RiZEbzRkNG5pUddZXbotE6htNfLgA5L1wIBgllrM
4DVkG0yNKmzqPNzfPTbdUgWCfjaQShHy1GP8KNEwFxM31F2wvQxsEb77o1S44Out
mlsi83tti6P7KjDk7w2j2zZO1X0xI8pflv3TBkJT1Am8vnk6rVnNO4pCpop3+kma
pDUEzBQmSQA5R1ECAwEAAaNTMFEwHQYDVR0OBBYEFDxFcFgPEsgsDixfKxB0uYGN
aJmzMB8GA1UdIwQYMBaAFDxFcFgPEsgsDixfKxB0uYGNaJmzMA8GA1UdEwEB/wQF
MAMBAf8wDQYJKoZIhvcNAQELBQADggEBAFih4lUbLlhKwIAV9x3/W7Mih8xUEdZr
olquZgaHedFet+ByAHvoES3pec7AVYTOD53mjgyZubD6INnVHzKyS4AG9ydD73o4
cmm3DKxBaesvlHeTn0MOKsoM8QCxeyFJmiUPpgDBok/PFnbGR9+JcsrlGJAnsSKD
vWpiwYcBauZ9nnK5yDe5M9XNFPkNDZzbKvWU7Sw3ziMT/+bRJse5vTrYcyOnNGgy
gZNz2nimKy1U8XZVAVwOV0rdGEFrfMln8DkRW86rGK/EncaVsl0SSP/rmjQgiX8Q
3CZraQGujJP932HSwUfdCX9yh+rTjE3MEnbqMoLzJa4BXB2aDQWtywU=
-----END CERTIFICATE-----
`

func TestParseCerts(t *testing.T) {
	certs, err := ParseCerts(exampleCert)
	assert.Nil(t, err)
	assert.Equal(t, len(certs), 1)
	assert.Equal(t, "localhost", certs[0].Subject.CommonName)
}

func TestParseConcatenatedCerts(t *testing.T) {
	certs, err := ParseCerts(exampleCert + "\n" + exampleCert)
	assert.Nil(t, err)
	assert.Equal(t, len(certs), 2)
	assert.Equal(t, "localhost", certs[0].Subject.CommonName)
}

func TestParseGarbageCerts(t *testing.T) {
	_, err := ParseCerts("garbage")
	assert.NotNil(t, err)
	assert.Equal(t, "no valid PEM certificates found in: garbage", err.Error())
}
