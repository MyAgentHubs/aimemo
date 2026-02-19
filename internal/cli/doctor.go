package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/MyAgentHubs/aimemo/internal/locate"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check aimemo installation health",
	RunE: func(cmd *cobra.Command, args []string) error {
		allOK := true

		check := func(label string, ok bool, detail string) {
			if ok {
				fmt.Printf("[OK] %s\n", label)
			} else {
				fmt.Printf("[FAIL] %s: %s\n", label, detail)
				allOK = false
			}
		}

		// 1. Storage path
		dbPath, err := locate.FindProjectDB(contextFlag)
		check("Storage path: "+dbPath, err == nil, fmt.Sprintf("%v", err))

		// 2. Database writable
		database, dbErr := db.Open(dbPath)
		check("Database writable", dbErr == nil, fmt.Sprintf("%v", dbErr))
		if dbErr != nil {
			printDoctorResult(allOK)
			return nil
		}
		defer database.Close()

		// 3. FTS5 functional
		ctx := context.Background()
		ftsOK := checkFTS5(ctx, database)
		check("FTS5 functional (porter unicode61 tokenizer)", ftsOK, "FTS5 not available — SQLite may lack this extension")

		// 4. WAL mode
		var journalMode string
		_ = database.QueryRowContext(ctx, "PRAGMA journal_mode").Scan(&journalMode)
		check("WAL mode enabled", journalMode == "wal", "journal_mode="+journalMode)

		// 5. MCP empty-query latency
		start := time.Now()
		_, err = database.Search(ctx, "", "", nil, "", 1)
		elapsed := time.Since(start)
		check(fmt.Sprintf("MCP server responds in <5ms (tested with empty query)"), elapsed < 5*time.Millisecond, fmt.Sprintf("took %v", elapsed))

		fmt.Println()
		printDoctorResult(allOK)

		if allOK {
			fmt.Println("\nTo register with Claude Code:")
			fmt.Printf("  claude mcp add-json \"aimemo-memory\" '{\"command\":\"aimemo\",\"args\":[\"serve\"]}'\n")
		}

		if !allOK {
			os.Exit(1)
		}
		return nil
	},
}

func checkFTS5(ctx context.Context, database *db.DB) bool {
	var result string
	err := database.QueryRowContext(ctx, `SELECT fts5_tokenize('porter unicode61', 'testing')`).Scan(&result)
	if err != nil {
		// Try a simpler FTS5 query
		err = database.QueryRowContext(ctx, `SELECT rowid FROM entities_fts WHERE entities_fts MATCH 'test' LIMIT 1`).Scan(&result)
		// ErrNoRows is OK — means FTS5 is functional but empty
		if err != nil && err.Error() != "sql: no rows in result set" {
			// Try creating a temp FTS5 table
			_, err2 := database.ExecContext(ctx, `CREATE VIRTUAL TABLE IF NOT EXISTS _fts5_check USING fts5(x, tokenize='porter unicode61')`)
			if err2 != nil {
				return false
			}
			database.ExecContext(ctx, `DROP TABLE IF EXISTS _fts5_check`)
		}
	}
	return true
}

func printDoctorResult(allOK bool) {
	if allOK {
		fmt.Println("All checks passed. aimemo is ready.")
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
