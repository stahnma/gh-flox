package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stahnma/gh-flox/internal/cache"
	"github.com/stahnma/gh-flox/internal/config"
	ghub "github.com/stahnma/gh-flox/internal/github"
)

// mockClient implements ghub.Client for testing commands.
type mockClient struct {
	searchCodeFn    func(ctx context.Context, query string, opts *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error)
	getRepositoryFn func(ctx context.Context, owner, repo string) (*gh.Repository, *gh.Response, error)
	isOrgMemberFn   func(ctx context.Context, org, user string) (bool, *gh.Response, error)
}

func (m *mockClient) SearchCode(ctx context.Context, query string, opts *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
	return m.searchCodeFn(ctx, query, opts)
}

func (m *mockClient) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, *gh.Response, error) {
	return m.getRepositoryFn(ctx, owner, repo)
}

func (m *mockClient) IsOrgMember(ctx context.Context, org, user string) (bool, *gh.Response, error) {
	return m.isOrgMemberFn(ctx, org, user)
}

func emptyResponse() *gh.Response {
	return &gh.Response{Response: &http.Response{StatusCode: 200}}
}

func makeCodeResult(owner, name string) *gh.CodeResult {
	return &gh.CodeResult{
		Repository: &gh.Repository{
			Owner: &gh.User{Login: gh.Ptr(owner)},
			Name:  gh.Ptr(name),
		},
	}
}

func newTestApp(client ghub.Client) *App {
	return &App{
		Config: config.Config{
			NoCache: true,
		},
		Cache:           cache.New(),
		GHClient:        client,
		MembershipCache: ghub.NewMembershipCache(),
		GitSHA:          "abc1234",
		GitDirty:        "",
	}
}

func defaultMockClient() *mockClient {
	stars := 42
	return &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return &gh.CodeSearchResult{
				CodeResults: []*gh.CodeResult{
					makeCodeResult("alice", "project1"),
					makeCodeResult("bob", "project2"),
				},
			}, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, emptyResponse(), nil
		},
	}
}

// --- Version ---

func TestVersionCommand(t *testing.T) {
	app := newTestApp(nil)
	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "abc1234") {
		t.Errorf("expected SHA in output, got:\n%s", out)
	}
	if strings.Contains(out, "Dirty") {
		t.Error("expected no dirty flag when GitDirty is empty")
	}
}

func TestVersionCommand_Dirty(t *testing.T) {
	app := newTestApp(nil)
	app.GitDirty = "true"
	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Dirty: true") {
		t.Errorf("expected dirty flag, got:\n%s", out)
	}
}

// --- ClearCache ---

func TestClearCacheCommand(t *testing.T) {
	app := newTestApp(nil)
	app.Config.CacheFile = t.TempDir() + "/cache.gob"
	app.Config.NoCache = false
	app.Cache.Set("key", "val")

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"clearcache"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "Cache cleared") {
		t.Errorf("expected 'Cache cleared', got:\n%s", buf.String())
	}
	if _, found := app.Cache.Get("key"); found {
		t.Error("cache should be flushed")
	}
}

// --- Stars ---

func TestStarsCommand(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"stars"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "42") {
		t.Errorf("expected star count, got:\n%s", out)
	}
	if !strings.Contains(out, "flox/flox") {
		t.Errorf("expected repo name, got:\n%s", out)
	}
}

func TestStarsCommand_SlackMode(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)
	app.Config.SlackMode = true

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"stars"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, ":star2:") {
		t.Errorf("expected slack emoji, got:\n%s", out)
	}
}

// --- Repos ---

func TestReposCommand(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"repos"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total unique repositories found: 2") {
		t.Errorf("expected repo count, got:\n%s", out)
	}
}

func TestReposCommand_Verbose(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"repos", "--verbose", "--full"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "alice/project1") {
		t.Errorf("expected repo listing, got:\n%s", out)
	}
	if !strings.Contains(out, "Total stars") {
		t.Errorf("expected star total in verbose, got:\n%s", out)
	}
}

// --- Readmes ---

func TestReadmesCommand(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"readmes"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total repositories with 'flox install' in README found: 2") {
		t.Errorf("expected readme count, got:\n%s", out)
	}
}

func TestReadmesCommand_AdditionalRepos(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)
	app.AdditionalRepos = []string{"extra/repo1"}

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"readmes", "--full"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	// 2 from search + 1 additional = 3
	if !strings.Contains(out, "3") {
		t.Errorf("expected 3 repos (2 + 1 additional), got:\n%s", out)
	}
}

// --- FloxIndex ---

func TestFloxIndexCommand(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"floxindex", "--full"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	out := buf.String()
	if !strings.Contains(out, "Total floxindex (sum of stars)") {
		t.Errorf("expected floxindex output, got:\n%s", out)
	}
}

// --- Export ---

func TestExportCommand(t *testing.T) {
	client := defaultMockClient()
	app := newTestApp(client)

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"export", "--full"})

	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}

	var result []ghub.RepoInfo
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput:\n%s", err, buf.String())
	}

	if len(result) == 0 {
		t.Fatal("expected non-empty export")
	}

	// Check we have both types.
	types := make(map[string]bool)
	for _, r := range result {
		types[r.Type] = true
		if r.Date == "" {
			t.Error("expected non-empty date")
		}
		if r.Repository == "" {
			t.Error("expected non-empty repository")
		}
	}
	if !types["dotflox"] {
		t.Error("expected dotflox type in export")
	}
	if !types["readme"] {
		t.Error("expected readme type in export")
	}
}

// --- No client error ---

func TestReposCommand_NoClient(t *testing.T) {
	app := &App{
		Config:          config.Config{NoCache: true},
		Cache:           cache.New(),
		MembershipCache: ghub.NewMembershipCache(),
	}

	cmd := app.NewRootCommand()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"repos"})

	err := cmd.Execute()
	if err == nil {
		t.Error("expected error when GITHUB_TOKEN is empty and no client")
	}
}
