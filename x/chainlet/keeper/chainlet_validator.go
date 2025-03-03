package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type Foo struct {
	r ChainletCounterFetcher
}
type ChainletCounterFetcher interface {
	GetChainletCount(ctx context.Context) (uint64, error)
	ChainletExists(ctx context.Context, chainId string) bool
}

func (f *Foo) ValidateLaunch(ctx sdk.Context, msg *types.MsgLaunchChainlet, p types.Params) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	numberOfChainlets, err := f.r.GetChainletCount(ctx)
	if err != nil {
		return err
	}

	if numberOfChainlets >= p.MaxChainlets {
		return types.ErrTooManyChainlets
	}

	if f.r.ChainletExists(ctx, msg.ChainId) {
		return types.ErrChainletExists
	}
	return nil
}
