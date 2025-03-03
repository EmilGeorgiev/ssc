package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type ChainletValidator interface {
	ValidateLaunch(ctx context.Context, ch types.Chainlet, p types.Params) error
}

type AccountBilling interface {
	BillAccount(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error
}

type ChainLauncherImplementation struct {
	Keeper
	accountBilling AccountBilling
	validator      ChainletValidator
}

func (chl ChainLauncherImplementation) LaunchChainlet(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error {
	if err := chl.validator.ValidateLaunch(ctx, chainlet, p); err != nil {
		return err
	}

	acc, err := chl.accountBilling.CreateNewAccount(ctx, chainlet, p)
	if err != nil {
		return err
	}

	if err = chl.accountBilling.BillAccount(ctx, chainlet, p); err != nil {
		return err
	}

	// Add as a CCV consumer
	// Registers the chainlet as a consumer in the cross-chain validation (CCV)
	// if err := chl.RegisterChainletAsConsumerInCCV(ctx, chainlet.ChainId, chainlet.SpawnTime); err != nil {
	//		return err
	// }
	if err := chl.addConsumer(ctx, chainlet.ChainId, chainlet.SpawnTime); err != nil {
		return err
	}

	// chl.CreateNewChainlet(ctx, chainlet)
	return chl.NewChainlet(ctx, chainlet)
}
