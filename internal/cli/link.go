package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var linkCmd = &cobra.Command{
	Use:   "link <from> <relation> <to>",
	Short: "Create a typed relation between two entities",
	Args:  cobra.ExactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		from := args[0]
		relation := args[1]
		to := args[2]

		database, _, err := openDB()
		if err != nil {
			return err
		}
		defer database.Close()

		ctx := context.Background()
		if err := database.UpsertRelationByName(ctx, from, to, relation); err != nil {
			return fmt.Errorf("link: %w", err)
		}
		fmt.Printf("%s -[%s]-> %s\n", from, relation, to)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd)
}
