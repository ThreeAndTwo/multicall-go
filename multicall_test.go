package multicall_go

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"testing"
)

func TestExampleViwCall(t *testing.T) {
	eth, err := getETHClient("https://mainnet.infura.io/v3/17ed7fe26d014e5b9be7dfff5368c69d")
	vc := NewViewCall(
		"key.1",
		"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7",
		"balances(uint256)(uint256)",
		[]interface{}{1},
	)
	vcs := ViewCalls{vc}
	_multiCall := common.HexToAddress("0x5eb3fa2dfecdde21c950813c665e9364fa609bd2")
	mc, _ := New(eth, _multiCall)
	res, err := mc.Call(vcs, nil)
	if err != nil {
		panic(err)
	}

	resJson, _ := json.Marshal(res)
	fmt.Println(string(resJson))
	fmt.Println(res)
	fmt.Println(err)

}

func TestExampleViwCallBytes(t *testing.T) {
	eth, err := getETHClient("https://mainnet.infura.io/v3/17ed7fe26d014e5b9be7dfff5368c69d")
	decode, _ := hexutil.Decode("0x4903b0d10000000000000000000000000000000000000000000000000000000000000001")
	vc := NewViewCallBytes(
		"key.1",
		"0xbebc44782c7db0a1a60cb6fe97d0b483032ff1c7",
		[]string{"uint256"},
		decode,
	)

	vcs := ViewCallsBytes{vc}
	_multiCall := common.HexToAddress("0x5eb3fa2dfecdde21c950813c665e9364fa609bd2")
	mc, _ := New(eth, _multiCall)
	res, err := mc.CallBytes(vcs, nil)
	if err != nil {
		panic(err)
	}

	resJson, _ := json.Marshal(res)
	fmt.Println(string(resJson))
	fmt.Println(res)
	fmt.Println(err)
}

func getETHClient(url string) (*ethclient.Client, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	return client, nil
}
