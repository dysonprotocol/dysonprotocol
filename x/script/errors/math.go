package errors

import "cosmossdk.io/errors"

// mathCodespace is the codespace for all errors defined in math package
const mathCodespace = "script_math"

// ErrInvalidDecString defines an error for an invalid decimal string
var ErrInvalidDecString = errors.Register(mathCodespace, 10, "invalid decimal string")
