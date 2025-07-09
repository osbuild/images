package defs

import (
	"fmt"
	"regexp"

	"github.com/osbuild/images/pkg/distro"
)

// ParseID parse the given nameVer into a distro.ID. It will also
// apply any matching `transform_re`. This is needed to support distro
// names like "rhel-810" without dots.
//
// If no transformations are needed it will return "nil"
func ParseID(nameVer string) (*distro.ID, error) {
	distros, err := loadDistros()
	if err != nil {
		return nil, err
	}

	for _, d := range distros.Distros {
		re, err := regexp.Compile(d.TransformRE)
		if err != nil {
			return nil, err
		}
		if l := re.FindStringSubmatch(nameVer); len(l) == 4 {
			transformed := fmt.Sprintf("%s-%s.%s", l[re.SubexpIndex("name")], l[re.SubexpIndex("major")], l[re.SubexpIndex("minor")])
			return distro.ParseID(transformed)
		}
	}
	return nil, nil
}
