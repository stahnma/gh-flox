package commands

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newReadmesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "readmes [flags]",
		Short: "List repositories with 'flox install' in the README",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runReadmes(cmd)
		},
	}
	cmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	cmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")
	return cmd
}

func (a *App) runReadmes(cmd *cobra.Command) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	ctx := context.Background()
	showFull, _ := cmd.Flags().GetBool("full")
	verbose, _ := cmd.Flags().GetBool("verbose")
	w := cmd.OutOrStdout()

	repos, _, err := ghub.FindReadmeRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, ghub.SearchOptions{
		ShowFull:  showFull,
		Verbose:   verbose,
		NoCache:   a.Config.NoCache,
		DebugMode: a.Config.DebugMode,
	})
	if err != nil {
		return fmt.Errorf("finding repositories: %w", err)
	}

	// Merge with additional repos, deduplicating by full name
	repoMap := make(map[string]ghub.Repo)
	for _, r := range repos {
		repoMap[r.FullName()] = r
	}
	for _, name := range a.AdditionalRepos {
		parts := strings.Split(name, "/")
		if len(parts) == 2 {
			if _, exists := repoMap[name]; !exists {
				repoMap[name] = ghub.Repo{Owner: parts[0], Name: parts[1]}
			}
		}
	}

	// Fetch star counts if verbose for all unique repos (including additional)
	totalStars := 0
	if verbose {
		for name, repo := range repoMap {
			stars, err := ghub.GetStarCount(ctx, a.GHClient, a.Cache, repo.Owner, repo.Name, a.Config.NoCache, a.Config.DebugMode)
			if err == nil {
				repo.Stars = stars
				repoMap[name] = repo
				totalStars += stars
			}
		}
	}

	// Convert to sorted list
	repoList := make([]ghub.Repo, 0, len(repoMap))
	for _, r := range repoMap {
		repoList = append(repoList, r)
	}
	sort.Slice(repoList, func(i, j int) bool {
		return repoList[i].FullName() < repoList[j].FullName()
	})

	// Output summary
	if verbose {
		if a.Config.SlackMode {
			fmt.Fprintf(w, "Total repositories with 'flox install' in README found: *%d*, Total stars: *%d*\n", len(repoList), totalStars)
			fmt.Fprintln(w, "```")
		} else {
			fmt.Fprintf(w, "Total repositories with 'flox install' in README found: %d, Total stars: %d\n", len(repoList), totalStars)
		}
	} else {
		if a.Config.SlackMode {
			fmt.Fprintf(w, "Total repositories with 'flox install' in README found: *%d*\n", len(repoList))
		} else {
			fmt.Fprintf(w, "Total repositories with 'flox install' in README found: %d\n", len(repoList))
		}
	}

	// Output repo list if verbose
	if verbose {
		for _, repo := range repoList {
			fmt.Fprintf(w, "%s,%d\n", repo.FullName(), repo.Stars)
		}
		if a.Config.SlackMode {
			fmt.Fprintln(w, "```")
		}
	}

	return nil
}
