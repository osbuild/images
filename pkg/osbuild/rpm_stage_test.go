package osbuild

import (
	"fmt"
	"testing"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRPMStageOptionsClone(t *testing.T) {
	tests := []struct {
		name string
		opts *RPMStageOptions
	}{
		{
			name: "nil",
			opts: nil,
		},
		{
			name: "empty",
			opts: &RPMStageOptions{},
		},
		{
			name: "all-fields",
			opts: &RPMStageOptions{
				DBPath:           "/var/lib/rpm",
				GPGKeys:          []string{"key1", "key2"},
				GPGKeysFromTree:  []string{"/etc/pki/rpm-gpg/RPM-GPG-KEY-fedora"},
				DisableDracut:    true,
				Exclude:          &Exclude{Docs: true},
				OSTreeBooted:     common.ToPtr(true),
				KernelInstallEnv: &KernelInstallEnv{BootRoot: "/boot"},
				InstallLangs:     []string{"en_US", "de_DE"},
			},
		},
		{
			name: "only-slices",
			opts: &RPMStageOptions{
				GPGKeys:         []string{"single-key"},
				GPGKeysFromTree: []string{"/path/to/key"},
				InstallLangs:    []string{"en_US"},
			},
		},
		{
			name: "only-pointers",
			opts: &RPMStageOptions{
				Exclude:          &Exclude{Docs: false},
				OSTreeBooted:     common.ToPtr(false),
				KernelInstallEnv: &KernelInstallEnv{BootRoot: ""},
			},
		},
		{
			name: "empty-slices",
			opts: &RPMStageOptions{
				GPGKeys:         []string{},
				GPGKeysFromTree: []string{},
				InstallLangs:    []string{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clone := tt.opts.Clone()
			assert.Equal(t, tt.opts, clone)

			if tt.opts == nil {
				assert.Nil(t, clone)
				return
			}

			assert.NotSame(t, tt.opts, clone)

			// Verify deep copy of slices (modifying clone shouldn't affect original)
			if len(clone.GPGKeys) > 0 {
				clone.GPGKeys[0] = "modified"
				assert.NotEqual(t, tt.opts.GPGKeys[0], clone.GPGKeys[0])
			}
			if len(clone.GPGKeysFromTree) > 0 {
				clone.GPGKeysFromTree[0] = "modified"
				assert.NotEqual(t, tt.opts.GPGKeysFromTree[0], clone.GPGKeysFromTree[0])
			}
			if len(clone.InstallLangs) > 0 {
				clone.InstallLangs[0] = "modified"
				assert.NotEqual(t, tt.opts.InstallLangs[0], clone.InstallLangs[0])
			}

			// Verify deep copy of pointer fields
			if clone.Exclude != nil {
				assert.NotSame(t, tt.opts.Exclude, clone.Exclude)
				clone.Exclude.Docs = !clone.Exclude.Docs
				assert.NotEqual(t, tt.opts.Exclude.Docs, clone.Exclude.Docs)
			}
			if clone.OSTreeBooted != nil {
				assert.NotSame(t, tt.opts.OSTreeBooted, clone.OSTreeBooted)
			}
			if clone.KernelInstallEnv != nil {
				assert.NotSame(t, tt.opts.KernelInstallEnv, clone.KernelInstallEnv)
				clone.KernelInstallEnv.BootRoot = "modified"
				assert.NotEqual(t, tt.opts.KernelInstallEnv.BootRoot, clone.KernelInstallEnv.BootRoot)
			}
		})
	}
}

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

func TestGPGKeysForPackages(t *testing.T) {
	// Define key values as variables for reuse
	key1 := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nkey1\n-----END PGP PUBLIC KEY BLOCK-----"
	key2 := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nkey2\n-----END PGP PUBLIC KEY BLOCK-----"
	keyA := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nkeyA\n-----END PGP PUBLIC KEY BLOCK-----"
	keyB := "-----BEGIN PGP PUBLIC KEY BLOCK-----\nkeyB\n-----END PGP PUBLIC KEY BLOCK-----"

	repoWithKey1 := &rpmmd.RepoConfig{GPGKeys: []string{key1}}
	repoWithKey2 := &rpmmd.RepoConfig{GPGKeys: []string{key2}}
	repoWithMultipleKeys := &rpmmd.RepoConfig{GPGKeys: []string{keyA, keyB}}
	repoNoKeys := &rpmmd.RepoConfig{GPGKeys: nil}

	tests := map[string]struct {
		pkgs        rpmmd.PackageList
		expected    []string
		expectError string
	}{
		"empty-package-list": {
			pkgs:     rpmmd.PackageList{},
			expected: nil,
		},
		"repo-without-gpg-keys-checkgpg-false": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoNoKeys, CheckGPG: false},
			},
			expected: nil,
		},
		// NOTE: for now we collect keys even for packages/repos with CheckGPG=false.
		"single-package-with-keys": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoWithKey1},
			},
			expected: []string{key1},
		},
		"multiple-packages-same-repo-deduplicated": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoWithKey1, CheckGPG: true},
				{Name: "pkg2", Repo: repoWithKey1, CheckGPG: true},
				{Name: "pkg3", Repo: repoWithKey1, CheckGPG: true},
			},
			expected: []string{key1},
		},
		"multiple-packages-different-repos": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoWithKey1, CheckGPG: true},
				{Name: "pkg3", Repo: repoNoKeys, CheckGPG: false},
				{Name: "pkg2", Repo: repoWithKey2, CheckGPG: true},
			},
			expected: []string{key1, key2},
		},
		"repo-with-multiple-keys": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoWithMultipleKeys, CheckGPG: true},
			},
			expected: []string{keyA, keyB},
		},
		// Error cases
		"error-checkgpg-true-no-keys": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoNoKeys, CheckGPG: true},
			},
			expectError: fmt.Sprintf(
				"package \"pkg1\" requires GPG check but repo %q has no GPG keys configured", repoNoKeys.Id),
		},
		"error-nil-repo-among-valid": {
			pkgs: rpmmd.PackageList{
				{Name: "pkg1", Repo: repoWithKey1},
				{Name: "pkg2", Repo: nil},
			},
			expectError: "package \"pkg2\" has nil Repo pointer. This is a bug in depsolving.",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			result, err := GPGKeysForPackages(tc.pkgs)

			if tc.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.expectError)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tc.expected, result)
		})
	}
}
