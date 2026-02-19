package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/spf13/cobra"
)

var (
	searchType   string
	searchTags   []string
	searchLimit  int
	searchSort   string
	outputJSON   bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search memory by full-text query",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := ""
		if len(args) > 0 {
			query = args[0]
		}

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		results, err := database.Search(ctx, query, searchType, searchTags, searchSort, searchLimit)
		if err != nil {
			return fmt.Errorf("search: %w", err)
		}

		var journalResults []db.JournalEntry
		if query != "" {
			journalResults, err = database.SearchJournal(ctx, query, searchLimit)
			if err != nil {
				return fmt.Errorf("journal search: %w", err)
			}
		}

		if outputJSON {
			return printJSON(map[string]any{
				"entities": results,
				"journal":  journalResults,
			})
		}
		if len(results) == 0 && len(journalResults) == 0 {
			fmt.Println("No results found.")
			return nil
		}
		printSearchResults(results)
		printJournalResults(journalResults)
		return nil
	},
}

func printSearchResults(results []db.SearchResult) {
	if len(results) == 0 {
		return
	}
	for _, r := range results {
		printEntity(&r.Entity)
	}
}

func printJournalResults(entries []db.JournalEntry) {
	if len(entries) == 0 {
		return
	}
	fmt.Println("── journal ──")
	for _, e := range entries {
		fmt.Printf("  %s\n", e.Content)
	}
}

func printEntity(e *db.Entity) {
	tags := ""
	if len(e.Tags) > 0 {
		tags = " [" + strings.Join(e.Tags, ", ") + "]"
	}
	fmt.Printf("• %s (%s)%s\n", e.Name, e.EntityType, tags)
	for _, obs := range e.Observations {
		fmt.Printf("  - %s\n", obs)
	}
}

func printJSON(v any) error {
	enc := json.NewEncoder(outputWriter())
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func outputWriter() interface{ Write([]byte) (int, error) } {
	return stdoutWriter{}
}

type stdoutWriter struct{}

func (stdoutWriter) Write(p []byte) (int, error) {
	fmt.Print(string(p))
	return len(p), nil
}

func init() {
	searchCmd.Flags().StringVar(&searchType, "type", "", "Filter by entity type")
	searchCmd.Flags().StringArrayVar(&searchTags, "tag", nil, "Filter by tag (AND); can be repeated")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 10, "Max results")
	searchCmd.Flags().StringVar(&searchSort, "sort", "recent", "Sort: recent|accessed|name")
	searchCmd.Flags().BoolVar(&outputJSON, "json", false, "Output as JSON")
	rootCmd.AddCommand(searchCmd)
}
