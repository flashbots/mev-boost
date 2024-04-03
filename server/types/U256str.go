package types

import (
	"github.com/flashbots/go-boost-utils/types"
)

type U256Str = types.U256Str

func IntToU256(i uint64) U256Str {
	return types.IntToU256(i)
}
