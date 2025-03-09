package keeper

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/sagaxyz/ssc/x/chainlet/types"
	"github.com/sagaxyz/ssc/x/chainlet/types/versions"

	cosmossdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const SagaAddress = "saga1h8r6gm4jehflfn2nn7mtw53l37skrke5kyax8l"

// CCVConsumerRegisterer register the chainlet as a consumer in CCV
type CCVConsumerRegisterer interface {
	registerChainletAsConsumerInCCV(ctx sdk.Context, chainId string, spawnTime time.Time) error
}

// AccountService is responsible for creating and bill a new account when a new chainlet is launched.
type AccountService interface {
	CreateNewAccount(sdk.Context, types.Chainlet, types.Params) (Account, error)
	//BillAccount(sdk.Context, Account) error
}

type ChainletRepository interface {
	FetchChainletCount(ctx sdk.Context) uint64
	FetchChainlet(ctx sdk.Context, chainId string) (chainlet types.Chainlet, err error)
	Create(sdk.Context, types.Chainlet) error
	setChainletInfo(ctx sdk.Context, chainlet types.Chainlet)
}

type ChainletStackRepository interface {
	FetchChainletStack(ctx sdk.Context, name string) (stack types.ChainletStack, err error)
	chainletStackVersionAvailable(ctx sdk.Context, name, version string) error
	CreateChainletStack(ctx sdk.Context, cs types.ChainletStack) error
	DisableChainletStackVersion(ctx sdk.Context, stack types.ChainletStack, version string) error
	AddChainletStackVersion(ctx sdk.Context, stack types.ChainletStack, version types.ChainletStackParams) error
}

type DefaultChainletService struct {
	repo           ChainletRepository
	stackRepo      ChainletStackRepository
	accountService AccountService
	ccvRegisterer  CCVConsumerRegisterer
	accountCharge  AccountCharge
}

func NewDefaultChainletService(cr ChainletRepository, sr ChainletStackRepository,
	as AccountService, ccv CCVConsumerRegisterer, ach AccountCharge) DefaultChainletService {
	return DefaultChainletService{
		repo:           cr,
		stackRepo:      sr,
		accountService: as,
		ccvRegisterer:  ccv,
		accountCharge:  ach,
	}
}

func (c DefaultChainletService) CreateChainletStack(ctx sdk.Context, stack types.ChainletStack) error {
	for _, version := range stack.Versions {
		if !versions.Check(version.Version) {
			return fmt.Errorf("version string '%s' invalid", version.Version)
		}
	}

	if _, err := c.stackRepo.FetchChainletStack(ctx, stack.DisplayName); err == nil {
		return fmt.Errorf("cannot add chainlet stack %v as it already exists", stack.DisplayName)
	}

	return c.stackRepo.CreateChainletStack(ctx, stack)
}

func (c DefaultChainletService) AddChainletStackVersion(ctx sdk.Context, stackName string, version types.ChainletStackParams) error {
	stack, err := c.stackRepo.FetchChainletStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("cannot get chainlet stack %s: %w", stackName, err)
	}

	for _, v := range stack.Versions {
		if v.Image == version.Image || v.Version == version.Version || v.Checksum == version.Checksum {
			return fmt.Errorf("cannot update chainlet stack %s with duplicate values for image, version, or checksum", stack.DisplayName)
		}
	}

	if !versions.Check(version.Version) {
		return fmt.Errorf("cannot update chainlet stack %s because version string '%s' invalid", stack.DisplayName, version.Version)
	}

	stack.Versions = append(stack.Versions, version)
	return c.stackRepo.AddChainletStackVersion(ctx, stack, version)
}

func (c DefaultChainletService) DisableChainletStackVersion(ctx sdk.Context, dchsv DisableChainletStackVersion) error {
	stack, err := c.stackRepo.FetchChainletStack(ctx, dchsv.DisplayName)
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

	return c.stackRepo.DisableChainletStackVersion(ctx, stack, dchsv.Version)
}

func (c DefaultChainletService) LaunchChainlet(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error {
	numberOfChainlets := c.repo.FetchChainletCount(ctx)
	if numberOfChainlets >= p.MaxChainlets {
		return types.ErrTooManyChainlets
	}

	if _, err := c.repo.FetchChainlet(ctx, chainlet.ChainId); err == nil {
		return cosmossdkerrors.Wrapf(types.ErrChainletExists, "chainlet with chainId %s already exists", chainlet.ChainId)
	}

	if err := c.stackRepo.chainletStackVersionAvailable(ctx, chainlet.ChainletStackName, chainlet.ChainletStackVersion); err != nil {
		return err
	}

	account, err := c.accountService.CreateNewAccount(ctx, chainlet, p)
	if err != nil {
		return err
	}

	if err = c.accountCharge.ApplyTo(ctx, account); err != nil {
		return err
	}

	if err = c.ccvRegisterer.registerChainletAsConsumerInCCV(ctx, chainlet.ChainId, chainlet.SpawnTime); err != nil {
		return err
	}

	return c.repo.Create(ctx, chainlet)
}

func (c DefaultChainletService) UpdateChainletVersion(ctx sdk.Context, chainId, creator, stackVersion string) error {
	chainlet, err := c.repo.FetchChainlet(ctx, chainId)
	if err != nil {
		return err
	}

	if !slices.Contains(chainlet.Maintainers, creator) && creator != SagaAddress {
		return fmt.Errorf("address %s not whitelisted for creating or updating chainlet stacks", creator)
	}
	majorUpgrade, err := versions.CheckUpgrade(chainlet.ChainletStackVersion, stackVersion)
	if err != nil {
		return err
	}
	// Only this Saga-controlled address is allowed to perform (manual) major upgrades until they're automated using IBC
	if majorUpgrade && creator != SagaAddress {
		return errors.New("major upgrades not implemented")
	}

	if err = c.stackRepo.chainletStackVersionAvailable(ctx, chainlet.ChainletStackName, stackVersion); err != nil {
		return err
	}

	chainlet.ChainletStackVersion = stackVersion
	c.repo.setChainletInfo(ctx, chainlet)
	return nil
}

type DisableChainletStackVersion struct {
	Creator     string
	DisplayName string
	Version     string
}
