package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/sagaxyz/ssc/x/chainlet/types"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type DefaultBillingAccount struct {
	repo          ChainletStackRepository
	escrowKeeper  types.EscrowKeeper
	billingKeeper types.BillingKeeper
}

func NewDefaultBillingAccount(r ChainletStackRepository, ek types.EscrowKeeper, bk types.BillingKeeper) DefaultBillingAccount {
	return DefaultBillingAccount{
		repo:          r,
		escrowKeeper:  ek,
		billingKeeper: bk,
	}
}

func (b DefaultBillingAccount) CreateNewAccount(ctx sdk.Context, chainlet types.Chainlet, p types.Params) (acc Account, err error) {
	stack, err := b.repo.FetchChainletStack(ctx, chainlet.ChainletStackName)
	if err != nil {
		err = types.ErrInvalidChainletStack
		return
	}

	epochfee, err := sdk.ParseCoinNormalized(stack.Fees.EpochFee)
	if err != nil {
		err = types.ErrInvalidCoin
		return
	}

	multiplier, ok := math.NewIntFromString(p.NEpochDeposit)
	if !ok {
		err = fmt.Errorf("bad multiplier")
		return
	}
	deposit := sdk.Coin{
		Amount: epochfee.Amount.Mul(multiplier),
		Denom:  epochfee.Denom,
	}

	owner, err := sdk.AccAddressFromBech32(chainlet.Launcher)
	if err != nil {
		return
	}
	if err = b.escrowKeeper.NewChainletAccount(ctx, owner, chainlet.ChainId, deposit); err != nil {
		return
	}

	return Account{chainlet: chainlet, stack: stack}, nil

}

// Account keeps infromation need for creating and billing account. It is used to reduce duplication
type Account struct {
	chainlet types.Chainlet
	stack    types.ChainletStack
}

func (b DefaultBillingAccount) BillAccount(ctx sdk.Context, account Account) error {
	epochfee, err := sdk.ParseCoinNormalized(account.stack.Fees.EpochFee)
	if err != nil {
		return types.ErrInvalidCoin
	}
	setupfee, err := sdk.ParseCoinNormalized(account.stack.Fees.SetupFee)
	if err != nil {
		return types.ErrInvalidCoin
	}

	totalFee := epochfee.Add(setupfee)

	err = b.billingKeeper.BillAccount(ctx, totalFee, account.chainlet, account.stack.Fees.EpochLength, "launching chainlet")
	if err != nil {
		return cosmossdkerrors.Wrapf(types.ErrBillingFailure, "failed to bill new account %s", err.Error())
	}
	return nil
}
