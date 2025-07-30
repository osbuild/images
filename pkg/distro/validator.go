package distro

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
)

func jsonTagFor(f reflect.StructField) string {
	tag := f.Tag.Get("json")
	return strings.Split(tag, ",")[0]
}

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

func fieldByTag(p reflect.Value, tag string) (reflect.Value, error) {
	for idx := 0; idx < p.Type().NumField(); idx++ {
		c := p.Type().Field(idx)
		if jsonTagFor(c) == tag {
			return p.Field(idx), nil
		}
	}

	return reflect.Value{}, fmt.Errorf("%s does not have a field with JSON tag %q", p.Type().Name(), tag)
}

func validateRequiredConfig(required []string, conf reflect.Value) error {
	// create two maps from the required list:
	//
	// 1. requiredMap contains the keys that must exist at this level as
	//    non-zero values. A key in this map can be the name of a substructure
	//    of the blueprint, like "Kernel", in which case that indicates that
	//    the "Kernel" section should be non-zero, regardless of which subparts
	//    of that structure are required or supported.
	//    This differs from the supportedMap in validateSupportedConfig() in
	//    that the requiredMap also lists keys that have required subparts,
	//    whether they are wholly required or not.
	//
	// 2. subMap contains the keys that have sub-parts that are required. Each
	//    substructure will have to be checked recursively until we reach
	//    required leaf nodes.

	requiredMap := make(map[string]bool)
	subMap := make(map[string][]string)
	for _, key := range required {
		if strings.Contains(key, ".") {
			// nested key: add to sub
			parts := strings.SplitN(key, ".", 2)
			subList := subMap[parts[0]]
			subList = append(subList, parts[1])
			subMap[parts[0]] = subList

			// if any subkey is required, then the top level one is as well
			requiredMap[parts[0]] = true
		} else {
			requiredMap[key] = true
		}
	}

	for key := range requiredMap {
		// requiredMap contains keys that are required at this level, whether
		// they have subkeys or not.
		// Their values should be non-zero but only for certain types:
		//   Struct, Pointer, Slice, and String
		// The Zero value for other types could be a valid value, so we
		// shouldn't assume that a zero value is the same as a missing one.
		value, err := fieldByTag(conf, key)
		if err != nil {
			return &blueprint.BlueprintError{Message: err.Error(), RevPath: []string{key}}
		}
		switch value.Kind() {
		case reflect.Ptr, reflect.Struct, reflect.String, reflect.Slice:
			// Required should only be used for Pointer, String, and Slice types.
			// For other types, the zero value can be valid and not indicate a
			// missing value.
			if value.IsZero() {
				return &blueprint.BlueprintError{Message: "required", RevPath: []string{key}}
			}
		}
	}

	for key := range subMap {
		// subMap contains keys that should contain specific subkeys.
		// If the key's value is Zero, that's an error, but that should be
		// caught by the requiredMap checks above.
		// If it's a Struct, descend into it.
		// If it's s Slice, descend into each element.
		value, err := fieldByTag(conf, key)
		if err != nil {
			return &blueprint.BlueprintError{Message: err.Error(), RevPath: []string{key}}
		}
		if value.Kind() == reflect.Ptr {
			// Dereference pointer before validating.
			// We don't need to worry about Zero values because of the previous
			// check in iteration through requiredMap above.
			value = value.Elem()
		}
		switch value.Kind() {
		case reflect.Struct:
			// Descend into map
			if err := validateRequiredConfig(subMap[key], value); err != nil {
				cerr := err.(*blueprint.BlueprintError)
				cerr.RevPath = append(cerr.RevPath, key)
				return cerr
			}
		case reflect.Slice:
			// iterate over slice and validate each element
			for idx := 0; idx < value.Len(); idx++ {
				if err := validateRequiredConfig(subMap[key], value.Index(idx)); err != nil {
					cerr := err.(*blueprint.BlueprintError)
					cerr.RevPath = append(cerr.RevPath, fmt.Sprintf("%s[%d]", key, idx))
					return cerr
				}
			}
		}

		if value.Kind() == reflect.Ptr {
			// if it's a pointer and it wasn't a Zero value, dereference and descend
			if err := validateRequiredConfig(subMap[key], value.Elem()); err != nil {
				cerr := err.(*blueprint.BlueprintError)
				cerr.RevPath = append(cerr.RevPath, key)
				return cerr
			}
		}
	}
	return nil
}

func ValidateConfig(t ImageType, bp blueprint.Blueprint) error {
	bpv := reflect.ValueOf(bp)
	if err := validateSupportedConfig(t.SupportedBlueprintOptions(), bpv); err != nil {
		return err
	}
	return validateRequiredConfig(t.RequiredBlueprintOptions(), bpv)
}
