package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/MyAgentHubs/aimemo/internal/db"
)

var allTools = []Tool{
	{
		Name: "memory_context",
		Description: `Call this FIRST before responding to any coding task, bug fix, feature request, or question about the project. Without it you have no memory of previous sessions and will repeat work already done or miss known constraints.

WHEN TO CALL (call immediately, do not wait for user to ask):
- User says "help me with", "fix", "implement", "add", "why does", "how does", or starts any work
- User says "continue", "where were we", "what's the status", "last time we..."
- Any unfamiliar codebase or project name appears in the conversation

COST OF SKIPPING: You lose all prior context — previous decisions, known bugs, in-progress work.
RETURNS: Recent activity (last 24h by default), top entities, DB path.
EXAMPLE: memory_context({}) — call with no arguments to load default context.`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"since": map[string]any{"type": "string", "description": "Time window: 2h|24h|7d|ISO date (default 24h)"},
				"limit": map[string]any{"type": "integer", "description": "Max recent observations (default 20)"},
			},
		},
	},
	{
		Name: "memory_store",
		Description: `Store facts, decisions, and progress notes so you remember them in future sessions. Also used to write journal entries at session end. Store proactively — do not wait to be asked.

WHEN TO CALL:
- Immediately after completing a task, fixing a bug, or making an architectural decision — store it before moving on
- When you discover something important: a non-obvious code pattern, a constraint, a gotcha
- When the user says "done", "thanks", "that's all", "good", or the conversation is wrapping up — write a journal entry
- Before ending a session: journal entry with what was completed, what is in progress, any blockers

ENTITY TYPES: project, module, bug, decision, person, concept, system — use whatever fits.
JOURNAL: Use the journal field (not entities) for session logs. Journal entries are append-only and never deduplicated.

EXAMPLES:
- Store a fact: memory_store({entities: [{name: "auth-service", entityType: "module", observations: ["Uses JWT with 1h expiry", "Refresh token stored in Redis"]}]})
- End-of-session log: memory_store({journal: "Completed: JWT refresh flow. In progress: rate limiting. Blocker: Redis connection pooling under load."})`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entities": map[string]any{
					"type":        "array",
					"description": "Entities to store (mutually exclusive with journal)",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"name":         map[string]any{"type": "string"},
							"entityType":   map[string]any{"type": "string"},
							"observations": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
							"tags":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						},
						"required": []string{"name", "entityType", "observations"},
					},
				},
				"journal": map[string]any{"type": "string", "description": "Journal entry (no dedup; mutually exclusive with entities)"},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Tags for journal entry"},
				"context": map[string]any{"type": "string", "description": "Named memory context"},
			},
		},
	},
	{
		Name: "memory_search",
		Description: `Search stored memory by keyword, exact name, or browse all entities. Also reads journal logs.

WHEN TO CALL: When you need to recall something specific — a past decision, a bug fix, a person's role.
VS memory_context: memory_context gives you recent activity; memory_search finds specific things by keyword.

EXAMPLES:
- Keyword search: memory_search({query: "redis connection"})
- Exact lookup: memory_search({name: "auth-service"})
- List all: memory_search({query: ""})
- Read journal: memory_search({journal: true, since: "7d"})`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query":   map[string]any{"type": "string", "description": "FTS search query; empty string = list all"},
				"name":    map[string]any{"type": "string", "description": "Exact entity name lookup (priority over query)"},
				"journal": map[string]any{"type": "boolean", "description": "Read journal entries instead of entities"},
				"since":   map[string]any{"type": "string", "description": "Time filter for journal: 2h|24h|7d|ISO date"},
				"context": map[string]any{"type": "string", "description": "Named memory context"},
				"type":    map[string]any{"type": "string", "description": "Filter by entity type"},
				"tags":    map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "AND tag filter"},
				"limit":   map[string]any{"type": "integer", "description": "Max results (default 10, max 50)"},
				"sort":    map[string]any{"type": "string", "enum": []string{"recent", "accessed", "name"}, "description": "Sort order for list mode"},
			},
		},
	},
	{
		Name: "memory_forget",
		Description: `Correct wrong information: retract a single bad observation, or soft-delete a whole entity. Soft-delete is reversible; use permanent:true only when sure.

WHEN TO CALL: When you stored something incorrect, or a project/entity is no longer relevant.
EXAMPLES:
- Remove one wrong fact: memory_forget({name: "auth-service", observation: "Uses JWT with 1h expiry"})
- Soft-delete entity: memory_forget({name: "old-feature"})`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"name":        map[string]any{"type": "string", "description": "Entity name"},
				"observation": map[string]any{"type": "string", "description": "Exact observation to retract; omit to delete entity"},
				"permanent":   map[string]any{"type": "boolean", "description": "Hard delete (irreversible); default false"},
				"context":     map[string]any{"type": "string", "description": "Named memory context"},
			},
			"required": []string{"name"},
		},
	},
	{
		Name: "memory_link",
		Description: `Connect two entities with a named relationship. Use this to map dependencies, ownership, and associations between things you have stored.

WHEN TO CALL: When you identify that two stored entities are related — a bug fixed by a commit, a module depending on another, a decision driving a design.
RELATION TYPES: uses, fixes, depends_on, implements, owns, blocks, related_to — use active-voice verbs.
EXAMPLES:
- memory_link({from: "payment-service", to: "Redis", relation: "uses"})
- memory_link({from: "fix/race-condition", to: "refund-handler", relation: "fixes"})`,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"from":     map[string]any{"type": "string", "description": "Source entity name"},
				"to":       map[string]any{"type": "string", "description": "Target entity name"},
				"relation": map[string]any{"type": "string", "description": "Relation type, e.g. uses|fixes|depends_on"},
				"context":  map[string]any{"type": "string", "description": "Named memory context"},
			},
			"required": []string{"from", "to", "relation"},
		},
	},
}

