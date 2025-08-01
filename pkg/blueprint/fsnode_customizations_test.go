package blueprint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/internal/common"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/pathpolicy"
)

func TestDirectoryCustomizationToFsNodeDirectory(t *testing.T) {
	ensureDirCreation := func(dir *fsnode.Directory, err error) *fsnode.Directory {
		t.Helper()
		assert.NoError(t, err)
		assert.NotNil(t, dir)
		return dir
	}

	testCases := []struct {
		Name    string
		Dir     DirectoryCustomization
		WantDir *fsnode.Directory
		Error   bool
	}{
		{
			Name:  "empty",
			Dir:   DirectoryCustomization{},
			Error: true,
		},
		{
			Name: "path-only",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", nil, nil, nil, false)),
		},
		{
			Name: "path-invalid",
			Dir: DirectoryCustomization{
				Path: "etc/dir",
			},
			Error: true,
		},
		{
			Name: "path-and-mode",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
				Mode: "0700",
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", common.ToPtr(os.FileMode(0700)), nil, nil, false)),
		},
		{
			Name: "path-and-mode-no-leading-zero",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
				Mode: "700",
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", common.ToPtr(os.FileMode(0700)), nil, nil, false)),
		},
		{
			Name: "path-and-mode-invalid",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
				Mode: "12345",
			},
			Error: true,
		},
		{
			Name: "path-user-group-string",
			Dir: DirectoryCustomization{
				Path:  "/etc/dir",
				User:  "root",
				Group: "root",
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", nil, "root", "root", false)),
		},
		{
			Name: "path-user-group-int64",
			Dir: DirectoryCustomization{
				Path:  "/etc/dir",
				User:  int64(0),
				Group: int64(0),
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", nil, int64(0), int64(0), false)),
		},
		{
			Name: "path-and-user-invalid-string",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
				User: "r@@t",
			},
			Error: true,
		},
		{
			Name: "path-and-user-invalid-int64",
			Dir: DirectoryCustomization{
				Path: "/etc/dir",
				User: -1,
			},
			Error: true,
		},
		{
			Name: "path-and-group-invalid-string",
			Dir: DirectoryCustomization{
				Path:  "/etc/dir",
				Group: "r@@t",
			},
			Error: true,
		},
		{
			Name: "path-and-group-invalid-int64",
			Dir: DirectoryCustomization{
				Path:  "/etc/dir",
				Group: -1,
			},
			Error: true,
		},
		{
			Name: "path-and-ensure-parent-dirs",
			Dir: DirectoryCustomization{
				Path:          "/etc/dir",
				EnsureParents: true,
			},
			WantDir: ensureDirCreation(fsnode.NewDirectory("/etc/dir", nil, nil, nil, true)),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			dir, err := tc.Dir.ToFsNodeDirectory()
			if tc.Error {
				assert.Error(t, err)
				assert.Nil(t, dir)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tc.WantDir, dir)
			}
		})
	}
}

func TestDirectoryCustomizationsToFsNodeDirectories(t *testing.T) {
	ensureDirCreation := func(dir *fsnode.Directory, err error) *fsnode.Directory {
		t.Helper()
		assert.NoError(t, err)
		assert.NotNil(t, dir)
		return dir
	}

	testCases := []struct {
		Name     string
		Dirs     []DirectoryCustomization
		WantDirs []*fsnode.Directory
		Error    bool
	}{
		{
			Name:     "empty",
			Dirs:     []DirectoryCustomization{},
			WantDirs: nil,
		},
		{
			Name: "single-directory",
			Dirs: []DirectoryCustomization{
				{
					Path:          "/etc/dir",
					User:          "root",
					Group:         "root",
					Mode:          "0700",
					EnsureParents: true,
				},
			},
			WantDirs: []*fsnode.Directory{
				ensureDirCreation(fsnode.NewDirectory(
					"/etc/dir",
					common.ToPtr(os.FileMode(0700)),
					"root",
					"root",
					true,
				)),
			},
		},
		{
			Name: "multiple-directories",
			Dirs: []DirectoryCustomization{
				{
					Path:  "/etc/dir",
					User:  "root",
					Group: "root",
				},
				{
					Path:  "/etc/dir2",
					User:  int64(0),
					Group: int64(0),
				},
			},
			WantDirs: []*fsnode.Directory{
				ensureDirCreation(fsnode.NewDirectory("/etc/dir", nil, "root", "root", false)),
				ensureDirCreation(fsnode.NewDirectory("/etc/dir2", nil, int64(0), int64(0), false)),
			},
		},
		{
			Name: "multiple-directories-with-errors",
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/../dir",
				},
				{
					Path: "/etc/dir2",
					User: "r@@t",
				},
			},
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			dirs, err := DirectoryCustomizationsToFsNodeDirectories(tc.Dirs)
			if tc.Error {
				assert.Error(t, err)
				assert.Nil(t, dirs)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tc.WantDirs, dirs)
			}
		})
	}
}

