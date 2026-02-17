package testarch

import (
	"fmt"
	"os"

	"github.com/osbuild/images/pkg/arch"
)

// GenerateCIArtifactName generates a new identifier for CI artifacts which is based
// on environment variables specified by Jenkins
// note: in case of migration to sth else like Github Actions, change it to whatever variables GH Action provides
func GenerateCIArtifactName(prefix string) (string, error) {
	distroCode := os.Getenv("DISTRO_CODE")
	branchName := os.Getenv("BRANCH_NAME")
	buildId := os.Getenv("BUILD_ID")
	if branchName == "" || buildId == "" || distroCode == "" {
		return "", fmt.Errorf("The environment variables must specify BRANCH_NAME, BUILD_ID, and DISTRO_CODE")
	}

	archStr := arch.Current().String()

	return fmt.Sprintf("%s%s-%s-%s-%s", prefix, distroCode, archStr, branchName, buildId), nil
}
