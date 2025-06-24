package hashutil

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

// Sha256sum() is a convenience wrapper to generate
// the sha256 hex digest of a file. The hash is the
// same as from the sha256sum util.
func Sha256sum(uri string) (string, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return "", fmt.Errorf("invalid URI for sha256sum: %w", err)
	}

	var r io.ReadCloser
	switch u.Scheme {
	case "", "file":
		var err error
		r, err = os.Open(u.Path)
		if err != nil {
			return "", fmt.Errorf("sha256sum: %w", err)
		}
	case "http", "https":
		resp, err := http.Get(u.String())
		if err != nil {
			return "", fmt.Errorf("sha256sum cannot get %s: %w", uri, err)
		}
		if resp.StatusCode != http.StatusOK {
			return "", fmt.Errorf("sha256sum cannot fetch %s: %s", uri, resp.Status)
		}
		r = resp.Body
	default:
		return "", fmt.Errorf("sha256sum does not support scheme %q", u.Scheme)
	}
	defer r.Close()

	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
