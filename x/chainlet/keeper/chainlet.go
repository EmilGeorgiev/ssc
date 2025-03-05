package keeper

import (
	"cosmossdk.io/store/prefix"
	"encoding/binary"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sagaxyz/ssc/x/chainlet/types"
)

func (k *Keeper) Chainlet(ctx sdk.Context, chainId string) (chainlet types.Chainlet, err error) {
	byteKey := []byte(chainId)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletKey)
	if !store.Has(byteKey) {
		err = fmt.Errorf("key %s not found", chainId)
		return
	}

	chainletBytes := store.Get(byteKey)
	if len(chainletBytes) == 0 {
		panic(fmt.Sprintf("no data at chainlet %s", chainId))
	}
	k.cdc.MustUnmarshal(chainletBytes, &chainlet)

	return
}

func (k *Keeper) setChainletInfo(ctx sdk.Context, chainlet types.Chainlet) {
	lcStore := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletKey)
	byteLCKey := []byte(chainlet.ChainId)
	updatedValue := k.cdc.MustMarshal(&chainlet)
	lcStore.Set(byteLCKey, updatedValue)
}

func (k *Keeper) InitializeChainletCount(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, uint64(0))
	store.Set(types.NumChainletsKey, bz)
}

func (k *Keeper) incrementChainletCount(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.NumChainletsKey)
	count := binary.BigEndian.Uint64(bz)
	count++
	bz = make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.NumChainletsKey, bz)
}

func (k *Keeper) AutoUpgradeChainlets(ctx sdk.Context) error {
	iter := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletKey).Iterator(nil, nil)
	for ; iter.Valid(); iter.Next() {
		var chainlet types.Chainlet
		k.cdc.MustUnmarshal(iter.Value(), &chainlet)

		if !chainlet.AutoUpgradeStack {
			ctx.Logger().Debug(fmt.Sprintf("skipping auto-upgrade for chainlet %s\n", chainlet.ChainId))
			continue
		}

		latestVersion, err := k.LatestVersion(ctx, chainlet.ChainletStackName, chainlet.ChainletStackVersion)
		if err != nil {
			iter.Close()
			return err
		}
		if latestVersion == chainlet.ChainletStackVersion {
			iter.Close()
			return nil
		}

		if err = k.chainletStackVersionAvailable(ctx, chainlet.ChainletStackName, latestVersion); err != nil {
			iter.Close()
			//TODO change to panic in the future, should never happen if the loaded versions are consistent with the state
			return fmt.Errorf("chainlet stack %s has unavailable version %s loaded", chainlet.ChainletStackName, latestVersion)
		}

		if chainlet.ChainletStackVersion == latestVersion {
			ctx.Logger().Debug(fmt.Sprintf("chainlet %s: %s is at its latest available version\n", chainlet.ChainId, chainlet.ChainletStackVersion))
			continue
		}

		ctx.Logger().Info(fmt.Sprintf("upgrading chainlet %s: %s to %s\n", chainlet.ChainId, chainlet.ChainletStackVersion, latestVersion))
		chainlet.ChainletStackVersion = latestVersion
		defer k.setChainletInfo(ctx, chainlet)
	}
	iter.Close()

	return nil
}

func (k *Keeper) Create(ctx sdk.Context, chainlet types.Chainlet) error {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletKey)

	key := []byte(chainlet.ChainId)
	value := k.cdc.MustMarshal(&chainlet)
	store.Set(key, value)
	k.incrementChainletCount(ctx)
	return nil
}

func (k *Keeper) GetChainletCount2(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.NumChainletsKey)
	ctx.Logger().Info("GetChainletCount", "count", binary.BigEndian.Uint64(bz))
	return binary.BigEndian.Uint64(bz)
}

func (k *Keeper) StartExistingChainlet(ctx sdk.Context, chainId string) error {
	c, err := k.Chainlet(ctx, chainId)
	if err != nil {
		return fmt.Errorf("cannot start existing chainlet %s: %v", chainId, err)
	}

	c.Status = types.Status_STATUS_ONLINE
	k.setChainletInfo(ctx, c)

	return nil
}

func (k *Keeper) StopChainlet(ctx sdk.Context, chainId string) error {
	c, err := k.Chainlet(ctx, chainId)
	if err != nil {
		return fmt.Errorf("cannot stop chainlet %s: %v", chainId, err)
	}
	c.Status = types.Status_STATUS_OFFLINE
	k.setChainletInfo(ctx, c)
	ctx.Logger().Info(fmt.Sprintf("Successfully stopped chainlet %s", chainId))
	return nil
}
