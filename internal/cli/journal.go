package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var journalSince string
var journalLimit int

// appendCmd appends a journal entry.
var appendCmd = &cobra.Command{
	Use:   "append <message>",
	Short: "Append a timestamped journal entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content := args[0]

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		entry, err := database.AppendJournal(ctx, content, nil)
		if err != nil {
			return fmt.Errorf("append journal: %w", err)
		}
		t := time.UnixMilli(entry.CreatedAt).Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] %s\n", t, content)
		return nil
	},
}

// journalCmd reads journal entries.
var journalCmd = &cobra.Command{
	Use:   "journal",
	Short: "Read journal entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		entries, err := database.ListJournal(ctx, journalSince, journalLimit)
		if err != nil {
			return fmt.Errorf("journal: %w", err)
		}

		if len(entries) == 0 {
			fmt.Println("No journal entries found.")
			return nil
		}

		// Reverse to show oldest first
		for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
			entries[i], entries[j] = entries[j], entries[i]
		}

		for _, e := range entries {
			t := time.UnixMilli(e.CreatedAt).Format("2006-01-02 15:04:05")
			fmt.Printf("[%s] %s\n", t, e.Content)
		}
		return nil
	},
}

func init() {
	journalCmd.Flags().StringVar(&journalSince, "since", "24h", "Time window: 2h|24h|7d|ISO date")
	journalCmd.Flags().IntVar(&journalLimit, "limit", 50, "Max entries")
	rootCmd.AddCommand(appendCmd)
	rootCmd.AddCommand(journalCmd)
}
