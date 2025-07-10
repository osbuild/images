package osbuild

import (
	"fmt"

	"github.com/osbuild/images/pkg/rpmmd"
)

const dnf4VersionlockType = "org.osbuild.dnf4.versionlock"

type DNF4VersionlockOptions struct {
	Add []string `json:"add"`
}

func (*DNF4VersionlockOptions) isStageOptions() {}

func (o *DNF4VersionlockOptions) validate() error {
	if len(o.Add) == 0 {
		return fmt.Errorf("%s: at least one package must be included in the 'add' list", dnf4VersionlockType)
	}

	return nil
}

func NewDNF4VersionlockStage(options *DNF4VersionlockOptions) *Stage {
	if err := options.validate(); err != nil {
		panic(err)
	}
	return &Stage{
		Type:    dnf4VersionlockType,
		Options: options,
	}
}

// GenDNF4VersionlockStageOptions creates DNF4VersionlockOptions for the provided
// packages at the specific EVR that is contained in the package spec list.
func GenDNF4VersionlockStageOptions(lockPackageNames []string, packageSpecs []rpmmd.PackageSpec) (*DNF4VersionlockOptions, error) {
	pkgNEVRs := make([]string, len(lockPackageNames))
	for idx, pkgName := range lockPackageNames {
		pkg, err := rpmmd.GetPackage(packageSpecs, pkgName)
		if err != nil {
			return nil, fmt.Errorf("%s: package %q not found in package list", dnf4VersionlockType, pkgName)
		}
		nevr := fmt.Sprintf("%s-%d:%s-%s", pkg.Name, pkg.Epoch, pkg.Version, pkg.Release)
		pkgNEVRs[idx] = nevr
	}

	return &DNF4VersionlockOptions{
		Add: pkgNEVRs,
	}, nil
}
