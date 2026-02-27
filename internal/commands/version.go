package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show the current version",
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Git SHA: %s\n", a.GitSHA)
			if a.GitDirty != "" {
				fmt.Fprintf(w, "Git Dirty: true\n")
			}
			return nil
		},
	}
}
