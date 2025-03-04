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

type ChainletActionsValidator struct {
	k                 Keeper
	chainletRepo      ChainletRepository
	chainletStackRepo ChainletStackRepository
	aclKeeper         types.AclKeeper
}

func (f ChainletActionsValidator) ValidateChainletStackCreation(ctx sdk.Context, stack types.ChainletStack) error {
	for _, version := range stack.Versions {
		if !versions.Check(version.Version) {
			return fmt.Errorf("version string '%s' invalid", version.Version)
		}
	}

	if isExists := f.chainletStackRepo.ChainletStackExist(ctx, stack.DisplayName); isExists {
		// cannot add a duplicate chainlet stack so return an error
		return fmt.Errorf("cannot add chainlet stack %v as it already exists", stack.DisplayName)
	}

	return nil
}

func (f ChainletActionsValidator) ValidateUpdateChainletStack(stack types.ChainletStack, version types.ChainletStackParams) error {
	// Validate that the incoming fields can be updated
	if err := validateUpdate(stack, version); err != nil {
		return fmt.Errorf("cannot update chainlet stack %s: %w", stack.DisplayName, err)
	}
	return nil
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
