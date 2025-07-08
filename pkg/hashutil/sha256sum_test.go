package hashutil_test

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/osbuild/images/pkg/hashutil"
)

// test vectors from https://www.di-mgt.com.au/sha_testvectors.html
var testVectors = map[string]string{
	"":    "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
	"abc": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
	"abcdbcdecdefdefgefghfghighijhijkijkljklmklmnlmnomnopnopq": "248d6a61d20638b8e5c026930c3e6039a33ce45964ff2167f6ecedd419db06c1",
}

func TestSha256sumLocalFile(t *testing.T) {
	for inp, expected := range testVectors {
		t.Run(fmt.Sprintf("test-%s", inp), func(t *testing.T) {
			tmpf := filepath.Join(t.TempDir(), "inp.txt")
			err := os.WriteFile(tmpf, []byte(inp), 0644)
			assert.NoError(t, err)
			hash, err := hashutil.Sha256sum(tmpf)
			assert.NoError(t, err)
			assert.Equal(t, expected, hash)
		})
	}
}

func TestSha256sumHttp(t *testing.T) {
	ln, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	srv := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, err := w.Write([]byte(r.URL.Path[1.:]))
			assert.NoError(t, err)
		}),
	}
	defer srv.Close()
	go srv.Serve(ln)

	for inp, expected := range testVectors {
		t.Run(fmt.Sprintf("test-http-%s", inp), func(t *testing.T) {
			hash, err := hashutil.Sha256sum("http://" + ln.Addr().String() + "/" + inp)
			assert.NoError(t, err)
			assert.Equal(t, expected, hash)
		})
	}
}

func TestSha256sumErrorLocal(t *testing.T) {
	_, err := hashutil.Sha256sum("non-existing-file")
	assert.EqualError(t, err, `sha256sum: open non-existing-file: no such file or directory`)
}

func TestSha256sumErrorHttp(t *testing.T) {
	_, err := hashutil.Sha256sum("http://example.com/non-existing-file")
	assert.EqualError(t, err, `sha256sum cannot fetch http://example.com/non-existing-file: 404 Not Found`)
}

func TestSha256sumErrorScheme(t *testing.T) {
	_, err := hashutil.Sha256sum("unknown://example.com/file")
	assert.EqualError(t, err, `sha256sum does not support scheme "unknown"`)
}
