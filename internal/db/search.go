package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// SearchResult extends Entity with a search score.
type SearchResult struct {
	Entity
	Score float64 `json:"score"`
}

// ftsEscape wraps user query in double quotes for FTS5 safety.
func ftsEscape(q string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	// Escape internal double quotes
	q = strings.ReplaceAll(q, `"`, `""`)
	return `"` + q + `"`
}

// Search performs FTS5 search across entities and observations.
// If query is empty, lists all active entities.
// limit=0 uses the default (10). Callers are responsible for enforcing max limits.
func (db *DB) Search(ctx context.Context, query, entityType string, tags []string, sort string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 10
	}

	if query == "" {
		return db.listAll(ctx, entityType, tags, sort, limit)
	}

	escaped := ftsEscape(query)
	// bm25() in CTEs is unreliable across SQLite versions. Use IN-subquery approach
	// to find matching entity IDs, then rank by importance score.
	sqlQuery := `
SELECT DISTINCT
    e.id, e.name, e.entity_type, e.tags, e.created_at, e.updated_at, e.deleted_at,
    e.access_count, e.last_accessed,
    (0.6 / LOG(((unixepoch('now') * 1000 - e.updated_at) / 3600000.0) + 2) + 0.4 * LOG(e.access_count + 1)) AS importance_rank
FROM entities e
WHERE e.deleted_at IS NULL
  AND (
    e.id IN (SELECT rowid FROM entities_fts WHERE entities_fts MATCH ?)
    OR e.id IN (
        SELECT o.entity_id FROM observations o
        WHERE o.id IN (SELECT rowid FROM observations_fts WHERE observations_fts MATCH ?)
    )
  )`

	args := []interface{}{escaped, escaped}

	if entityType != "" {
		sqlQuery += " AND e.entity_type = ?"
		args = append(args, entityType)
	}

	if len(tags) > 0 {
		sqlQuery += fmt.Sprintf(
			" AND (SELECT COUNT(*) FROM json_each(e.tags) WHERE value IN (%s)) = %d",
			placeholders(len(tags)), len(tags),
		)
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	sqlQuery += " ORDER BY importance_rank DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	return db.scanSearchRows(ctx, rows)
}

// SearchByName does an exact (case-insensitive) name lookup with observation loading.
func (db *DB) SearchByName(ctx context.Context, name string) (*Entity, error) {
	return db.GetEntity(ctx, name)
}

// listAll returns entities sorted by sort order.
func (db *DB) listAll(ctx context.Context, entityType string, tags []string, sort string, limit int) ([]SearchResult, error) {
	orderBy := "e.updated_at DESC"
	switch sort {
	case "accessed":
		orderBy = "COALESCE(e.last_accessed, 0) DESC"
	case "name":
		orderBy = "e.name ASC"
	}

	query := `
		SELECT DISTINCT e.id, e.name, e.entity_type, e.tags, e.created_at, e.updated_at, e.deleted_at,
		       e.access_count, e.last_accessed, 0.0 AS final_rank
		FROM entities e
		WHERE e.deleted_at IS NULL`
	args := []interface{}{}

	if entityType != "" {
		query += " AND e.entity_type = ?"
		args = append(args, entityType)
	}

	if len(tags) > 0 {
		query += fmt.Sprintf(
			" AND (SELECT COUNT(*) FROM json_each(e.tags) WHERE value IN (%s)) = %d",
			placeholders(len(tags)), len(tags),
		)
		for _, tag := range tags {
			args = append(args, tag)
		}
	}

	query += " ORDER BY " + orderBy + " LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return db.scanSearchRows(ctx, rows)
}

// scanSearchRows scans rows into SearchResult slice and loads observations.
// Observations are loaded in a second pass after closing the search rows,
// to avoid deadlock on single-connection DBs.
func (db *DB) scanSearchRows(ctx context.Context, rows *sql.Rows) ([]SearchResult, error) {
	var results []SearchResult
	for rows.Next() {
		var e Entity
		var tagsJSON string
		var deletedAt sql.NullInt64
		var lastAccessed sql.NullInt64
		var score float64

		if err := rows.Scan(
			&e.ID, &e.Name, &e.EntityType, &tagsJSON,
			&e.CreatedAt, &e.UpdatedAt, &deletedAt,
			&e.AccessCount, &lastAccessed, &score,
		); err != nil {
			return nil, err
		}

		if deletedAt.Valid {
			e.DeletedAt = &deletedAt.Int64
		}
		if lastAccessed.Valid {
			e.LastAccessed = &lastAccessed.Int64
		}

		var tags []string
		if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil || tags == nil {
			tags = []string{}
		}
		e.Tags = tags

		results = append(results, SearchResult{Entity: e, Score: score})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	rows.Close() // release the connection before loading observations

	// Second pass: load observations (requires a new query)
	for i := range results {
		obs, err := db.ListObservationsByEntityID(ctx, results[i].ID)
		if err != nil {
			return nil, err
		}
		results[i].Observations = obs
	}
	return results, nil
}

// Stats returns counts of entities, observations, and journal entries.
type Stats struct {
	EntityCount      int    `json:"entity_count"`
	ObservationCount int    `json:"observation_count"`
	RelationCount    int    `json:"relation_count"`
	JournalCount     int    `json:"journal_count"`
	StoragePath      string `json:"storage_path"`
}

// GetStats returns database statistics.
func (db *DB) GetStats(ctx context.Context) (Stats, error) {
	var s Stats
	err := db.QueryRowContext(ctx, `
		SELECT
			(SELECT COUNT(*) FROM entities WHERE deleted_at IS NULL),
			(SELECT COUNT(*) FROM observations o JOIN entities e ON o.entity_id = e.id WHERE e.deleted_at IS NULL),
			(SELECT COUNT(*) FROM relations r JOIN entities fe ON r.from_id = fe.id JOIN entities te ON r.to_id = te.id WHERE fe.deleted_at IS NULL AND te.deleted_at IS NULL),
			(SELECT COUNT(*) FROM journal)
	`).Scan(&s.EntityCount, &s.ObservationCount, &s.RelationCount, &s.JournalCount)
	return s, err
}
