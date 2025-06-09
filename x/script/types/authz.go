package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/x/authz"
)

var _ authz.Authorization = &ExecAuthorization{}

// NewExecAuthorization creates a new ExecAuthorization object.
func NewExecAuthorization(scriptAddress string, functionNames []string) *ExecAuthorization {
	return &ExecAuthorization{
		ScriptAddress: scriptAddress,
		FunctionNames: functionNames,
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a ExecAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgExec{})
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a ExecAuthorization) ValidateBasic() error {
	if a.ScriptAddress == "" {
		return sdkerrors.ErrInvalidAddress.Wrap("script address cannot be empty")
	}

	// Validate script address format
	_, err := sdk.AccAddressFromBech32(a.ScriptAddress)
	if err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid script address: %s", err)
	}

	// Check for duplicate function names
	seen := make(map[string]bool)
	for _, fn := range a.FunctionNames {
		if fn == "" {
			return sdkerrors.ErrInvalidRequest.Wrap("function name cannot be empty")
		}
		if seen[fn] {
			return sdkerrors.ErrInvalidRequest.Wrapf("duplicate function name: %s", fn)
		}
		seen[fn] = true
	}

	return nil
}

// Accept implements Authorization.Accept.
func (a ExecAuthorization) Accept(ctx context.Context, msg sdk.Msg) (authz.AcceptResponse, error) {
	execMsg, ok := msg.(*MsgExec)
	if !ok {
		return authz.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("type mismatch: expected MsgExec")
	}

	// Check if the script address matches
	if execMsg.ScriptAddress != a.ScriptAddress {
		return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf("script address mismatch: expected %s, got %s", a.ScriptAddress, execMsg.ScriptAddress)
	}

	// Direct execution (empty function name) is always allowed
	if execMsg.FunctionName == "" {
		return authz.AcceptResponse{Accept: true}, nil
	}

	// If a function name is specified, check if it's in the allowed list
	// Empty FunctionNames list means only direct execution is allowed
	if len(a.FunctionNames) == 0 {
		return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrap("function calls not authorized, only direct script execution allowed")
	}

	// Check if the function is in the allowed list
	isAllowed := false
	for _, allowedFn := range a.FunctionNames {
		if allowedFn == execMsg.FunctionName {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return authz.AcceptResponse{}, sdkerrors.ErrUnauthorized.Wrapf("function %s is not authorized for execution", execMsg.FunctionName)
	}

	// Accept the execution
	return authz.AcceptResponse{Accept: true}, nil
}
