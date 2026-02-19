package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show memory statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, dbPath, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		stats, err := database.GetStats(ctx)
		if err != nil {
			return fmt.Errorf("stats: %w", err)
		}

		fmt.Printf("Storage:      %s\n", dbPath)
		fmt.Printf("Entities:     %d\n", stats.EntityCount)
		fmt.Printf("Observations: %d\n", stats.ObservationCount)
		fmt.Printf("Relations:    %d\n", stats.RelationCount)
		fmt.Printf("Journal:      %d entries\n", stats.JournalCount)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
