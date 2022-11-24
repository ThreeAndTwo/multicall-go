package multicall2

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"reflect"
	"regexp"
	"strings"
)

type ViewCall struct {
	id        string
	target    string
	method    string
	arguments []interface{}
}

type ViewCallBytes struct {
	id, abi        string
	method, target string
	outputTy       []string
	inputData      []byte
}

type ViewCallsBytes []ViewCallBytes

type ViewCalls []ViewCall

func NewViewCall(id, target, method string, arguments []interface{}) ViewCall {
	return ViewCall{
		id:        id,
		target:    target,
		method:    method,
		arguments: arguments,
	}
}

func NewViewCallBytes(id, abi, method, target string, inputData []byte) ViewCallBytes {
	return ViewCallBytes{
		id:        id,
		abi:       abi,
		method:    method,
		target:    target,
		inputData: inputData,
	}
}

func (call ViewCall) Validate() error {
	if _, err := call.argsCallData(); err != nil {
		return err
	}
	return nil
}

func (cb ViewCallBytes) Validate() error {
	return nil
}

var insideParens = regexp.MustCompile("\\(.*?\\)")
var numericArg = regexp.MustCompile("u?int(256)|(8)")

func (call ViewCall) argumentTypes() []string {
	rawArgs := insideParens.FindAllString(call.method, -1)[0]
	rawArgs = strings.Replace(rawArgs, "(", "", -1)
	rawArgs = strings.Replace(rawArgs, ")", "", -1)
	if rawArgs == "" {
		return []string{}
	}
	args := strings.Split(rawArgs, ",")
	for index, arg := range args {
		args[index] = strings.Trim(arg, " ")
	}
	return args
}

func (call ViewCall) returnTypes() []string {
	rawArgs := insideParens.FindAllString(call.method, -1)[1]
	rawArgs = strings.Replace(rawArgs, "(", "", -1)
	rawArgs = strings.Replace(rawArgs, ")", "", -1)
	args := strings.Split(rawArgs, ",")
	for index, arg := range args {
		args[index] = strings.Trim(arg, " ")
	}
	return args
}

func (call ViewCall) callData() ([]byte, error) {
	argsSuffix, err := call.argsCallData()
	if err != nil {
		return nil, err
	}
	methodPrefix, err := call.methodCallData()
	if err != nil {
		return nil, err
	}

	payload := make([]byte, 0)
	payload = append(payload, methodPrefix...)
	payload = append(payload, argsSuffix...)

	return payload, nil
}

func (call ViewCall) methodCallData() ([]byte, error) {
	methodParts := strings.Split(call.method, ")(")
	var method string
	if len(methodParts) > 1 {
		method = fmt.Sprintf("%s)", methodParts[0])
	} else {
		method = methodParts[0]
	}
	hash := crypto.Keccak256([]byte(method))
	return hash[0:4], nil
}

func (call ViewCall) argsCallData() ([]byte, error) {
	argTypes := call.argumentTypes()
	if len(argTypes) != len(call.arguments) {
		return nil, fmt.Errorf("number of argument types doesn't match with number of arguments for %s with method %s", call.id, call.method)
	}
	argumentValues := make([]interface{}, len(call.arguments))
	arguments := make(abi.Arguments, len(call.arguments))

	for index, argTypeStr := range argTypes {
		argType, err := abi.NewType(argTypeStr, "", nil)
		if err != nil {
			return nil, err
		}

		arguments[index] = abi.Argument{Type: argType}
		argumentValues[index], err = call.getArgument(index, argTypeStr)
		if err != nil {
			return nil, err
		}
	}

	return arguments.Pack(argumentValues...)
}

