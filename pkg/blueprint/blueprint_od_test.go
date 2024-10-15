package blueprint_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/osbuild/images/pkg/blueprint"
)

func makeBlueprintFile(t *testing.T, content string) string {
	fakeBlueprintPath := filepath.Join(t.TempDir(), "bp.json")

	err := os.WriteFile(fakeBlueprintPath, []byte(content), 0644)
	require.NoError(t, err)

	return fakeBlueprintPath
}

func TestParseUsersPlural(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "users": [
     {"name": "user-1"}
   ]
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "user-1", bp.Customizations.User[0].Name)
}

func TestParseUsersSingularDeprecated(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "user": [
     {"name": "user-1"}
   ]
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "user-1", bp.Customizations.User[0].Name)
}

func TestParseUsersPluralSingularError(t *testing.T) {
	_, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "user": [
     {"name": "user-1"}
   ],
   "users": [
     {"name": "user-1"}
   ]
 }
}`))
	assert.ErrorContains(t, err, "both 'user' and 'users' keys are set")
}

func TestParseGroupsPlural(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "groups": [
     {"name": "group-1"}
   ]
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "group-1", bp.Customizations.Group[0].Name)
}

func TestParseGroupSingularDeprecated(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "group": [
     {"name": "group-1"}
   ]
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "group-1", bp.Customizations.Group[0].Name)
}

func TestParseGroupPluralSingularError(t *testing.T) {
	_, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "group": [
     {"name": "grp-1"}
   ],
   "groups": [
     {"name": "grp-1"}
   ]
 }
}`))
	assert.ErrorContains(t, err, "both 'group' and 'groups' keys are set")
}

func TestParseContainersStorageDeprecatedDash(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "containers-storage": {
     "destination-path": "/storage/path"
   }
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "/storage/path", *bp.Customizations.ContainersStorage.StoragePath)
}

func TestParseContainersStorageUnderscore(t *testing.T) {
	bp, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "containers_storage": {
     "destination_path": "/storage/path"
   }
 }
}`))
	require.NoError(t, err)
	assert.Equal(t, "/storage/path", *bp.Customizations.ContainersStorage.StoragePath)
}

func TestParseContainersStorageBothDashUnderscoreError(t *testing.T) {
	_, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "containers_storage": {
     "destination_path": "/storage/path"
   },
   "containers-storage": {
     "destination_path": "/storage/path"
   }
 }
}`))
	assert.ErrorContains(t, err, "both 'containers-storage' and 'constainers_storage' keys are set")
}

func TestParseDestinationPathBothDashUnderscoreError(t *testing.T) {
	_, err := blueprint.Load(makeBlueprintFile(t, `{
 "name": "bp-name",
 "customizations": {
   "containers_storage": {
     "destination_path": "/storage/path",
     "destination-path": "/storage/path"
   }
 }
}`))
	assert.ErrorContains(t, err, "both 'destination-path' and 'destination_path' set")
}
