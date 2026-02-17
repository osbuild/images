package osbuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/osbuild/images/internal/test"
	"github.com/osbuild/images/pkg/remotefile"
	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDoer is a remotefile.Doer that returns predefined bodies and status codes per URL.
type mockDoer struct {
	responses map[string]struct {
		body   []byte
		status int
	}
}

func (m *mockDoer) Do(req *http.Request) (*http.Response, error) {
	r, ok := m.responses[req.URL.String()]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}
	return &http.Response{
		StatusCode: r.status,
		Body:       io.NopCloser(bytes.NewReader(r.body)),
	}, nil
}

func TestPackageSourceValidation(t *testing.T) {
	assert := assert.New(t)

	type testCase struct {
		pkg   rpmmd.Package
		valid bool
	}

	cases := []testCase{
		{
			pkg: rpmmd.Package{
				Name:            "openssl-libs",
				Epoch:           1,
				Version:         "3.0.1",
				Release:         "5.el9",
				Arch:            "x86_64",
				RemoteLocations: []string{"https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm"},
				Checksum:        rpmmd.Checksum{Type: "invalid", Value: "fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666"},
				Secrets:         "",
				CheckGPG:        false,
				IgnoreSSL:       true,
			},
			valid: false,
		},
		{
			pkg: rpmmd.Package{
				Name:            "openssl-whatever",
				Epoch:           1,
				Version:         "3.0.1",
				Release:         "5.el9",
				Arch:            "x86_64",
				RemoteLocations: []string{"https://example.com/repo/Packages/openssl-libs-3.0.1-5.el9.x86_64.rpm"},
				Checksum:        rpmmd.Checksum{Type: "", Value: "fcf2515ec9115551c99d552da721803ecbca23b7ae5a974309975000e8bef666"},
				Secrets:         "",
				CheckGPG:        false,
				IgnoreSSL:       true,
			},
			valid: false,
		},
		{
			pkg: rpmmd.Package{
				Name:            "package-with-no-remote-location",
				Epoch:           0,
				Version:         "0.4.11",
				Release:         "7.el9",
				Arch:            "x86_64",
				RemoteLocations: []string{},
				Checksum:        rpmmd.Checksum{Type: "sha256", Value: "4be41142a5fb2b4cd6d812e126838cffa57b7c84e5a79d65f66bb9cf1d2830a3"},
				Secrets:         "",
				CheckGPG:        false,
				IgnoreSSL:       true,
			},
			valid: false,
		},
		{
			pkg: rpmmd.Package{
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
			valid: true,
		},
		{
			pkg: rpmmd.Package{
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
			valid: true,
		},
		{
			pkg: rpmmd.Package{
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
			valid: true,
		},
		{
			pkg: rpmmd.Package{
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

func TestResolveAddURLs(t *testing.T) {
	type response struct {
		body   []byte
		status int
	}
	type wantItem struct {
		body []byte
		url  string
	}
	cases := []struct {
		name      string
		responses map[string]response
		urls      []string
		wantErr   error
		wantItems []wantItem
	}{
		{
			name:      "empty URLs",
			responses: map[string]response{},
			urls:      nil,
			wantErr:   nil,
			wantItems: nil,
		},
		{
			name: "single URL",
			responses: map[string]response{
				"https://example.com/key1": {body: []byte("key1\n"), status: http.StatusOK},
			},
			urls:      []string{"https://example.com/key1"},
			wantErr:   nil,
			wantItems: []wantItem{{body: []byte("key1\n"), url: "https://example.com/key1"}},
		},
		{
			name: "multiple URLs",
			responses: map[string]response{
				"https://example.com/key1": {body: []byte("key1\n"), status: http.StatusOK},
				"https://example.com/key2": {body: []byte("key2\n"), status: http.StatusOK},
			},
			urls:    []string{"https://example.com/key1", "https://example.com/key2"},
			wantErr: nil,
			wantItems: []wantItem{
				{body: []byte("key1\n"), url: "https://example.com/key1"},
				{body: []byte("key2\n"), url: "https://example.com/key2"},
			},
		},
		{
			name:      "resolve error",
			responses: map[string]response{},
			urls:      []string{"https://example.com/notfound"},
			wantErr:   errors.New("failed to resolve remote files"),
		},
		{
			name: "non-OK status",
			responses: map[string]response{
				"https://example.com/error": {body: []byte("error"), status: http.StatusInternalServerError},
			},
			urls:    []string{"https://example.com/error"},
			wantErr: errors.New("unexpected status 500"),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doer := &mockDoer{
				responses: make(map[string]struct {
					body   []byte
					status int
				}),
			}
			for u, r := range tc.responses {
				doer.responses[u] = struct {
					body   []byte
					status int
				}{body: r.body, status: r.status}
			}
			var doerIf remotefile.Doer = doer
			test.MockGlobal(t, &resolveDoer, doerIf)
			source := NewCurlSource()
			err := source.ResolveAddURLs(context.Background(), tc.urls...)
			if tc.wantErr != nil {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr.Error())
				assert.Empty(t, source.Items)
				return
			}
			require.NoError(t, err)
			require.Len(t, source.Items, len(tc.wantItems))
			for _, wi := range tc.wantItems {
				sum := sha256.Sum256(wi.body)
				checksum := "sha256:" + hex.EncodeToString(sum[:])
				item, ok := source.Items[checksum].(URL)
				require.True(t, ok, "item for %s", checksum)
				assert.Equal(t, wi.url, string(item))
			}
		})
	}
}