func (call ViewCall) getArgument(index int, argumentType string) (interface{}, error) {
	arg := call.arguments[index]
	if argumentType == "address" {
		address, ok := arg.(string)
		if !ok {
			return nil, fmt.Errorf("expected address argument to be a string")
		}
		return toByteArray(address)
	} else if numericArg.MatchString(argumentType) {
		if num, ok := arg.(json.Number); ok {
			if v, err := num.Int64(); err != nil {
				return big.NewInt(v), nil
			} else if v, err := num.Float64(); err != nil {
				return big.NewInt(int64(v)), nil
			} else {
			}
		} else {
			_int64 := reflect.TypeOf(int64(0))
			argType := reflect.TypeOf(arg)
			kind := argType.Kind()
			if kind == reflect.String {
				if val, ok := new(big.Int).SetString(call.arguments[index].(string), 10); !ok {
					return nil, fmt.Errorf("could not parse %s as a base 10 number", call.arguments[index])
				} else {
					return val, nil
				}
			} else if argType.ConvertibleTo(_int64) {
				return big.NewInt(reflect.ValueOf(arg).Convert(_int64).Int()), nil
			}
		}
	}
	return arg, nil
}

func (call ViewCall) decode(raw []byte) ([]interface{}, error) {
	retTypes := call.returnTypes()
	args := make(abi.Arguments, 0, 0)
	for index, retTypeStr := range retTypes {
		retType, err := abi.NewType(retTypeStr, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Name: fmt.Sprintf("ret%d", index), Type: retType})
	}
	decoded := make(map[string]interface{})
	err := args.UnpackIntoMap(decoded, raw)
	if err != nil {
		return nil, err
	}
	returns := make([]interface{}, len(retTypes))
	for index := range retTypes {
		key := fmt.Sprintf("ret%d", index)
		item := decoded[key]
		if bigint, ok := item.(*big.Int); ok {
			returns[index] = bigint
		} else {
			returns[index] = decoded[key]
		}
	}
	return returns, nil
}

type callArgs struct {
	Target   [20]byte
	CallData []byte
}

func (cbs ViewCallsBytes) callData() ([]byte, error) {
	payloadArgs := make([]callArgs, 0, len(cbs))
	for _, call := range cbs {

		targetBytes, err := toByteArray(call.target)
		if err != nil {
			return nil, err
		}
		payloadArgs = append(payloadArgs, callArgs{targetBytes, call.inputData})
	}

	tupleArray, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Type: "address", Name: "Target"},
		{Type: "bytes", Name: "CallData"},
	})
	if err != nil {
		return nil, err
	}
	boolean, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{
		{Type: tupleArray, Name: "calls"},
		{Type: boolean, Name: "strict"},
	}
	return args.Pack(payloadArgs, false)
}

func (calls ViewCalls) callData() ([]byte, error) {
	payloadArgs := make([]callArgs, 0, len(calls))
	for _, call := range calls {
		callData, err := call.callData()
		if err != nil {
			return nil, err
		}
		targetBytes, err := toByteArray(call.target)
		if err != nil {
			return nil, err
		}
		payloadArgs = append(payloadArgs, callArgs{targetBytes, callData})
	}

	tupleArray, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Type: "address", Name: "Target"},
		{Type: "bytes", Name: "CallData"},
	})
	if err != nil {
		return nil, err
	}
	boolean, err := abi.NewType("bool", "", nil)
	if err != nil {
		return nil, err
	}
	args := abi.Arguments{
		{Type: tupleArray, Name: "calls"},
		{Type: boolean, Name: "strict"},
	}
	return args.Pack(payloadArgs, false)
}

type retType struct {
	Success bool
	Data    []uint8
}

type wrapperRet struct {
	BlockNumber *big.Int
	Returns     []struct {
		Success bool    `json:"Success"`
		Data    []uint8 `json:"Data"`
	}
}

func decodeWrapper(raw string) (*wrapperRet, error) {
	rawBytes, err := hex.DecodeString(strings.Replace(raw, "0x", "", -1))
	if err != nil {
		return nil, err
	}

	uint256Type, err := abi.NewType("uint256", "", nil)
	if err != nil {
		return nil, err
	}
	returnType, err := abi.NewType("tuple[]", "", []abi.ArgumentMarshaling{
		{Name: "Success", Type: "bool"},
		{Name: "Data", Type: "bytes"},
	})
	if err != nil {
		return nil, err
	}
	wrapperArgs := abi.Arguments{
		{
			Name: "BlockNumber",
			Type: uint256Type,
		},
		{
			Name: "Returns",
			Type: returnType,
		},
	}
	decoded := wrapperRet{}
	decodedMap := map[string]interface{}{}
	err = wrapperArgs.UnpackIntoMap(decodedMap, rawBytes)
	if err != nil {
		return nil, err
	}
	decoded.BlockNumber = decodedMap["BlockNumber"].(*big.Int)
	decoded.Returns = decodedMap["Returns"].([]struct {
		Success bool    `json:"Success"`
		Data    []uint8 `json:"Data"`
	})
	return &decoded, err
}

