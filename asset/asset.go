package asset

import (
	"encoding/json"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

const ASSET_PREFIX = "asset"

var ErrAssetNotExist = errors.New("asset not exists")

type Asset struct {
	Name    string
	Owner   string
	Creator string
	Price   uint64
}

func Put(asset Asset, stub shim.ChaincodeStubInterface) error {
	key, err := stub.CreateCompositeKey(ASSET_PREFIX, []string{asset.Name})
	if err != nil {
		return errors.Errorf("failed to create composite key")
	}

	bytes, err := json.Marshal(asset)
	if err != nil {
		return errors.Errorf("failed to marshal asset object: %s", err)
	}

	err = stub.PutState(key, bytes)
	if err != nil {
		return errors.Errorf("failed to store asset to ledger: %s", err)
	}

	return nil
}

func Get(name string, stub shim.ChaincodeStubInterface) (*Asset, error) {
	key, err := stub.CreateCompositeKey(ASSET_PREFIX, []string{name})
	if err != nil {
		return nil, errors.Errorf("failed to create composite key")
	}

	res, err := stub.GetState(key)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, ErrAssetNotExist
	}

	return Unmarshal(res)
}

func Exist(name string, stub shim.ChaincodeStubInterface) (bool, error) {
	_, err := Get(name, stub)
	if err == ErrAssetNotExist {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	return true, nil
}

func List(stub shim.ChaincodeStubInterface) ([]Asset, error) {
	iter, err := stub.GetStateByPartialCompositeKey(ASSET_PREFIX, nil)
	if err != nil {
		return nil, errors.Errorf("failed to read by partial composite key: %s", err)
	}
	defer iter.Close()

	var assets []Asset

	for iter.HasNext() {
		kv, err := iter.Next()
		if err != nil {
			return nil, errors.Errorf("failed to read query result: %s", err)
		}

		a := Asset{}
		err = json.Unmarshal(kv.Value, &a)
		if err != nil {
			return nil, errors.Errorf("failed to unmarshal asset '%s': %s", kv.Key, err)
		}

		assets = append(assets, a)
	}

	return assets, nil
}

func Unmarshal(data []byte) (*Asset, error) {
	a := &Asset{}
	err := json.Unmarshal(data, a)
	if err != nil {
		return nil, errors.Errorf("failed to unmarshal asset: %s", err)
	}

	return a, nil
}
