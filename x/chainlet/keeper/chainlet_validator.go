package keeper

import (
	cosmossdkerrors "cosmossdk.io/errors"
	"errors"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
	"github.com/sagaxyz/ssc/x/chainlet/types/versions"
	"slices"
)

const SagaAddress = "saga1h8r6gm4jehflfn2nn7mtw53l37skrke5kyax8l"

type ChainletActionsValidator struct {
	chainletRepo      ChainletRepository
	chainletStackRepo ChainletStackRepository
}

func NewChainletActionsValidator(cr ChainletRepository, sr ChainletStackRepository) ChainletActionsValidator {
	return ChainletActionsValidator{
		chainletRepo:      cr,
		chainletStackRepo: sr,
	}
}

func (f ChainletActionsValidator) ValidateChainletLaunch(ctx sdk.Context, ch types.Chainlet, p types.Params) error {
	numberOfChainlets := f.chainletRepo.GetChainletCount2(ctx)
	if numberOfChainlets >= p.MaxChainlets {
		return types.ErrTooManyChainlets
	}

	if _, err := f.chainletRepo.Chainlet(ctx, ch.ChainId); err == nil {
		return cosmossdkerrors.Wrapf(types.ErrChainletExists, "chainlet with chainId %s already exists", ch.ChainId)
	}

	return f.chainletStackRepo.chainletStackVersionAvailable(ctx, ch.ChainletStackName, ch.ChainletStackVersion)
}

func (f ChainletActionsValidator) ValidateChainletUpdate(ctx sdk.Context, ogChainlet types.Chainlet, creator, stackVersion string) error {
	if !slices.Contains(ogChainlet.Maintainers, creator) && creator != SagaAddress {
		return fmt.Errorf("address %s not whitelisted for creating or updating chainlet stacks", creator)
	}
	majorUpgrade, err := versions.CheckUpgrade(ogChainlet.ChainletStackVersion, stackVersion)
	if err != nil {
		return err
	}
	// Only this Saga-controlled address is allowed to perform (manual) major upgrades until they're automated using IBC
	if majorUpgrade && creator != SagaAddress {
		return errors.New("major upgrades not implemented")
	}

	return f.chainletStackRepo.chainletStackVersionAvailable(ctx, ogChainlet.ChainletStackName, stackVersion)
}
