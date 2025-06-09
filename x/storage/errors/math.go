package errors

import "cosmossdk.io/errors"

// mathCodespace is the codespace for all errors defined in math package
const mathCodespace = "storage_math"

// ErrInvalidDecString defines an error for an invalid decimal string
var ErrInvalidDecString = errors.Register(mathCodespace, 10, "invalid decimal string")
