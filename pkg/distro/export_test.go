package distro

import "reflect"

// We wrap our internal functions in exported functions instead of defining
// aliases so we can return an error type instead of validationError. Our
// recursive functions need to return validationError so that the path can be
// constructed when returning up the stack. But if the function at the top
// returns a nil valued validationError, it will fail the NoError() check. This
// is not a problem outside of testing since the public entrypoint,
// ValidateConfig(), returns nil when everything is ok.

func ValidateSupportedConfig(supported []string, conf reflect.Value) error {
	if err := validateSupportedConfig(supported, conf); err != nil {
		return err
	}
	return nil
}

func ValidateRequiredConfig(required []string, conf reflect.Value) error {
	if err := validateRequiredConfig(required, conf); err != nil {
		return err
	}
	return nil
}
