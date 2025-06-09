package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	gogoprotoany "github.com/cosmos/gogoproto/types/any"
)

var (
	_ gogoprotoany.UnpackInterfacesMessage = (*Task)(nil)
	_ gogoprotoany.UnpackInterfacesMessage = (*MsgCreateTask)(nil)
)

// UnpackInterfaces implements the UnpackInterfacesMessage interface for Task
func (t Task) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	for i, anyMsg := range t.Msgs {
		var sdkMsg sdk.Msg
		err := unpacker.UnpackAny(anyMsg, &sdkMsg)
		if err != nil {
			return fmt.Errorf("failed to unpack msg at index %d: %w", i, err)
		}
	}

	return nil
}

// UnpackInterfaces implements the UnpackInterfacesMessage interface for MsgCreateTask
func (msg MsgCreateTask) UnpackInterfaces(unpacker gogoprotoany.AnyUnpacker) error {
	for i, anyMsg := range msg.Msgs {
		var sdkMsg sdk.Msg
		err := unpacker.UnpackAny(anyMsg, &sdkMsg)
		if err != nil {
			return fmt.Errorf("failed to unpack msg at index %d: %w", i, err)
		}
	}

	return nil
}

// GetMessages unpacks the Msgs into sdk.Msg's
func (task Task) GetMessages() ([]sdk.Msg, error) {
	return tx.GetMsgs(task.Msgs, "Task")
}

// GetMessageResults unpacks the MsgResults into sdk.Msg's
func (task Task) GetMessageResults() ([]sdk.Msg, error) {
	if len(task.MsgResults) == 0 {
		return []sdk.Msg{}, nil
	}

	results := make([]sdk.Msg, 0, len(task.MsgResults))
	for i, resultMsg := range task.MsgResults {
		if resultMsg != nil {
			cached := resultMsg.GetCachedValue()
			if cached == nil {
				return nil, fmt.Errorf("result at index %d has nil cached value", i)
			}
			if msg, ok := cached.(sdk.Msg); ok {
				results = append(results, msg)
			}
		}
	}
	return results, nil
}

// SetMessages sets the Msgs field by converting sdk.Msg to Any
func (task *Task) SetMessages(msgs []sdk.Msg) error {
	anys, err := tx.SetMsgs(msgs)
	if err != nil {
		return fmt.Errorf("failed to pack messages: %w", err)
	}
	task.Msgs = anys
	return nil
}

// SetMessageResults sets the MsgResults field by converting sdk.Msg to Any
func (task *Task) SetMessageResults(results []sdk.Msg) error {
	anys, err := tx.SetMsgs(results)
	if err != nil {
		return fmt.Errorf("failed to pack results: %w", err)
	}
	task.MsgResults = anys
	return nil
}

// SetMessages sets the Msgs field by converting sdk.Msg to Any
func (msg *MsgCreateTask) SetMessages(msgs []sdk.Msg) error {
	anys, err := tx.SetMsgs(msgs)
	if err != nil {
		return fmt.Errorf("failed to pack messages: %w", err)
	}
	msg.Msgs = anys
	return nil
}

// GetMessages unpacks the Msgs into sdk.Msg's
func (msg MsgCreateTask) GetMessages() ([]sdk.Msg, error) {
	return tx.GetMsgs(msg.Msgs, "MsgCreateTask")
}
