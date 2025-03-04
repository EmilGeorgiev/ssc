package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type ChainletStackService struct {
	validator ChainletValidator
	repo      ChainletRepository
}

func (c ChainletStackService) Create(ctx sdk.Context, stack types.ChainletStack, p types.Params) error {
	if err := c.validator.ValidateChainletStackCreation(ctx, stack, p); err != nil {
		return err
	}

	return c.repo.CreateChainletStack(ctx, stack)
}

type DisableChainletStackVersion struct {
	Creator     string
	DisplayName string
	Version     string
}

func (c ChainletStackService) DisableChainletStackVersion(ctx sdk.Context, dchsv DisableChainletStackVersion, p types.Params) error {
	if err := c.validator.ValidateDisableChainletStackVersion(ctx, dchsv.Creator, p); err != nil {
		return err
	}

	stack, err := c.repo.getChainletStack(ctx, dchsv.DisplayName)
	if err != nil {
		return fmt.Errorf("cannot get chainlet stack %s: %w", dchsv.DisplayName, err)
	}

	var found bool
	for i, version := range stack.Versions {
		if version.Version != dchsv.Version {
			continue
		}

		if !version.Enabled {
			return nil // Already disabled
		}
		stack.Versions[i].Enabled = false
		found = true
		break
	}
	if !found {
		return fmt.Errorf("cannot find chainlet stack %s version %s", dchsv.DisplayName, dchsv.Version)
	}

	return c.repo.DisableChainletStackVersion2(ctx, stack, dchsv.Version)
}

func (c ChainletStackService) UpdateChainletStackVersion(ctx sdk.Context, creator, stackName string, version types.ChainletStackParams, p types.Params) error {
	if err := c.validator.ValdiateUpdateChainletStack(ctx, creator, stackName, version, p); err != nil {
		return err
	}

	stack, err := c.repo.getChainletStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("cannot get chainlet stack %s: %w", stackName, err)
	}

	stack.Versions = append(stack.Versions, version)

	return c.repo.AddChainletStackVersion2(ctx, stack, version)
}

func (c ChainletStackService) UpdateChainlet(ctx sdk.Context, chainId, creator, stackVersion string) error {
	if err := c.validator.ValidateChailetUpdate(ctx, chainId, creator, stackVersion); err != nil {
		return err
	}

	chainlet, err := c.repo.Chainlet(ctx, chainId)
	if err != nil {
		return err
	}

	chainlet.ChainletStackVersion = stackVersion
	if err = c.repo.UpgradeChainlet2(ctx, chainlet); err != nil {
		return fmt.Errorf("error while updating chainlet: %s", err)
	}

	return nil
}
