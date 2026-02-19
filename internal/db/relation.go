package db

import (
	"context"
	"fmt"
)

// Relation represents a typed edge between two entities.
type Relation struct {
	ID        int64  `json:"id"`
	FromID    int64  `json:"from_id"`
	FromName  string `json:"from"`
	ToID      int64  `json:"to_id"`
	ToName    string `json:"to"`
	Relation  string `json:"relation"`
	CreatedAt int64  `json:"created_at"`
}

// UpsertRelation creates a typed relation between two entities (by ID).
// It silently ignores duplicate (from, to, relation) triples.
func (db *DB) UpsertRelation(ctx context.Context, fromID, toID int64, relation string) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO relations (from_id, to_id, relation)
		VALUES (?, ?, ?)
		ON CONFLICT(from_id, to_id, relation) DO NOTHING
	`, fromID, toID, relation)
	return err
}

// UpsertRelationByName creates a relation between named entities, auto-creating them if needed.
func (db *DB) UpsertRelationByName(ctx context.Context, fromName, toName, relation string) error {
	fromID, err := db.ensureEntity(ctx, fromName)
	if err != nil {
		return fmt.Errorf("ensure entity %q: %w", fromName, err)
	}
	toID, err := db.ensureEntity(ctx, toName)
	if err != nil {
		return fmt.Errorf("ensure entity %q: %w", toName, err)
	}
	return db.UpsertRelation(ctx, fromID, toID, relation)
}

// ensureEntity returns the entity ID for name, creating it if it doesn't exist.
// Unlike UpsertEntity, this does NOT overwrite existing type/tags.
func (db *DB) ensureEntity(ctx context.Context, name string) (int64, error) {
	// Try insert-ignore first
	_, err := db.ExecContext(ctx, `
		INSERT OR IGNORE INTO entities (name, entity_type, tags) VALUES (?, 'concept', '[]')
	`, name)
	if err != nil {
		return 0, err
	}
	// Always get by name
	var id int64
	err = db.QueryRowContext(ctx, `SELECT id FROM entities WHERE name = ?`, name).Scan(&id)
	return id, err
}

// ListRelationsByEntity returns all relations involving a given entity name.
func (db *DB) ListRelationsByEntity(ctx context.Context, name string) ([]Relation, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT r.id, r.from_id, fe.name, r.to_id, te.name, r.relation, r.created_at
		FROM relations r
		JOIN entities fe ON r.from_id = fe.id
		JOIN entities te ON r.to_id = te.id
		WHERE (lower(fe.name) = lower(?) OR lower(te.name) = lower(?))
		  AND fe.deleted_at IS NULL AND te.deleted_at IS NULL
		ORDER BY r.created_at ASC
	`, name, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rels []Relation
	for rows.Next() {
		var r Relation
		if err := rows.Scan(&r.ID, &r.FromID, &r.FromName, &r.ToID, &r.ToName, &r.Relation, &r.CreatedAt); err != nil {
			return nil, err
		}
		rels = append(rels, r)
	}
	return rels, rows.Err()
}
