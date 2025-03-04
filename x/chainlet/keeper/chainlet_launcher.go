package keeper

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

// ChainletValidator validate that the chainlet can be launched and all
// the data, which is provided, is valid.
type ChainletValidator interface {
	ValidateChainletLaunch(sdk.Context, types.Chainlet, types.Params) error
	ValidateChainletStackCreation(sdk.Context, types.ChainletStack, types.Params) error
	ValidateDisableChainletStackVersion(ctx sdk.Context, creator string, p types.Params) error
	ValdiateUpdateChainletStack(ctx sdk.Context, creator, stackName string, params types.ChainletStackParams, p types.Params) error
	ValidateChailetUpdate(ctx sdk.Context, chainId, creator, stackVersion string) error
}

// AccountService is responsible for creating and bill a new account when a new chainlet is launched.
type AccountService interface {
	CreateNewAccount(sdk.Context, types.Chainlet, types.Params) (Account, error)
	BillAccount(sdk.Context, Account) error
}

// CCVConsumerRegisterer register the chainlet as a consumer in CCV
type CCVConsumerRegisterer interface {
	RegisterChainletAsConsumerInCCV(ctx sdk.Context, chainId string, spawnTime time.Time) error
}

type ChainletRepo interface {
	Create(sdk.Context, types.Chainlet) error
}

type ChainLauncherImplementation struct {
	Keeper
	accountService    AccountService
	chainletValidator ChainletValidator
	ccvRegisterer     CCVConsumerRegisterer
	chainletRepo      ChainletRepo
}

func (chl ChainLauncherImplementation) Launch(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error {
	if err := chl.chainletValidator.ValidateChainletLaunch(ctx, chainlet, p); err != nil {
		return err
	}

	account, err := chl.accountService.CreateNewAccount(ctx, chainlet, p)
	if err != nil {
		return err
	}

	if err = chl.accountService.BillAccount(ctx, account); err != nil {
		return err
	}

	if err = chl.ccvRegisterer.RegisterChainletAsConsumerInCCV(ctx, chainlet.ChainId, chainlet.SpawnTime); err != nil {
		return err
	}

	return chl.chainletRepo.Create(ctx, chainlet)
}
