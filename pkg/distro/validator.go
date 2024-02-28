package distro

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
)

func validateSubConfig(supported []string, conf reflect.Value) error {
	// split supported list into two maps:
	//
	// 1. supportedMap contains the keys that can exist at this level as
	//    non-zero values. A key in this map can be the name of a large part of
	//    the blueprint, like "User", in which case that indicates that the
	//    whole substructure is supported.
	//
	// 2. subMap contains the keys that have sub-parts that are supported. Each
	//    substructure will have to be checked recursively until we reach leaf
	//    nodes.

	supportedMap := make(map[string]bool)
	subMap := make(map[string][]string)
	for _, key := range supported {
		if strings.Contains(key, ".") {
			// nested key: add top level component as key in subMap and the
			// rest as the value.
			parts := strings.SplitN(key, ".", 2)
			subList := subMap[parts[0]]
			subList = append(subList, parts[1])
			subMap[parts[0]] = subList
		} else {
			// leaf node in supported list: will be checked for non-zero value
			supportedMap[key] = true
		}
	}

	confT := conf.Type()
	for i := 0; i < confT.NumField(); i++ {
		name := confT.Field(i).Name
		if subList, listed := subMap[name]; listed {
			subStruct := conf.Field(i)
			if subStruct.IsZero() {
				// nothing to validate: continue
				continue
			}
			if subStruct.Kind() == reflect.Ptr {
				// dereference pointer before validating
				subStruct = subStruct.Elem()
			}
			if subStruct.Kind() == reflect.Slice {
				// iterate over slice and validate each element as a substructure
				for idx := 0; idx < subStruct.Len(); idx++ {
					if err := validateSubConfig(subList, subStruct.Index(idx)); err != nil {
						cerr := err.(*blueprint.CustomizationError)
						cerr.RevPath = append(cerr.RevPath, fmt.Sprintf("%s[%d]", name, idx))
						return cerr
					}
				}
			} else {
				// single element
				if err := validateSubConfig(subList, subStruct); err != nil {
					cerr := err.(*blueprint.CustomizationError)
					cerr.RevPath = append(cerr.RevPath, name)
					return cerr
				}
			}
		} else {
			// not listed: check if it's non-zero
			empty := conf.Field(i).IsZero()
			if !empty && !supportedMap[name] {
				return &blueprint.CustomizationError{Message: "not supported by image type", RevPath: []string{name}}
			}
		}
	}

	return nil
}

func ValidateConfig(t ImageType, bp blueprint.Blueprint, _ ImageOptions) error {
	bpv := reflect.ValueOf(bp)
	return validateSubConfig(t.SupportedCustomizations(), bpv)
}
