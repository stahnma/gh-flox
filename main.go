package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/go-github/v43/github"
	"github.com/patrickmn/go-cache"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/oauth2"
)

// Embed the JSON file
//
//go:embed additional_repos.json
var additionalReposJSON []byte

var (
	GitSHA   string
	GitDirty string
)

var (
	resultCache     *cache.Cache
	cacheFile       = "/tmp/cache.gob" // Use /tmp for Lambda
	debugMode       = false
	noCache         bool
	additionalRepos []string
)

var rootCmd = &cobra.Command{
	Use:   os.Args[0],
	Short: "Tool for querying GitHub for flox things.",
}

var reposCmd = &cobra.Command{
	Use:   "repos [flags]",
	Short: "List repositories with .flox/env/manifest.toml",
	Run: func(cmd *cobra.Command, args []string) {
		runReposCommand(cmd, args)
	},
}

var starsCmd = &cobra.Command{
	Use:   "stars",
	Short: "Show star count for the flox/flox repository",
	Run: func(cmd *cobra.Command, args []string) {
		runStarsCommand(cmd, args)
	},
}

var readmesCmd = &cobra.Command{
	Use:   "readmes [flags]",
	Short: "List repositories with 'flox install' in the README",
	Run: func(cmd *cobra.Command, args []string) {
		runReadmesCommand(cmd, args)
	},
}

var floxIndexCmd = &cobra.Command{
	Use:   "floxindex [flags]",
	Short: "Calculate the total star count for all flox-related repositories",
	Run: func(cmd *cobra.Command, args []string) {
		runFloxIndexCommand(cmd, args)
	},
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show the current version",
	Run: func(cmd *cobra.Command, args []string) {
		runVersionCommand(cmd, args)
	},
}

var clearCacheCmd = &cobra.Command{
	Use:   "clearcache",
	Short: "Clear the cache",
	Run: func(cmd *cobra.Command, args []string) {
		runClearCacheCommand(cmd, args)
	},
}

var exportCmd = &cobra.Command{
	Use:   "export [flags]",
	Short: "Export data in JSON format",
	Run: func(cmd *cobra.Command, args []string) {
		runExportJSONCommand(cmd, args)
	},
}

// RepoInfo struct to hold repository information for export
type RepoInfo struct {
	Date       string `json:"date"`
	Repository string `json:"repository"`
	Type       string `json:"type"`
	StarCount  int    `json:"starcount"`
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

	// Adding flags to floxindex command
	floxIndexCmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")

	// Adding flags to export command
	exportCmd.Flags().BoolP("full", "f", false, "Include repositories from excluded organizations")

	// Adding global --no-cache flag
	rootCmd.PersistentFlags().BoolVar(&noCache, "no-cache", false, "Disable caching")

	// Register commands
	rootCmd.AddCommand(reposCmd)
	rootCmd.AddCommand(starsCmd)
	rootCmd.AddCommand(readmesCmd)
	rootCmd.AddCommand(floxIndexCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(clearCacheCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(downloadManifestsCmd)

	// Initialize cache
	var err error
	resultCache, err = loadCacheFromFile(cacheFile)
	if err != nil {
		log.Fatalf("Error loading cache: %v", err)
	}

	// Set debug mode based on environment variable
	debugEnv := os.Getenv("DEBUG")
	debugMode = !(debugEnv == "" || debugEnv == "0" || strings.ToLower(debugEnv) == "false")

	// Parse the embedded JSON
	err = json.Unmarshal(additionalReposJSON, &additionalRepos)
	if err != nil {
		log.Fatalf("Error parsing additional repositories JSON: %v", err)
	}
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
	if os.Getenv("LAMBDA_TASK_ROOT") != "" {
		lambda.Start(lambdaHandler)
	} else {
		if err := rootCmd.Execute(); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Save cache to file
		if !noCache {
			if err := saveCacheToFile(resultCache, cacheFile); err != nil {
				log.Fatalf("Error saving cache: %v", err)
			}
		}
	}
}

func lambdaHandler(ctx context.Context, event interface{}) (string, error) {
	args := []string{"export"}
	rootCmd.SetArgs(args)

	output := new(bytes.Buffer)
	rootCmd.SetOutput(output)

	// Capture the standard output
	stdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := rootCmd.Execute()
	if err != nil {
		return "", err
	}

	w.Close()
	os.Stdout = stdout
	outputStr, _ := ioutil.ReadAll(r)

	// Ensure the buffer has content
	if len(outputStr) == 0 {
		log.Printf("export command produced no output")
		return "", fmt.Errorf("export command produced no output")
	}

	// Upload the output to S3
	s3Bucket := os.Getenv("S3_BUCKET_NAME")
	s3ObjectKey := os.Getenv("S3_OBJECT_KEY")

	if s3Bucket == "" || s3ObjectKey == "" {
		return "", fmt.Errorf("S3_BUCKET_NAME and S3_OBJECT_KEY environment variables must be set")
	}

	// Append the current date to the S3_OBJECT_KEY
	date := time.Now().Format("2006-Jan-02")
	s3ObjectKey = fmt.Sprintf(s3ObjectKey, date)

	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(os.Getenv("AWS_REGION")))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %v", err)
	}

	svc := s3.NewFromConfig(cfg)

	_, err = svc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3ObjectKey),
		Body:   bytes.NewReader(outputStr),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %v", err)
	}

	return "Lambda executed successfully and output uploaded to S3", nil
}

func runReposCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	showFull, _ := cmd.Flags().GetBool("full")
	verbose, _ := cmd.Flags().GetBool("verbose")

	repos, stars, err := findAllFloxManifestRepos(ctx, client, showFull, verbose)
	if err != nil {
		log.Fatalf("Error finding repositories: %v", err)
	}

	if verbose {
		fmt.Printf("Total unique repositories found: %d, Total stars: %d\n", len(repos), stars)
	} else {
		fmt.Printf("Total unique repositories found: %d\n", len(repos))
	}

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
}

func runStarsCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	showStars(ctx, client, "flox", "flox")
}

func runReadmesCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	showFull, _ := cmd.Flags().GetBool("full")
	verbose, _ := cmd.Flags().GetBool("verbose")

	repos, _, err := findAllFloxReadmeRepos(ctx, client, showFull, verbose)
	if err != nil {
		log.Fatalf("Error finding repositories: %v", err)
	}

	// Use a map to track unique repositories and their star counts
	repoSet := make(map[string]int)
	for _, repo := range repos {
		parts := strings.Split(repo, ",")
		repoName := parts[0]
		repoSet[repoName] = 0 // Initialize with 0 stars
	}

	// Include additional repositories from JSON in all modes
	for _, repoName := range additionalRepos {
		repoSet[repoName] = 0
	}

	totalStars := 0
	// If verbose, calculate stars for all unique repositories
	if verbose {
		for repoName := range repoSet {
			parts := strings.Split(repoName, "/")
			if len(parts) == 2 {
				starsCount, err := getStarCount(ctx, client, parts[0], parts[1])
				if err == nil {
					repoSet[repoName] = starsCount
					totalStars += starsCount
				}
			}
		}
	}

	// Convert map back to a sorted slice
	var repoList []string
	for repoName := range repoSet {
		if verbose {
			repoList = append(repoList, fmt.Sprintf("%s,%d", repoName, repoSet[repoName]))
		} else {
			repoList = append(repoList, repoName)
		}
	}
	sort.Strings(repoList)

	// Check if SLACK_MODE is enabled
	slackMode := viper.GetBool("SLACK_MODE")

	// Output the results
	if verbose {
		if slackMode {
			fmt.Printf("Total repositories with 'flox install' in README found: *%d*, Total stars: *%d*\n", len(repoList), totalStars)
			fmt.Println("```")
		} else {
			fmt.Printf("Total repositories with 'flox install' in README found: %d, Total stars: %d\n", len(repoList), totalStars)
		}
	} else {
		if slackMode {
			fmt.Printf("Total repositories with 'flox install' in README found: *%d*\n", len(repoList))
		} else {
			fmt.Printf("Total repositories with 'flox install' in README found: %d\n", len(repoList))
		}
	}

	if verbose {
		for _, repo := range repoList {
			fmt.Println(repo)
		}
		if slackMode {
			fmt.Println("```")
		}
	}
}

func runFloxIndexCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	showFull, _ := cmd.Flags().GetBool("full")

	totalStars, err := calculateFloxIndex(ctx, client, showFull)
	if err != nil {
		log.Fatalf("Error calculating floxindex: %v", err)
	}

	fmt.Printf("Total floxindex (sum of stars): %d\n", totalStars)
}

func runVersionCommand(cmd *cobra.Command, args []string) {
	fmt.Printf("Git SHA: %s\n", GitSHA)
	if GitDirty != "" {
		fmt.Printf("Git Dirty: true\n")
	}
}

func runClearCacheCommand(cmd *cobra.Command, args []string) {
	resultCache.Flush()
	if err := saveCacheToFile(resultCache, cacheFile); err != nil {
		log.Fatalf("Error saving cache: %v", err)
	}
	fmt.Println("Cache cleared.")
}

func runExportJSONCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	showFull, _ := cmd.Flags().GetBool("full")

	var allRepos []RepoInfo
	date := time.Now().Format("2006-Jan-02")

	// Get data from repos command
	repos, _, err := findAllFloxManifestRepos(ctx, client, showFull, true)
	if err != nil {
		log.Fatalf("Error finding repositories: %v", err)
	}

	for _, repo := range repos {
		parts := strings.Split(repo, ",")
		if len(parts) == 2 {
			starCount := 0
			fmt.Sscanf(parts[1], "%d", &starCount)
			allRepos = append(allRepos, RepoInfo{
				Date:       date,
				Repository: parts[0],
				Type:       "dotflox",
				StarCount:  starCount,
			})
		}
	}

	// Get data from readmes command
	readmeRepos, _, err := findAllFloxReadmeRepos(ctx, client, showFull, true)
	if err != nil {
		log.Fatalf("Error finding repositories: %v", err)
	}

	for _, repo := range readmeRepos {
		parts := strings.Split(repo, ",")
		if len(parts) == 2 {
			starCount := 0
			fmt.Sscanf(parts[1], "%d", &starCount)
			allRepos = append(allRepos, RepoInfo{
				Date:       date,
				Repository: parts[0],
				Type:       "readme",
				StarCount:  starCount,
			})
		}
	}

	jsonOutput, err := json.MarshalIndent(allRepos, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	if viper.GetBool("SLACK_MODE") {
		fmt.Println("```")
	}
	fmt.Println(string(jsonOutput))
	if viper.GetBool("SLACK_MODE") {
		fmt.Println("```")
	}
}

func saveCacheToFile(cache *cache.Cache, filename string) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(cache.Items()); err != nil {
		return err
	}
	return ioutil.WriteFile(filename, buf.Bytes(), 0644)
}

func loadCacheFromFile(filename string) (*cache.Cache, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return cache.New(4*time.Hour, 6*time.Hour), nil
		}
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	items := map[string]cache.Item{}
	if err := dec.Decode(&items); err != nil {
		return nil, err
	}
	return cache.NewFrom(4*time.Hour, 6*time.Hour, items), nil
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

