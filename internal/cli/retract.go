package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var retractCmd = &cobra.Command{
	Use:   "retract <entity-name> <observation>",
	Short: "Remove a specific observation from an entity",
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
		remaining, err := database.RetractObservation(ctx, name, content)
		if err != nil {
			return fmt.Errorf("retract: %w", err)
		}

		fmt.Printf("Retracted from %q:\n  - %s\n", name, content)
		if len(remaining) > 0 {
			fmt.Printf("Remaining observations (%d):\n", len(remaining))
			for _, obs := range remaining {
				fmt.Printf("  • %s\n", obs)
			}
		} else {
			fmt.Println("No observations remaining.")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(retractCmd)
}
