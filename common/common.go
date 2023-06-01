package common

import (
	"math/big"
	"os"
	"strconv"

	"github.com/flashbots/go-boost-utils/types"
)

const (
	GenesisTimeMainnet = 1606824023
	SlotTimeSecMainnet = 12
)

func GetEnv(key, defaultValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return defaultValue
}

func GetEnvInt(key string, defaultValue int) int {
	if value, ok := os.LookupEnv(key); ok {
		val, err := strconv.Atoi(value)
		if err == nil {
			return val
		}
	}
	return defaultValue
}

func GetEnvFloat64(key string, defaultValue float64) float64 {
	if value, ok := os.LookupEnv(key); ok {
		val, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return val
		}
	}
	return defaultValue
}

// FloatEthTo256Wei converts a float (precision 10) denominated in eth to a U256Str denominated in wei
func FloatEthTo256Wei(val float64) (*types.U256Str, error) {
	weiU256 := new(types.U256Str)
	ethFloat := new(big.Float)
	weiFloat := new(big.Float)
	weiFloatLessPrecise := new(big.Float)
	weiInt := new(big.Int)

	ethFloat.SetFloat64(val)
	weiFloat.Mul(ethFloat, big.NewFloat(1e18))
	weiFloatLessPrecise.SetString(weiFloat.String())
	weiFloatLessPrecise.Int(weiInt)

	err := weiU256.FromBig(weiInt)
	return weiU256, err
}
