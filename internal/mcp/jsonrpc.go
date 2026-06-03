package mcp

import (
	"encoding/json"
	"fmt"
)

const jsonrpcVersion = "2.0"

type jsonrpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      *int64 `json:"id,omitempty"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonrpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *jsonrpcError   `json:"error,omitempty"`
}

type jsonrpcError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

func (e *jsonrpcError) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("mcp json-rpc error %d: %s", e.Code, e.Message)
}

func marshalJSONRPCRequest(id int64, method string, params any) ([]byte, error) {
	return json.Marshal(jsonrpcRequest{
		JSONRPC: jsonrpcVersion,
		ID:      &id,
		Method:  method,
		Params:  params,
	})
}

func marshalJSONRPCNotification(method string, params any) ([]byte, error) {
	return json.Marshal(jsonrpcRequest{
		JSONRPC: jsonrpcVersion,
		Method:  method,
		Params:  params,
	})
}
