package keeper

import (
	"context"
	"fmt"
)

type handlers struct {
	*Keeper
}

// NewHandlers creates a new handlers instance
func NewHandlers(k *Keeper) handlers {
	return handlers{k}
}
func (h handlers) BeforeGlobal(ctx context.Context, msg interface{}) error {
	fmt.Println("BeforeGlobal", msg)
	return nil
}

func (h handlers) AfterGlobal(ctx context.Context, msg, msgResp interface{}) error {
	fmt.Println("AfterGlobal", msg, msgResp)
	return nil
}
