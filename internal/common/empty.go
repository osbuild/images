package common

import (
	"fmt"
	"reflect"
)

// compare a struct to its zero (empty) value. note that if a struct contains
// maps or slices that an empty slice is different from a slice of length 0.
// the latter is *not* an empty value and thus a struct won't compare empty
func IsEmptyStruct(object interface{}) (bool, error) {
	kind := reflect.ValueOf(object).Kind()
	if kind != reflect.Struct {
		return false, fmt.Errorf("non struct argument %v", kind)
	}

	if reflect.ValueOf(object).IsZero() {
		return true, nil
	}

	return false, nil
}
