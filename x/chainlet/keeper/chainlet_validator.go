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
	aclKeeper         types.AclKeeper
}

func NewChainletActionsValidator(cr ChainletRepository, sr ChainletStackRepository, acl types.AclKeeper) ChainletActionsValidator {
	return ChainletActionsValidator{
		chainletRepo:      cr,
		chainletStackRepo: sr,
		aclKeeper:         acl,
	}
}

func (f ChainletActionsValidator) ValidateChainletLaunch(ctx sdk.Context, ch types.Chainlet, p types.Params) error {
	numberOfChainlets := f.chainletRepo.GetChainletCount2(ctx)
	if numberOfChainlets >= p.MaxChainlets {
		return types.ErrTooManyChainlets
	}

	if isExist := f.chainletRepo.ChainletExists(ctx, ch.ChainId); isExist {
		return cosmossdkerrors.Wrapf(types.ErrChainletExists, "chainlet with chainId %s already exists", ch.ChainId)
	}

	avail, err := f.chainletStackVersionAvailable(ctx, ch.ChainletStackName, ch.ChainletStackVersion)
	if err != nil {
		return cosmossdkerrors.Wrapf(types.ErrInvalidChainletStack, "cannot use stack %s version %s: %s", ch.ChainletStackName, ch.ChainletStackVersion, err)
	}
	if !avail {
		return cosmossdkerrors.Wrapf(types.ErrInvalidChainletStack, "stack %s version %s not available", ch.ChainletStackName, ch.ChainletStackVersion)
	}

	return nil
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

	avail, err := f.chainletStackVersionAvailable(ctx, ogChainlet.ChainletStackName, stackVersion)
	if err != nil {
		return cosmossdkerrors.Wrapf(types.ErrInvalidChainletStack, "cannot upgrade to stack %s version %s: %s", ogChainlet.ChainletStackName, stackVersion, err)
	}
	if !avail {
		return cosmossdkerrors.Wrapf(types.ErrInvalidChainletStack, "stack %s version %s not available", ogChainlet.ChainletStackName, ogChainlet.ChainletStackVersion)
	}
	return nil
}

func (f ChainletActionsValidator) chainletStackVersionAvailable(ctx sdk.Context, name, version string) (bool, error) {
	stack, err := f.chainletStackRepo.getChainletStack(ctx, name)
	if err != nil {
		return false, fmt.Errorf("cannot get chainlet stack with name %s: %w", name, err)
	}

	for _, v := range stack.Versions {
		if v.Version != version {
			continue
		}
		if !v.Enabled {
			return false, nil
		}
		return true, nil
	}

	return false, fmt.Errorf("stack version %s is not found", version)
}
