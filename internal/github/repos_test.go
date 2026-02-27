package github

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stahnma/gh-flox/internal/cache"
)

func TestExcludedOrgs(t *testing.T) {
	for _, org := range []string{"flox", "flox-examples"} {
		if !excludedOrgs[org] {
			t.Errorf("expected %q to be in excludedOrgs", org)
		}
	}
	if excludedOrgs["random-org"] {
		t.Error("unexpected org in excludedOrgs")
	}
}

// --- MembershipCache ---

func TestMembershipCache_Miss(t *testing.T) {
	mc := NewMembershipCache()
	client := &mockClient{
		isOrgMemberFn: func(_ context.Context, org, user string) (bool, *gh.Response, error) {
			if org == "flox" && user == "alice" {
				return true, emptyResponse(), nil
			}
			return false, emptyResponse(), nil
		},
	}

	got, err := mc.Check(context.Background(), client, "alice", "flox")
	if err != nil {
		t.Fatal(err)
	}
	if !got {
		t.Error("expected alice to be a member of flox")
	}
}

func TestMembershipCache_Hit(t *testing.T) {
	mc := NewMembershipCache()
	calls := 0
	client := &mockClient{
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			calls++
			return true, emptyResponse(), nil
		},
	}

	// First call populates cache.
	mc.Check(context.Background(), client, "bob", "flox")
	// Second call should use cache.
	mc.Check(context.Background(), client, "bob", "flox")

	if calls != 1 {
		t.Errorf("expected 1 API call, got %d", calls)
	}
}

func TestMembershipCache_Error(t *testing.T) {
	mc := NewMembershipCache()
	client := &mockClient{
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, nil, errors.New("network error")
		},
	}

	_, err := mc.Check(context.Background(), client, "alice", "flox")
	if err == nil {
		t.Error("expected error from Check")
	}
}

// --- GetStarCount ---

func TestGetStarCount_CacheMiss(t *testing.T) {
	c := cache.New()
	stars := 42
	client := &mockClient{
		getRepositoryFn: func(_ context.Context, owner, repo string) (*gh.Repository, *gh.Response, error) {
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	got, err := GetStarCount(context.Background(), client, c, "owner", "repo", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != 42 {
		t.Errorf("got %d stars, want 42", got)
	}

	// Verify it was cached.
	val, found := c.Get("starCount:owner/repo")
	if !found {
		t.Fatal("star count not cached")
	}
	if val.(int) != 42 {
		t.Errorf("cached value = %d, want 42", val.(int))
	}
}

func TestGetStarCount_CacheHit(t *testing.T) {
	c := cache.New()
	c.Set("starCount:owner/repo", 99)
	calls := 0
	client := &mockClient{
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			calls++
			return nil, nil, errors.New("should not be called")
		},
	}

	got, err := GetStarCount(context.Background(), client, c, "owner", "repo", false, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != 99 {
		t.Errorf("got %d stars, want 99", got)
	}
	if calls != 0 {
		t.Errorf("API should not have been called, but was called %d times", calls)
	}
}

func TestGetStarCount_NoCache(t *testing.T) {
	c := cache.New()
	c.Set("starCount:owner/repo", 99)
	stars := 50
	client := &mockClient{
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	got, err := GetStarCount(context.Background(), client, c, "owner", "repo", true, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != 50 {
		t.Errorf("got %d, want 50 (should bypass cache)", got)
	}
}

func TestGetStarCount_Error(t *testing.T) {
	c := cache.New()
	client := &mockClient{
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			return nil, nil, errors.New("api error")
		},
	}

	_, err := GetStarCount(context.Background(), client, c, "owner", "repo", true, false)
	if err == nil {
		t.Error("expected error from GetStarCount")
	}
}

// --- FetchManifestPath ---

func TestFetchManifestPath_Found(t *testing.T) {
	path := ".flox/env/manifest.toml"
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return &gh.CodeSearchResult{
				CodeResults: []*gh.CodeResult{
					{Path: &path},
				},
			}, emptyResponse(), nil
		},
	}

	got, err := FetchManifestPath(context.Background(), client, "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestFetchManifestPath_NotFound(t *testing.T) {
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return &gh.CodeSearchResult{CodeResults: []*gh.CodeResult{}}, emptyResponse(), nil
		},
	}

	got, err := FetchManifestPath(context.Background(), client, "owner", "repo")
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("expected empty path, got %q", got)
	}
}

func TestFetchManifestPath_Error(t *testing.T) {
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return nil, nil, errors.New("search error")
		},
	}

	_, err := FetchManifestPath(context.Background(), client, "owner", "repo")
	if err == nil {
		t.Error("expected error")
	}
}
