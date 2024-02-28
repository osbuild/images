package distro

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
)

func validateSupportedConfig(supported []string, conf reflect.Value) error {

	// Construct two maps:
	//  - subMap represents the keys on the current level of the recursion that
	//  have sub-keys in the list of supported customizations:
	//  - supportedMap represents the keys on the current level that are fully
	//  supported.
	//
	// For example, for the following customizations
	//   customizations.kernel.name
	//   customizations.locale
	//
	// subMap will be
	//   {"customizations": ["kernel.name", "locale"]}
	//
	// When the function is then recursively called with just the "locale"
	// element, supportedMap will be
	//   {"locale": true}

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
		tag := confT.Field(i).Tag.Get("json")
		tag = strings.Split(tag, ",")[0] // strip things like omitempty
		if subList, listed := subMap[tag]; listed {
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
					if err := validateSupportedConfig(subList, subStruct.Index(idx)); err != nil {
						cerr := err.(*blueprint.BlueprintError)
						cerr.RevPath = append(cerr.RevPath, fmt.Sprintf("%s[%d]", tag, idx))
						return cerr
					}
				}
			} else {
				// single element
				if err := validateSupportedConfig(subList, subStruct); err != nil {
					cerr := err.(*blueprint.BlueprintError)
					cerr.RevPath = append(cerr.RevPath, tag)
					return cerr
				}
			}
		} else {
			// not listed: check if it's non-zero
			empty := conf.Field(i).IsZero()
			if !empty && !supportedMap[tag] {
				return &blueprint.BlueprintError{Message: "not supported", RevPath: []string{tag}}
			}
		}
	}

	return nil
}

func ValidateConfig(t ImageType, bp blueprint.Blueprint) error {
	bpv := reflect.ValueOf(bp)
	return validateSupportedConfig(t.SupportedBlueprintOptions(), bpv)
}
