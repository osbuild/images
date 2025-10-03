package platform

type BootMode uint64

const (
	BOOT_NONE BootMode = iota
	BOOT_LEGACY
	BOOT_UEFI
	BOOT_HYBRID
)

func (m BootMode) String() string {
	switch m {
	case BOOT_NONE:
		return "none"
	case BOOT_LEGACY:
		return "legacy"
	case BOOT_UEFI:
		return "uefi"
	case BOOT_HYBRID:
		return "hybrid"
	default:
		panic("invalid boot mode")
	}
}

var BootModeMap = make(map[string]BootMode)

func init() {
	BootModeMap["none"] = BOOT_NONE
	BootModeMap["legacy"] = BOOT_LEGACY
	BootModeMap["uefi"] = BOOT_UEFI
	BootModeMap["hybrid"] = BOOT_HYBRID
}
