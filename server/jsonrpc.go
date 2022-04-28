package server

import (
	"encoding/json"
	"fmt"
)

type errorWithCode interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func newRPCError(msg string, code int) rpcError {
	return rpcError{
		Message: msg,
		Code:    code,
	}
}

func (err rpcError) Error() string {
	if err.Message == "" {
		return fmt.Sprintf("json-rpc error %d", err.Code)
	}

	return err.Message
}

// ErrorCode returns the ID of the error.
func (err rpcError) ErrorCode() int { return err.Code }

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
