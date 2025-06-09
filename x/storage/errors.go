package storage

import (
	"cosmossdk.io/errors"
)

var (
	// ErrInvalidOwner is returned when the owner address is invalid
	ErrInvalidOwner = errors.Register("storage", 1, "invalid owner address")

	// ErrEmptyIndex is returned when the index is empty
	ErrEmptyIndex = errors.Register("storage", 2, "index cannot be empty")
)
