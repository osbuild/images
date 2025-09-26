package osbuild

import (
	"testing"

	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRPMStage(t *testing.T) {
	expectedStage := &Stage{
		Type:    "org.osbuild.rpm",
		Options: &RPMStageOptions{},
		Inputs:  &RPMStageInputs{},
	}
	actualStage := NewRPMStage(&RPMStageOptions{}, &RPMStageInputs{})
	assert.Equal(t, expectedStage, actualStage)
}

func TestNewRpmStageSourceFilesInputs(t *testing.T) {

	assert := assert.New(t)
	require := require.New(t)

	pkgs := rpmmd.PackageList{
		{
			Name:            "openssl-libs",
			Epoch:           1,
			Version:         "3.0.1",
			Release:         "5.el9",
			Arch:            "x86_64",
			RemoteLocations: []string{"https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm"},
			Checksum:        rpmmd.Checksum{Type: "sha256", Value: "fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666"},
			Secrets:         "",
			CheckGPG:        false,
			IgnoreSSL:       true,
		},
		{
			Name:            "openssl-pkcs11",
			Epoch:           0,
			Version:         "0.4.11",
			Release:         "7.el9",
			Arch:            "x86_64",
			RemoteLocations: []string{"https://example.com/repo/Packages/openssl-pkcs11-0.4.11-7.el9.x86_64.rpm"},
			Checksum:        rpmmd.Checksum{Type: "sha256", Value: "4be41142a5fb2b4cd6d812e126838cffa57b7c84e5a79d65f66bb9cf1d2830a3"},
			Secrets:         "",
			CheckGPG:        false,
			IgnoreSSL:       true,
		},
		{
			Name:            "p11-kit",
			Epoch:           0,
			Version:         "0.24.1",
			Release:         "2.el9",
			Arch:            "x86_64",
			RemoteLocations: []string{"https://example.com/repo/Packages/p11-kit-0.24.1-2.el9.x86_64.rpm"},
			Checksum:        rpmmd.Checksum{Type: "sha256", Value: "da167e41efd19cf25fd1c708b6f123d0203824324b14dd32401d49f2aa0ef0a6"},
			Secrets:         "",
			CheckGPG:        false,
			IgnoreSSL:       true,
		},
		{
			Name:            "package-with-sha1-checksum",
			Epoch:           1,
			Version:         "3.4.2.",
			Release:         "10.el9",
			Arch:            "x86_64",
			RemoteLocations: []string{"https://example.com/repo/Packages/package-with-sha1-checksum-4.3.2-10.el9.x86_64.rpm"},
			Checksum:        rpmmd.Checksum{Type: "sha1", Value: "6e01b8076a2ab729d564048bf2e3a97c7ac83c13"},
			Secrets:         "",
			CheckGPG:        true,
			IgnoreSSL:       true,
		},
		{
			Name:            "package-with-md5-checksum",
			Epoch:           1,
			Version:         "3.4.2.",
			Release:         "5.el9",
			Arch:            "x86_64",
			RemoteLocations: []string{"https://example.com/repo/Packages/package-with-md5-checksum-4.3.2-5.el9.x86_64.rpm"},
			Checksum:        rpmmd.Checksum{Type: "md5", Value: "8133f479f38118c5f9facfe2a2d9a071"},
			Secrets:         "",
			CheckGPG:        true,
			IgnoreSSL:       true,
		},
	}
	inputs := NewRpmStageSourceFilesInputs(pkgs)

	refsArrayPtr, convOk := inputs.Packages.References.(*FilesInputSourceArrayRef)
	require.True(convOk)
	require.NotNil(refsArrayPtr)

	refsArray := *refsArrayPtr

	for idx := range refsArray {
		refItem := refsArray[idx]
		pkg := pkgs[idx]
		assert.Equal(pkg.Checksum.String(), refItem.ID)

		if pkg.CheckGPG {
			// GPG check enabled: metadata expected
			require.NotNil(refItem.Options)
			require.NotNil(refItem.Options.Metadata)

			md, convOk := refItem.Options.Metadata.(*RPMStageReferenceMetadata)
			require.True(convOk)
			require.NotNil(md)
			assert.Equal(md.CheckGPG, pkg.CheckGPG)
		}
	}
}
