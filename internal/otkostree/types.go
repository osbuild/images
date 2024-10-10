package otkostree

type ResolvedConst struct {
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
