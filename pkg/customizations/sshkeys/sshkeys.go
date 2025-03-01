package sshkeys

import "github.com/osbuild/images/pkg/blueprint"

type SSHKey struct {
	User string
	Key  string
}

func SSHKeysFromBP(sshKeyCustomizations []blueprint.SSHKeyCustomization) []SSHKey {
	sshkeys := make([]SSHKey, len(sshKeyCustomizations))

	for idx := range sshKeyCustomizations {
		// currently, they have the same structure, so we convert directly
		// this will fail to compile as soon as one of the two changes
		sshkeys[idx] = SSHKey(sshKeyCustomizations[idx])
	}

	return sshkeys
}
