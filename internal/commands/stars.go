package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newStarsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stars",
		Short: "Show star count for the flox/flox repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runStars(cmd)
		},
	}
}

func (a *App) runStars(cmd *cobra.Command) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	ctx := context.Background()
	w := cmd.OutOrStdout()

	stars, err := ghub.GetStarCount(ctx, a.GHClient, a.Cache, "flox", "flox", a.Config.NoCache, a.Config.DebugMode)
	if err != nil {
		return fmt.Errorf("retrieving star count: %w", err)
	}

	if a.Config.SlackMode {
		fmt.Fprintf(w, "The repository :star2: `flox/flox` has %d stars :star2:.\n", stars)
	} else {
		fmt.Fprintf(w, "The repository flox/flox has %d stars \n", stars)
	}
	return nil
}
