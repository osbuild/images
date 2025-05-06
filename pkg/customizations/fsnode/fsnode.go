package fsnode

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

const usernameRegex = `^[A-Za-z0-9_.][A-Za-z0-9_.-]{0,31}$`
const groupnameRegex = `^[A-Za-z0-9_][A-Za-z0-9_-]{0,31}$`

func validate(path string, mode *os.FileMode, user any, group any) error {
	// Check that the path is valid
	if path == "" {
		return fmt.Errorf("path must not be empty")
	}
	if path[0] != '/' {
		return fmt.Errorf("path must be absolute")
	}
	if path[len(path)-1] == '/' {
		return fmt.Errorf("path must not end with a slash")
	}
	if path != filepath.Clean(path) {
		return fmt.Errorf("path must be canonical")
	}

	// Check that the mode is valid
	if mode != nil && *mode&os.ModeType != 0 {
		return fmt.Errorf("mode must not contain file type bits")
	}

	// Check that the user and group are valid
	switch user := user.(type) {
	case string:
		nameRegex := regexp.MustCompile(usernameRegex)
		if !nameRegex.MatchString(user) {
			return fmt.Errorf("user name %q doesn't conform to validating regex (%s)", user, nameRegex.String())
		}
	case float64:
		if user != float64(int64(user)) {
			return fmt.Errorf("user ID must be int")
		}
		if user < 0 {
			return fmt.Errorf("user ID must be non-negative")
		}
	case int:
		if user < 0 {
			return fmt.Errorf("user ID must be non-negative")
		}
	case int64:
		if user < 0 {
			return fmt.Errorf("user ID must be non-negative")
		}
	case nil:
		// user is not set
	default:
		return fmt.Errorf("user must be either a string or an int64, got %T", user)
	}

	switch group := group.(type) {
	case string:
		nameRegex := regexp.MustCompile(groupnameRegex)
		if !nameRegex.MatchString(group) {
			return fmt.Errorf("group name %q doesn't conform to validating regex (%s)", group, nameRegex.String())
		}
	case float64:
		if group != float64(int64(group)) {
			return fmt.Errorf("group ID must be int")
		}
		if group < 0 {
			return fmt.Errorf("group ID must be non-negative")
		}
	case int:
		if group < 0 {
			return fmt.Errorf("group ID must be non-negative")
		}
	case int64:
		if group < 0 {
			return fmt.Errorf("group ID must be non-negative")
		}
	case nil:
		// group is not set
	default:
		return fmt.Errorf("group must be either a string or an int64, got %T", group)
	}

	return nil
}
