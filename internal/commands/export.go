package commands

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/spf13/cobra"
	"github.com/stahnma/gh-flox/internal/format"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [flags]",
		Short: "Export data in JSON format",
		RunE: func(cmd *cobra.Command, args []string) error {
			showFull, _ := cmd.Flags().GetBool("full")
			return a.ExportJSON(context.Background(), cmd.OutOrStdout(), showFull)
		},
	}
	cmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")
	return cmd
}

// ExportJSON runs the export logic, writing JSON to w.
func (a *App) ExportJSON(ctx context.Context, w io.Writer, showFull bool) error {
	if err := a.ensureClient(); err != nil {
		return err
	}

	var allRepos []ghub.RepoInfo
	date := time.Now().Format("2006-Jan-02")

	// Get repos with .flox/env/manifest.toml
	repos, _, err := ghub.FindManifestRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, showFull, true, a.Config.NoCache, a.Config.DebugMode)
	if err != nil {
		return fmt.Errorf("finding manifest repositories: %w", err)
	}
	for _, repo := range repos {
		allRepos = append(allRepos, ghub.RepoInfo{
			Date:       date,
			Repository: repo.FullName(),
			Type:       "dotflox",
			StarCount:  repo.Stars,
		})
	}

	// Get repos with 'flox install' in README
	readmeRepos, _, err := ghub.FindReadmeRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, showFull, true, a.Config.NoCache, a.Config.DebugMode)
	if err != nil {
		return fmt.Errorf("finding readme repositories: %w", err)
	}
	for _, repo := range readmeRepos {
		allRepos = append(allRepos, ghub.RepoInfo{
			Date:       date,
			Repository: repo.FullName(),
			Type:       "readme",
			StarCount:  repo.Stars,
		})
	}

	return format.WriteJSON(w, allRepos, a.Config.SlackMode)
}
