package rpmmd_test

import (
	"encoding/json"
	"testing"

	"github.com/osbuild/images/pkg/rpmmd"
	"github.com/stretchr/testify/assert"
)

func TestRepoConfigMarshalEmpty(t *testing.T) {
	repoCfg := &rpmmd.RepoConfig{}
	js, err := json.Marshal(repoCfg)
	assert.NoError(t, err)
	assert.Equal(t, string(js), `{}`)
}