func findAllFloxManifestRepos(ctx context.Context, client *github.Client, showFull bool, verbose bool) ([]string, int, error) {
	cacheKey := fmt.Sprintf("floxManifestRepos:%t:%t", showFull, verbose)
	if !noCache {
		if cachedRepos, found := resultCache.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			return cachedRepos.([]string), sumStars(cachedRepos.([]string)), nil
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	seen := make(map[string]bool)
	var repositories []string
	var totalStars int
	excludedOrgs := map[string]bool{"flox": true, "flox-examples": true}
	membershipCache := make(map[string]bool)

	query := ".flox/env/manifest.toml in:path"
	options := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		repos, response, err := client.Search.Code(ctx, query, options)
		if err != nil {
			return nil, 0, err
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

					if verbose {
						stars, err := getStarCount(ctx, client, parts[3], parts[4])
						if err == nil {
							repoName = fmt.Sprintf("%s,%d", repoName, stars)
							totalStars += stars
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
	if !noCache {
		resultCache.Set(cacheKey, repositories, cache.DefaultExpiration)
	}
	return repositories, totalStars, nil
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

func findAllFloxReadmeRepos(ctx context.Context, client *github.Client, showFull bool, verbose bool) ([]string, int, error) {
	cacheKey := fmt.Sprintf("floxReadmeRepos:%t:%t", showFull, verbose)
	if !noCache {
		if cachedRepos, found := resultCache.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			return cachedRepos.([]string), sumStars(cachedRepos.([]string)), nil
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	seen := make(map[string]bool)
	var repositories []string
	var totalStars int
	excludedOrgs := map[string]bool{"flox": true, "flox-examples": true}
	membershipCache := make(map[string]bool)

	query := "\"flox install\" in:file filename:README"
	options := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 100}}

	for {
		results, response, err := client.Search.Code(ctx, query, options)
		if err != nil {
			return nil, 0, err
		}

		for _, item := range results.CodeResults {
			repoName := fmt.Sprintf("%s/%s", *item.Repository.Owner.Login, *item.Repository.Name)
			if _, exists := seen[repoName]; !exists {
				if !showFull {
					isMember, err := isOrgMember(ctx, client, *item.Repository.Owner.Login, "flox", membershipCache)
					if err != nil {
						fmt.Printf("Error checking membership: %v\n", err)
						continue
					}
					if isMember {
						continue
					}
					if excludedOrgs[*item.Repository.Owner.Login] {
						continue
					}
				}
				if verbose {
					stars, err := getStarCount(ctx, client, *item.Repository.Owner.Login, *item.Repository.Name)
					if err == nil {
						repoName = fmt.Sprintf("%s,%d", repoName, stars)
						totalStars += stars
					}
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
	if !noCache {
		resultCache.Set(cacheKey, repositories, cache.DefaultExpiration)
	}
	return repositories, totalStars, nil
}

func calculateFloxIndex(ctx context.Context, client *github.Client, showFull bool) (int, error) {
	totalStars := 0

	// Calculate stars for repos with .flox/env/manifest.toml
	repos, _, err := findAllFloxManifestRepos(ctx, client, showFull, false)
	if err != nil {
		return 0, err
	}

	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			stars, err := getStarCount(ctx, client, parts[0], parts[1])
			if err != nil {
				return 0, err
			}
			totalStars += stars
		}
	}

	// Calculate stars for repos with 'flox install' in the README
	readmeRepos, _, err := findAllFloxReadmeRepos(ctx, client, showFull, false)
	if err != nil {
		return 0, err
	}

	for _, repo := range readmeRepos {
		parts := strings.Split(repo, "/")
		if len(parts) == 2 {
			stars, err := getStarCount(ctx, client, parts[0], parts[1])
			if err != nil {
				return 0, err
			}
			totalStars += stars
		}
	}

	// Include stars from additional repositories
	for _, repoName := range additionalRepos {
		parts := strings.Split(repoName, "/")
		if len(parts) == 2 {
			stars, err := getStarCount(ctx, client, parts[0], parts[1])
			if err != nil {
				return 0, err
			}
			totalStars += stars
		}
	}

	return totalStars, nil
}

func getStarCount(ctx context.Context, client *github.Client, owner, repo string) (int, error) {
	cacheKey := fmt.Sprintf("starCount:%s/%s", owner, repo)
	if !noCache {
		if cachedStars, found := resultCache.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			return cachedStars.(int), nil
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	repository, _, err := client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return 0, err
	}
	starCount := *repository.StargazersCount
	if !noCache {
		resultCache.Set(cacheKey, starCount, cache.DefaultExpiration)
	}
	return starCount, nil
}

func sumStars(repos []string) int {
	totalStars := 0
	for _, repo := range repos {
		parts := strings.Split(repo, ",")
		if len(parts) == 2 {
			var stars int
			fmt.Sscanf(parts[1], "%d", &stars)
			totalStars += stars
		}
	}
	return totalStars
}

var downloadManifestsCmd = &cobra.Command{
	Use:   "download-manifests",
	Short: "Download manifest.toml files from repositories with .flox directory",
	Run: func(cmd *cobra.Command, args []string) {
		runDownloadManifestsCommand(cmd, args)
	},
}

func runDownloadManifestsCommand(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	client := setupGitHubClient(ctx)
	verbose, _ := cmd.Flags().GetBool("verbose")

	// Get repositories
	repos, _, err := findAllFloxManifestRepos(ctx, client, false, verbose)
	if err != nil {
		log.Fatalf("Error finding repositories: %v", err)
	}

	// Create the "manifests" directory if it doesn't exist
	err = os.MkdirAll("manifests", 0755)
	if err != nil {
		log.Fatalf("Error creating manifests directory: %v", err)
	}

	// Iterate through each repository
	for _, repo := range repos {
		parts := strings.Split(repo, "/")
		if len(parts) < 2 {
			log.Printf("Skipping invalid repository name: %s", repo)
			continue
		}
		owner, repoName := parts[0], parts[1]
		filePath := fetchManifestFile(ctx, client, owner, repoName)
		if filePath != "" {
			fmt.Printf("Downloaded manifest.toml for %s/%s to %s\n", owner, repoName, filePath)
		}
	}
}

func fetchManifestFile(ctx context.Context, client *github.Client, owner, repo string) string {
	// Search for manifest.toml in the repository
	query := fmt.Sprintf("manifest.toml repo:%s/%s path:.flox/env", owner, repo)
	options := &github.SearchOptions{ListOptions: github.ListOptions{PerPage: 1}}
	results, _, err := client.Search.Code(ctx, query, options)
	if err != nil {
		log.Printf("Error searching for manifest.toml in %s/%s: %v", owner, repo, err)
		return ""
	}

	if len(results.CodeResults) == 0 {
		log.Printf("No manifest.toml found in %s/%s", owner, repo)
		return ""
	}

	// Construct the raw file URL
	filePath := *results.CodeResults[0].Path
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/main/%s", owner, repo, filePath)

	// Make the HTTP GET request to fetch the file content
	resp, err := http.Get(rawURL)
	if err != nil {
		log.Printf("Error fetching raw content from %s: %v", rawURL, err)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch file from %s, status code: %d", rawURL, resp.StatusCode)
		return ""
	}

	// Save the file to the "manifests" directory
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