func TestDirectoryCustomizationUnmarshalTOML(t *testing.T) {
	testCases := []struct {
		Name  string
		TOML  string
		Want  []DirectoryCustomization
		Error bool
	}{
		{
			Name: "directory-with-path",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.directories]]
path = "/etc/dir"
`,
			Want: []DirectoryCustomization{
				{
					Path: "/etc/dir",
				},
			},
		},
		{
			Name: "multiple-directories",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.directories]]
path = "/etc/dir1"
mode = "0700"
user = "root"
group = "root"
ensure_parents = true

[[customizations.directories]]
path = "/etc/dir2"
mode = "0755"
user = 0
group = 0
ensure_parents = true

[[customizations.directories]]
path = "/etc/dir3"
`,
			Want: []DirectoryCustomization{
				{
					Path:          "/etc/dir1",
					Mode:          "0700",
					User:          "root",
					Group:         "root",
					EnsureParents: true,
				},
				{
					Path:          "/etc/dir2",
					Mode:          "0755",
					User:          int64(0),
					Group:         int64(0),
					EnsureParents: true,
				},
				{
					Path: "/etc/dir3",
				},
			},
		},
		{
			Name: "invalid-directories",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.directories]]
path = "/etc/../dir1"

[[customizations.directories]]
path = "/etc/dir2"
mode = "12345"

[[customizations.directories]]
path = "/etc/dir3"
user = "r@@t"

[[customizations.directories]]
path = "/etc/dir4"
group = "r@@t"

[[customizations.directories]]
path = "/etc/dir5"
user = -1

[[customizations.directories]]
path = "/etc/dir6"
group = -1


[[customizations.directories]]
`,
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var blueprint Blueprint
			err := toml.Unmarshal([]byte(tc.TOML), &blueprint)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, blueprint.Customizations)
				assert.Len(t, blueprint.Customizations.Directories, len(tc.Want))
				assert.EqualValues(t, tc.Want, blueprint.Customizations.GetDirectories())
			}
		})
	}
}

func TestDirectoryCustomizationUnmarshalJSON(t *testing.T) {
	testCases := []struct {
		Name  string
		JSON  string
		Want  []DirectoryCustomization
		Error bool
	}{
		{
			Name: "directory-with-path",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"directories": [
			{
				"path": "/etc/dir"
			}
		]
	}
}`,
			Want: []DirectoryCustomization{
				{
					Path: "/etc/dir",
				},
			},
		},
		{
			Name: "multiple-directories",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"directories": [
			{
				"path": "/etc/dir1",
				"mode": "0700",
				"user": "root",
				"group": "root",
				"ensure_parents": true
			},
			{
				"path": "/etc/dir2",
				"mode": "0755",
				"user": 0,
				"group": 0,
				"ensure_parents": true
			},
			{
				"path": "/etc/dir3"
			}
		]
	}
}`,
			Want: []DirectoryCustomization{
				{
					Path:          "/etc/dir1",
					Mode:          "0700",
					User:          "root",
					Group:         "root",
					EnsureParents: true,
				},
				{
					Path:          "/etc/dir2",
					Mode:          "0755",
					User:          int64(0),
					Group:         int64(0),
					EnsureParents: true,
				},
				{
					Path: "/etc/dir3",
				},
			},
		},
		{
			Name: "invalid-directories",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"directories": [
			{
				"path": "/etc/../dir1"
			},
			{
				"path": "/etc/dir2",
				"mode": "12345"
			},
			{
				"path": "/etc/dir3",
				"user": "r@@t"
			},
			{
				"path": "/etc/dir4",
				"group": "r@@t"
			},
			{
				"path": "/etc/dir5",
				"user": -1
			},
			{
				"path": "/etc/dir6",
				"group": -1
			}
			{}
		]
	}
}`,
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var blueprint Blueprint
			err := json.Unmarshal([]byte(tc.JSON), &blueprint)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, blueprint.Customizations)
				assert.Len(t, blueprint.Customizations.Directories, len(tc.Want))
				assert.EqualValues(t, tc.Want, blueprint.Customizations.GetDirectories())
			}
		})
	}

}

