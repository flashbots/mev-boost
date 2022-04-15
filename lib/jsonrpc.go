package lib

import (
	"encoding/json"
	"fmt"
)

//
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (err rpcError) Error() string {
	return fmt.Sprintf("Error %d (%s)", err.Code, err.Message)
}

type rpcResponse struct {
	ID      string          `json:"id"`
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcRequest struct {
	ID      string        `json:"id"`
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
}

func parseRPCResponse(data []byte) (ret *rpcResponse, err error) {
	ret = new(rpcResponse)
	err = json.Unmarshal(data, &ret)
	return
}
