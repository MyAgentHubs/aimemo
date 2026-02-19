package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/MyAgentHubs/aimemo/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server (stdio transport)",
	Long:  `Start the aimemo MCP server. Usually auto-spawned by your AI client.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		database, dbPath, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		slog.Info("aimemo MCP server starting", "db", dbPath)
		fmt.Fprintf(os.Stderr, "aimemo MCP server ready (db: %s)\n", dbPath)

		server := mcp.NewServer(database, dbPath)
		return server.ServeStdio()
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
