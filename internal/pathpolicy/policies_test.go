package pathpolicy

import "testing"

func TestMountpointPolicies(t *testing.T) {
	type testCase struct {
		path    string
		allowed bool
	}

	testCases := []testCase{
		{"/", true},

		{"/bin", false},
		{"/dev", false},
		{"/etc", false},
		{"/lib", false},
		{"/lib64", false},
		{"/lost+found", false},
		{"/proc", false},
		{"/run", false},
		{"/sbin", false},
		{"/sys", false},
		{"/sysroot", false},

		{"/mnt", true},
		{"/root", true},

		{"/custom", true},
		{"/custom/dir", true},

		{"/boot", true},
		{"/boot/dir", true},
		{"/boot/efi", false},

		{"/var", true},
		{"/var/lib", true},
		{"/var/log", true},
		{"/var/tmp", true},
		{"/var/run", false},
		{"/var/lock", false},

		{"/opt", true},
		{"/opt/fancyapp", true},

		{"/srv", true},
		{"/srv/www", true},

		{"/usr", true},
		{"/usr/bin", false},
		{"/usr/sbin", false},
		{"/usr/local", false},
		{"/usr/local/bin", false},
		{"/usr/lib", false},
		{"/usr/lib64", false},

		{"/tmp", true},
		{"/tmp/foo", true},

		{"/app", true},
		{"/app/bin", true},

		{"/data", true},
		{"/data/foo", true},

		{"/home", true},
		{"/home/user", true},
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			err := MountpointPolicies.Check(tc.path)
			if err != nil && tc.allowed {
				t.Errorf("expected %s to be allowed, but got error: %v", tc.path, err)
			} else if err == nil && !tc.allowed {
				t.Errorf("expected %s to be denied, but got no error", tc.path)
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
