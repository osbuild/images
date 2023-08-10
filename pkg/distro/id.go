package distro

import (
	"fmt"
	"strconv"
	"strings"
)

type ID struct {
	Name         string
	MajorVersion int
	MinorVersion int
}

type ParseError struct {
	ToParse string
	Msg     string
	Inner   error
}

func (e ParseError) Error() string {
	msg := fmt.Sprintf("error when parsing distro name (%s): %v", e.ToParse, e.Msg)

	if e.Inner != nil {
		msg += fmt.Sprintf(", inner error:\n%v", e.Inner)
	}

	return msg
}

func ParseName(id string) (ID, error) {
	idParts := strings.Split(id, "-")

	if len(idParts) > 2 {
		return ID{}, ParseError{ToParse: id, Msg: fmt.Sprintf("too many dashes (%d)", len(idParts)-1)}
	}

	name := idParts[0]
	version := idParts[1]

	versionParts := strings.Split(version, ".")

	if len(versionParts) > 2 {
		return ID{}, ParseError{ToParse: id, Msg: fmt.Sprintf("too many dots in the version (%d)", len(versionParts)-1)}
	}

	majorVersion, err := strconv.Atoi(versionParts[0])
	if err != nil {
		return ID{}, ParseError{ToParse: id, Msg: "parsing major version failed", Inner: err}
	}

	var minorVersion int

	if len(versionParts) > 1 {
		minorVersion, err = strconv.Atoi(versionParts[1])
		if err != nil {
			return ID{}, ParseError{ToParse: id, Msg: "parsing minor version failed", Inner: err}
		}
	}

	return ID{
		Name:         name,
		MajorVersion: majorVersion,
		MinorVersion: minorVersion,
	}, nil
}
