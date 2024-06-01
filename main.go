package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

var (
	GitSHA   string
	GitDirty string
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "Tool for querying GitHub for flox things.",
}

var reposCmd = &cobra.Command{
	Use:   "repos [flags]",
	Short: "List repositories with .flox/env/manifest.toml",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := setupGitHubClient(ctx)
		showFull, _ := cmd.Flags().GetBool("full")
		verbose, _ := cmd.Flags().GetBool("verbose")

		repos, err := findAllFloxManifestRepos(ctx, client, showFull, verbose)
		if err != nil {
			log.Fatalf("Error finding repositories: %v", err)
		}

		fmt.Printf("Total unique repositories found: %d\n", len(repos))

		if verbose && viper.GetBool("SLACK_MODE") {
			fmt.Println("```")
			for _, repo := range repos {
				fmt.Println(repo)
			}
			fmt.Println("```")
		} else if verbose {
			for _, repo := range repos {
				fmt.Println(repo)
			}
		}
	},
}

var starsCmd = &cobra.Command{
	Use:   "stars",
	Short: "Show star count for the flox/flox repository",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := setupGitHubClient(ctx)
		showStars(ctx, client, "flox", "flox")
	},
}

var readmesCmd = &cobra.Command{
	Use:   "readmes [flags]",
	Short: "List repositories with 'flox install' in the README",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := context.Background()
		client := setupGitHubClient(ctx)
		showFull, _ := cmd.Flags().GetBool("full")
		verbose, _ := cmd.Flags().GetBool("verbose")

		repos, err := findAllFloxReadmeRepos(ctx, client, showFull)
		if err != nil {
			log.Fatalf("Error finding repositories: %v", err)
		}

		fmt.Printf("Total repositories with 'flox install' in README found: %d\n", len(repos))

		if verbose {
			for _, repo := range repos {
				fmt.Println(repo)
			}
		}
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Git SHA: %s\n", GitSHA)
		if GitDirty != "" {
			fmt.Printf("Git Dirty: true\n")
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.CompletionOptions.DisableDefaultCmd = true

	// Adding flags to repos command
	reposCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	reposCmd.Flags().BoolP("full", "f", false, "Show full list including those made by flox and employees")

	// Adding flags to readmes command
	readmesCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	readmesCmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")

	// Register commands
	rootCmd.AddCommand(reposCmd)
	rootCmd.AddCommand(starsCmd)
	rootCmd.AddCommand(readmesCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	viper.AutomaticEnv()
	viper.SetDefault("SLACK_MODE", false)
	slackMode := viper.GetString("SLACK_MODE")
	viper.Set("SLACK_MODE", slackMode != "false" && slackMode != "FALSE" && slackMode != "0")
}

func setupGitHubClient(ctx context.Context) *github.Client {
	token := viper.GetString("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN must be set")
	}
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, ts)
	return github.NewClient(httpClient)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func isOrgMember(ctx context.Context, client *github.Client, username, org string, cache map[string]bool) (bool, error) {
	if member, ok := cache[username]; ok {
		return member, nil
	}
	member, _, err := client.Organizations.IsMember(ctx, org, username)
	if err != nil {
		fmt.Println("Error during membership check:", err)
		return false, err
	}
	cache[username] = member
	return member, nil
}

func findAllFloxManifestRepos(ctx context.Context, client *github.Client, showFull bool, verbose bool) ([]string, error) {
	seen := make(map[string]bool)
	var repositories []string
	excludedOrgs := map[string]bool{"flox": true, "flox-examples": true}
	membershipCache := make(map[string]bool)

	query := ".flox/env/manifest.toml in:path"
	options := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, response, err := client.Search.Code(ctx, query, options)
		if err != nil {
			return nil, err
		}

		for _, item := range repos.CodeResults {
			parts := strings.Split(*item.HTMLURL, "/")
			if len(parts) > 2 {
				repoName := fmt.Sprintf("%s/%s", parts[3], parts[4])
				if _, exists := seen[repoName]; !exists {
					seen[repoName] = true

					if !showFull {
						isMember, e := isOrgMember(ctx, client, parts[3], "flox", membershipCache)
						if e != nil {
							fmt.Printf("Error checking membership: %v\n", e)
						}
						if isMember {
							continue
						}
						if excludedOrgs[parts[3]] {
							continue
						}
					}
					repositories = append(repositories, repoName)
				}
			}
		}

		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}

	sort.Strings(repositories)
	return repositories, nil
}

func showStars(ctx context.Context, client *github.Client, owner, repo string) {
	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		log.Fatalf("Error retrieving repository: %v", err)
	}
	if viper.GetBool("SLACK_MODE") {
		fmt.Printf("The repository :star2: `%s/%s` has %d stars :star2:.\n", owner, repo, *repository.StargazersCount)
	} else {
		fmt.Printf("The repository %s/%s has %d stars \n", owner, repo, *repository.StargazersCount)
	}
}

func findAllFloxReadmeRepos(ctx context.Context, client *github.Client, showFull bool) ([]string, error) {
	seen := make(map[string]bool)
	var repositories []string
	excludedOrgs := map[string]bool{"flox": true, "flox-examples": true}

	query := "\"flox install\" in:file filename:README"
	options := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		results, response, err := client.Search.Code(ctx, query, options)
		if err != nil {
			return nil, err
		}

		for _, item := range results.CodeResults {
			repoName := fmt.Sprintf("%s/%s", *item.Repository.Owner.Login, *item.Repository.Name)
			if _, exists := seen[repoName]; !exists {
				if !showFull && excludedOrgs[*item.Repository.Owner.Login] {
					continue
				}
				seen[repoName] = true
				repositories = append(repositories, repoName)
			}
		}

		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}

	sort.Strings(repositories)
	return repositories, nil
}
