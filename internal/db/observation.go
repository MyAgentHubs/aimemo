package db

import (
	"context"
	"fmt"
)

// AddObservation adds an observation to an entity, deduplicating via UNIQUE constraint.
func (db *DB) AddObservation(ctx context.Context, entityID int64, content string) error {
	if len(content) > 10*1024 {
		return fmt.Errorf("observation content exceeds 10KB limit")
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO observations (entity_id, content)
		VALUES (?, ?)
		ON CONFLICT(entity_id, content) DO NOTHING
	`, entityID, content)
	return err
}

// ListObservationsByEntityID returns all observation contents for an entity.
func (db *DB) ListObservationsByEntityID(ctx context.Context, entityID int64) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT content FROM observations WHERE entity_id = ? ORDER BY created_at ASC
	`, entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var obs []string
	for rows.Next() {
		var content string
		if err := rows.Scan(&content); err != nil {
			return nil, err
		}
		obs = append(obs, content)
	}
	return obs, rows.Err()
}

// RetractObservation removes a specific observation from an entity by exact content match.
// Returns remaining observations after deletion.
func (db *DB) RetractObservation(ctx context.Context, entityName, content string) ([]string, error) {
	// Get entity id
	var entityID int64
	err := db.QueryRowContext(ctx, `
		SELECT id FROM entities WHERE lower(name) = lower(?) AND deleted_at IS NULL
	`, entityName).Scan(&entityID)
	if err != nil {
		return nil, fmt.Errorf("entity %q not found", entityName)
	}

	res, err := db.ExecContext(ctx, `
		DELETE FROM observations WHERE entity_id = ? AND content = ?
	`, entityID, content)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, fmt.Errorf("observation not found in entity %q", entityName)
	}

	return db.ListObservationsByEntityID(ctx, entityID)
}
