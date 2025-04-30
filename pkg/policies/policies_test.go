package policies

import "testing"

func TestMountpointPolicies(t *testing.T) {
	type testCase struct {
		path        string
		allowedFS   bool
		allowedDisk bool
	}

	testCases := []testCase{
		{"/", true, true},

		{"/bin", false, false},
		{"/dev", false, false},
		{"/etc", false, false},
		{"/lib", false, false},
		{"/lib64", false, false},
		{"/lost+found", false, false},
		{"/proc", false, false},
		{"/run", false, false},
		{"/sbin", false, false},
		{"/sys", false, false},
		{"/sysroot", false, false},

		{"/mnt", true, true},
		{"/root", true, true},

		{"/custom", true, true},
		{"/custom/dir", true, true},

		{"/boot", true, true},
		{"/boot/dir", true, true},
		// Note that /boot/efi is allowed for disk customizations
		// but not for filesystem customizations
		{"/boot/efi", false, true},

		{"/var", true, true},
		{"/var/lib", true, true},
		{"/var/log", true, true},
		{"/var/tmp", true, true},
		{"/var/run", false, false},
		{"/var/lock", false, false},

		{"/opt", true, true},
		{"/opt/fancyapp", true, true},

		{"/srv", true, true},
		{"/srv/www", true, true},

		{"/usr", true, true},
		{"/usr/bin", false, false},
		{"/usr/sbin", false, false},
		{"/usr/local", false, false},
		{"/usr/local/bin", false, false},
		{"/usr/lib", false, false},
		{"/usr/lib64", false, false},

		{"/tmp", true, true},
		{"/tmp/foo", true, true},

		{"/app", true, true},
		{"/app/bin", true, true},

		{"/data", true, true},
		{"/data/foo", true, true},

		{"/home", true, true},
		{"/home/user", true, true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			// check FS customizations policy
			err := MountpointPoliciesFS.Check(tc.path)
			if err != nil && tc.allowedFS {
				t.Errorf("expected %s to be allowed for FS, but got error: %v", tc.path, err)
			} else if err == nil && !tc.allowedFS {
				t.Errorf("expected %s to be denied for FS, but got no error", tc.path)
			}
			// check disk customizations policy
			err = MountpointPoliciesDisk.Check(tc.path)
			if err != nil && tc.allowedDisk {
				t.Errorf("expected %s to be allowed for disk, but got error: %v", tc.path, err)
			} else if err == nil && !tc.allowedDisk {
				t.Errorf("expected %s to be denied for disk, but got no error", tc.path)
			}
		})
	}
}

func TestOstreeMountpointPolicies(t *testing.T) {
	type testCase struct {
		path    string
		allowed bool
	}

	testCases := []testCase{
		{"/ostree", false},
		{"/ostree/foo", false},

		{"/foo", true},
		{"/foo/bar", true},

		{"/var", true},
		{"/var/myfiles", true},
		{"/var/roothome", false},

		{"/home", false},
		{"/home/shadowman", false},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			err := OstreeMountpointPolicies.Check(tc.path)
			if err != nil && tc.allowed {
				t.Errorf("expected %s to be allowed, but got error: %v", tc.path, err)
			} else if err == nil && !tc.allowed {
				t.Errorf("expected %s to be denied, but got no error", tc.path)
			}
		})
	}
}
