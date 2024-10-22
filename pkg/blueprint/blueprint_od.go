package blueprint

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type blueprintOnDisk struct {
	Name           string                `json:"name" toml:"name"`
	Description    string                `json:"description" toml:"description"`
	Version        string                `json:"version,omitempty" toml:"version,omitempty"`
	Packages       []Package             `json:"packages" toml:"packages"`
	Modules        []Package             `json:"modules" toml:"modules"`
	Groups         []Group               `json:"groups" toml:"groups"`
	Containers     []Container           `json:"containers,omitempty" toml:"containers,omitempty"`
	Customizations *customizationsOnDisk `json:"customizations,omitempty" toml:"customizations"`
	Distro         string                `json:"distro" toml:"distro"`

	// EXPERIMENTAL
	Minimal bool `json:"minimal" toml:"minimal"`
}

type customizationsOnDisk struct {
	Hostname *string              `json:"hostname,omitempty" toml:"hostname,omitempty"`
	Kernel   *KernelCustomization `json:"kernel,omitempty" toml:"kernel,omitempty"`

	// deprecated because singular, replaced with "users"
	DeprecatedUser []UserCustomization `json:"user,omitempty" toml:"user,omitempty"`
	Users          []UserCustomization `json:"users,omitempty" toml:"users,omitempty"`

	// deprecated because singular, replaced with "groups"
	DeprecatedGroup []GroupCustomization `json:"group,omitempty" toml:"group,omitempty"`
	Groups          []GroupCustomization `json:"groups,omitempty" toml:"groups,omitempty"`

	Timezone           *TimezoneCustomization    `json:"timezone,omitempty" toml:"timezone,omitempty"`
	Locale             *LocaleCustomization      `json:"locale,omitempty" toml:"locale,omitempty"`
	Firewall           *FirewallCustomization    `json:"firewall,omitempty" toml:"firewall,omitempty"`
	Services           *ServicesCustomization    `json:"services,omitempty" toml:"services,omitempty"`
	Filesystem         []FilesystemCustomization `json:"filesystem,omitempty" toml:"filesystem,omitempty"`
	InstallationDevice string                    `json:"installation_device,omitempty" toml:"installation_device,omitempty"`
	FDO                *FDOCustomization         `json:"fdo,omitempty" toml:"fdo,omitempty"`
	OpenSCAP           *OpenSCAPCustomization    `json:"openscap,omitempty" toml:"openscap,omitempty"`
	Ignition           *IgnitionCustomization    `json:"ignition,omitempty" toml:"ignition,omitempty"`
	Directories        []DirectoryCustomization  `json:"directories,omitempty" toml:"directories,omitempty"`
	Files              []FileCustomization       `json:"files,omitempty" toml:"files,omitempty"`
	Repositories       []RepositoryCustomization `json:"repositories,omitempty" toml:"repositories,omitempty"`
	FIPS               *bool                     `json:"fips,omitempty" toml:"fips,omitempty"`

	// deprecated because of "-" instead of "_" in key
	DeprecatedContainersStorage *containerStorageCustomizationOnDisk `json:"containers-storage,omitempty" toml:"containers-storage,omitempty"`
	ContainersStorage           *containerStorageCustomizationOnDisk `json:"containers_storage,omitempty" toml:"containers_storage,omitempty"`

	Installer *InstallerCustomization `json:"installer,omitempty" toml:"installer,omitempty"`
	RPM       *RPMCustomization       `json:"rpm,omitempty" toml:"rpm,omitempty"`
	RHSM      *RHSMCustomization      `json:"rhsm,omitempty" toml:"rhsm,omitempty"`
}

type containerStorageCustomizationOnDisk struct {
	// deprecated because of the "-"
	DeprecatedStoragePath *string `json:"destination-path,omitempty" toml:"destination-path,omitempty"`
	// destination is always `containers-storage`, so we won't expose this
	StoragePath *string `json:"destination_path,omitempty" toml:"destination_path,omitempty"`
}