func TestFileCustomizationToFsNodeFile(t *testing.T) {
	ensureFileCreation := func(file *fsnode.File, err error) *fsnode.File {
		t.Helper()
		assert.NoError(t, err)
		assert.NotNil(t, file)
		return file
	}

	testCases := []struct {
		Name  string
		File  FileCustomization
		Want  *fsnode.File
		Error bool
	}{
		{
			Name:  "empty",
			File:  FileCustomization{},
			Error: true,
		},
		{
			Name: "path-only",
			File: FileCustomization{
				Path: "/etc/file",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, nil, nil, nil)),
		},
		{
			Name: "path-invalid",
			File: FileCustomization{
				Path: "../etc/file",
			},
			Error: true,
		},
		{
			Name: "path-and-mode",
			File: FileCustomization{
				Path: "/etc/file",
				Mode: "0700",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", common.ToPtr(os.FileMode(0700)), nil, nil, nil)),
		},
		{
			Name: "path-and-mode-no-leading-zero",
			File: FileCustomization{
				Path: "/etc/file",
				Mode: "700",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", common.ToPtr(os.FileMode(0700)), nil, nil, nil)),
		},
		{
			Name: "path-and-mode-invalid",
			File: FileCustomization{
				Path: "/etc/file",
				Mode: "12345",
			},
			Error: true,
		},
		{
			Name: "path-user-group-string",
			File: FileCustomization{
				Path:  "/etc/file",
				User:  "root",
				Group: "root",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, "root", "root", nil)),
		},
		{
			Name: "path-user-group-int64",
			File: FileCustomization{
				Path:  "/etc/file",
				User:  int64(0),
				Group: int64(0),
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, int64(0), int64(0), nil)),
		},
		{
			Name: "path-and-user-invalid-string",
			File: FileCustomization{
				Path: "/etc/file",
				User: "r@@t",
			},
			Error: true,
		},
		{
			Name: "path-and-user-invalid-int64",
			File: FileCustomization{
				Path: "/etc/file",
				User: int64(-1),
			},
			Error: true,
		},
		{
			Name: "path-and-group-string",
			File: FileCustomization{
				Path:  "/etc/file",
				Group: "root",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, nil, "root", nil)),
		},
		{
			Name: "path-and-group-int64",
			File: FileCustomization{
				Path:  "/etc/file",
				Group: int64(0),
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, nil, int64(0), nil)),
		},
		{
			Name: "path-and-group-invalid-string",
			File: FileCustomization{
				Path:  "/etc/file",
				Group: "r@@t",
			},
			Error: true,
		},
		{
			Name: "path-and-group-invalid-int64",
			File: FileCustomization{
				Path:  "/etc/file",
				Group: int64(-1),
			},
			Error: true,
		},
		{
			Name: "path-and-data",
			File: FileCustomization{
				Path: "/etc/file",
				Data: "hello world",
			},
			Want: ensureFileCreation(fsnode.NewFile("/etc/file", nil, nil, nil, []byte("hello world"))),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			file, err := tc.File.ToFsNodeFile()
			if tc.Error {
				assert.Error(t, err)
				assert.Nil(t, file)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tc.Want, file)
			}
		})
	}
}

