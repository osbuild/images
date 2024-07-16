package oscap

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/osbuild/images/pkg/blueprint"
)

type Profile string

func (p Profile) String() string {
	return string(p)
}

const (
	AnssiBp28Enhanced     Profile = "xccdf_org.ssgproject.content_profile_anssi_bp28_enhanced"
	AnssiBp28High         Profile = "xccdf_org.ssgproject.content_profile_anssi_bp28_high"
	AnssiBp28Intermediary Profile = "xccdf_org.ssgproject.content_profile_anssi_bp28_intermediary"
	AnssiBp28Minimal      Profile = "xccdf_org.ssgproject.content_profile_anssi_bp28_minimal"
	CcnAdvanced           Profile = "xccdf_org.ssgproject.content_profile_ccn_advanced"
	CcnBasic              Profile = "xccdf_org.ssgproject.content_profile_ccn_basic"
	CcnIntermediate       Profile = "xccdf_org.ssgproject.content_profile_ccn_intermediate"
	Cis                   Profile = "xccdf_org.ssgproject.content_profile_cis"
	CisServerL1           Profile = "xccdf_org.ssgproject.content_profile_cis_server_l1"
	CisWorkstationL1      Profile = "xccdf_org.ssgproject.content_profile_cis_workstation_l1"
	CisWorkstationL2      Profile = "xccdf_org.ssgproject.content_profile_cis_workstation_l2"
	Cui                   Profile = "xccdf_org.ssgproject.content_profile_cui"
	E8                    Profile = "xccdf_org.ssgproject.content_profile_e8"
	Hippa                 Profile = "xccdf_org.ssgproject.content_profile_hipaa"
	IsmO                  Profile = "xccdf_org.ssgproject.content_profile_ism_o"
	Ospp                  Profile = "xccdf_org.ssgproject.content_profile_ospp"
	PciDss                Profile = "xccdf_org.ssgproject.content_profile_pci-dss"
	Standard              Profile = "xccdf_org.ssgproject.content_profile_standard"
	Stig                  Profile = "xccdf_org.ssgproject.content_profile_stig"
	StigGui               Profile = "xccdf_org.ssgproject.content_profile_stig_gui"

	// datastream fallbacks
	defaultFedoraDatastream  string = "/usr/share/xml/scap/ssg/content/ssg-fedora-ds.xml"
	defaultCentos8Datastream string = "/usr/share/xml/scap/ssg/content/ssg-centos8-ds.xml"
	defaultCentos9Datastream string = "/usr/share/xml/scap/ssg/content/ssg-cs9-ds.xml"
	defaultRHEL8Datastream   string = "/usr/share/xml/scap/ssg/content/ssg-rhel8-ds.xml"
	defaultRHEL9Datastream   string = "/usr/share/xml/scap/ssg/content/ssg-rhel9-ds.xml"

	// oscap related directories
	DataDir string = "/oscap_data"
)

type RemediationConfig struct {
	Datastream         string
	ProfileID          string
	TailoringPath      string
	CompressionEnabled bool
}

type TailoringConfig struct {
	RemediationConfig
	TailoredProfileID string
	JSONFilepath      string
	Selected          []string
	Unselected        []string
}

func newTailoringConfigs(profileID string, tc blueprint.OpenSCAPTailoringCustomizations, remediationConfig RemediationConfig) (*RemediationConfig, *TailoringConfig, error) {
	// the tailoring config needs to know about the original base profile id,
	// but the remediation config needs to know the updated profile id.
	remediationConfig.ProfileID = fmt.Sprintf("%s_osbuild_tailoring", remediationConfig.ProfileID)
	remediationConfig.TailoringPath = filepath.Join(DataDir, "tailoring.xml")

	tailoringConfig := &TailoringConfig{
		RemediationConfig: RemediationConfig{
			ProfileID:     profileID,
			Datastream:    remediationConfig.Datastream,
			TailoringPath: remediationConfig.TailoringPath,
		},
		Selected:          tc.Selected,
		Unselected:        tc.Unselected,
		TailoredProfileID: remediationConfig.ProfileID,
	}

	return &remediationConfig, tailoringConfig, nil
}

func newJsonConfigs(profileID string, json blueprint.OpenSCAPJSONTailoringCustomizations, remediationConfig RemediationConfig) (*RemediationConfig, *TailoringConfig, error) {
	if json.Filepath == "" {
		return nil, nil, fmt.Errorf("Filepath to an JSON tailoring file is required")
	}

	if json.ProfileID == "" {
		return nil, nil, fmt.Errorf("Tailoring profile ID is required for an JSON tailoring file")
	}

	// the tailoring config needs to know about the original base profile id,
	// but the remediation config needs to know the updated profile id.
	remediationConfig.ProfileID = json.ProfileID
	remediationConfig.TailoringPath = filepath.Join(DataDir, "tailoring.xml")

	tailoringConfig := &TailoringConfig{
		RemediationConfig: RemediationConfig{
			ProfileID:     profileID,
			Datastream:    remediationConfig.Datastream,
			TailoringPath: remediationConfig.TailoringPath,
		},
		JSONFilepath:      json.Filepath,
		TailoredProfileID: json.ProfileID,
	}

	return &remediationConfig, tailoringConfig, nil
}

func NewConfigs(oscapConfig blueprint.OpenSCAPCustomization, defaultDatastream *string) (*RemediationConfig, *TailoringConfig, error) {
	var datastream = oscapConfig.DataStream
	if datastream == "" {
		if defaultDatastream == nil {
			return nil, nil, fmt.Errorf("No OSCAP datastream specified and the distro does not have any default set")
		}
		datastream = *defaultDatastream
	}

	tc := oscapConfig.Tailoring
	json := oscapConfig.JSONTailoring

	remediationConfig := RemediationConfig{
		Datastream:         datastream,
		ProfileID:          oscapConfig.ProfileID,
		CompressionEnabled: true,
	}

	switch {
	case tc != nil && json != nil:
		return nil, nil, fmt.Errorf("Multiple tailoring types set, only one type can be chosen (JSON/Override rules)")
	case tc != nil:
		return newTailoringConfigs(oscapConfig.ProfileID, *tc, remediationConfig)
	case json != nil:
		return newJsonConfigs(oscapConfig.ProfileID, *json, remediationConfig)
	default:
		return &remediationConfig, nil, nil
	}
}

func DefaultFedoraDatastream() string {
	return defaultFedoraDatastream
}

func DefaultRHEL8Datastream(isRHEL bool) string {
	if isRHEL {
		return defaultRHEL8Datastream
	}
	return defaultCentos8Datastream
}

func DefaultRHEL9Datastream(isRHEL bool) string {
	if isRHEL {
		return defaultRHEL9Datastream
	}
	return defaultCentos9Datastream
}

func IsProfileAllowed(profile string, allowlist []Profile) bool {
	for _, a := range allowlist {
		if a.String() == profile {
			return true
		}
		// this enables a user to specify
		// the full profile or the short
		// profile id
		if strings.HasSuffix(a.String(), profile) {
			return true
		}
	}

	return false
}
