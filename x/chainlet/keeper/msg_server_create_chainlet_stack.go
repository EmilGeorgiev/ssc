package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sagaxyz/ssc/x/chainlet/types"
)

// CreateChainletStack creates a chainlet stack which is essentially a blueprint or template for a chainlet. It contains
// the configuration and metadata—such as the container image, version, checksum, display name, description, and fee
// information—that define how a chainlet should be deployed and managed.
//
// When you create a chainlet stack, you’re not immediately launching a live blockchain. Instead, you’re registering the
// settings and parameters that will be used later to instantiate a chainlet. Think of it as predefining the software
// stack for your dedicated blockchain.
//
// The chainlet stack supports multiple versions. This allows for updates and improvements over time. Once a chainlet
// stack is registered, further operations like LaunchChainlet, UpdateChainletStack, or UpgradeChainlet refer to this
// pre-registered configuration.
func (k msgServer) CreateChainletStack(goCtx context.Context, msg *types.MsgCreateChainletStack) (*types.MsgCreateChainletStackResponse, error) {
	err := msg.ValidateBasic()
	if err != nil {
		return &types.MsgCreateChainletStackResponse{}, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	p := k.GetParams(ctx)
	if p.ChainletStackProtections {
		addr, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			return &types.MsgCreateChainletStackResponse{}, err
		}
		if !k.aclKeeper.Allowed(ctx, addr) {
			return nil, fmt.Errorf("address %s not allowed to create chainlet stacks", msg.Creator)
		}
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
	err = k.NewChainletStack(ctx, chainletStack)
	if err != nil {
		return nil, fmt.Errorf("error while adding chainlet stack: %s", err)
	}

	return &types.MsgCreateChainletStackResponse{}, ctx.EventManager().EmitTypedEvent(&types.EventNewChainletStack{
		Creator: msg.Creator,
		Name:    msg.DisplayName,
		Version: msg.Version,
	})
}
