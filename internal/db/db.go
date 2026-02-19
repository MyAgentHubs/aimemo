package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

// DB wraps sql.DB with aimemo-specific helpers.
type DB struct {
	*sql.DB
}

// Open opens (or creates) the SQLite database at path, configures WAL mode and FTS5, and runs migrations.
func Open(path string) (*DB, error) {
	sqldb, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// Configure connection pool for SQLite (single writer model)
	sqldb.SetMaxOpenConns(1)

	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA foreign_keys=ON",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA cache_size=-64000",
		"PRAGMA busy_timeout=5000",
	}
	for _, p := range pragmas {
		if _, err := sqldb.Exec(p); err != nil {
			sqldb.Close()
			return nil, fmt.Errorf("pragma %q: %w", p, err)
		}
	}

	db := &DB{sqldb}
	if err := db.migrate(); err != nil {
		sqldb.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// migrate runs the schema creation statements and ensures FTS indexes are populated.
func (db *DB) migrate() error {
	if _, err := db.Exec(Schema); err != nil {
		return err
	}
	// Rebuild journal_fts to index any pre-existing journal rows that were
	// inserted before the journal_fts table existed.
	_, err := db.Exec(`INSERT INTO journal_fts(journal_fts) VALUES('rebuild')`)
	return err
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.DB.Close()
}
