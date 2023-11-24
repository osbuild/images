package arch

type Arch uint64

const ( // architecture enum
	ARCH_AARCH64 Arch = iota
	ARCH_PPC64LE
	ARCH_S390X
	ARCH_X86_64
)

func (a Arch) String() string {
	switch a {
	case ARCH_AARCH64:
		return "aarch64"
	case ARCH_PPC64LE:
		return "ppc64le"
	case ARCH_S390X:
		return "s390x"
	case ARCH_X86_64:
		return "x86_64"
	default:
		panic("invalid architecture")
	}
}
