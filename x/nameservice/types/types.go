package types

import (
	"regexp"

	sdkerrors "cosmossdk.io/errors"
)

var (
	// NameRegex defines the regex for valid name strings
	// Must be lowercase alphanumeric, start with a letter, may contain dashes, and must end with ".dys"
	NameRegex = regexp.MustCompile("^[a-z][a-z0-9-]*\\.dys$")
)

// ValidateBasic performs basic validation of NFTData
func (d *NFTData) ValidateBasic() error {

	// Validate metadata size if present
	if len(d.Metadata) > 1024 { // 1KB limit
		return ErrMetadataTooLarge.Wrap("metadata exceeds 1KB limit")
	}

	return nil
}

// ValidateName validates a name string
func ValidateName(name string) error {
	if len(name) == 0 {
		return sdkerrors.Wrap(ErrInvalidName, "name cannot be empty")
	}

	// Check name format
	if !NameRegex.MatchString(name) {
		return sdkerrors.Wrap(ErrInvalidName, "invalid name format: must be lowercase, start with a letter, contain only alphanumeric and dash characters, and end with .dys")
	}

	return nil
}
