package main

import (
	"fmt"
	"os"

	"github.com/osbuild/images/pkg/ostree"
)

func main() {
	fmt.Println("Resolving ostree source, configuration:")
	fmt.Printf("CA: %s\n", os.Getenv("OSBUILD_COMPOSER_OSTREE_CA"))
	fmt.Printf("Client cert: %s\n", os.Getenv("OSBUILD_COMPOSER_OSTREE_CLIENT_CERT"))
	fmt.Printf("Client key: %s\n", os.Getenv("OSBUILD_COMPOSER_OSTREE_CLIENT_KEY"))

	spec := ostree.SourceSpec{
		URL: "https://builder.home.lan/ccb2194f-9876-4e76-9e64-a338a32df230/",
		Ref: "fedora/40/x86_64/iot",
	}
	cs, err := ostree.Resolve(spec)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Resolved checksum: %s", cs.Checksum)
}
