package types

import (
	sdkerrors "cosmossdk.io/errors"
)

// x/nameservice module sentinel errors
var (
	ErrInvalidName        = sdkerrors.Register("nameservice", 1, "invalid name")
	ErrInvalidOwner       = sdkerrors.Register("nameservice", 2, "invalid owner")
	ErrInvalidValuation   = sdkerrors.Register("nameservice", 3, "invalid valuation")
	ErrInvalidDuration    = sdkerrors.Register("nameservice", 4, "invalid duration")
	ErrMetadataTooLarge   = sdkerrors.Register("nameservice", 5, "metadata too large")
	ErrNameNotFound       = sdkerrors.Register("nameservice", 6, "name not found")
	ErrCommitmentNotFound = sdkerrors.Register("nameservice", 7, "commitment not found")
	ErrBidNotFound        = sdkerrors.Register("nameservice", 8, "bid not found")
	ErrUnauthorized       = sdkerrors.Register("nameservice", 9, "unauthorized")
	ErrInvalidBid         = sdkerrors.Register("nameservice", 10, "invalid bid")
)
