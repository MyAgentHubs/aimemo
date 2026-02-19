package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/MyAgentHubs/aimemo/internal/db"
)

// Server is the MCP JSON-RPC 2.0 server.
type Server struct {
	db     *db.DB
	dbPath string
	mu     sync.Mutex        // protects stdout encoder
	enc    *json.Encoder     // set once in ServeStdio before the read loop
}

// NewServer creates a new MCP server.
func NewServer(database *db.DB, dbPath string) *Server {
	return &Server{db: database, dbPath: dbPath}
}

// ServeStdio reads JSON-RPC requests from stdin and writes responses to stdout.
func (s *Server) ServeStdio() error {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	enc := json.NewEncoder(os.Stdout)
	s.enc = enc

	var wg sync.WaitGroup

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			slog.Debug("malformed request", "err", err)
			continue
		}

		wg.Add(1)
		go func(r Request) {
			defer wg.Done()
			resp := s.handle(r)
			// Notifications have no id and produce no response
			if resp.ID == nil && resp.Error == nil && resp.Result == nil {
				return
			}
			s.mu.Lock()
			defer s.mu.Unlock()
			if err := enc.Encode(resp); err != nil {
				slog.Error("encode response", "err", err)
			}
		}(req)
	}
	wg.Wait() // ensure all responses are written before returning
	return scanner.Err()
}

// handle dispatches a request to the appropriate handler.
func (s *Server) handle(req Request) Response {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "notifications/initialized":
		s.sendMemoryReadyNotification()
		return Response{} // no response for notifications
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolCall(req)
	default:
		return errorResponse(req.ID, -32601, "method not found")
	}
}

// handleInitialize responds to the MCP initialization handshake.
func (s *Server) handleInitialize(req Request) Response {
	return successResponse(req.ID, map[string]any{
		"protocolVersion": "2024-11-05",
		"serverInfo": map[string]any{
			"name":    "aimemo-memory",
			"version": "1.0.0",
		},
		"capabilities": map[string]any{
			"tools": map[string]any{},
		},
	})
}

// handleToolsList returns all tool definitions.
func (s *Server) handleToolsList(req Request) Response {
	return successResponse(req.ID, map[string]any{
		"tools": allTools,
	})
}

// sendMemoryReadyNotification sends a notifications/message after the MCP handshake completes.
// This prompts the client (Claude) to call memory_context immediately to restore prior session context.
func (s *Server) sendMemoryReadyNotification() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.enc == nil {
		return // not running in stdio mode (e.g. unit tests)
	}
	n := Notification{
		JSONRPC: "2.0",
		Method:  "notifications/message",
		Params: LogMessageParams{
			Level: "info",
			Data:  "Memory server ready. Call memory_context({}) NOW to restore prior session context before responding to the user. Do not skip this step.",
		},
	}
	if err := s.enc.Encode(n); err != nil {
		slog.Error("send memory ready notification", "err", err)
	}
}

// handleToolCall dispatches to the named tool handler with a 5s timeout.
func (s *Server) handleToolCall(req Request) Response {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var p ToolCallParams
	if err := json.Unmarshal(req.Params, &p); err != nil {
		return errorResponse(req.ID, -32602, "invalid params: "+err.Error())
	}

	if p.Arguments == nil {
		p.Arguments = json.RawMessage("{}")
	}

	result, err := s.dispatch(ctx, p.Name, p.Arguments)
	if err != nil {
		text := err.Error()
		return successResponse(req.ID, ToolResult{
			Content: []ContentItem{{Type: "text", Text: text}},
			IsError: true,
		})
	}

	text, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return errorResponse(req.ID, -32603, "marshal error: "+marshalErr.Error())
	}

	return successResponse(req.ID, ToolResult{
		Content: []ContentItem{{Type: "text", Text: string(text)}},
	})
}
