package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var getCmd = &cobra.Command{
	Use:   "get <entity-name>",
	Short: "Show details for a specific entity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		e, err := database.GetEntity(ctx, name)
		if err != nil {
			return fmt.Errorf("get entity: %w", err)
		}
		if e == nil {
			// Check whether the entity exists but is soft-deleted
			var count int
			_ = database.QueryRowContext(ctx,
				`SELECT COUNT(*) FROM entities WHERE lower(name) = lower(?) AND deleted_at IS NOT NULL`,
				name,
			).Scan(&count)
			if count > 0 {
				return fmt.Errorf("entity %q is soft-deleted; re-add it to restore, or run 'aimemo forget %s --permanent' to hard-delete", name, name)
			}
			return fmt.Errorf("entity %q not found", name)
		}

		tags := strings.Join(e.Tags, ", ")
		if tags == "" {
			tags = "(none)"
		}

		fmt.Printf("Name:         %s\n", e.Name)
		fmt.Printf("Type:         %s\n", e.EntityType)
		fmt.Printf("Tags:         %s\n", tags)
		fmt.Printf("Access count: %d\n", e.AccessCount)
		fmt.Printf("Observations (%d):\n", len(e.Observations))
		for _, obs := range e.Observations {
			fmt.Printf("  - %s\n", obs)
		}

		rels, err := database.ListRelationsByEntity(ctx, name)
		if err != nil {
			return err
		}
		if len(rels) > 0 {
			fmt.Printf("Relations (%d):\n", len(rels))
			for _, r := range rels {
				fmt.Printf("  %s -[%s]-> %s\n", r.FromName, r.Relation, r.ToName)
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(getCmd)
}
