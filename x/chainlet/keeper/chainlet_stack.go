package keeper

import (
	"cosmossdk.io/store/prefix"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/sagaxyz/ssc/x/chainlet/types"
)

func (k *Keeper) chainletStackVersionAvailable(ctx sdk.Context, name, version string) (bool, error) {
	stack, err := k.getChainletStack(ctx, name)
	if err != nil {
		return false, fmt.Errorf("cannot get chainlet stack with name %s: %w", name, err)
	}

	//TODO avoid loop
	for _, v := range stack.Versions {
		if v.Version != version {
			continue
		}
		if !v.Enabled {
			return false, nil
		}
		return true, nil
	}

	return false, fmt.Errorf("stack version %s is not found", version)
}

func (k *Keeper) getChainletStack(ctx sdk.Context, name string) (stack types.ChainletStack, err error) {
	byteKey := []byte(name)

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletStackKey)
	if !store.Has(byteKey) {
		err = fmt.Errorf("stack %s not found", name)
		return
	}

	data := store.Get(byteKey)
	k.cdc.MustUnmarshal(data, &stack)
	return
}

func (k *Keeper) CreateChainletStack(ctx sdk.Context, cs types.ChainletStack) error {
	for _, version := range cs.Versions {
		if version.Enabled {
			err := k.AddVersion(ctx, cs.DisplayName, version.Version)
			if err != nil {
				return err
			}
		}
	}

	// Our key is the display name e.g. SagaEVM. Associated with this key can be many versions
	// Versions is a slice of object type TemplateMetadata
	byteKey := []byte(cs.DisplayName)
	value := k.cdc.MustMarshal(&cs)
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletStackKey)
	store.Set(byteKey, value)

	return nil
}

func (k *Keeper) AddChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version types.ChainletStackParams) error {
	// Store in the version tree for automatic updates
	if version.Enabled {
		err := k.AddVersion(ctx, stack.DisplayName, version.Version)
		if err != nil {
			return err
		}
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletStackKey)
	updatedValue := k.cdc.MustMarshal(&stack)
	store.Set([]byte(stack.DisplayName), updatedValue)
	return nil
}

func (k *Keeper) DisableChainletStackVersion2(ctx sdk.Context, stack types.ChainletStack, version string) error {
	err := k.RemoveVersion(ctx, stack.DisplayName, version)
	if err != nil {
		return err
	}

	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ChainletStackKey)
	updatedValue := k.cdc.MustMarshal(&stack)
	store.Set([]byte(stack.DisplayName), updatedValue)
	return nil
}
