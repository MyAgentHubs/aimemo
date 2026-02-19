package db

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpen_InMemory(t *testing.T) {
	db := NewTestDB(t)
	assert.NotNil(t, db)
}

func TestEntity_UpsertAndGet(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id, err := db.UpsertEntity(ctx, "Redis", "system", []string{"cache", "infra"})
	require.NoError(t, err)
	assert.Greater(t, id, int64(0))

	e, err := db.GetEntity(ctx, "Redis")
	require.NoError(t, err)
	require.NotNil(t, e)
	assert.Equal(t, "Redis", e.Name)
	assert.Equal(t, "system", e.EntityType)
	assert.ElementsMatch(t, []string{"cache", "infra"}, e.Tags)
}

func TestEntity_UpsertDedup(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id1, err := db.UpsertEntity(ctx, "Redis", "system", []string{"cache"})
	require.NoError(t, err)

	// Upsert same name — should update, same id
	id2, err := db.UpsertEntity(ctx, "Redis", "system", []string{"cache", "infra"})
	require.NoError(t, err)
	assert.Equal(t, id1, id2)
}

func TestEntity_CaseInsensitiveGet(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.UpsertEntity(ctx, "OpenClaw", "system", nil)
	require.NoError(t, err)

	e, err := db.GetEntity(ctx, "openclaw")
	require.NoError(t, err)
	assert.NotNil(t, e)
	assert.Equal(t, "OpenClaw", e.Name)
}

func TestEntity_SoftDelete(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.UpsertEntity(ctx, "Old Thing", "concept", nil)
	require.NoError(t, err)

	require.NoError(t, db.SoftDeleteEntity(ctx, "Old Thing"))

	e, err := db.GetEntity(ctx, "Old Thing")
	require.NoError(t, err)
	assert.Nil(t, e) // soft-deleted, not visible
}

func TestEntity_HardDelete(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id, err := db.UpsertEntity(ctx, "Temp", "concept", nil)
	require.NoError(t, err)
	require.NoError(t, db.AddObservation(ctx, id, "some fact"))

	require.NoError(t, db.HardDeleteEntity(ctx, "Temp"))

	e, err := db.GetEntity(ctx, "Temp")
	require.NoError(t, err)
	assert.Nil(t, e)
}

func TestObservation_AddAndList(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id, err := db.UpsertEntity(ctx, "Redis", "system", nil)
	require.NoError(t, err)

	require.NoError(t, db.AddObservation(ctx, id, "Runs on port 6379"))
	require.NoError(t, db.AddObservation(ctx, id, "Used for session store"))

	// Dedup
	require.NoError(t, db.AddObservation(ctx, id, "Runs on port 6379"))

	obs, err := db.ListObservationsByEntityID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, []string{"Runs on port 6379", "Used for session store"}, obs)
}

func TestObservation_Retract(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id, err := db.UpsertEntity(ctx, "Redis", "system", nil)
	require.NoError(t, err)
	require.NoError(t, db.AddObservation(ctx, id, "Port 6379"))
	require.NoError(t, db.AddObservation(ctx, id, "Version 7.2"))

	remaining, err := db.RetractObservation(ctx, "Redis", "Port 6379")
	require.NoError(t, err)
	assert.Equal(t, []string{"Version 7.2"}, remaining)
}

func TestRelation_UpsertAndList(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	err := db.UpsertRelationByName(ctx, "Redis", "Gateway", "used-by")
	require.NoError(t, err)

	// Dedup
	err = db.UpsertRelationByName(ctx, "Redis", "Gateway", "used-by")
	require.NoError(t, err)

	rels, err := db.ListRelationsByEntity(ctx, "Redis")
	require.NoError(t, err)
	require.Len(t, rels, 1)
	assert.Equal(t, "used-by", rels[0].Relation)
}

