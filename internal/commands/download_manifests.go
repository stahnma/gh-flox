package commands

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/spf13/cobra"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

func (a *App) newDownloadManifestsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "download-manifests",
		Short: "Download manifest.toml files from repositories with .flox directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			return a.runDownloadManifests(cmd)
		},
	}
}

func (a *App) runDownloadManifests(cmd *cobra.Command) error {
	if err := a.ensureClient(); err != nil {
		return err
	}
	ctx := context.Background()
	w := cmd.OutOrStdout()

	repos, _, err := ghub.FindManifestRepos(ctx, a.GHClient, a.Cache, a.MembershipCache, false, false, a.Config.NoCache, a.Config.DebugMode)
	if err != nil {
		return fmt.Errorf("finding repositories: %w", err)
	}

	if err := os.MkdirAll("manifests", 0755); err != nil {
		return fmt.Errorf("creating manifests directory: %w", err)
	}

	for _, repo := range repos {
		filePath := a.fetchManifestFile(ctx, repo.Owner, repo.Name)
		if filePath != "" {
			fmt.Fprintf(w, "Downloaded manifest.toml for %s/%s to %s\n", repo.Owner, repo.Name, filePath)
		}
	}
	return nil
}

func (a *App) fetchManifestFile(ctx context.Context, owner, repo string) string {
	manifestPath, err := ghub.FetchManifestPath(ctx, a.GHClient, owner, repo)
	if err != nil {
		log.Printf("Error searching for manifest.toml in %s/%s: %v", owner, repo, err)
		return ""
	}
	if manifestPath == "" {
		log.Printf("No manifest.toml found in %s/%s", owner, repo)
		return ""
	}

	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, manifestPath)

	httpClient := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequestWithContext(ctx, "GET", rawURL, nil)
	if err != nil {
		log.Printf("Error creating request for %s: %v", rawURL, err)
		return ""
	}
	if a.Config.GitHubToken != "" {
		req.Header.Set("Authorization", "token "+a.Config.GitHubToken)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error fetching raw content from %s: %v", rawURL, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch file from %s, status code: %d", rawURL, resp.StatusCode)
		return ""
	}

	localFilePath := fmt.Sprintf("manifests/%s_%s_manifest.toml", owner, repo)
	file, err := os.Create(localFilePath)
	if err != nil {
		log.Printf("Error creating local file for %s/%s: %v", owner, repo, err)
		return ""
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		log.Printf("Error saving manifest.toml for %s/%s: %v", owner, repo, err)
		return ""
	}

	return localFilePath
}
