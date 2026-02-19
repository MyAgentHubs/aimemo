package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	listType  string
	listTags  []string
	listLimit int
	listSort  string
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all entities in memory",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		results, err := database.Search(ctx, "", listType, listTags, listSort, listLimit)
		if err != nil {
			return fmt.Errorf("list: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No entities in memory. Use 'aimemo add' or let your AI agent store them.")
			return nil
		}
		printSearchResults(results)
		fmt.Printf("\nTotal: %d entities\n", len(results))
		return nil
	},
}

func init() {
	listCmd.Flags().StringVar(&listType, "type", "", "Filter by entity type")
	listCmd.Flags().StringArrayVar(&listTags, "tag", nil, "Filter by tag (AND); can be repeated")
	listCmd.Flags().IntVar(&listLimit, "limit", 50, "Max results")
	listCmd.Flags().StringVar(&listSort, "sort", "recent", "Sort: recent|accessed|name")
	rootCmd.AddCommand(listCmd)
}