func TestFileCustomizationsToFsNodeFiles(t *testing.T) {
	ensureFileCreation := func(file *fsnode.File, err error) *fsnode.File {
		t.Helper()
		assert.NoError(t, err)
		assert.NotNil(t, file)
		return file
	}

	testCases := []struct {
		Name  string
		Files []FileCustomization
		Want  []*fsnode.File
		Error bool
	}{
		{
			Name:  "empty",
			Files: []FileCustomization{},
			Want:  nil,
		},
		{
			Name: "single-file",
			Files: []FileCustomization{
				{
					Path:  "/etc/file",
					User:  "root",
					Group: "root",
					Mode:  "0700",
					Data:  "hello world",
				},
			},
			Want: []*fsnode.File{
				ensureFileCreation(fsnode.NewFile(
					"/etc/file",
					common.ToPtr(os.FileMode(0700)),
					"root",
					"root",
					[]byte("hello world"),
				)),
			},
		},
		{
			Name: "multiple-files",
			Files: []FileCustomization{
				{
					Path:  "/etc/file",
					Data:  "hello world",
					User:  "root",
					Group: "root",
				},
				{
					Path:  "/etc/file2",
					Data:  "hello world",
					User:  int64(0),
					Group: int64(0),
				},
			},
			Want: []*fsnode.File{
				ensureFileCreation(fsnode.NewFile("/etc/file", nil, "root", "root", []byte("hello world"))),
				ensureFileCreation(fsnode.NewFile("/etc/file2", nil, int64(0), int64(0), []byte("hello world"))),
			},
		},
		{
			Name: "multiple-files-with-errors",
			Files: []FileCustomization{
				{
					Path: "/etc/../file",
					Data: "hello world",
				},
				{
					Path: "/etc/file2",
					Data: "hello world",
					User: "r@@t",
				},
			},
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			files, err := FileCustomizationsToFsNodeFiles(tc.Files)
			if tc.Error {
				assert.Error(t, err)
				assert.Nil(t, files)
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tc.Want, files)
			}
		})
	}
}

func TestFileCustomizationUnmarshalTOML(t *testing.T) {
	tmpdir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpdir, "some-file.txt"), nil, 0644)
	require.NoError(t, err)

	testCases := []struct {
		Name  string
		TOML  string
		Want  []FileCustomization
		Error bool
	}{
		{
			Name: "file-with-path",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.files]]
path = "/etc/file"
`,
			Want: []FileCustomization{
				{
					Path: "/etc/file",
				},
			},
		},
		{
			Name: "file-with-uri",
			TOML: fmt.Sprintf(`
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.files]]
path = "/etc/file"
uri = "file://%s/some-file.txt"
`, tmpdir),
			Want: []FileCustomization{
				{
					Path: "/etc/file",
					URI:  fmt.Sprintf("file://%s/some-file.txt", tmpdir),
				},
			},
		},
		{
			Name: "multiple-files",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.files]]
path = "/etc/file1"
mode = "0600"
user = "root"
group = "root"
data = "hello world"

[[customizations.files]]
path = "/etc/file2"
mode = "0644"
data = "hello world 2"

[[customizations.files]]
path = "/etc/file3"
user = 0
group = 0
data = "hello world 3"
`,
			Want: []FileCustomization{
				{
					Path:  "/etc/file1",
					Mode:  "0600",
					User:  "root",
					Group: "root",
					Data:  "hello world",
				},
				{
					Path: "/etc/file2",
					Mode: "0644",
					Data: "hello world 2",
				},
				{
					Path:  "/etc/file3",
					User:  int64(0),
					Group: int64(0),
					Data:  "hello world 3",
				},
			},
		},
		{
			Name: "invalid-files",
			TOML: `
name = "test"
description = "Test"
version = "0.0.0"

[[customizations.files]]
path = "/etc/../file1"

[[customizations.files]]
path = "/etc/file2"
mode = "12345"

[[customizations.files]]
path = "/etc/file3"
user = "r@@t"

[[customizations.files]]
path = "/etc/file4"
group = "r@@t"

[[customizations.files]]
path = "/etc/file5"
user = -1

[[customizations.files]]
path = "/etc/file6"
group = -1
`,
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var blueprint Blueprint
			err := toml.Unmarshal([]byte(tc.TOML), &blueprint)
			if tc.Error {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, blueprint.Customizations)
				assert.Len(t, blueprint.Customizations.Files, len(tc.Want))
				assert.EqualValues(t, tc.Want, blueprint.Customizations.Files)
			}
		})
	}
}

