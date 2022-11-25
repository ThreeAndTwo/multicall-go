package multicall2

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"math/big"
	"reflect"
	"strconv"
	"strings"
)

const symbol = "."

type ParseABI struct {
	_abi   *abi.ABI
	method string
}

func InitABI(abiStr, method string) (*ParseABI, error) {
	parsed, err := abi.JSON(strings.NewReader(abiStr))
	if err != nil {
		return nil, err
	}

	return &ParseABI{
		_abi:   &parsed,
		method: method,
	}, nil
}

func (p *ParseABI) ParseOutputData(data []byte) (map[string]interface{}, error) {
	unpackData, err := p._abi.Unpack(p.method, data)
	if err != nil {
		return nil, err
	}
	return parseData(unpackData), nil
}

func parseData(data []interface{}) map[string]interface{} {
	dataMap := make(map[string]interface{})
	for k, v := range data {
		if reflect.Struct != reflect.TypeOf(v).Kind() {
			dataMap[strconv.Itoa(k)] = v
			continue
		}
		recursionOutput(strconv.Itoa(k), v, dataMap)
	}
	return dataMap
}

func recursionOutput(originIndex string, val interface{}, dataMap map[string]interface{}) {
	vv := reflect.ValueOf(val)
	num := vv.NumField()

	for i := 0; i < num; i++ {
		mapKey := originIndex + symbol + strconv.Itoa(i)
		switch vv.Field(i).Type().Kind() {
		case reflect.String:
			dataMap[mapKey] = vv.Field(i).Interface().(string)
		case reflect.Ptr:
			dataMap[mapKey] = vv.Field(i).Interface().(*big.Int)
		case reflect.Bool:
			dataMap[mapKey] = vv.Field(i).Interface().(bool)
		case reflect.Uint:
			dataMap[mapKey] = vv.Field(i).Interface().(uint)
		case reflect.Uint8:
			dataMap[mapKey] = vv.Field(i).Interface().(uint8)
		case reflect.Uint16:
			dataMap[mapKey] = vv.Field(i).Interface().(uint16)
		case reflect.Uint32:
			dataMap[mapKey] = vv.Field(i).Interface().(uint32)
		case reflect.Uint64:
			dataMap[mapKey] = vv.Field(i).Interface().(uint64)
		case reflect.Int:
			dataMap[mapKey] = vv.Field(i).Interface().(int)
		case reflect.Int8:
			dataMap[mapKey] = vv.Field(i).Interface().(int8)
		case reflect.Int16:
			dataMap[mapKey] = vv.Field(i).Interface().(int16)
		case reflect.Int32:
			dataMap[mapKey] = vv.Field(i).Interface().(int32)
		case reflect.Int64:
			dataMap[mapKey] = vv.Field(i).Interface().(int64)
		case reflect.Struct:
			recursionOutput(originIndex+symbol+strconv.Itoa(i), vv.Field(i).Interface(), dataMap)
		}
	}
}
