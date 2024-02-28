package distro

import (
	"fmt"
	"reflect"
	"slices"
	"strings"

	"github.com/osbuild/blueprint/pkg/blueprint"
)

type ImageTypeValidator interface {
	// A list of customization options that this image requires.
	RequiredBlueprintOptions() []string

	// A list of customization options that this image supports.
	SupportedBlueprintOptions() []string
}

type validationError struct {
	// Reverse path to the customization that caused the error.
	revPath []string
	message string
}

func (e validationError) Error() string {
	path := e.revPath
	slices.Reverse(path)
	return fmt.Sprintf("%s: %s", strings.Join(path, "."), e.message)
}

func validateSupportedConfig(supported []string, conf reflect.Value) *validationError {

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
	for fieldIdx := 0; fieldIdx < confT.NumField(); fieldIdx++ {
		field := confT.Field(fieldIdx)
		if field.Anonymous {
			// embedded struct: flatten with the parent
			if err := validateSupportedConfig(supported, conf.Field(fieldIdx)); err != nil {
				return err
			}
			continue
		}
		tag := field.Tag.Get("json")
		tag = strings.Split(tag, ",")[0] // strip things like omitempty
		if subList, listed := subMap[tag]; listed {
			subStruct := conf.Field(fieldIdx)
			if subStruct.IsZero() {
				// nothing to validate: continue
				continue
			}
			if subStruct.Kind() == reflect.Ptr {
				// dereference pointer before validating
				subStruct = subStruct.Elem()
			}

			switch subStruct.Kind() {
			case reflect.Slice:
				// iterate over slice and validate each element as a substructure
				for sliceIdx := 0; sliceIdx < subStruct.Len(); sliceIdx++ {
					if err := validateSupportedConfig(subList, subStruct.Index(sliceIdx)); err != nil {
						err.revPath = append(err.revPath, fmt.Sprintf("%s[%d]", tag, sliceIdx))
						return err
					}
				}
			case reflect.Bool, reflect.String, reflect.Int, reflect.Struct:
				// single element
				if err := validateSupportedConfig(subList, subStruct); err != nil {
					err.revPath = append(err.revPath, tag)
					return err
				}
			default:
				return &validationError{message: fmt.Sprintf("internal error: unexpected field type found %v: %v", subStruct.Kind(), subStruct)}
			}
		} else {
			// not listed: check if it's non-zero
			empty := conf.Field(fieldIdx).IsZero()
			if !empty && !supportedMap[tag] {
				return &validationError{message: "not supported", revPath: []string{tag}}
			}
		}
	}

	return nil
}

func ValidateConfig(t ImageTypeValidator, bp blueprint.Blueprint) error {
	bpv := reflect.ValueOf(bp)
	if err := validateSupportedConfig(t.SupportedBlueprintOptions(), bpv); err != nil {
		return err
	}

	// explicitly return nil when there is no error, otherwise the error type
	// will be validationError instead of nil
	// https://go.dev/doc/faq#nil_error
	return nil
}
