package mcp

import "encoding/json"

// JSON-RPC 2.0 types for MCP protocol.

// Request is an incoming JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is an outgoing JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool describes an MCP tool for the tools/list response.
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

// ToolCallParams is the params for a tools/call request.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// ToolResult is the content item in a tools/call response.
type ToolResult struct {
	Content []ContentItem `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

// ContentItem is a single piece of content in a tool result.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// InitializeParams is the params for the initialize handshake.
type InitializeParams struct {
	ProtocolVersion string         `json:"protocolVersion"`
	ClientInfo      map[string]any `json:"clientInfo,omitempty"`
	Capabilities    map[string]any `json:"capabilities,omitempty"`
}

// errorResponse creates an error response for a given request ID.
func errorResponse(id any, code int, message string) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

// Notification is a server-to-client JSON-RPC 2.0 notification (no id, no response expected).
type Notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

// LogMessageParams is the params for notifications/message (MCP logging notification).
type LogMessageParams struct {
	Level  string `json:"level"`
	Data   string `json:"data"`
}

// successResponse creates a success response.
func successResponse(id any, result any) Response {
	return Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
}
