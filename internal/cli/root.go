package cli

import (
	"fmt"
	"os"

	"github.com/MyAgentHubs/aimemo/internal/config"
	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/MyAgentHubs/aimemo/internal/locate"
	"github.com/spf13/cobra"
)

var (
	contextFlag string
	cfgFile     string
	cfg         config.Config
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "aimemo",
	Short: "Zero-dependency MCP memory server for AI agents",
	Long: `aimemo gives AI agents (Claude Code, Cursor, Windsurf) persistent,
searchable memory stored in a local SQLite database.

Run 'aimemo init' in a project directory to create project-local memory.
Run 'aimemo serve' to start the MCP server (usually done automatically by your AI client).`,
	SilenceUsage: true,
}

// Execute runs the root command.
func Execute(version string) {
	rootCmd.Version = version
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&contextFlag, "context", "", "Named memory context (e.g. 'work', 'personal')")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Config file (default: ~/.aimemo/config.toml)")
}

func initConfig() {
	path := cfgFile
	if path == "" {
		p, _ := locate.ConfigPath()
		path = p
	}
	var err error
	cfg, err = config.Load(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: config load error: %v\n", err)
		cfg = config.Default()
	}
}

// openDB opens the database for the current context.
func openDB() (*db.DB, string, error) {
	dbPath, err := locate.FindProjectDB(contextFlag)
	if err != nil {
		return nil, "", fmt.Errorf("find db: %w", err)
	}
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, "", fmt.Errorf("open db %s: %w", dbPath, err)
	}
	return database, dbPath, nil
}
