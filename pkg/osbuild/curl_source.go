package osbuild

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	"github.com/osbuild/images/pkg/remotefile"
	"github.com/osbuild/images/pkg/rpmmd"
)

const SourceNameCurl = "org.osbuild.curl"

var curlDigestPattern = regexp.MustCompile(`(md5|sha1|sha256|sha384|sha512):[0-9a-f]{32,128}`)

type CurlSource struct {
	Items map[string]CurlSourceItem `json:"items"`
}

func (CurlSource) isSource() {}

// CurlSourceItem can be either a URL string or a URL paired with a secrets
// provider
type CurlSourceItem interface {
	isCurlSourceItem()
}

func NewCurlSource() *CurlSource {
	return &CurlSource{
		Items: make(map[string]CurlSourceItem),
	}
}

func NewCurlPackageItem(pkg rpmmd.Package) (CurlSourceItem, error) {
	if !curlDigestPattern.MatchString(pkg.Checksum.String()) {
		return nil, fmt.Errorf("curl package source item with name %q has invalid digest %q", pkg.Name, pkg.Checksum)
	}
	if len(pkg.RemoteLocations) == 0 {
		return nil, fmt.Errorf("curl source: package %q has no remote locations", pkg.Name)
	}
	item := new(CurlSourceOptions)
	item.URL = pkg.RemoteLocations[0]
	switch pkg.Secrets {
	case "org.osbuild.rhsm":
		item.Secrets = &URLSecrets{
			Name: "org.osbuild.rhsm",
		}
	case "org.osbuild.mtls":
		item.Secrets = &URLSecrets{
			Name: "org.osbuild.mtls",
		}
	}
	item.Insecure = pkg.IgnoreSSL
	return item, nil
}

// AddPackage adds a pkg to the curl source to download. Will return an error
// if any of the supplied options are invalid or missing.
func (source *CurlSource) AddPackage(pkg rpmmd.Package) error {
	item, err := NewCurlPackageItem(pkg)
	if err != nil {
		return err
	}
	source.Items[pkg.Checksum.String()] = item
	return nil
}

type URL string

func (URL) isCurlSourceItem() {}

type CurlSourceOptions struct {
	URL      string      `json:"url"`
	Secrets  *URLSecrets `json:"secrets,omitempty"`
	Insecure bool        `json:"insecure,omitempty"`
}

func (CurlSourceOptions) isCurlSourceItem() {}

type URLSecrets struct {
	Name string `json:"name"`
}

var resolveDoer remotefile.Doer = &http.Client{}

// ResolveAddURLs downloads each URL via the remotefile package, computes the
// checksum, and adds a new item to the source.
func (source *CurlSource) ResolveAddURLs(ctx context.Context, urls ...string) error {
	if len(urls) == 0 {
		return nil
	}

	resolver := remotefile.NewResolver(ctx, remotefile.WithDoer(resolveDoer))
	resolver.Add(urls...)
	specs, err := resolver.Finish()
	if err != nil {
		return err
	}

	for _, spec := range specs {
		sum := sha256.Sum256(spec.Content)
		checksum := "sha256:" + hex.EncodeToString(sum[:])
		source.Items[checksum] = URL(spec.URL)
	}

	return nil
}

// Unmarshal method for CurlSource for handling the CurlSourceItem interface:
// Tries each of the implementations until it finds the one that works.
func (cs *CurlSource) UnmarshalJSON(data []byte) (err error) {
	cs.Items = make(map[string]CurlSourceItem)
	type csSimple struct {
		Items map[string]URL `json:"items"`
	}
	simple := new(csSimple)
	b := bytes.NewReader(data)
	dec := json.NewDecoder(b)
	dec.DisallowUnknownFields()
	if err = dec.Decode(simple); err == nil {
		for k, v := range simple.Items {
			cs.Items[k] = v
		}
		return
	}

	type csWithSecrets struct {
		Items map[string]CurlSourceOptions `json:"items"`
	}
	withSecrets := new(csWithSecrets)
	b.Reset(data)
	if err = dec.Decode(withSecrets); err == nil {
		for k, v := range withSecrets.Items {
			cs.Items[k] = v
		}
		return
	}

	return
}
