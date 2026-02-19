package locate

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var contextSanitize = regexp.MustCompile(`[^a-z0-9-]`)

// SanitizeContext sanitizes a context name to [a-z0-9-] for safe use in filenames.
func SanitizeContext(context string) string {
	lower := strings.ToLower(context)
	sanitized := contextSanitize.ReplaceAllString(lower, "-")
	return strings.Trim(sanitized, "-")
}

// dbName returns the database filename for a given context.
func dbName(context string) string {
	if context == "" || context == "default" {
		return "memory.db"
	}
	return fmt.Sprintf("memory-%s.db", SanitizeContext(context))
}

// globalDBPath returns the path to the global database.
func globalDBPath(context string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	dir := filepath.Join(home, ".aimemo")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("cannot create global .aimemo dir: %w", err)
	}
	return filepath.Join(dir, dbName(context)), nil
}

// FindProjectDB walks up from cwd looking for a .aimemo/ directory.
// Falls back to ~/.aimemo/ if not found.
func FindProjectDB(context string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return globalDBPath(context)
	}

	home, _ := os.UserHomeDir()
	dir := cwd

	for {
		aimemoDir := filepath.Join(dir, ".aimemo")
		if info, err := os.Stat(aimemoDir); err == nil && info.IsDir() {
			return filepath.Join(aimemoDir, dbName(context)), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir || dir == home {
			break
		}
		dir = parent
	}

	return globalDBPath(context)
}

// ConfigPath returns the path to the user's config file.
func ConfigPath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "aimemo", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".aimemo", "config.toml"), nil
}
