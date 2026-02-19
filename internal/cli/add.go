package cli

import (
	"context"
	"fmt"

	"github.com/MyAgentHubs/aimemo/internal/db"
	"github.com/spf13/cobra"
)

var addTags []string

var addCmd = &cobra.Command{
	Use:   "add <name> <type> [observations...]",
	Short: "Add an entity with observations",
	Long: `Add an entity with one or more observations.

Example:
  aimemo add "Redis" system "Runs on port 6379" "Used for session store" --tag cache --tag infra`,
	Args: cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		entityType := args[1]
		observations := args[2:]

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		results, err := database.StoreEntities(ctx, []db.EntityInput{
			{Name: name, EntityType: entityType, Observations: observations, Tags: addTags},
		})
		if err != nil {
			return fmt.Errorf("add entity: %w", err)
		}
		if len(results) > 0 {
			fmt.Printf("Stored entity: %s (%s)\n", results[0].Name, results[0].EntityType)
			for _, obs := range observations {
				fmt.Printf("  + %s\n", obs)
			}
		}
		return nil
	},
}

func init() {
	addCmd.Flags().StringArrayVar(&addTags, "tag", nil, "Tag (can be repeated)")
	rootCmd.AddCommand(addCmd)
}
