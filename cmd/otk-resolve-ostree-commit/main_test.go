package main_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	resolver "github.com/osbuild/images/cmd/otk-resolve-ostree-commit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var commitMap = map[string]string{
	"centos/9/x86_64/edge": "d04105393ca0617856b34f897842833d785522e41617e17dca2063bf97e294ef",
	"fake/ref/f":           "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	"fake/ref/9":           "9999999999999999999999999999999999999999999999999999999999999999",
	"test/ref/alpha":       "9b1ea9a8e10dc27d4ea40545bec028ad8e360dd26d18de64b0f6217833a8443d",
	"test/ref/one":         "7433e1b49fb136d61dcca49ebe34e713fdbb8e29bf328fe90819628f71b86105",
}

// Create a test server that responds with the commit ID that corresponds to
// the ref.
func createTestServer(refIDs map[string]string) *httptest.Server {
	handler := http.NewServeMux()
	handler.HandleFunc("/refs/heads/", func(w http.ResponseWriter, r *http.Request) {
		reqRef := strings.TrimPrefix(r.URL.Path, "/refs/heads/")
		id, ok := refIDs[reqRef]
		if !ok {
			http.NotFound(w, r)
			return
		}
		fmt.Fprint(w, id)
	})

	return httptest.NewServer(handler)
}

func TestResolver(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	repoServer := createTestServer(commitMap)
	defer repoServer.Close()

	url := repoServer.URL
	for ref, id := range commitMap {
		inputReq, err := json.Marshal(map[string]map[string]string{
			"tree": {
				"url": url,
				"ref": ref,
			},
		})
		require.NoError(err)

		inpBuf := bytes.NewBuffer(inputReq)
		outBuf := &bytes.Buffer{}

		assert.NoError(resolver.Run(inpBuf, outBuf))

		var output map[string]map[string]map[string]string
		require.NoError(json.Unmarshal(outBuf.Bytes(), &output))

		expOutput := map[string]map[string]map[string]string{
			"tree": {
				"const": {
					"url":      url,
					"ref":      ref,
					"checksum": id,
				},
			},
		}
		assert.Equal(expOutput, output)
	}
}

func TestResolverByID(t *testing.T) {
	require := require.New(t)
	assert := assert.New(t)

	for _, id := range commitMap {
		inputReq, err := json.Marshal(map[string]map[string]string{
			"tree": {
				"ref": id,
			},
		})
		require.NoError(err)

		inpBuf := bytes.NewBuffer(inputReq)
		outBuf := &bytes.Buffer{}

		assert.NoError(resolver.Run(inpBuf, outBuf))

		var output map[string]map[string]map[string]string
		require.NoError(json.Unmarshal(outBuf.Bytes(), &output))

		expOutput := map[string]map[string]map[string]string{
			"tree": {
				"const": {
					"ref":      id,
					"checksum": id,
					"url":      "",
				},
			},
		}
		assert.Equal(expOutput, output)
	}
}
func TestResolverIDwithURL(t *testing.T) {

	require := require.New(t)
	assert := assert.New(t)

	// the URL is not used when the ref is a commit ID, but it should be returned in the output
	url := "https://doesnt-matter.example.org"
	for _, id := range commitMap {
		inputReq, err := json.Marshal(map[string]map[string]string{
			"tree": {
				"ref": id,
				"url": url,
			},
		})
		require.NoError(err)

		inpBuf := bytes.NewBuffer(inputReq)
		outBuf := &bytes.Buffer{}

		assert.NoError(resolver.Run(inpBuf, outBuf))

		var output map[string]map[string]map[string]string
		require.NoError(json.Unmarshal(outBuf.Bytes(), &output))

		expOutput := map[string]map[string]map[string]string{
			"tree": {
				"const": {
					"ref":      id,
					"checksum": id,
					"url":      url,
				},
			},
		}
		assert.Equal(expOutput, output)
	}
}

func TestResolverErrors(t *testing.T) {

	repoServer := createTestServer(commitMap)
	defer repoServer.Close()

	type testCase struct {
		url          string
		ref          string
		errSubstring string
	}

	testCases := map[string]testCase{
		"bad-ref": {
			url:          "doesn't matter",
			ref:          "---",
			errSubstring: "Invalid ostree ref or commit",
		},
		"ref-not-found": {
			url:          repoServer.URL,
			ref:          "good/ref/but/does-not-exist",
			errSubstring: "returned status: 404 Not Found",
		},
	}

	for name := range testCases {
		tc := testCases[name]
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			inputReq, err := json.Marshal(map[string]map[string]string{
				"tree": {
					"url": tc.url,
					"ref": tc.ref,
				},
			})
			require.NoError(err)

			inpBuf := bytes.NewBuffer(inputReq)
			outBuf := &bytes.Buffer{}

			assert.ErrorContains(resolver.Run(inpBuf, outBuf), tc.errSubstring)
		})
	}
}
