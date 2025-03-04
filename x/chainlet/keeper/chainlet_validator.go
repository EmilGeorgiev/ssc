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

type Foo struct {
	k            Keeper
	chainletRepo ChainletRepository
	aclKeeper    types.AclKeeper
}

type ChainletRepository interface {
	GetChainletCount2(ctx sdk.Context) uint64
	ChainletExists(ctx sdk.Context, chainId string) bool
	ChainletStackExist(ctx sdk.Context, displayName string) bool
	getChainletStack(ctx sdk.Context, name string) (stack types.ChainletStack, err error)
	CreateChainletStack(ctx sdk.Context, cs types.ChainletStack) error
	DisableChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version string) error
	AddChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version types.ChainletStackParams) error
	UpgradeChainlet2(ctx sdk.Context, ch types.Chainlet) error
	Chainlet(ctx sdk.Context, chainId string) (chainlet types.Chainlet, err error)
}

type ChainletStackRepository interface {
	getChainletStack(ctx sdk.Context, name string) (stack types.ChainletStack, err error)
}

func (f *Foo) ValidateChainletLaunch(ctx sdk.Context, ch types.Chainlet, p types.Params) error {
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

func (f *Foo) ValidateChainletStackCreation(ctx sdk.Context, stack types.ChainletStack, p types.Params) error {
	if p.ChainletStackProtections {
		addr, err := sdk.AccAddressFromBech32(stack.Creator)
		if err != nil {
			return err
		}
		if !f.aclKeeper.Allowed(ctx, addr) {
			return fmt.Errorf("address %s not allowed to create chainlet stacks", stack.Creator)
		}
	}

	for _, version := range stack.Versions {
		if !versions.Check(version.Version) {
			return fmt.Errorf("version string '%s' invalid", version.Version)
		}
	}

	if isExists := f.chainletRepo.ChainletStackExist(ctx, stack.DisplayName); isExists {
		// cannot add a duplicate chainlet stack so return an error
		return fmt.Errorf("cannot add chainlet stack %v as it already exists", stack.DisplayName)
	}

	return nil
}

func (f *Foo) ValidateDisableChainletStackVersion(ctx sdk.Context, creator string, p types.Params) error {
	if p.ChainletStackProtections {
		//var addr sdk.AccAddress
		addr, err := sdk.AccAddressFromBech32(creator)
		if err != nil {
			return err
		}
		if !f.aclKeeper.Allowed(ctx, addr) {
			return fmt.Errorf("address %s not allowed to disable chainlet stacks", creator)
		}
	}
	return nil
}

func (f Foo) ValdiateUpdateChainletStack(ctx sdk.Context, creator, stackName string, version types.ChainletStackParams, p types.Params) error {
	// validate auth
	if p.ChainletStackProtections {
		//var addr sdk.AccAddress
		addr, err := sdk.AccAddressFromBech32(creator)
		if err != nil {
			return err
		}
		if !f.aclKeeper.Allowed(ctx, addr) {
			return fmt.Errorf("address %s not allowed to disable chainlet stacks", creator)
		}
	}

	stack, err := f.chainletRepo.getChainletStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("cannot get chainlet stack %s: %w", stackName, err)
	}

	// Validate that the incoming fields can be updated
	err = validateUpdate(stack, version)
	if err != nil {
		return fmt.Errorf("cannot update chainlet stack %s: %w", stackName, err)
	}
	return nil
}

func (f Foo) ValidateChailetUpdate(ctx sdk.Context, chainId, creator, stackVersion string) error {
	ogChainlet, err := f.chainletRepo.Chainlet(ctx, chainId)
	if err != nil {
		return err
	}

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

func (f *Foo) chainletStackVersionAvailable(ctx sdk.Context, name, version string) (bool, error) {
	stack, err := f.chainletRepo.getChainletStack(ctx, name)
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