func (calls ViewCalls) decodeRaw(raw string) (*Result, error) {
	decoded, err := decodeWrapper(raw)
	if err != nil {
		return nil, err
	}
	result := &Result{}
	result.BlockNumber = decoded.BlockNumber.Uint64()
	result.Calls = make(map[string]CallResult)

	for index, call := range calls {
		callResult := CallResult{
			Success: decoded.Returns[index].Success,
			Raw:     decoded.Returns[index].Data,
			Decoded: []interface{}{},
		}
		result.Calls[call.id] = callResult
	}

	return result, nil
}

func (cbs ViewCallsBytes) decode(raw string) (*BytesResult, error) {
	decoded, err := decodeWrapper(raw)
	if err != nil {
		return nil, err
	}
	result := &BytesResult{}
	result.BlockNumber = decoded.BlockNumber.Uint64()
	calls := make(map[string]CallBytesResult)
	result.Calls = calls
	for index, cb := range cbs {
		callResult := CallBytesResult{
			Success: decoded.Returns[index].Success,
			Raw:     decoded.Returns[index].Data,
		}
		if decoded.Returns[index].Success {
			returnValues, err := cb.decodeRaw(decoded.Returns[index].Data)
			if err != nil {
				return nil, err
			}
			callResult.Decoded = returnValues
		}
		result.Calls[cb.id] = callResult
	}
	return result, nil
}

func (cb ViewCallBytes) decodeRaw(raw []byte) (map[string]interface{}, error) {
	_abi, err := InitABI(cb.abi, cb.method)
	if err != nil {
		return nil, err
	}

	data, err := _abi.ParseOutputData(raw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (cb ViewCallBytes) decode(raw []byte) ([]interface{}, error) {
	args := make(abi.Arguments, 0, 0)
	for index, retTypeStr := range cb.outputTy {
		retType, err := abi.NewType(retTypeStr, "", nil)
		if err != nil {
			return nil, err
		}
		args = append(args, abi.Argument{Name: fmt.Sprintf("ret%d", index), Type: retType})
	}

	values, _ := args.UnpackValues(raw)
	aa, _ := json.Marshal(values)
	fmt.Println("array:", string(aa))

	decoded := make(map[string]interface{})
	err := args.UnpackIntoMap(decoded, raw)
	if err != nil {
		return nil, err
	}

	returns := make([]interface{}, len(cb.outputTy))
	for index := range cb.outputTy {
		key := fmt.Sprintf("ret%d", index)
		item := decoded[key]
		if bigint, ok := item.(*big.Int); ok {
			returns[index] = bigint
		} else {
			returns[index] = decoded[key]
		}
	}
	return returns, nil
}

func (calls ViewCalls) decode(raw string) (*Result, error) {
	decoded, err := decodeWrapper(raw)
	if err != nil {
		return nil, err
	}
	result := &Result{}
	result.BlockNumber = decoded.BlockNumber.Uint64()
	result.Calls = make(map[string]CallResult)
	for index, call := range calls {
		callResult := CallResult{
			Success: decoded.Returns[index].Success,
			Raw:     decoded.Returns[index].Data,
		}
		if decoded.Returns[index].Success {
			returnValues, err := call.decode(decoded.Returns[index].Data)
			if err != nil {
				return nil, err
			}
			callResult.Decoded = returnValues
		}
		result.Calls[call.id] = callResult
	}

	return result, nil
}

func toByteArray(address string) ([20]byte, error) {
	var addressBytes [20]byte
	address = strings.Replace(address, "0x", "", -1)
	addressBytesSlice, err := hex.DecodeString(address)
	if err != nil {
		return addressBytes, err
	}

	copy(addressBytes[:], addressBytesSlice[:])
	return addressBytes, nil
}
