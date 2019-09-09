package balance

import (
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/pkg/errors"
)

const INITIAL_FUND = 10000

const FUND_PREFIX = "fund"

var ErrFundNotExist = errors.New("fund not exists")

func Init(mspid string, stub shim.ChaincodeStubInterface) error {
	_, err := Get(mspid, stub)
	if err == ErrFundNotExist {
		return Put(mspid, INITIAL_FUND, stub)
	}

	if err != nil {
		return errors.Errorf("failed to read data from ledger: %s", err)
	}

	return errors.Errorf("balance for '%s' already initialized", mspid)
}

func Put(mspid string, value uint64, stub shim.ChaincodeStubInterface) error {
	fundkey, err := stub.CreateCompositeKey(FUND_PREFIX, []string{mspid})
	if err != nil {
		return errors.Errorf("failed to create composite key: %s", err)
	}

	return stub.PutState(fundkey, []byte(strconv.FormatUint(value, 10)))
}

func Get(mspid string, stub shim.ChaincodeStubInterface) (uint64, error) {
	fundkey, err := stub.CreateCompositeKey(FUND_PREFIX, []string{mspid})
	if err != nil {
		return 0, errors.Errorf("failed to create composite key: %s", err)
	}

	balance, err := stub.GetState(fundkey)
	if err != nil {
		return 0, errors.Errorf("failed to read balance from ledger: %s", err)
	}

	if balance == nil {
		return 0, ErrFundNotExist
	}

	bal, err := strconv.ParseUint(string(balance), 10, 0)
	if err != nil {
		return 0, errors.Errorf("balance for org '%s' is ill-formed", err)
	}

	return bal, nil
}