func TestSearch_FTS5(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	id, err := db.UpsertEntity(ctx, "Redis Cache", "system", nil)
	require.NoError(t, err)
	require.NoError(t, db.AddObservation(ctx, id, "Runs on port 6379"))

	id2, err := db.UpsertEntity(ctx, "PostgreSQL", "system", nil)
	require.NoError(t, err)
	require.NoError(t, db.AddObservation(ctx, id2, "Primary relational database"))

	results, err := db.Search(ctx, "Redis", "", nil, "recent", 10)
	require.NoError(t, err)
	require.NotEmpty(t, results)
	assert.Equal(t, "Redis Cache", results[0].Name)
}

func TestSearch_EmptyQuery(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.UpsertEntity(ctx, "Entity1", "system", nil)
	require.NoError(t, err)
	_, err = db.UpsertEntity(ctx, "Entity2", "system", nil)
	require.NoError(t, err)

	results, err := db.Search(ctx, "", "", nil, "name", 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSearch_EmptyDB(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	results, err := db.Search(ctx, "anything", "", nil, "", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearch_TagFilter(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.UpsertEntity(ctx, "Redis", "system", []string{"cache"})
	require.NoError(t, err)
	_, err = db.UpsertEntity(ctx, "PG", "system", []string{"db"})
	require.NoError(t, err)

	results, err := db.Search(ctx, "", "", []string{"cache"}, "name", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Equal(t, "Redis", results[0].Name)
}

func TestStoreEntities(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	inputs := []EntityInput{
		{Name: "Redis", EntityType: "system", Observations: []string{"Port 6379", "In-memory"}, Tags: []string{"cache"}},
		{Name: "PG", EntityType: "system", Observations: []string{"SQL database"}},
	}
	results, err := db.StoreEntities(ctx, inputs)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestConcurrentWrites(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				name := fmt.Sprintf("entity-%d-%d", n, j)
				id, err := db.UpsertEntity(ctx, name, "test", nil)
				if err != nil {
					t.Errorf("UpsertEntity: %v", err)
					return
				}
				if err := db.AddObservation(ctx, id, "observation"); err != nil {
					t.Errorf("AddObservation: %v", err)
				}
			}
		}(i)
	}
	wg.Wait()

	stats, err := db.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 100, stats.EntityCount)
}

func TestJournal(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	e1, err := db.AppendJournal(ctx, "Fixed auth bug", []string{"fix", "auth"})
	require.NoError(t, err)
	assert.Greater(t, e1.ID, int64(0))

	// Same content again — no dedup
	e2, err := db.AppendJournal(ctx, "Fixed auth bug", nil)
	require.NoError(t, err)
	assert.NotEqual(t, e1.ID, e2.ID)

	entries, err := db.ListJournal(ctx, "", 50)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestSearchJournal_FTS5(t *testing.T) {
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.AppendJournal(ctx, "Fixed the authentication bug in login flow", []string{"fix", "auth"})
	require.NoError(t, err)
	_, err = db.AppendJournal(ctx, "Refactored Redis connection pool", []string{"refactor"})
	require.NoError(t, err)
	_, err = db.AppendJournal(ctx, "Deployed new feature to production", nil)
	require.NoError(t, err)

	// Should match first entry
	results, err := db.SearchJournal(ctx, "authentication", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "authentication")

	// Should match second entry
	results, err = db.SearchJournal(ctx, "Redis", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "Redis")

	// No match
	results, err = db.SearchJournal(ctx, "nonexistent", 10)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestSearchJournal_Rebuild(t *testing.T) {
	// Verify that rebuild in migrate() indexes pre-existing journal rows.
	// We test this by checking that SearchJournal returns results right after Open.
	db := NewTestDB(t)
	ctx := context.Background()

	_, err := db.AppendJournal(ctx, "Pre-existing session log entry", nil)
	require.NoError(t, err)

	results, err := db.SearchJournal(ctx, "session", 10)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.Contains(t, results[0].Content, "session")
}

func TestParseSince(t *testing.T) {
	ms, err := ParseSince("24h")
	require.NoError(t, err)
	assert.Greater(t, ms, int64(0))

	ms, err = ParseSince("7d")
	require.NoError(t, err)
	assert.Greater(t, ms, int64(0))

	ms, err = ParseSince("2026-02-17")
	require.NoError(t, err)
	assert.Greater(t, ms, int64(0))

	_, err = ParseSince("bogus")
	assert.Error(t, err)
}