func TestFileCustomizationUnmarshalJSON(t *testing.T) {
	tmpdir := t.TempDir()
	err := os.WriteFile(filepath.Join(tmpdir, "some-file.txt"), nil, 0644)
	require.NoError(t, err)

	testCases := []struct {
		Name  string
		JSON  string
		Want  []FileCustomization
		Error bool
	}{
		{
			Name: "file-with-path",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"files": [
			{
				"path": "/etc/file"
			}
		]
	}
}`,
			Want: []FileCustomization{
				{
					Path: "/etc/file",
				},
			},
		},
		{
			Name: "file-with-uri",
			JSON: fmt.Sprintf(`
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"files": [
			{
				"path": "/etc/file",
                                "uri": "file://%s/some-file.txt"
			}
		]
	}
}`, tmpdir),
			Want: []FileCustomization{
				{
					Path: "/etc/file",
					URI:  fmt.Sprintf("file://%s/some-file.txt", tmpdir),
				},
			},
		},
		{
			Name: "multiple-files",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"files": [
			{
				"path": "/etc/file1",
				"mode": "0600",
				"user": "root",
				"group": "root",
				"data": "hello world"
			},
			{
				"path": "/etc/file2",
				"mode": "0644",
				"data": "hello world 2"
			},
			{
				"path": "/etc/file3",
				"user": 0,
				"group": 0,
				"data": "hello world 3"
			}
		]
	}
}`,
			Want: []FileCustomization{
				{
					Path:  "/etc/file1",
					Mode:  "0600",
					User:  "root",
					Group: "root",
					Data:  "hello world",
				},
				{
					Path: "/etc/file2",
					Mode: "0644",
					Data: "hello world 2",
				},
				{
					Path:  "/etc/file3",
					User:  int64(0),
					Group: int64(0),
					Data:  "hello world 3",
				},
			},
		},
		{
			Name: "invalid-files",
			JSON: `
{
	"name": "test",
	"description": "Test",
	"version": "0.0.0",
	"customizations": {
		"files": [
			{
				"path": "/etc/../file1"
			},
			{
				"path": "/etc/file2",
				"mode": "12345"
			},
			{
				"path": "/etc/file3",
				"user": "r@@t"
			},
			{
				"path": "/etc/file4",
				"group": "r@@t"
			},
			{
				"path": "/etc/file5",
				"user": -1
			},
			{
				"path": "/etc/file6",
				"group": -1
			}
		]
	}
}`,
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			var blueprint Blueprint
			err := json.Unmarshal([]byte(tc.JSON), &blueprint)
			if tc.Error {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, blueprint.Customizations)
				assert.Len(t, blueprint.Customizations.Files, len(tc.Want))
				assert.EqualValues(t, tc.Want, blueprint.Customizations.Files)
			}
		})
	}
}