// dispatch routes a tool call to the appropriate handler.
func (s *Server) dispatch(ctx context.Context, name string, args json.RawMessage) (any, error) {
	switch name {
	case "memory_context":
		return s.handleMemoryContext(ctx, args)
	case "memory_store":
		return s.handleMemoryStore(ctx, args)
	case "memory_search":
		return s.handleMemorySearch(ctx, args)
	case "memory_forget":
		return s.handleMemoryForget(ctx, args)
	case "memory_link":
		return s.handleMemoryLink(ctx, args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", name)
	}
}

// handleMemoryStore dispatches to journal or entity mode.
func (s *Server) handleMemoryStore(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Entities []db.EntityInput `json:"entities"`
		Journal  string           `json:"journal"`
		Tags     []string         `json:"tags"`
		Context  string           `json:"context"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Journal != "" {
		entry, err := s.db.AppendJournal(ctx, p.Journal, p.Tags)
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"stored": "journal",
			"id":     entry.ID,
		}, nil
	}

	if len(p.Entities) == 0 {
		return nil, fmt.Errorf("entities or journal is required")
	}

	results, err := s.db.StoreEntities(ctx, p.Entities)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"stored":   "entities",
		"count":    len(results),
		"entities": results,
	}, nil
}

// handleMemorySearch dispatches to journal or entity search mode.
func (s *Server) handleMemorySearch(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Query   string   `json:"query"`
		Name    string   `json:"name"`
		Journal bool     `json:"journal"`
		Since   string   `json:"since"`
		Context string   `json:"context"`
		Type    string   `json:"type"`
		Tags    []string `json:"tags"`
		Limit   int      `json:"limit"`
		Sort    string   `json:"sort"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}

	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 50 {
		p.Limit = 50
	}

	// Journal mode
	if p.Journal {
		entries, err := s.db.ListJournal(ctx, p.Since, p.Limit)
		if err != nil {
			return nil, err
		}
		if entries == nil {
			entries = []db.JournalEntry{}
		}
		return map[string]any{
			"journal": entries,
			"count":   len(entries),
		}, nil
	}

	// Exact name lookup
	if p.Name != "" {
		e, err := s.db.SearchByName(ctx, p.Name)
		if err != nil {
			return nil, err
		}
		if e == nil {
			return map[string]any{"entities": []any{}, "count": 0}, nil
		}
		return map[string]any{
			"entities": []any{e},
			"count":    1,
		}, nil
	}

	// FTS or list-all search
	results, err := s.db.Search(ctx, p.Query, p.Type, p.Tags, p.Sort, p.Limit)
	if err != nil {
		return nil, err
	}
	if results == nil {
		results = []db.SearchResult{}
	}
	resp := map[string]any{
		"entities": results,
		"count":    len(results),
	}

	// When a keyword query is given, also search journal entries via FTS.
	if p.Query != "" {
		journalResults, err := s.db.SearchJournal(ctx, p.Query, p.Limit)
		if err != nil {
			return nil, err
		}
		if journalResults == nil {
			journalResults = []db.JournalEntry{}
		}
		resp["journal"] = journalResults
		resp["journal_count"] = len(journalResults)
	}

	return resp, nil
}

// handleMemoryForget dispatches to retract or delete.
func (s *Server) handleMemoryForget(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Name        string `json:"name"`
		Observation string `json:"observation"`
		Permanent   bool   `json:"permanent"`
		Context     string `json:"context"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.Name == "" {
		return nil, fmt.Errorf("name is required")
	}

	if p.Observation != "" {
		remaining, err := s.db.RetractObservation(ctx, p.Name, p.Observation)
		if err != nil {
			return nil, err
		}
		if remaining == nil {
			remaining = []string{}
		}
		return map[string]any{
			"action":                  "retract_observation",
			"entity":                  p.Name,
			"deleted":                 p.Observation,
			"remaining_observations":  remaining,
		}, nil
	}

	if p.Permanent {
		if err := s.db.HardDeleteEntity(ctx, p.Name); err != nil {
			return nil, err
		}
		return map[string]any{
			"action": "hard_delete",
			"entity": p.Name,
		}, nil
	}

	if err := s.db.SoftDeleteEntity(ctx, p.Name); err != nil {
		return nil, err
	}
	return map[string]any{
		"action": "soft_delete",
		"entity": p.Name,
	}, nil
}

// handleMemoryLink creates a typed relation between two entities.
func (s *Server) handleMemoryLink(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		From     string `json:"from"`
		To       string `json:"to"`
		Relation string `json:"relation"`
		Context  string `json:"context"`
	}
	if err := json.Unmarshal(args, &p); err != nil {
		return nil, fmt.Errorf("invalid params: %w", err)
	}
	if p.From == "" || p.To == "" || p.Relation == "" {
		return nil, fmt.Errorf("from, to, and relation are required")
	}

	if err := s.db.UpsertRelationByName(ctx, p.From, p.To, p.Relation); err != nil {
		return nil, err
	}
	return map[string]any{
		"from":     p.From,
		"to":       p.To,
		"relation": p.Relation,
		"created":  time.Now().UnixMilli(),
	}, nil
}
