package oscap

import (
	"fmt"
	"path/filepath"

	"github.com/osbuild/images/pkg/blueprint"
	"github.com/osbuild/images/pkg/customizations/fsnode"
	"github.com/osbuild/images/pkg/distro"
	"github.com/osbuild/images/pkg/osbuild"
)

func createRequiredDirectories(createTailoring bool) ([]*fsnode.Directory, error) {
	var directories []*fsnode.Directory

	// although the osbuild stage will create this directory,
	// it's probably better to ensure that it is created here
	dataDirNode, err := fsnode.NewDirectory(dataDirPath, nil, nil, nil, true)
	if err != nil {
		return nil, fmt.Errorf("unexpected error creating OpenSCAP data directory: %s", err)
	}

	directories = append(directories, dataDirNode)

	if createTailoring {
		tailoringDirNode, err := fsnode.NewDirectory(tailoringDirPath, nil, nil, nil, true)
		if err != nil {
			return nil, fmt.Errorf("unexpected error creating OpenSCAP tailoring directory: %s", err)
		}

		directories = append(directories, tailoringDirNode)
	}

	return directories, nil
}

func getTailoringProfileID(profileID string) string {
	return fmt.Sprintf("%s_osbuild_tailoring", profileID)
}

func CreateTailoringStageOptions(oscapConfig *blueprint.OpenSCAPCustomization, d distro.Distro) *osbuild.OscapAutotailorStageOptions {
	if oscapConfig == nil {
		return nil
	}

	datastream := getDatastream(oscapConfig.Datastream, d)

	tailoringConfig := oscapConfig.Tailoring
	if tailoringConfig == nil {
		return nil
	}

	newProfile := getTailoringProfileID(oscapConfig.ProfileID)
	path := filepath.Join(tailoringDirPath, "tailoring.xml")

	return osbuild.NewOscapAutotailorStageOptions(
		path,
		osbuild.OscapAutotailorConfig{
			ProfileID:  oscapConfig.ProfileID,
			Datastream: datastream,
			Selected:   tailoringConfig.Selected,
			Unselected: tailoringConfig.Unselected,
			NewProfile: newProfile,
		},
	)
}

func CreateRemediationStageOptions(
	oscapConfig *blueprint.OpenSCAPCustomization,
	isOSTree bool,
	d distro.Distro,
) (*osbuild.OscapRemediationStageOptions, []*fsnode.Directory, error) {
	if oscapConfig == nil {
		return nil, nil, nil
	}

	if isOSTree {
		return nil, nil, fmt.Errorf("unexpected oscap options for ostree image type")
	}

	datastream := getDatastream(oscapConfig.Datastream, d)

	profileID := oscapConfig.ProfileID
	if oscapConfig.Tailoring != nil {
		profileID = getTailoringProfileID(profileID)
	}

	directories, err := createRequiredDirectories(oscapConfig.Tailoring == nil)
	if err != nil {
		return nil, nil, err
	}

	return osbuild.NewOscapRemediationStageOptions(
		dataDirPath,
		osbuild.OscapConfig{
			Datastream:  datastream,
			ProfileID:   profileID,
			Compression: true,
		},
	), directories, nil
}
