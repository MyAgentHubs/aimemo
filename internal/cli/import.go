package cli

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/spf13/cobra"
)

var importFromJSONL string

var importCmd = &cobra.Command{
	Use:   "import [file]",
	Short: "Import from mcp-knowledge-graph JSONL format",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if importFromJSONL == "" && len(args) > 0 {
			importFromJSONL = args[0]
		}
		if importFromJSONL == "" {
			return fmt.Errorf("provide a file path: aimemo import <file> or --from-jsonl <file>")
		}

		data, err := os.ReadFile(importFromJSONL)
		if err != nil {
			return fmt.Errorf("open file: %w", err)
		}

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()

		type record struct {
			Type         string   `json:"type"`
			Name         string   `json:"name"`
			EntityType   string   `json:"entityType"`
			Observations []string `json:"observations"`
			Tags         []string `json:"tags"`
			From         string   `json:"from"`
			To           string   `json:"to"`
			RelationType string   `json:"relationType"`
		}

		// Detect format: if the file (ignoring leading whitespace) starts with '['
		// it is a JSON array (produced by `export --format json`); otherwise treat
		// it as JSONL (one JSON object per line), which preserves compatibility with
		// mcp-knowledge-graph.
		var records []record
		if trimmed := bytes.TrimSpace(data); len(trimmed) > 0 && trimmed[0] == '[' {
			// JSON array format
			if err := json.Unmarshal(trimmed, &records); err != nil {
				return fmt.Errorf("parse JSON array: %w", err)
			}
		} else {
			// JSONL format — parse line by line
			scanner := bufio.NewScanner(bytes.NewReader(data))
			scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				var rec record
				if err := json.Unmarshal([]byte(line), &rec); err != nil {
					// will be counted as skipped below; append a zero record so the
					// warning is emitted in the unified processing loop
					fmt.Fprintf(os.Stderr, "Warning: skipping malformed line: %v\n", err)
					continue
				}
				records = append(records, rec)
			}
			if err := scanner.Err(); err != nil {
				return fmt.Errorf("scan: %w", err)
			}
		}

		var entityCount, relationCount, skipCount int

		for _, rec := range records {
			switch rec.Type {
			case "entity":
				if rec.Name == "" {
					skipCount++
					continue
				}
				entityType := rec.EntityType
				if entityType == "" {
					entityType = "concept"
				}
				_, err := database.StoreEntities(ctx, []db.EntityInput{
					{Name: rec.Name, EntityType: entityType, Observations: rec.Observations, Tags: rec.Tags},
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to import entity %q: %v\n", rec.Name, err)
					skipCount++
				} else {
					entityCount++
				}

			case "relation":
				if rec.From == "" || rec.To == "" || rec.RelationType == "" {
					skipCount++
					continue
				}
				if err := database.UpsertRelationByName(ctx, rec.From, rec.To, rec.RelationType); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to import relation: %v\n", err)
					skipCount++
				} else {
					relationCount++
				}

			default:
				fmt.Fprintf(os.Stderr, "Warning: unknown record type %q — skipping\n", rec.Type)
				skipCount++
			}
		}

		fmt.Printf("Import complete:\n")
		fmt.Printf("  Entities:  %d\n", entityCount)
		fmt.Printf("  Relations: %d\n", relationCount)
		if skipCount > 0 {
			fmt.Printf("  Skipped:   %d (malformed)\n", skipCount)
		}
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&importFromJSONL, "from-jsonl", "", "Path to mcp-knowledge-graph JSONL file")
	rootCmd.AddCommand(importCmd)
}
