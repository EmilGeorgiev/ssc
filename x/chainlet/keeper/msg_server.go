package keeper

import (
	"context"
	"cosmossdk.io/math"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sagaxyz/ssc/x/chainlet/types"
)

type ChainletService interface {
	CreateChainletStack(sdk.Context, types.ChainletStack) error
	AddChainletStackVersion(ctx sdk.Context, stackName string, version types.ChainletStackParams) error
	DisableChainletStackVersion(sdk.Context, DisableChainletStackVersion) error
	LaunchChainlet(sdk.Context, types.Chainlet, types.Params) error
	UpdateChainletVersion(ctx sdk.Context, chainId, creator, stackVersion string) error
}

type msgServer struct {
	*Keeper
	chainletService ChainletService
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper *Keeper, s ChainletService) types.MsgServer {
	return &msgServer{Keeper: keeper, chainletService: s}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CreateChainletStack(goCtx context.Context, msg *types.MsgCreateChainletStack) (*types.MsgCreateChainletStackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.validateMsgAndACL(ctx, msg.Creator, msg); err != nil {
		return &types.MsgCreateChainletStackResponse{}, err
	}

	metaData := types.ChainletStackParams{
		Image:    msg.Image,
		Version:  msg.Version,
		Checksum: msg.Checksum,
		Enabled:  true,
	}
	metaDataUpsert := []types.ChainletStackParams{metaData}
	chainletStack := types.ChainletStack{
		Creator:     msg.Creator,
		DisplayName: msg.DisplayName,
		Description: msg.Description,
		Versions:    metaDataUpsert,
		Fees:        msg.Fees,
	}
	if err := k.chainletService.CreateChainletStack(ctx, chainletStack); err != nil {
		return nil, fmt.Errorf("error while adding chainlet stack: %s", err)
	}

	return &types.MsgCreateChainletStackResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventNewChainletStack{
		Creator: msg.Creator,
		Name:    msg.DisplayName,
		Version: msg.Version,
	})
}

func (k msgServer) DisableChainletStackVersion(goCtx context.Context, msg *types.MsgDisableChainletStackVersion) (resp *types.MsgDisableChainletStackVersionResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err = k.validateMsgAndACL(ctx, msg.Creator, msg); err != nil {
		return
	}

	v := DisableChainletStackVersion{
		Creator:     msg.Creator,
		DisplayName: msg.DisplayName,
		Version:     msg.Version,
	}
	if err = k.chainletService.DisableChainletStackVersion(ctx, v); err != nil {
		return nil, err
	}

	return &types.MsgDisableChainletStackVersionResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventChainletStackVersionDisabled{
		Name:    msg.DisplayName,
		Version: msg.Version,
	})
}

func (k msgServer) UpdateChainletStack(goCtx context.Context, msg *types.MsgUpdateChainletStack) (resp *types.MsgUpdateChainletStackResponse, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err = k.validateMsgAndACL(ctx, msg.Creator, msg); err != nil {
		return
	}

	version := types.ChainletStackParams{
		Image:    msg.Image,
		Version:  msg.Version,
		Checksum: msg.Checksum,
		Enabled:  true,
	}
	err = k.chainletService.AddChainletStackVersion(ctx, msg.DisplayName, version)
	if err != nil {
		return nil, fmt.Errorf("error while adding chainlet stack version: %w", err)
	}

	return &types.MsgUpdateChainletStackResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventNewChainletStackVersion{
		Name:    msg.DisplayName,
		Version: msg.Version,
	})
}

func (k msgServer) LaunchChainlet(goCtx context.Context, msg *types.MsgLaunchChainlet) (*types.MsgLaunchChainletResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	p := k.GetParams(ctx)

	if err := msg.ValidateBasic(); err != nil {
		return &types.MsgLaunchChainletResponse{}, err
	}

	chainlet, err := k.buildChainlet(ctx, msg, p)
	if err != nil {
		return &types.MsgLaunchChainletResponse{}, err
	}

	if err = k.chainletService.LaunchChainlet(ctx, chainlet, p); err != nil {
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

func (k msgServer) UpgradeChainlet(goCtx context.Context, msg *types.MsgUpgradeChainlet) (*types.MsgUpgradeChainletResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := msg.ValidateBasic(); err != nil {
		return &types.MsgUpgradeChainletResponse{}, err
	}

	if err := k.chainletService.UpdateChainletVersion(ctx, msg.ChainId, msg.Creator, msg.StackVersion); err != nil {
		return &types.MsgUpgradeChainletResponse{}, err
	}

	return &types.MsgUpgradeChainletResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventUpdateChainlet{
		ChainId:      msg.ChainId,
		StackVersion: msg.StackVersion,
	})
}

func (k msgServer) buildChainlet(ctx sdk.Context, msg *types.MsgLaunchChainlet, p types.Params) (types.Chainlet, error) {
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

func (k msgServer) validateMsgAndACL(ctx sdk.Context, msgCreator string, validator validator) error {
	if err := validator.ValidateBasic(); err != nil {
		return err
	}
	p := k.GetParams(ctx)
	if p.ChainletStackProtections {
		addr, err := sdk.AccAddressFromBech32(msgCreator)
		if err != nil {
			return err
		}
		if !k.aclKeeper.Allowed(ctx, addr) {
			return fmt.Errorf("address %s not allowed to perform the action", msgCreator)
		}
	}
	return nil
}

type validator interface {
	ValidateBasic() error
}
