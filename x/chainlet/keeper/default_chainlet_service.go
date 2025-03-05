package keeper

import (
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
	"github.com/sagaxyz/ssc/x/chainlet/types/versions"
	"time"
)

// ChainletValidator validate that the chainlet can be launched and all
// the data, which is provided, is valid.
type ChainletValidator interface {
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
	Chainlet(ctx sdk.Context, chainId string) (chainlet types.Chainlet, err error)
	Create(sdk.Context, types.Chainlet) error
	setChainletInfo(ctx sdk.Context, chainlet *types.Chainlet)
}

type ChainletStackRepository interface {
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
	for _, version := range stack.Versions {
		if !versions.Check(version.Version) {
			return fmt.Errorf("version string '%s' invalid", version.Version)
		}
	}

	if _, err := c.stackRepo.getChainletStack(ctx, stack.DisplayName); err == nil {
		return fmt.Errorf("cannot add chainlet stack %v as it already exists", stack.DisplayName)
	}

	return c.stackRepo.CreateChainletStack(ctx, stack)
}

func (c DefaultChainletService) AddChainletStackVersion(ctx sdk.Context, stackName string, version types.ChainletStackParams) error {
	stack, err := c.stackRepo.getChainletStack(ctx, stackName)
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
	c.repo.setChainletInfo(ctx, &chainlet)
	return nil
}

type DisableChainletStackVersion struct {
	Creator     string
	DisplayName string
	Version     string
}
