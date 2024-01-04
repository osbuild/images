package distro

type ImageTypeValidator interface {
	// A list of customization options that this image requires.
	RequiredBlueprintOptions() []string

	// A list of customization options that this image supports.
	SupportedBlueprintOptions() []string
}
