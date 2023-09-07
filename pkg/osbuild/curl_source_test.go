package osbuild

import (
	"testing"

	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
)

func TestPackageSourceValidation(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		pkg   rpmmd.PackageSpec
		valid bool
	}

	cases := []testCase{
		{
			pkg: rpmmd.PackageSpec{
				Name:           "openssl-libs",
				Epoch:          1,
				Version:        "3.0.1",
				Release:        "5.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
				Checksum:       "invalid:fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666",
				Secrets:        "",
				CheckGPG:       false,
				IgnoreSSL:      true,
			},
			valid: false,
		},
		{
			pkg: rpmmd.PackageSpec{
				Name:           "openssl-whatever",
				Epoch:          1,
				Version:        "3.0.1",
				Release:        "5.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm",
				Checksum:       "fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666",
				Secrets:        "",
				CheckGPG:       false,
				IgnoreSSL:      true,
			},
			valid: false,
		},
		{
			pkg: rpmmd.PackageSpec{
				Name:           "openssl-pkcs11",
				Epoch:          0,
				Version:        "0.4.11",
				Release:        "7.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/openssl-pkcs11-0.4.11-7.el9.x86_64.rpm",
				Checksum:       "sha256:4be41142a5fb2b4cd6d812e126838cffa57b7c84e5a79d65f66bb9cf1d2830a3",
				Secrets:        "",
				CheckGPG:       false,
				IgnoreSSL:      true,
			},
			valid: true,
		},
		{
			pkg: rpmmd.PackageSpec{
				Name:           "p11-kit",
				Epoch:          0,
				Version:        "0.24.1",
				Release:        "2.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/p11-kit-0.24.1-2.el9.x86_64.rpm",
				Checksum:       "sha256:da167e41efd19cf25fd1c708b6f123d0203824324b14dd32401d49f2aa0ef0a6",
				Secrets:        "",
				CheckGPG:       false,
				IgnoreSSL:      true,
			},
			valid: true,
		},
		{
			pkg: rpmmd.PackageSpec{
				Name:           "package-with-sha1-checksum",
				Epoch:          1,
				Version:        "3.4.2.",
				Release:        "10.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/package-with-sha1-checksum-4.3.2-10.el9.x86_64.rpm",
				Checksum:       "sha1:6e01b8076a2ab729d564048bf2e3a97c7ac83c13",
				Secrets:        "",
				CheckGPG:       true,
				IgnoreSSL:      true,
			},
			valid: true,
		},
		{
			pkg: rpmmd.PackageSpec{
				Name:           "package-with-md5-checksum",
				Epoch:          1,
				Version:        "3.4.2.",
				Release:        "5.el9",
				Arch:           "x86_64",
				RemoteLocation: "https://example.com/repo/Packages/package-with-md5-checksum-4.3.2-5.el9.x86_64.rpm",
				Checksum:       "md5:8133f479f38118c5f9facfe2a2d9a071",
				Secrets:        "",
				CheckGPG:       true,
				IgnoreSSL:      true,
			},
			valid: true,
		},
	}

	curl := NewCurlSource()
	for _, tc := range cases {
		if tc.valid {
			assert.NoError(curl.AddPackage(tc.pkg))
		} else {
			assert.Error(curl.AddPackage(tc.pkg))
		}
	}
}
