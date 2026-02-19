package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var forgetPermanent bool

var forgetCmd = &cobra.Command{
	Use:   "forget <entity-name>",
	Short: "Soft-delete (or permanently delete) an entity",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		if forgetPermanent {
			if err := database.HardDeleteEntity(ctx, name); err != nil {
				return fmt.Errorf("hard delete: %w", err)
			}
			fmt.Printf("Permanently deleted entity: %s\n", name)
		} else {
			if err := database.SoftDeleteEntity(ctx, name); err != nil {
				return fmt.Errorf("soft delete: %w", err)
			}
			fmt.Printf("Soft-deleted entity: %s (recoverable)\n", name)
		}
		return nil
	},
}

func init() {
	forgetCmd.Flags().BoolVar(&forgetPermanent, "permanent", false, "Hard delete (irreversible)")
	rootCmd.AddCommand(forgetCmd)
}
