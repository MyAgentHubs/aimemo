package mcp

import (
	"encoding/json"
	"testing"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	database := db.NewTestDB(t)
	return NewServer(database, ":memory:")
}

func TestHandle_Initialize(t *testing.T) {
	s := newTestServer(t)
	req := Request{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
	}
	resp := s.handle(req)
	assert.Nil(t, resp.Error)
	assert.NotNil(t, resp.Result)
}

func TestHandle_ToolsList(t *testing.T) {
	s := newTestServer(t)
	req := Request{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}
	resp := s.handle(req)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(map[string]any)
	require.True(t, ok)
	tools, ok := result["tools"].([]Tool)
	require.True(t, ok)
	assert.Len(t, tools, 5)
}

func TestHandle_ToolsCall_MemoryStore(t *testing.T) {
	s := newTestServer(t)

	params, _ := json.Marshal(ToolCallParams{
		Name: "memory_store",
		Arguments: json.RawMessage(`{
			"entities": [{"name": "Redis", "entityType": "system", "observations": ["Port 6379"]}]
		}`),
	})

	req := Request{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  params,
	}
	resp := s.handle(req)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
	assert.NotEmpty(t, result.Content[0].Text)
}

func TestHandle_ToolsCall_MemorySearch_Empty(t *testing.T) {
	s := newTestServer(t)

	params, _ := json.Marshal(ToolCallParams{
		Name:      "memory_search",
		Arguments: json.RawMessage(`{"query": "anything"}`),
	})

	req := Request{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  params,
	}
	resp := s.handle(req)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError, "empty DB search should not error")
}

func TestHandle_ToolsCall_MemoryContext_Empty(t *testing.T) {
	s := newTestServer(t)

	params, _ := json.Marshal(ToolCallParams{
		Name:      "memory_context",
		Arguments: json.RawMessage(`{}`),
	})

	req := Request{
		JSONRPC: "2.0",
		ID:      5,
		Method:  "tools/call",
		Params:  params,
	}
	resp := s.handle(req)
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError, "empty DB context should not error")

	// Verify content is valid JSON with expected fields
	var content map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &content))
	assert.Equal(t, float64(0), content["entity_count"])
	assert.NotNil(t, content["incomplete_tasks"])
}

func TestHandle_ToolsCall_StoreAndSearch(t *testing.T) {
	s := newTestServer(t)

	// Store
	storeParams, _ := json.Marshal(ToolCallParams{
		Name: "memory_store",
		Arguments: json.RawMessage(`{
			"entities": [{"name": "PostgreSQL", "entityType": "database", "observations": ["Primary SQL DB"]}]
		}`),
	})
	storeReq := Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: storeParams}
	storeResp := s.handle(storeReq)
	assert.Nil(t, storeResp.Error)

	// Search by name
	searchParams, _ := json.Marshal(ToolCallParams{
		Name:      "memory_search",
		Arguments: json.RawMessage(`{"name": "PostgreSQL"}`),
	})
	searchReq := Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: searchParams}
	searchResp := s.handle(searchReq)
	assert.Nil(t, searchResp.Error)

	result, ok := searchResp.Result.(ToolResult)
	require.True(t, ok)
	var content map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &content))
	assert.Equal(t, float64(1), content["count"])
}

func TestHandle_ToolsCall_MemoryForget(t *testing.T) {
	s := newTestServer(t)

	// Store entity
	storeParams, _ := json.Marshal(ToolCallParams{
		Name: "memory_store",
		Arguments: json.RawMessage(`{
			"entities": [{"name": "Redis", "entityType": "system", "observations": ["Port 6379", "Version 7.2"]}]
		}`),
	})
	s.handle(Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: storeParams})

	// Retract observation
	forgetParams, _ := json.Marshal(ToolCallParams{
		Name:      "memory_forget",
		Arguments: json.RawMessage(`{"name": "Redis", "observation": "Port 6379"}`),
	})
	forgetResp := s.handle(Request{JSONRPC: "2.0", ID: 2, Method: "tools/call", Params: forgetParams})
	assert.Nil(t, forgetResp.Error)

	result, ok := forgetResp.Result.(ToolResult)
	require.True(t, ok)
	var content map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &content))
	assert.Equal(t, "retract_observation", content["action"])
}

func TestHandle_ToolsCall_MemoryLink(t *testing.T) {
	s := newTestServer(t)

	linkParams, _ := json.Marshal(ToolCallParams{
		Name:      "memory_link",
		Arguments: json.RawMessage(`{"from": "Redis", "to": "Gateway", "relation": "used-by"}`),
	})
	resp := s.handle(Request{JSONRPC: "2.0", ID: 1, Method: "tools/call", Params: linkParams})
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	assert.False(t, result.IsError)
}

func TestHandle_Notification_Initialized(t *testing.T) {
	s := newTestServer(t)
	req := Request{
		JSONRPC: "2.0",
		Method:  "notifications/initialized",
		// No ID — it's a notification
	}
	resp := s.handle(req)
	// Should return empty response (no id, no result, no error)
	assert.Nil(t, resp.ID)
	assert.Nil(t, resp.Result)
	assert.Nil(t, resp.Error)
}

func TestHandle_MethodNotFound(t *testing.T) {
	s := newTestServer(t)
	req := Request{JSONRPC: "2.0", ID: 1, Method: "unknown/method"}
	resp := s.handle(req)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, -32601, resp.Error.Code)
}

func TestHandle_Journal(t *testing.T) {
	s := newTestServer(t)

	// Append two journal entries
	for i := 0; i < 2; i++ {
		storeParams, _ := json.Marshal(ToolCallParams{
			Name:      "memory_store",
			Arguments: json.RawMessage(`{"journal": "Fixed a bug", "tags": ["fix"]}`),
		})
		resp := s.handle(Request{JSONRPC: "2.0", ID: i, Method: "tools/call", Params: storeParams})
		assert.Nil(t, resp.Error)
	}

	// Read journal
	searchParams, _ := json.Marshal(ToolCallParams{
		Name:      "memory_search",
		Arguments: json.RawMessage(`{"journal": true, "since": "24h"}`),
	})
	resp := s.handle(Request{JSONRPC: "2.0", ID: 99, Method: "tools/call", Params: searchParams})
	assert.Nil(t, resp.Error)

	result, ok := resp.Result.(ToolResult)
	require.True(t, ok)
	var content map[string]any
	require.NoError(t, json.Unmarshal([]byte(result.Content[0].Text), &content))
	assert.Equal(t, float64(2), content["count"])
}
