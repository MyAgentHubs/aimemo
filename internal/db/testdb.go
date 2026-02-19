package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// NewTestDB creates an in-memory SQLite database for testing.
func NewTestDB(t *testing.T) *DB {
	t.Helper()
	db, err := Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })
	return db
}
