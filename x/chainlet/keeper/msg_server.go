package keeper

import (
	"context"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type msgServer struct {
	*Keeper
	launcher       ChainletLauncher
	msgValidator   ChainletValidator
	accountBilling AccountBilling
}

type ChainletValidator interface {
	ValidateLaunch(ctx context.Context, msg *types.MsgLaunchChainlet, p types.Params) error
}

type AccountBilling interface {
	BillAccount(ctx sdk.Context, chainlet types.Chainlet, p types.Params) error
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) LaunchChainlet2(goCtx context.Context, msg *types.MsgLaunchChainlet) (*types.MsgLaunchChainletResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := k.GetParams(ctx)
	if err := k.msgValidator.ValidateLaunch(ctx, msg, p); err != nil {
		return &types.MsgLaunchChainletResponse{}, err
	}

	chainlet, err := k.buildChain(ctx, msg, p)
	if err != nil {
		return &types.MsgLaunchChainletResponse{}, err
	}

	if err = k.launcher.LaunchChainlet(ctx, chainlet, p); err != nil {
		return &types.MsgLaunchChainletResponse{}, err
	}

	return &types.MsgLaunchChainletResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventLaunchChainlet{
		ChainName:    msg.ChainletName,
		Launcher:     msg.Creator,
		ChainId:      msg.ChainId,
		Stack:        msg.ChainletStackName,
		StackVersion: msg.ChainletStackVersion,
	})
}

func (k msgServer) buildChain(ctx sdk.Context, msg *types.MsgLaunchChainlet, p types.Params) (types.Chainlet, error) {
	for idx, bal := range msg.Params.GenAcctBalances.List {
		amount, err := math.ParseUint(bal.Balance + "000000000000000000")
		if err != nil {
			return types.Chainlet{}, err
		}
		if amount.IsZero() {
			msg.Params.GenAcctBalances.List = append(msg.Params.GenAcctBalances.List[:idx], msg.Params.GenAcctBalances.List[idx+1:]...)
			continue
		}
		msg.Params.GenAcctBalances.List[idx].Balance = amount.String()
	}

	return types.Chainlet{
		SpawnTime:            ctx.BlockTime().Add(p.LaunchDelay),
		Launcher:             msg.Creator,
		Maintainers:          msg.Maintainers,
		ChainletStackName:    msg.ChainletStackName,
		ChainletStackVersion: msg.ChainletStackVersion,
		ChainletName:         msg.ChainletName,
		ChainId:              msg.ChainId,
		Denom:                msg.Denom,
		Params:               msg.Params,
		Status:               types.Status_STATUS_ONLINE,
		AutoUpgradeStack:     !msg.DisableAutomaticStackUpgrades,
	}, nil
}