func Load(path string) (*Blueprint, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	switch ext := filepath.Ext(path); ext {
	case ".json":
		return parseJSONFromReader(f, path)
		// TODO: add parseTOMLFromReader
	default:
		return nil, fmt.Errorf("unsupported file format %q", ext)
	}
}

func parseJSONFromReader(r io.Reader, what string) (*Blueprint, error) {
	var bpod blueprintOnDisk

	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&bpod); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("cannot support multiple blueprints from %q", what)
	}

	return bpFromBpod(&bpod)
}

func bpFromBpod(bpod *blueprintOnDisk) (*Blueprint, error) {
	var bp Blueprint

	bp.Name = bpod.Name
	bp.Description = bpod.Description
	bp.Version = bpod.Version
	bp.Packages = bpod.Packages
	bp.Modules = bpod.Modules
	bp.Groups = bpod.Groups
	bp.Containers = bpod.Containers
	bp.Distro = bpod.Distro
	if bpod.Customizations != nil {
		cust, err := bpCustomizationsFromOD(bpod.Customizations)
		if err != nil {
			return nil, err
		}
		bp.Customizations = cust
	}
	bp.Minimal = bpod.Minimal

	return &bp, nil
}

func bpCustomizationsFromOD(cod *customizationsOnDisk) (*Customizations, error) {
	var cus Customizations

	cus.Hostname = cod.Hostname
	cus.Kernel = cod.Kernel
	// XXX: add compat mode here for plural
	switch {
	case cod.DeprecatedUser != nil && cod.Users != nil:
		return nil, fmt.Errorf("both 'user' and 'users' keys are set")
	case cod.DeprecatedUser != nil:
		// warn here?
		cus.User = cod.DeprecatedUser
	case cod.Users != nil:
		cus.User = cod.Users
	}
	switch {
	case cod.DeprecatedGroup != nil && cod.Groups != nil:
		return nil, fmt.Errorf("both 'group' and 'groups' keys are set")
	case cod.DeprecatedGroup != nil:
		cus.Group = cod.DeprecatedGroup
	case cod.Groups != nil:
		cus.Group = cod.Groups
	}

	cus.Timezone = cod.Timezone
	cus.Locale = cod.Locale
	cus.Firewall = cod.Firewall
	cus.Services = cod.Services
	cus.Filesystem = cod.Filesystem
	cus.InstallationDevice = cod.InstallationDevice
	cus.FDO = cod.FDO
	cus.OpenSCAP = cod.OpenSCAP
	cus.Ignition = cod.Ignition
	cus.Directories = cod.Directories
	cus.Files = cod.Files
	cus.Repositories = cod.Repositories
	cus.FIPS = cod.FIPS
	switch {
	case cod.DeprecatedContainersStorage != nil && cod.ContainersStorage != nil:
		return nil, fmt.Errorf("both 'containers-storage' and 'constainers_storage' keys are set")
	case cod.DeprecatedContainersStorage != nil:
		cs, err := bpContainersStorageFromOD(cod.DeprecatedContainersStorage)
		if err != nil {
			return nil, err
		}
		cus.ContainersStorage = cs
	case cod.ContainersStorage != nil:
		cs, err := bpContainersStorageFromOD(cod.ContainersStorage)
		if err != nil {
			return nil, err
		}
		cus.ContainersStorage = cs
	}
	cus.Installer = cod.Installer
	cus.RPM = cod.RPM
	cus.RHSM = cod.RHSM
	return &cus, nil
}

func bpContainersStorageFromOD(csd *containerStorageCustomizationOnDisk) (*ContainerStorageCustomization, error) {
	var cs ContainerStorageCustomization

	switch {
	case csd.DeprecatedStoragePath != nil && csd.StoragePath != nil:
		return nil, fmt.Errorf("both 'destination-path' and 'destination_path' set")
	case csd.DeprecatedStoragePath != nil:
		cs.StoragePath = csd.DeprecatedStoragePath
	case csd.StoragePath != nil:
		cs.StoragePath = csd.StoragePath
	}
	return &cs, nil
}
