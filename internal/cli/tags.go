package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags in use",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		rows, err := database.QueryContext(ctx, `
			SELECT DISTINCT value FROM entities, json_each(entities.tags)
			WHERE deleted_at IS NULL ORDER BY value
		`)
		if err != nil {
			return fmt.Errorf("query tags: %w", err)
		}
		defer rows.Close()

		count := 0
		for rows.Next() {
			var tag string
			if err := rows.Scan(&tag); err != nil {
				return err
			}
			fmt.Println(tag)
			count++
		}
		if count == 0 {
			fmt.Println("No tags in use.")
		}
		return rows.Err()
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}
