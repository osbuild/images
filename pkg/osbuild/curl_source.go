package osbuild

import (
	"bytes"
	"encoding/json"

	"github.com/osbuild/images/pkg/rpmmd"
)

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

func NewCurlPackageItem(pkg rpmmd.PackageSpec) CurlSourceItem {
	item := new(CurlSourceOptions)
	item.URL = pkg.RemoteLocation
	if pkg.Secrets == "org.osbuild.rhsm" {
		item.Secrets = &URLSecrets{
			Name: "org.osbuild.rhsm",
		}
	}
	item.Insecure = pkg.IgnoreSSL
	return item
}

// AddPackage adds a pkg to the curl source to download. Will panic if any of
// the supplied options are invalid or missing.
func (source *CurlSource) AddPackage(pkg rpmmd.PackageSpec) {
	item := NewCurlPackageItem(pkg)
	source.Items[pkg.Checksum] = item
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
