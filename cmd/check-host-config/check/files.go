package check

import (
	"fmt"
	"log"
	"strconv"

	"github.com/osbuild/images/internal/buildconfig"
)

func init() {
	RegisterCheck(Metadata{
		Name:                   "Files Check",
		ShortName:              "files",
		RequiresBlueprint:      true,
		RequiresCustomizations: true,
	}, filesCheck)
}

// resolveUserOrGroup converts an any value (string, int, or int64) to a uint32 UID/GID.
// If the value is a string, it looks up the user/group name. If it's numeric, it converts it directly.
// This function should only be called when value is not nil.
func resolveUserOrGroup(value any, isGroup bool) (uint32, error) {
	switch v := value.(type) {
	case string:
		if isGroup {
			return LookupGID(v)
		}
		return LookupUID(v)
	case int:
		if v < 0 || v > int(^uint32(0)) {
			return 0, fmt.Errorf("integer value %d out of range for uint32", v)
		}
		return uint32(v), nil
	case int64:
		if v < 0 || v > int64(^uint32(0)) {
			return 0, fmt.Errorf("integer value %d out of range for uint32", v)
		}
		return uint32(v), nil
	default:
		return 0, fmt.Errorf("unsupported type for user/group: %T (expected string, int, or int64)", value)
	}
}

func filesCheck(meta *Metadata, config *buildconfig.BuildConfig) error {
	expected := config.Blueprint.Customizations.Files

	if len(expected) == 0 {
		return Skip("no files to check")
	}

	for _, file := range expected {
		if !Exists(file.Path) {
			return Fail("file does not exist:", file.Path)
		}

		mode, uid, gid, err := FileInfo(file.Path)
		if err != nil {
			return Fail("failed to get file info:", file.Path)
		}

		if file.Mode != "" {
			userMode, err := strconv.ParseUint(file.Mode, 8, 32)
			if err != nil {
				return Fail("failed to parse file mode:", file.Path)
			}

			if int64(mode.Perm()) != int64(userMode) {
				return Fail("file mode does not match:", file.Path)
			}
		}

		if file.User != nil {
			expectedUid, err := resolveUserOrGroup(file.User, false)
			if err != nil {
				return Fail("failed to resolve user:", file.Path, err)
			}
			if uid != expectedUid {
				return Fail("file user does not match:", file.Path)
			}
		}

		if file.Group != nil {
			expectedGid, err := resolveUserOrGroup(file.Group, true)
			if err != nil {
				return Fail("failed to resolve group:", file.Path, err)
			}
			if gid != expectedGid {
				return Fail("file group does not match:", file.Path)
			}
		}

		if len(file.Data) > 0 {
			content, err := ReadFile(file.Path)
			if err != nil {
				return Fail("failed to read file:", file.Path)
			}

			if string(content) != file.Data {
				return Fail("file content does not match:", file.Path)
			}
		}

		if len(file.URI) > 0 {
			log.Printf("Not checking file content specified by URI: %s", file.URI)
		}
	}

	return Pass()
}
