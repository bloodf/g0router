package mcp

import "testing"

func TestStdioClientCloseNilProcess(t *testing.T) {
	client := &StdioClient{}
	if err := client.Close(); err != nil {
		t.Fatalf("Close nil process: %v", err)
	}
}

func TestStdioClientClose(t *testing.T) {
	server, process := newFakeStdioServer(t, func(req map[string]any) map[string]any {
		return rpcResult(req["id"], map[string]any{"protocolVersion": protocolVersion})
	})
	defer server.Close()
	client := NewStdioClient(process)
	if err := client.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}
