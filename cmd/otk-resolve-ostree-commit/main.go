package main

// All otk external inputs are nested under a top-level "tree"
type Tree struct {
	Tree Input `json:"tree"`
}

// Input represents the user-provided inputs that will be used to resolve an
// ostree commit ID.
type Input struct {
	// URL of the repo where the commit can be fetched.
	URL string `json:"url"`

	// Ref to resolve.
	Ref string `json:"ref"`

	// Whether to use RHSM secrets when resolving and fetching the commit.
	RHSM bool `json:"rhsm,omitempty"`
}

// Output contains everything needed to write a manifest that requires pulling
// an ostree commit.
type Output struct {
	Const OutputConst `json:"const"`
}

type OutputConst struct {
	// Ref of the commit (can be empty).
	Ref string `json:"ref,omitempty"`

	// URL of the repo where the commit can be fetched.
	URL string `json:"url"`

	// Secrets type to use when pulling the ostree commit content
	// (e.g. org.osbuild.rhsm.consumer).
	Secrets string `json:"secrets,omitempty"`

	// Checksum of the commit.
	Checksum string `json:"checksum"`
}

func main() {

}
