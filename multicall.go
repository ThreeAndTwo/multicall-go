package multicall_go

import (
	"context"
	"encoding/hex"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

type Multicall interface {
	CallRaw(calls ViewCalls, block *big.Int) (*Result, error)
	Call(calls ViewCalls, block *big.Int) (*Result, error)
	CallBytes(callsBytes ViewCallsBytes, block *big.Int) (*Result, error)
	Contract() *common.Address
}

type multicall struct {
	eth    *ethclient.Client
	config *Config
}

func New(ethClient *ethclient.Client, contract common.Address, opts ...Option) (Multicall, error) {
	config := &Config{
		MulticallAddress: &contract,
		Gas:              10000000,
	}

	for _, opt := range opts {
		opt(config)
	}

	return &multicall{
		eth:    ethClient,
		config: config,
	}, nil
}

type CallResult struct {
	Success bool
	Raw     []byte
	Decoded []interface{}
}

type Result struct {
	BlockNumber uint64
	Calls       map[string]CallResult
}

const AggregateMethod = "0x17352e13"

func (mc multicall) CallRaw(calls ViewCalls, block *big.Int) (*Result, error) {
	resultRaw, err := mc.makeRequest(calls, block)
	if err != nil {
		return nil, err
	}
	return calls.decodeRaw(resultRaw)
}

func (mc multicall) Call(calls ViewCalls, block *big.Int) (*Result, error) {
	resultRaw, err := mc.makeRequest(calls, block)
	if err != nil {
		return nil, err
	}
	println("resultRaw:", resultRaw)
	return calls.decode(resultRaw)
}

func (mc multicall) CallBytes(callsBytes ViewCallsBytes, block *big.Int) (*Result, error) {
	payloadArgs, err := callsBytes.callData()
	if err != nil {
		return nil, err
	}

	input, _ := hexutil.Decode(AggregateMethod + hex.EncodeToString(payloadArgs))
	msg := ethereum.CallMsg{
		To:   mc.config.MulticallAddress,
		Gas:  mc.config.Gas,
		Data: input,
	}
	data, err := mc.eth.CallContract(context.Background(), msg, block)
	if err != nil {
		return nil, err
	}

	encodeData := hexutil.Encode(data)
	return callsBytes.decode(encodeData)
}

func (mc multicall) makeRequest(calls ViewCalls, block *big.Int) (string, error) {
	payloadArgs, err := calls.callData()
	if err != nil {
		return "", err
	}
	input, _ := hexutil.Decode(AggregateMethod + hex.EncodeToString(payloadArgs))
	msg := ethereum.CallMsg{
		To:   mc.config.MulticallAddress,
		Gas:  mc.config.Gas,
		Data: input,
	}
	data, err := mc.eth.CallContract(context.Background(), msg, block)
	return hexutil.Encode(data), err
}

func (mc multicall) Contract() *common.Address {
	return mc.config.MulticallAddress
}
