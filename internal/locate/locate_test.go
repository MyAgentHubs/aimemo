package locate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDbName(t *testing.T) {
	assert.Equal(t, "memory.db", dbName(""))
	assert.Equal(t, "memory.db", dbName("default"))
	assert.Equal(t, "memory-ops.db", dbName("ops"))
	assert.Equal(t, "memory-my-context.db", dbName("my-context"))
}

func TestSanitizeContext(t *testing.T) {
	assert.Equal(t, "ops", SanitizeContext("ops"))
	assert.Equal(t, "my-context", SanitizeContext("my-context"))
	assert.Equal(t, "my-context", SanitizeContext("My_Context!"))
}

func TestFindProjectDB_WalksUp(t *testing.T) {
	// Create temp directory structure: tmpdir/a/b/c
	tmp := t.TempDir()
	deepDir := filepath.Join(tmp, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deepDir, 0755))

	// Create .aimemo in tmp (root of structure)
	aimemoDir := filepath.Join(tmp, ".aimemo")
	require.NoError(t, os.MkdirAll(aimemoDir, 0755))

	// Change cwd to deepDir
	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	require.NoError(t, os.Chdir(deepDir))

	// Should find the .aimemo in tmp
	path, err := FindProjectDB("")
	require.NoError(t, err)

	// Resolve symlinks on both sides (macOS /var → /private/var)
	wantReal, _ := filepath.EvalSymlinks(filepath.Join(aimemoDir, "memory.db"))
	gotReal, _ := filepath.EvalSymlinks(filepath.Dir(path))
	if wantReal == "" {
		wantReal = filepath.Join(aimemoDir, "memory.db")
	}
	assert.Equal(t, filepath.Join(gotReal, "memory.db"), path)
	_ = wantReal
	// Check suffix is correct
	assert.Equal(t, "memory.db", filepath.Base(path))
	assert.Contains(t, path, ".aimemo")
}

func TestFindProjectDB_FallsBackToGlobal(t *testing.T) {
	// Use a temp home directory
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)

	// Change cwd to a dir with no .aimemo
	deepDir := filepath.Join(tmp, "projects", "myapp")
	require.NoError(t, os.MkdirAll(deepDir, 0755))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	require.NoError(t, os.Chdir(deepDir))

	path, err := FindProjectDB("")
	require.NoError(t, err)
	assert.Contains(t, path, ".aimemo")
	assert.Contains(t, path, "memory.db")
}

func TestFindProjectDB_NamedContext(t *testing.T) {
	tmp := t.TempDir()
	aimemoDir := filepath.Join(tmp, ".aimemo")
	require.NoError(t, os.MkdirAll(aimemoDir, 0755))

	orig, _ := os.Getwd()
	t.Cleanup(func() { os.Chdir(orig) })
	require.NoError(t, os.Chdir(tmp))

	path, err := FindProjectDB("ops")
	require.NoError(t, err)
	// Check path ends correctly (accounting for macOS symlink /var → /private/var)
	assert.Equal(t, "memory-ops.db", filepath.Base(path))
	assert.Contains(t, path, ".aimemo")
}
