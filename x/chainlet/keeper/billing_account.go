package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	"github.com/sagaxyz/ssc/x/chainlet/types"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type ChainletRepository interface {
	GetChainletStack(goCtx context.Context, req *types.QueryGetChainletStackRequest) (*types.QueryGetChainletStackResponse, error)
}

type FooBillingAccount struct {
	repo          ChainletRepository
	escrowKeeper  types.EscrowKeeper
	billingKeeper types.BillingKeeper
}

func (b FooBillingAccount) CreateNewAccount(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error {
	stack, err := b.repo.GetChainletStack(ctx.Context(), &types.QueryGetChainletStackRequest{DisplayName: chainlet.ChainletStackName})
	if err != nil {
		return types.ErrInvalidChainletStack
	}

	epochfee, err := sdk.ParseCoinNormalized(stack.ChainletStack.Fees.EpochFee)
	if err != nil {
		return types.ErrInvalidCoin
	}

	multiplier, ok := math.NewIntFromString(p.NEpochDeposit)
	if !ok {
		return fmt.Errorf("bad multiplier")

	}
	deposit := sdk.Coin{
		Amount: epochfee.Amount.Mul(multiplier),
		Denom:  epochfee.Denom,
	}

	owner, err := sdk.AccAddressFromBech32(chainlet.Launcher)
	if err != nil {
		return err
	}
	return b.escrowKeeper.NewChainletAccount(ctx, owner, chainlet.ChainId, deposit)

}

func (b FooBillingAccount) BillAccount(ctx sdk.Context, chainlet types.Chainlet) error {
	stack, err := b.repo.GetChainletStack(ctx.Context(), &types.QueryGetChainletStackRequest{DisplayName: chainlet.ChainletStackName})
	if err != nil {
		return types.ErrInvalidChainletStack
	}

	epochfee, err := sdk.ParseCoinNormalized(stack.ChainletStack.Fees.EpochFee)
	if err != nil {
		return types.ErrInvalidCoin
	}
	setupfee, err := sdk.ParseCoinNormalized(stack.ChainletStack.Fees.SetupFee)
	if err != nil {
		return types.ErrInvalidCoin
	}

	totalFee := epochfee.Add(setupfee)

	err = b.billingKeeper.BillAccount(ctx, totalFee, chainlet, stack.ChainletStack.Fees.EpochLength, "launching chainlet")
	if err != nil {
		return cosmossdkerrors.Wrapf(types.ErrBillingFailure, "failed to bill new account %s", err.Error())
	}
	return nil
}

//func (b FooBillingAccount) getFee(epochFee, setupFee string, p types.Params) (deposit sdk.Coin, totalFee sdk.Coin, err error) {
//	epochfee, err := sdk.ParseCoinNormalized(epochFee)
//	if err != nil {
//		err = types.ErrInvalidCoin
//		return
//	}
//	setupfee, err := sdk.ParseCoinNormalized(setupFee)
//	if err != nil {
//		err = types.ErrInvalidCoin
//		return
//	}
//
//	multiplier, ok := math.NewIntFromString(p.NEpochDeposit)
//	if !ok {
//		err = fmt.Errorf("bad multiplier")
//		return
//	}
//
//	deposit = sdk.Coin{
//		Amount: epochfee.Amount.Mul(multiplier),
//		Denom:  epochfee.Denom,
//	}
//	deposit.Add(setupfee)
//	totalFee = epochfee.Add(setupfee)
//	return
//}
