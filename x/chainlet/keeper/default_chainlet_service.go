package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
	"time"
)

// ChainletValidator validate that the chainlet can be launched and all
// the data, which is provided, is valid.
type ChainletValidator interface {
	ValidateChainletStackCreation(sdk.Context, types.ChainletStack) error
	ValidateUpdateChainletStack(stack types.ChainletStack, params types.ChainletStackParams) error
	ValidateChainletLaunch(sdk.Context, types.Chainlet, types.Params) error
	ValidateChainletUpdate(ctx sdk.Context, chainlet types.Chainlet, creator, stackVersion string) error
}

// CCVConsumerRegisterer register the chainlet as a consumer in CCV
type CCVConsumerRegisterer interface {
	RegisterChainletAsConsumerInCCV(ctx sdk.Context, chainId string, spawnTime time.Time) error
}

// AccountService is responsible for creating and bill a new account when a new chainlet is launched.
type AccountService interface {
	CreateNewAccount(sdk.Context, types.Chainlet, types.Params) (Account, error)
	BillAccount(sdk.Context, Account) error
}

type ChainletRepository interface {
	GetChainletCount2(ctx sdk.Context) uint64
	ChainletExists(ctx sdk.Context, chainId string) bool
	UpgradeChainlet2(ctx sdk.Context, ch types.Chainlet) error
	Chainlet(ctx sdk.Context, chainId string) (chainlet types.Chainlet, err error)
	Create(sdk.Context, types.Chainlet) error
}

type ChainletStackRepository interface {
	ChainletStackExist(ctx sdk.Context, displayName string) bool
	getChainletStack(ctx sdk.Context, name string) (stack types.ChainletStack, err error)
	CreateChainletStack(ctx sdk.Context, cs types.ChainletStack) error
	DisableChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version string) error
	AddChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version types.ChainletStackParams) error
}

type DefaultChainletService struct {
	validator      ChainletValidator
	repo           ChainletRepository
	stackRepo      ChainletStackRepository
	accountService AccountService
	ccvRegisterer  CCVConsumerRegisterer
}

func NewDefaultChainletService(v ChainletValidator, cr ChainletRepository, sr ChainletStackRepository,
	as AccountService, ccv CCVConsumerRegisterer) DefaultChainletService {
	return DefaultChainletService{
		validator:      v,
		repo:           cr,
		stackRepo:      sr,
		accountService: as,
		ccvRegisterer:  ccv,
	}
}

func (c DefaultChainletService) CreateChainletStack(ctx sdk.Context, stack types.ChainletStack) error {
	if err := c.validator.ValidateChainletStackCreation(ctx, stack); err != nil {
		return err
	}

	return c.stackRepo.CreateChainletStack(ctx, stack)
}

func (c DefaultChainletService) AddChainletStackVersion(ctx sdk.Context, stackName string, version types.ChainletStackParams) error {
	stack, err := c.stackRepo.getChainletStack(ctx, stackName)
	if err != nil {
		return fmt.Errorf("cannot get chainlet stack %s: %w", stackName, err)
	}

	if err = c.validator.ValidateUpdateChainletStack(stack, version); err != nil {
		return err
	}

	stack.Versions = append(stack.Versions, version)
	return c.stackRepo.AddChainletStackVersion2(ctx, stack, version)
}

func (c DefaultChainletService) DisableChainletStackVersion(ctx sdk.Context, dchsv DisableChainletStackVersion) error {
	stack, err := c.stackRepo.getChainletStack(ctx, dchsv.DisplayName)
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

	return c.stackRepo.DisableChainletStackVersion2(ctx, stack, dchsv.Version)
}

func (c DefaultChainletService) LaunchChainlet(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error {
	if err := c.validator.ValidateChainletLaunch(ctx, chainlet, p); err != nil {
		return err
	}

	account, err := c.accountService.CreateNewAccount(ctx, chainlet, p)
	if err != nil {
		return err
	}

	if err = c.accountService.BillAccount(ctx, account); err != nil {
		return err
	}

	if err = c.ccvRegisterer.RegisterChainletAsConsumerInCCV(ctx, chainlet.ChainId, chainlet.SpawnTime); err != nil {
		return err
	}

	return c.repo.Create(ctx, chainlet)
}

func (c DefaultChainletService) UpdateChainletVersion(ctx sdk.Context, chainId, creator, stackVersion string) error {
	chainlet, err := c.repo.Chainlet(ctx, chainId)
	if err != nil {
		return err
	}

	if err = c.validator.ValidateChainletUpdate(ctx, chainlet, creator, stackVersion); err != nil {
		return err
	}

	chainlet.ChainletStackVersion = stackVersion
	if err = c.repo.UpgradeChainlet2(ctx, chainlet); err != nil {
		return fmt.Errorf("error while updating chainlet: %s", err)
	}

	return nil
}

type DisableChainletStackVersion struct {
	Creator     string
	DisplayName string
	Version     string
}
