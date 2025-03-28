package osbuild

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHMACStageValidation(t *testing.T) {
	type testCase struct {
		options *HMACStageOptions
		expErr  string
	}

	testCases := map[string]testCase{
		"good1": {
			options: &HMACStageOptions{
				Paths:     []string{"/greet1.txt", "/greet2.txt"},
				Algorithm: HMACSHA256,
			},
		},
		"good2": {
			options: &HMACStageOptions{
				Paths:     []string{"/path/to/some/file", "/boot/efi/EFI/Linux/vmlinuz-linux"},
				Algorithm: HMACSHA512,
			},
		},
		"nil": {},

		"nothing": {
			options: &HMACStageOptions{},
			expErr:  "'paths' is a required property",
		},
		"nopaths": {
			options: &HMACStageOptions{
				Algorithm: HMACSHA512,
			},
			expErr: "'paths' is a required property",
		},
		"noalgo": {
			options: &HMACStageOptions{
				Paths: []string{"/greet1.txt", "/greet2.txt"},
			},
			expErr: "'algorithm' is a required property",
		},
		"emptypaths": {
			options: &HMACStageOptions{
				Paths:     []string{},
				Algorithm: HMACSHA512,
			},
			expErr: "'paths' is a required property",
		},
		"badalgo": {
			options: &HMACStageOptions{
				Paths:     []string{"/path"},
				Algorithm: "md5",
			},
			expErr: "'md5' is not one of [sha1 sha224 sha256 sha384 sha512]",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			err := tc.options.validate()
			if expErr := tc.expErr; expErr == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, expErr)
			}
		})
	}
}
