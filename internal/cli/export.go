package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/spf13/cobra"
)

var exportFormat string

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export memory to JSON or Markdown",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		// Export all entities by using the max limit (50) and paginating if needed.
		// For simplicity use ListEntities directly with no limit cap.
		entities, err := database.Search(ctx, "", "", nil, "name", 1000)
		if err != nil {
			return fmt.Errorf("export: %w", err)
		}

		switch exportFormat {
		case "json":
			return exportJSON(ctx, database, entities)
		case "markdown", "md":
			return exportMarkdown(entities)
		default:
			return fmt.Errorf("unknown format %q: use json or markdown", exportFormat)
		}
	},
}

// exportEntry is the mcp-knowledge-graph compatible JSONL format.
type exportEntry struct {
	Type         string   `json:"type"`
	Name         string   `json:"name,omitempty"`
	EntityType   string   `json:"entityType,omitempty"`
	Observations []string `json:"observations,omitempty"`
	Tags         []string `json:"tags,omitempty"`
	From         string   `json:"from,omitempty"`
	To           string   `json:"to,omitempty"`
	RelationType string   `json:"relationType,omitempty"`
}

func exportJSON(ctx context.Context, database *db.DB, entities []db.SearchResult) error {
	var entries []exportEntry
	for _, r := range entities {
		entry := exportEntry{
			Type:         "entity",
			Name:         r.Name,
			EntityType:   r.EntityType,
			Observations: r.Observations,
			Tags:         r.Tags,
		}
		if entry.Observations == nil {
			entry.Observations = []string{}
		}
		if entry.Tags == nil {
			entry.Tags = []string{}
		}
		entries = append(entries, entry)

		rels, err := database.ListRelationsByEntity(ctx, r.Name)
		if err != nil {
			continue
		}
		for _, rel := range rels {
			if rel.FromName == r.Name {
				entries = append(entries, exportEntry{
					Type:         "relation",
					From:         rel.FromName,
					To:           rel.ToName,
					RelationType: rel.Relation,
				})
			}
		}
	}

	enc := json.NewEncoder(stdoutWriter{})
	enc.SetIndent("", "  ")
	return enc.Encode(entries)
}

func exportMarkdown(entities []db.SearchResult) error {
	fmt.Println("# Memory Export")
	fmt.Println()
	for _, r := range entities {
		tags := ""
		if len(r.Tags) > 0 {
			tags = " `" + strings.Join(r.Tags, "` `") + "`"
		}
		fmt.Printf("## %s (%s)%s\n\n", r.Name, r.EntityType, tags)
		for _, obs := range r.Observations {
			fmt.Printf("- %s\n", obs)
		}
		fmt.Println()
	}
	return nil
}

func init() {
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "Output format: json|markdown")
	rootCmd.AddCommand(exportCmd)
}