func TestValidateDirFileCustomizations(t *testing.T) {
	testCases := []struct {
		Name  string
		Files []FileCustomization
		Dirs  []DirectoryCustomization
		Error bool
	}{
		{
			Name: "empty-customizations",
		},
		{
			Name: "valid-file-customizations-only",
			Files: []FileCustomization{
				{
					Path: "/etc/file1",
				},
				{
					Path: "/etc/file2",
				},
			},
		},
		{
			Name: "valid-dir-customizations-only",
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/dir1",
				},
				{
					Path: "/etc/dir2",
				},
			},
		},
		{
			Name: "valid-file-and-dir-customizations",
			Files: []FileCustomization{
				{
					Path: "/etc/named/path1/path2/file",
				},
				{
					Path: "/etc/named/path1/file",
				},
				{
					Path: "/etc/named/path1/path2/file2",
				},
				{
					Path: "/etc/named/path1/file2",
				},
				{
					Path: "/etc/named/file",
				},
			},
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/named/path1/path2",
				},
				{
					Path: "/etc/named/path1",
				},
			},
		},
		// Errors
		{
			Name: "file-parent-of-file",
			Files: []FileCustomization{
				{
					Path: "/etc/file1/file2",
				},
				{
					Path: "/etc/file1",
				},
			},
			Error: true,
		},
		{
			Name: "file-parent-of-dir",
			Files: []FileCustomization{
				{
					Path: "/etc/file2",
				},
				{
					Path: "/etc/dir1",
				},
			},
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/dir1/dir2",
				},
				{
					Path: "/etc/dir3",
				},
			},
			Error: true,
		},
		{
			Name: "duplicate-file-paths",
			Files: []FileCustomization{
				{
					Path: "/etc/file1",
				},
				{
					Path: "/etc/file2",
				},
				{
					Path: "/etc/file1",
				},
			},
			Error: true,
		},
		{
			Name: "duplicate-dir-paths",
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/dir1",
				},
				{
					Path: "/etc/dir2",
				},
				{
					Path: "/etc/dir1",
				},
			},
			Error: true,
		},
		{
			Name: "duplicate-file-and-dir-paths",
			Files: []FileCustomization{
				{
					Path: "/etc/path1",
				},
				{
					Path: "/etc/path2",
				},
			},
			Dirs: []DirectoryCustomization{
				{
					Path: "/etc/path3",
				},
				{
					Path: "/etc/path2",
				},
			},
			Error: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := ValidateDirFileCustomizations(tc.Dirs, tc.Files)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckFileCustomizationsPolicy(t *testing.T) {
	policy := map[string]pathpolicy.PathPolicy{
		"/":               {Deny: true},
		"/etc":            {},
		"/etc/fstab":      {Deny: true},
		"/etc/os-release": {Deny: true},
		"/etc/hostname":   {Deny: true},
		"/etc/shadow":     {Deny: true},
		"/etc/passwd":     {Deny: true},
		"/etc/group":      {Deny: true},
	}
	pathPolicy := pathpolicy.NewPathPolicies(policy)

	testCases := []struct {
		Name  string
		Files []FileCustomization
		Error bool
	}{
		{
			Name: "disallowed-file",
			Files: []FileCustomization{
				{
					Path: "/etc/shadow",
				},
			},
			Error: true,
		},
		{
			Name: "disallowed-file-2",
			Files: []FileCustomization{
				{
					Path: "/home/user/.ssh/authorized_keys",
				},
			},
			Error: true,
		},
		{
			Name: "disallowed-file-3",
			Files: []FileCustomization{
				{
					Path: "/file",
				},
			},
			Error: true,
		},
		{
			Name: "allowed-file-named",
			Files: []FileCustomization{
				{
					Path: "/etc/named.conf",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := CheckFileCustomizationsPolicy(tc.Files, pathPolicy)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckDirectoryCustomizationsPolicy(t *testing.T) {
	policy := map[string]pathpolicy.PathPolicy{
		"/":    {Deny: true},
		"/etc": {},
	}
	pathPolicy := pathpolicy.NewPathPolicies(policy)

	testCases := []struct {
		Name        string
		Directories []DirectoryCustomization
		Error       bool
	}{
		{
			Name: "disallowed-directory",
			Directories: []DirectoryCustomization{
				{
					Path: "/dir",
				},
			},
			Error: true,
		},
		{
			Name: "disallowed-directory-2",
			Directories: []DirectoryCustomization{
				{
					Path: "/var/log/fancy-dir",
				},
			},
			Error: true,
		},
		{
			Name: "allowed-directory",
			Directories: []DirectoryCustomization{
				{
					Path: "/etc/systemd/system/sshd.service.d",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			err := CheckDirectoryCustomizationsPolicy(tc.Directories, pathPolicy)
			if tc.Error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestToFileNodeNoMixingOfPathAndData(t *testing.T) {
	fc := FileCustomization{
		URI:  "/some/ref",
		Data: "some-data",
	}
	_, err := fc.ToFsNodeFile()
	assert.EqualError(t, err, `cannot specify both data "some-data" and URI "/some/ref"`)
}

func TestToFileNodeForRef(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "test1.txt")
	err := os.WriteFile(testFile, nil, 0644)
	assert.NoError(t, err)

	fc := FileCustomization{
		Path: "/some/path",
		URI:  testFile,
	}
	file, err := fc.ToFsNodeFile()
	assert.NoError(t, err)
	assert.Equal(t, "/some/path", file.Path())
	assert.Equal(t, testFile, file.URI())
}
