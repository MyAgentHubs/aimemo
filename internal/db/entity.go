package db

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Entity represents a named entity in the knowledge graph.
type Entity struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	EntityType   string   `json:"entity_type"`
	Tags         []string `json:"tags"`
	CreatedAt    int64    `json:"created_at"`
	UpdatedAt    int64    `json:"updated_at"`
	DeletedAt    *int64   `json:"deleted_at,omitempty"`
	AccessCount  int64    `json:"access_count"`
	LastAccessed *int64   `json:"last_accessed,omitempty"`
	Observations []string `json:"observations,omitempty"`
}

// EntityInput is used for upserting entities.
type EntityInput struct {
	Name         string   `json:"name"`
	EntityType   string   `json:"entityType"`
	Observations []string `json:"observations"`
	Tags         []string `json:"tags"`
}

// scanEntity scans a row into an Entity (without Observations).
func scanEntity(row interface {
	Scan(...interface{}) error
}) (*Entity, error) {
	var e Entity
	var tagsJSON string
	var deletedAt sql.NullInt64
	var lastAccessed sql.NullInt64

	err := row.Scan(
		&e.ID, &e.Name, &e.EntityType, &tagsJSON,
		&e.CreatedAt, &e.UpdatedAt, &deletedAt,
		&e.AccessCount, &lastAccessed,
	)
	if err != nil {
		return nil, err
	}

	if deletedAt.Valid {
		e.DeletedAt = &deletedAt.Int64
	}
	if lastAccessed.Valid {
		e.LastAccessed = &lastAccessed.Int64
	}

	if err := json.Unmarshal([]byte(tagsJSON), &e.Tags); err != nil {
		e.Tags = []string{}
	}
	if e.Tags == nil {
		e.Tags = []string{}
	}
	return &e, nil
}

// UpsertEntity upserts an entity (insert or update name/type/tags/updated_at).
// Returns the entity ID.
func (db *DB) UpsertEntity(ctx context.Context, name, entityType string, tags []string) (int64, error) {
	if len(name) == 0 {
		return 0, fmt.Errorf("entity name cannot be empty")
	}
	if len(name) > 1024 {
		return 0, fmt.Errorf("entity name exceeds 1KB limit")
	}
	if len(entityType) > 256 {
		return 0, fmt.Errorf("entity type exceeds 256-byte limit")
	}
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return 0, err
	}

	// Upsert: insert or update on conflict (no WHERE so it always upserts)
	_, err = db.ExecContext(ctx, `
		INSERT INTO entities (name, entity_type, tags, updated_at)
		VALUES (?, ?, ?, unixepoch('now', 'subsec') * 1000)
		ON CONFLICT(name) DO UPDATE SET
			entity_type = excluded.entity_type,
			tags = excluded.tags,
			updated_at = excluded.updated_at,
			deleted_at = NULL
	`, name, entityType, string(tagsJSON))
	if err != nil {
		return 0, fmt.Errorf("upsert entity: %w", err)
	}

	// Always query for the ID — LastInsertId() is unreliable for UPSERT
	var id int64
	err = db.QueryRowContext(ctx, `SELECT id FROM entities WHERE name = ?`, name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get entity id: %w", err)
	}
	return id, nil
}

// GetEntity retrieves an entity by name (case-insensitive), with its observations.
func (db *DB) GetEntity(ctx context.Context, name string) (*Entity, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, entity_type, tags, created_at, updated_at, deleted_at, access_count, last_accessed
		FROM entities
		WHERE lower(name) = lower(?) AND deleted_at IS NULL
	`, name)

	e, err := scanEntity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	// Update access count
	now := time.Now().UnixMilli()
	_, _ = db.ExecContext(ctx, `
		UPDATE entities SET access_count = access_count + 1, last_accessed = ? WHERE id = ?
	`, now, e.ID)
	e.AccessCount++
	e.LastAccessed = &now

	obs, err := db.ListObservationsByEntityID(ctx, e.ID)
	if err != nil {
		return nil, err
	}
	e.Observations = obs
	return e, nil
}

// GetEntityByID retrieves an entity by ID.
func (db *DB) GetEntityByID(ctx context.Context, id int64) (*Entity, error) {
	row := db.QueryRowContext(ctx, `
		SELECT id, name, entity_type, tags, created_at, updated_at, deleted_at, access_count, last_accessed
		FROM entities WHERE id = ? AND deleted_at IS NULL
	`, id)
	e, err := scanEntity(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return e, err
}

// ListEntities lists active entities with optional filters.
func (db *DB) ListEntities(ctx context.Context, entityType string, tags []string, sort string, limit int) ([]*Entity, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	orderBy := "e.updated_at DESC"
	switch sort {
	case "accessed":
		orderBy = "e.last_accessed DESC"
	case "name":
		orderBy = "e.name ASC"
	}

	query := `
		SELECT DISTINCT e.id, e.name, e.entity_type, e.tags, e.created_at, e.updated_at, e.deleted_at, e.access_count, e.last_accessed
		FROM entities e
		WHERE e.deleted_at IS NULL
	`
	args := []interface{}{}

	if entityType != "" {
		query += " AND e.entity_type = ?"
		args = append(args, entityType)
	}

	if len(tags) > 0 {
		// AND tag filter
		query += fmt.Sprintf(`
			AND (SELECT COUNT(*) FROM json_each(e.tags) WHERE value IN (%s)) = %d
		`, placeholders(len(tags)), len(tags))
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

	var entities []*Entity
	for rows.Next() {
		e, err := scanEntity(rows)
		if err != nil {
			return nil, err
		}
		entities = append(entities, e)
	}
	return entities, rows.Err()
}

// SoftDeleteEntity soft-deletes an entity by name.
func (db *DB) SoftDeleteEntity(ctx context.Context, name string) error {
	now := time.Now().UnixMilli()
	res, err := db.ExecContext(ctx, `
		UPDATE entities SET deleted_at = ? WHERE lower(name) = lower(?) AND deleted_at IS NULL
	`, now, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entity %q not found", name)
	}
	return nil
}

// HardDeleteEntity permanently deletes an entity and all its observations/relations.
func (db *DB) HardDeleteEntity(ctx context.Context, name string) error {
	res, err := db.ExecContext(ctx, `DELETE FROM entities WHERE lower(name) = lower(?)`, name)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("entity %q not found", name)
	}
	return nil
}

// StoreEntities upserts a batch of entities with their observations.
func (db *DB) StoreEntities(ctx context.Context, inputs []EntityInput) ([]Entity, error) {
	var results []Entity
	for _, inp := range inputs {
		entityType := inp.EntityType
		if entityType == "" {
			entityType = "concept"
		}
		id, err := db.UpsertEntity(ctx, inp.Name, entityType, inp.Tags)
		if err != nil {
			return nil, fmt.Errorf("upsert %q: %w", inp.Name, err)
		}
		for _, obs := range inp.Observations {
			if err := db.AddObservation(ctx, id, obs); err != nil {
				return nil, fmt.Errorf("add observation to %q: %w", inp.Name, err)
			}
		}
		e, err := db.GetEntityByID(ctx, id)
		if err != nil {
			return nil, err
		}
		obs, _ := db.ListObservationsByEntityID(ctx, id)
		if e != nil {
			e.Observations = obs
			results = append(results, *e)
		}
	}
	return results, nil
}

// placeholders returns n comma-separated "?" for SQL IN clauses.
func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	s := "?"
	for i := 1; i < n; i++ {
		s += ",?"
	}
	return s
}
