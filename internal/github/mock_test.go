package github

import (
	"context"
	"net/http"

	gh "github.com/google/go-github/v68/github"
)

// mockClient implements Client for testing.
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

// emptyResponse returns a *gh.Response that signals no more pages.
func emptyResponse() *gh.Response {
	return &gh.Response{
		Response: &http.Response{StatusCode: 200},
	}
}

// makeCodeResult builds a CodeResult for the given owner/repo.
func makeCodeResult(owner, name string) *gh.CodeResult {
	return &gh.CodeResult{
		Repository: &gh.Repository{
			Owner: &gh.User{Login: gh.Ptr(owner)},
			Name:  gh.Ptr(name),
		},
	}
}
