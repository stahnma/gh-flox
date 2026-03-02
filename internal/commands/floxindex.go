package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newFloxIndexCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "floxindex [flags]",
		Short: "Calculate the total star count for all flox-related repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runFloxIndex(cmd)
		},
	}
	cmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")
	return cmd
}

func (a *App) runFloxIndex(cmd *cobra.Command) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	ctx := context.Background()
	showFull, _ := cmd.Flags().GetBool("full")
	w := cmd.OutOrStdout()

	totalStars, err := a.calculateFloxIndex(ctx, showFull)
	if err != nil {
		return fmt.Errorf("calculating floxindex: %w", err)
	}

	fmt.Fprintf(w, "Total floxindex (sum of stars): %d\n", totalStars)
	return nil
}

func (a *App) calculateFloxIndex(ctx context.Context, showFull bool) (int, error) {
	totalStars := 0

	// Stars for repos with .flox/env/manifest.toml
	opts := ghub.SearchOptions{
		ShowFull:  showFull,
		NoCache:   a.Config.NoCache,
		DebugMode: a.Config.DebugMode,
	}
	repos, err := ghub.FindManifestRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, opts)
	if err != nil {
		return 0, err
	}
	for _, repo := range repos {
		stars, err := ghub.GetStarCount(ctx, a.GHClient, a.Cache, repo.Owner, repo.Name, a.Config.NoCache, a.Config.DebugMode)
		if err != nil {
			return 0, err
		}
		totalStars += stars
	}

	// Stars for repos with 'flox install' in README
	readmeRepos, err := ghub.FindReadmeRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, opts)
	if err != nil {
		return 0, err
	}
	for _, repo := range readmeRepos {
		stars, err := ghub.GetStarCount(ctx, a.GHClient, a.Cache, repo.Owner, repo.Name, a.Config.NoCache, a.Config.DebugMode)
		if err != nil {
			return 0, err
		}
		totalStars += stars
	}

	// Stars from additional repositories
	for _, repoName := range a.AdditionalRepos {
		parts := strings.Split(repoName, "/")
		if len(parts) == 2 {
			stars, err := ghub.GetStarCount(ctx, a.GHClient, a.Cache, parts[0], parts[1], a.Config.NoCache, a.Config.DebugMode)
			if err != nil {
				return 0, err
			}
			totalStars += stars
		}
	}

	return totalStars, nil
}
