package github

import (
	"context"
	"errors"
	"testing"

	gh "github.com/google/go-github/v68/github"
	"github.com/stahnma/gh-flox/internal/cache"
)

func newSearchClient(results []*gh.CodeResult) *mockClient {
	return &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return &gh.CodeSearchResult{CodeResults: results}, emptyResponse(), nil
		},
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			stars := 10
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}
}

func TestFindManifestRepos_Basic(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("alice", "project1"),
		makeCodeResult("bob", "project2"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
	// Should be sorted alphabetically.
	if repos[0].FullName() != "alice/project1" {
		t.Errorf("first repo = %s, want alice/project1", repos[0].FullName())
	}
}

func TestFindManifestRepos_Dedup(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("alice", "project1"),
		makeCodeResult("alice", "project1"), // duplicate
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("got %d repos, want 1 (deduped)", len(repos))
	}
}

func TestFindManifestRepos_FilterExcludedOrgs(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("flox", "internal-tool"),
		makeCodeResult("flox-examples", "demo"),
		makeCodeResult("alice", "project1"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1 (excluded orgs filtered)", len(repos))
	}
	if repos[0].FullName() != "alice/project1" {
		t.Errorf("remaining repo = %s, want alice/project1", repos[0].FullName())
	}
}

func TestFindManifestRepos_FilterOrgMember(t *testing.T) {
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return &gh.CodeSearchResult{
				CodeResults: []*gh.CodeResult{
					makeCodeResult("employee", "repo1"),
					makeCodeResult("external", "repo2"),
				},
			}, emptyResponse(), nil
		},
		isOrgMemberFn: func(_ context.Context, org, user string) (bool, *gh.Response, error) {
			if user == "employee" {
				return true, emptyResponse(), nil
			}
			return false, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			stars := 5
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].FullName() != "external/repo2" {
		t.Errorf("repo = %s, want external/repo2", repos[0].FullName())
	}
}

func TestFindManifestRepos_ShowFull(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("flox", "internal"),
		makeCodeResult("alice", "project"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Errorf("showFull should return all repos, got %d", len(repos))
	}
}

func TestFindManifestRepos_FetchesStars(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("alice", "project1"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Fatalf("got %d repos, want 1", len(repos))
	}
	if repos[0].Stars != 10 {
		t.Errorf("stars = %d, want 10", repos[0].Stars)
	}
}

func TestFindManifestRepos_Caching(t *testing.T) {
	calls := 0
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			calls++
			return &gh.CodeSearchResult{
				CodeResults: []*gh.CodeResult{makeCodeResult("alice", "repo")},
			}, emptyResponse(), nil
		},
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			stars := 10
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	// First call.
	FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true})
	// Second call should use cache.
	FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true})

	if calls != 1 {
		t.Errorf("expected 1 API call, got %d (cache should prevent second)", calls)
	}
}

func TestFindManifestRepos_Error(t *testing.T) {
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return nil, nil, errors.New("search failed")
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	_, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err == nil {
		t.Error("expected error")
	}
}

// --- FindReadmeRepos ---

func TestFindReadmeRepos_Basic(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("charlie", "docs"),
		makeCodeResult("dave", "tutorial"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindReadmeRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("got %d repos, want 2", len(repos))
	}
}

func TestFindReadmeRepos_FilterExcludedOrgs(t *testing.T) {
	client := newSearchClient([]*gh.CodeResult{
		makeCodeResult("flox", "readme-repo"),
		makeCodeResult("external", "project"),
	})

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindReadmeRepos(context.Background(), client, c, mc, SearchOptions{NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 {
		t.Errorf("got %d repos, want 1", len(repos))
	}
}

func TestFindReadmeRepos_Caching(t *testing.T) {
	calls := 0
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			calls++
			return &gh.CodeSearchResult{
				CodeResults: []*gh.CodeResult{makeCodeResult("alice", "repo")},
			}, emptyResponse(), nil
		},
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			stars := 10
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	FindReadmeRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true})
	FindReadmeRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true})

	if calls != 1 {
		t.Errorf("expected 1 API call, got %d", calls)
	}
}

func TestFindReadmeRepos_Error(t *testing.T) {
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, _ *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			return nil, nil, errors.New("fail")
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	_, err := FindReadmeRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err == nil {
		t.Error("expected error")
	}
}

func TestFindManifestRepos_Pagination(t *testing.T) {
	page := 0
	client := &mockClient{
		searchCodeFn: func(_ context.Context, _ string, opts *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
			page++
			resp := emptyResponse()
			var results []*gh.CodeResult
			if page == 1 {
				results = []*gh.CodeResult{makeCodeResult("alice", "repo1")}
				resp.NextPage = 2
			} else {
				results = []*gh.CodeResult{makeCodeResult("bob", "repo2")}
				// NextPage = 0 (default) signals last page.
			}
			return &gh.CodeSearchResult{CodeResults: results}, resp, nil
		},
		isOrgMemberFn: func(_ context.Context, _, _ string) (bool, *gh.Response, error) {
			return false, emptyResponse(), nil
		},
		getRepositoryFn: func(_ context.Context, _, _ string) (*gh.Repository, *gh.Response, error) {
			stars := 10
			return &gh.Repository{StargazersCount: &stars}, emptyResponse(), nil
		},
	}

	c := cache.New()
	mc := NewMembershipCache()

	repos, err := FindManifestRepos(context.Background(), client, c, mc, SearchOptions{ShowFull: true, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Errorf("got %d repos, want 2 across 2 pages", len(repos))
	}
	if page != 2 {
		t.Errorf("expected 2 pages fetched, got %d", page)
	}
}
