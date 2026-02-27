package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/stahnma/gh-flox/internal/cache"
	"github.com/stahnma/gh-flox/internal/config"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

// App holds shared application state.
type App struct {
	Config          config.Config
	Cache           *cache.Cache
	GHClient        ghub.Client
	AdditionalRepos []string
	MembershipCache *ghub.MembershipCache
	GitSHA          string
	GitDirty        string
}

// NewApp creates a new App from the given configuration.
func NewApp(cfg config.Config, additionalRepos []string, gitSHA, gitDirty string) (*App, error) {
	c, err := cache.LoadFromFile(cfg.CacheFile)
	if err != nil {
		return nil, fmt.Errorf("loading cache: %w", err)
	}

	return &App{
		Config:          cfg,
		Cache:           c,
		AdditionalRepos: additionalRepos,
		MembershipCache: ghub.NewMembershipCache(),
		GitSHA:          gitSHA,
		GitDirty:        gitDirty,
	}, nil
}

// ensureClient creates the GitHub client if it doesn't exist.
func (a *App) ensureClient() error {
	if a.GHClient != nil {
		return nil
	}
	if a.Config.GitHubToken == "" {
		return fmt.Errorf("GITHUB_TOKEN must be set")
	}
	a.GHClient = ghub.NewClient(a.Config.GitHubToken)
	return nil
}

// SaveCache saves the cache to disk if caching is enabled.
func (a *App) SaveCache() error {
	if !a.Config.NoCache {
		return a.Cache.SaveToFile(a.Config.CacheFile)
	}
	return nil
}

// NewRootCommand creates the root cobra command with all subcommands.
func (a *App) NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   os.Args[0],
		Short: "Tool for querying GitHub for flox things.",
	}
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.SilenceErrors = true
	rootCmd.SilenceUsage = true
	rootCmd.PersistentFlags().BoolVar(&a.Config.NoCache, "no-cache", false, "Disable caching")

	rootCmd.AddCommand(a.newReposCommand())
	rootCmd.AddCommand(a.newStarsCommand())
	rootCmd.AddCommand(a.newReadmesCommand())
	rootCmd.AddCommand(a.newFloxIndexCommand())
	rootCmd.AddCommand(a.newVersionCommand())
	rootCmd.AddCommand(a.newClearCacheCommand())
	rootCmd.AddCommand(a.newExportCommand())
	rootCmd.AddCommand(a.newDownloadManifestsCommand())

	return rootCmd
}
