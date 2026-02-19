package mcp

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/MyAgentHubs/aimemo/internal/db"
)

// handleMemoryContext returns session orientation data.
// Runs sub-queries in parallel for <50ms with 10k entities.
func (s *Server) handleMemoryContext(ctx context.Context, args json.RawMessage) (any, error) {
	var p struct {
		Since string `json:"since"`
		Limit int    `json:"limit"`
	}
	if args != nil {
		_ = json.Unmarshal(args, &p)
	}
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Since == "" {
		p.Since = "24h"
	}

	type result struct {
		recentObs     []recentObservation
		topEntities   []db.SearchResult
		stats         db.Stats
		lastSession   int64
		recentJournal []db.JournalEntry
		err           error
	}

	ch := make(chan result, 1)

	go func() {
		var r result
		var wg sync.WaitGroup
		var mu sync.Mutex

		wg.Add(4)

		// Recent observations
		go func() {
			defer wg.Done()
			obs, err := recentObservations(ctx, s.db, p.Since, p.Limit)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = err
			} else {
				r.recentObs = obs
			}
			mu.Unlock()
		}()

		// Top entities by importance (list-all, sorted by recent, limit 10)
		go func() {
			defer wg.Done()
			entities, err := s.db.Search(ctx, "", "", nil, "recent", 10)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = err
			} else {
				r.topEntities = entities
			}
			mu.Unlock()
		}()

		// Stats
		go func() {
			defer wg.Done()
			stats, err := s.db.GetStats(ctx)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = err
			} else {
				r.stats = stats
			}
			mu.Unlock()
		}()

		// Recent journal entries
		go func() {
			defer wg.Done()
			entries, err := s.db.ListJournal(ctx, p.Since, 5)
			mu.Lock()
			if err != nil && r.err == nil {
				r.err = err
			} else {
				r.recentJournal = entries
			}
			mu.Unlock()
		}()

		wg.Wait()
		ch <- r
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		if r.recentObs == nil {
			r.recentObs = []recentObservation{}
		}
		if r.topEntities == nil {
			r.topEntities = []db.SearchResult{}
		}
		if r.recentJournal == nil {
			r.recentJournal = []db.JournalEntry{}
		}
		return map[string]any{
			"storage_path":        s.dbPath,
			"entity_count":        r.stats.EntityCount,
			"observation_count":   r.stats.ObservationCount,
			"recent_observations": r.recentObs,
			"top_entities":        r.topEntities,
			"recent_journal":      r.recentJournal,
			"incomplete_tasks":    []any{},
			"generated_at":        time.Now().UnixMilli(),
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// recentObservation is a flattened observation for memory_context response.
type recentObservation struct {
	EntityName string `json:"entity_name"`
	Content    string `json:"content"`
	CreatedAt  int64  `json:"created_at"`
}

// recentObservations fetches the N most recent observations across all entities.
func recentObservations(ctx context.Context, database *db.DB, since string, limit int) ([]recentObservation, error) {
	sinceMs, err := db.ParseSince(since)
	if err != nil {
		sinceMs = time.Now().Add(-24 * time.Hour).UnixMilli()
	}

	rows, err := database.QueryContext(ctx, `
		SELECT e.name, o.content, o.created_at
		FROM observations o
		JOIN entities e ON o.entity_id = e.id
		WHERE e.deleted_at IS NULL AND o.created_at >= ?
		ORDER BY o.created_at DESC
		LIMIT ?
	`, sinceMs, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var obs []recentObservation
	for rows.Next() {
		var r recentObservation
		if err := rows.Scan(&r.EntityName, &r.Content, &r.CreatedAt); err != nil {
			return nil, err
		}
		obs = append(obs, r)
	}
	return obs, rows.Err()
}
