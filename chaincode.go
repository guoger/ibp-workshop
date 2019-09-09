/*
 * SPDX-License-Identifier: Apache-2.0
 */

package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"github.com/guoger/ibp-workshop/asset"
	"github.com/guoger/ibp-workshop/balance"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/msp"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// SimpleChaincode example simple Chaincode implementation
type AssetChaincode struct {
}

func (a *AssetChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	return shim.Success(nil)
}

func (a *AssetChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	switch function {
	case "InitFund":
		return a.initFund(stub, args)
	case "GetBalance":
		return a.getBalance(stub, args)
	case "CreateAsset":
		return a.createAsset(stub, args)
	case "ListAsset":
		return a.listAsset(stub, args)
	case "BuyAsset":
		return a.buyAsset(stub, args)
	default:
		return shim.Error(fmt.Sprintf("unknown func: %s, expecting: [InitFund, GetBalance, CreateAsset, ListAsset, BuyAsset]", function))
	}
}

func getMspID(stub shim.ChaincodeStubInterface) (string, error) {
	creator, err := stub.GetCreator()
	if err != nil {
		return "", err
	}

	id := &msp.SerializedIdentity{}
	if err = proto.Unmarshal(creator, id); err != nil {
		return "", err
	}

	return id.Mspid, nil
}

func (a *AssetChaincode) initFund(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 0 {
		return shim.Error(fmt.Sprintf("got %d args, expect 0", len(args)))
	}

	mspid, err := getMspID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get MSP ID: %s", err))
	}

	err = balance.Init(mspid, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to initialize fund for org '%s': %s", mspid, err))
	}

	return shim.Success([]byte(fmt.Sprintf("Balance for org '%s' initialized to %d", mspid, balance.INITIAL_FUND)))
}

func (a *AssetChaincode) getBalance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 0 {
		return shim.Error(fmt.Sprintf("got %d args, expect 0", len(args)))
	}

	mspid, err := getMspID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get MSP ID: %s", err))
	}

	bal, err := balance.Get(mspid, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get balance for org '%s': %s", mspid, err))
	}

	fmt.Printf("Balance for Org %s is %d\n", mspid, bal)
	return shim.Success([]byte(fmt.Sprintf("Current balance for org %s is: %s", mspid, strconv.FormatUint(bal, 10))))
}

func (a *AssetChaincode) createAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error(fmt.Sprintf("got %d args, expecting 2", len(args)))
	}

	name := strings.ToLower(args[0])
	price, err := strconv.ParseUint(args[1], 10, 0)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to extract price from arg '%s': %s", args[1], err))
	}

	mspid, err := getMspID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get MSP ID: %s", err))
	}

	exist, err := asset.Exist(name, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to read asset from ledger: %s", err))
	}

	if exist {
		return shim.Error(fmt.Sprintf("asset '%s' already exists", name))
	}

	ast := asset.Asset{
		Name:    name,
		Owner:   mspid,
		Creator: mspid,
		Price:   price,
	}

	err = asset.Put(ast, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to create asset: %s", err))
	}

	fmt.Printf("Created Asset\n")
	return shim.Success([]byte(fmt.Sprintf("Asset '%s' created", name)))
}

func (a *AssetChaincode) buyAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return shim.Error(fmt.Sprintf("expect 1 arg, got %d", len(args)))
	}

	name := strings.ToLower(args[0])
	ast, err := asset.Get(name, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to read asset from ledger: %s", err))
	}

	mspid, err := getMspID(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get MSP ID: %s", err))
	}

	if ast.Owner == mspid {
		return shim.Error(fmt.Sprintf("you already own asset '%s'", name))
	}

	buyerBalance, err := balance.Get(mspid, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get buyer balance: %s", err))
	}

	if buyerBalance < ast.Price {
		return shim.Error(fmt.Sprintf("balance insufficient, have %d, need %d", buyerBalance, ast.Price))
	}

	sellerBalance, err := balance.Get(ast.Owner, stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to get seller balance: %s", err))
	}

	buyerBalance -= ast.Price
	if err = balance.Put(mspid, buyerBalance, stub); err != nil {
		return shim.Error(fmt.Sprintf("failed to update buyer balance: %s", err))
	}

	sellerBalance += ast.Price
	if err = balance.Put(ast.Owner, sellerBalance, stub); err != nil {
		return shim.Error(fmt.Sprintf("failed to update seller balance: %s", err))
	}

	ast.Owner = mspid
	if err = asset.Put(*ast, stub); err != nil {
		return shim.Error(fmt.Sprintf("failed to update asset: %s", err))
	}

	fmt.Printf("Buy Asset\n")
	return shim.Success([]byte(fmt.Sprintf("Purchase is completed, now you own %s!", name)))
}

func (a *AssetChaincode) listAsset(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	fmt.Printf("List Assets\n")

	if len(args) != 0 {
		return shim.Error(fmt.Sprintf("expect no arg, got %d", len(args)))
	}

	res, err := asset.List(stub)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to retrieve asset list: %s", err))
	}

	bytes, err := json.Marshal(res)
	if err != nil {
		return shim.Error(fmt.Sprintf("failed to marshal asset list: %s", err))
	}

	return shim.Success(bytes)
}

func main() {
	err := shim.Start(new(AssetChaincode))
	if err != nil {
		panic(err)
	}
}
