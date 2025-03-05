package keeper

import (
	"context"
	"cosmossdk.io/store/prefix"
	"encoding/binary"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/sagaxyz/ssc/x/chainlet/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = &Keeper{}

func (k *Keeper) GetChainlet(goCtx context.Context, req *types.QueryGetChainletRequest) (*types.QueryGetChainletResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	chainlet, err := k.Chainlet(ctx, req.ChainId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return &types.QueryGetChainletResponse{
		Chainlet: chainlet,
	}, nil
}

func (k Keeper) GetChainletCount(goCtx context.Context, req *types.QueryGetChainletCountRequest) (*types.QueryGetChainletCountResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.NumChainletsKey)
	ctx.Logger().Info("GetChainletCount", "count", binary.BigEndian.Uint64(bz))
	return &types.QueryGetChainletCountResponse{Count: binary.BigEndian.Uint64(bz)}, nil
}

func (k *Keeper) GetChainletStack(goCtx context.Context, req *types.QueryGetChainletStackRequest) (*types.QueryGetChainletStackResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if req.DisplayName == "" {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	stack, err := k.getChainletStack(ctx, req.DisplayName)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "TODO") //TODO
	}
	return &types.QueryGetChainletStackResponse{
		ChainletStack: stack,
	}, nil
}

func (k *Keeper) ListChainletStack(goCtx context.Context, req *types.QueryListChainletStackRequest) (*types.QueryListChainletStackResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	var chainletStacks []*types.ChainletStack
	var err error

	ctx := sdk.UnwrapSDKContext(goCtx)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), []byte(types.ChainletStackKey))
	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		var chainletStack types.ChainletStack
		if err := k.cdc.Unmarshal(value, &chainletStack); err != nil {
			return err
		}
		chainletStacks = append(chainletStacks, &chainletStack)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListChainletStackResponse{ChainletStacks: chainletStacks, Pagination: pageRes}, nil
}

func (k *Keeper) ListChainlets(goCtx context.Context, req *types.QueryListChainletsRequest) (*types.QueryListChainletsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	var chainlets []*types.Chainlet
	var err error

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletKey)
	pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
		var chainlet types.Chainlet
		if err := k.cdc.Unmarshal(value, &chainlet); err != nil {
			return err
		}
		chainlets = append(chainlets, &chainlet)
		return nil
	})

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryListChainletsResponse{Chainlets: chainlets, Pagination: pageRes}, nil
}

func (k *Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	return &types.QueryParamsResponse{Params: k.GetParams(ctx)}, nil
}
