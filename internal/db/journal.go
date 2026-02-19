package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JournalEntry is a single timestamped log entry.
type JournalEntry struct {
	ID        int64    `json:"id"`
	Content   string   `json:"content"`
	Tags      []string `json:"tags"`
	CreatedAt int64    `json:"created_at"`
}

// AppendJournal writes a new journal entry (no deduplication).
func (db *DB) AppendJournal(ctx context.Context, content string, tags []string) (*JournalEntry, error) {
	if len(content) > 10*1024 {
		return nil, fmt.Errorf("journal content exceeds 10KB limit")
	}
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, err
	}

	res, err := db.ExecContext(ctx, `
		INSERT INTO journal (content, tags) VALUES (?, ?)
	`, content, string(tagsJSON))
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()

	var entry JournalEntry
	err = db.QueryRowContext(ctx, `SELECT id, content, tags, created_at FROM journal WHERE id = ?`, id).
		Scan(&entry.ID, &entry.Content, &tagsJSON, &entry.CreatedAt)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(tagsJSON), &entry.Tags); err != nil {
		entry.Tags = []string{}
	}
	return &entry, nil
}

// ParseSince parses a duration string like "2h", "24h", "7d", or ISO date "2026-02-17".
// Returns the Unix millisecond timestamp for the start of the window.
func ParseSince(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		// default 24h
		return time.Now().Add(-24 * time.Hour).UnixMilli(), nil
	}

	// Try ISO date first: YYYY-MM-DD
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t.UnixMilli(), nil
	}

	// Try duration: Nh or Nd
	if strings.HasSuffix(s, "d") {
		var days int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "d"), "%d", &days); err == nil {
			return time.Now().Add(-time.Duration(days) * 24 * time.Hour).UnixMilli(), nil
		}
	}
	if strings.HasSuffix(s, "h") {
		var hours int
		if _, err := fmt.Sscanf(strings.TrimSuffix(s, "h"), "%d", &hours); err == nil {
			return time.Now().Add(-time.Duration(hours) * time.Hour).UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("cannot parse since %q: use formats like '2h', '24h', '7d', or '2026-02-17'", s)
}

// ListJournal returns journal entries optionally filtered by a time window.
// sinceStr is a duration like "24h", "7d", ISO date, or empty for all.
func (db *DB) ListJournal(ctx context.Context, sinceStr string, limit int) ([]JournalEntry, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, content, tags, created_at FROM journal`
	args := []interface{}{}

	if sinceStr != "" {
		sinceMs, err := ParseSince(sinceStr)
		if err != nil {
			return nil, err
		}
		query += " WHERE created_at >= ?"
		args = append(args, sinceMs)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanJournalRows(rows)
}

// SearchJournal performs FTS5 full-text search on journal content.
func (db *DB) SearchJournal(ctx context.Context, query string, limit int) ([]JournalEntry, error) {
	if limit <= 0 {
		limit = 10
	}
	escaped := ftsEscape(query)
	rows, err := db.QueryContext(ctx, `
		SELECT j.id, j.content, j.tags, j.created_at
		FROM journal j
		WHERE j.id IN (SELECT rowid FROM journal_fts WHERE journal_fts MATCH ?)
		ORDER BY j.created_at DESC
		LIMIT ?
	`, escaped, limit)
	if err != nil {
		return nil, fmt.Errorf("journal search: %w", err)
	}
	defer rows.Close()
	return scanJournalRows(rows)
}

func scanJournalRows(rows *sql.Rows) ([]JournalEntry, error) {
	var entries []JournalEntry
	for rows.Next() {
		var e JournalEntry
		var tagsJSON string
		if err := rows.Scan(&e.ID, &e.Content, &tagsJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(tagsJSON), &e.Tags); err != nil {
			e.Tags = []string{}
		}
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
