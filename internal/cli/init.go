package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize project-local memory in the current directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := os.MkdirAll(".aimemo", 0755); err != nil {
			return fmt.Errorf("create .aimemo: %w", err)
		}

		gitignore := `# aimemo memory database (binary, not diff-friendly)
memory.db
memory-*.db
# Export files are gitignore-exempt so you can commit them
!memory-export.json
!memory-export.md
`
		if err := os.WriteFile(".aimemo/.gitignore", []byte(gitignore), 0644); err != nil {
			return fmt.Errorf("write .gitignore: %w", err)
		}

		// Initialize the database to validate it works
		database, dbPath, err := openDB()
		if err != nil {
			return err
		}
		database.Close()

		fmt.Printf("Initialized aimemo memory in .aimemo/\n")
		fmt.Printf("Database: %s\n\n", dbPath)
		fmt.Printf("To register with Claude Code:\n")
		fmt.Printf("  claude mcp add-json \"aimemo-memory\" '{\"command\":\"aimemo\",\"args\":[\"serve\"]}'\n")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
