package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var observeCmd = &cobra.Command{
	Use:   "observe <entity-name> <observation>",
	Short: "Add an observation to an existing entity",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		content := args[1]

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
			return fmt.Errorf("entity %q not found — use 'aimemo add' to create it", name)
		}

		if err := database.AddObservation(ctx, e.ID, content); err != nil {
			return fmt.Errorf("add observation: %w", err)
		}
		fmt.Printf("Observation added to %q:\n  + %s\n", name, content)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(observeCmd)
}
