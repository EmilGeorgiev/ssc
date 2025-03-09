package keeper

import (
	"fmt"

	cosmossdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type AccountInitializer struct {
	repo         ChainletStackRepository
	escrowKeeper types.EscrowKeeper
}

func NewAccountInitializer(r ChainletStackRepository, ek types.EscrowKeeper) AccountInitializer {
	return AccountInitializer{
		repo:         r,
		escrowKeeper: ek,
	}
}

func (b AccountInitializer) CreateNewAccount(ctx sdk.Context, chainlet types.Chainlet, p types.Params) (acc Account, err error) {
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

// AccountCharge defines an interface for applying a billing charge to an account. This abstraction allows for different billing strategies.
//
// TODO: we charging an account when a chainlet is launched, restarted, or before each epoch—to be. This interface can
//
//	have multiple implementations for every case  and may eventually be moved into the billing module.
type AccountCharge interface {
	ApplyTo(sdk.Context, Account) error
}

type LaunchChainletFee struct {
	billingKeeper types.BillingKeeper
}

func NewLaunchChainletFee(billingKeeper types.BillingKeeper) LaunchChainletFee {
	return LaunchChainletFee{billingKeeper: billingKeeper}
}

func (lc LaunchChainletFee) ApplyTo(ctx sdk.Context, acc Account) error {
	epochfee, err := sdk.ParseCoinNormalized(acc.stack.Fees.EpochFee)
	if err != nil {
		return types.ErrInvalidCoin
	}
	setupfee, err := sdk.ParseCoinNormalized(acc.stack.Fees.SetupFee)
	if err != nil {
		return types.ErrInvalidCoin
	}

	totalFee := epochfee.Add(setupfee)

	err = lc.billingKeeper.BillAccount(ctx, totalFee, acc.chainlet, acc.stack.Fees.EpochLength, "launching chainlet")
	if err != nil {
		return cosmossdkerrors.Wrapf(types.ErrBillingFailure, "failed to bill new account %s", err.Error())
	}
	return nil
}

// Account keeps information need for creating and billing account. It is used to reduce duplication
type Account struct {
	chainlet types.Chainlet
	stack    types.ChainletStack
}
