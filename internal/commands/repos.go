package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newReposCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos [flags]",
		Short: "List repositories with .flox/env/manifest.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runRepos(cmd)
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	cmd.Flags().BoolP("full", "f", false, "Show full list including those made by flox and employees")
	return cmd
}

func (a *App) runRepos(cmd *cobra.Command) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	ctx := context.Background()
	showFull, _ := cmd.Flags().GetBool("full")
	verbose, _ := cmd.Flags().GetBool("verbose")
	w := cmd.OutOrStdout()

	repos, err := ghub.FindManifestRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, ghub.SearchOptions{
		ShowFull:  showFull,
		NoCache:   a.Config.NoCache,
		DebugMode: a.Config.DebugMode,
	})
	if err != nil {
		return fmt.Errorf("finding repositories: %w", err)
	}

	if verbose {
		totalStars := 0
		for _, repo := range repos {
			totalStars += repo.Stars
		}
		fmt.Fprintf(w, "Total unique repositories found: %d, Total stars: %d\n", len(repos), totalStars)
		if a.Config.SlackMode {
			fmt.Fprintln(w, "```")
		}
		for _, repo := range repos {
			fmt.Fprintf(w, "%s,%d\n", repo.FullName(), repo.Stars)
		}
		if a.Config.SlackMode {
			fmt.Fprintln(w, "```")
		}
	} else {
		fmt.Fprintf(w, "Total unique repositories found: %d\n", len(repos))
	}

	return nil
}
