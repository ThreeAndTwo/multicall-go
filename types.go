package multicall2

import "github.com/ethereum/go-ethereum/common"

type Option func(*Config)

type Config struct {
	MulticallAddress *common.Address
	Gas              uint64
}
