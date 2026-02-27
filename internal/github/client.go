package github

import (
	"context"

	gh "github.com/google/go-github/v68/github"
	"golang.org/x/oauth2"
)

// Client defines the GitHub API methods used by this application.
type Client interface {
	SearchCode(ctx context.Context, query string, opts *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error)
	GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, *gh.Response, error)
	IsOrgMember(ctx context.Context, org, user string) (bool, *gh.Response, error)
}

// realClient wraps the go-github client to implement Client.
type realClient struct {
	inner *gh.Client
}

// NewClient creates a new GitHub API client authenticated with the given token.
func NewClient(token string) Client {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	httpClient := oauth2.NewClient(context.Background(), ts)
	return &realClient{inner: gh.NewClient(httpClient)}
}

func (c *realClient) SearchCode(ctx context.Context, query string, opts *gh.SearchOptions) (*gh.CodeSearchResult, *gh.Response, error) {
	return c.inner.Search.Code(ctx, query, opts)
}

func (c *realClient) GetRepository(ctx context.Context, owner, repo string) (*gh.Repository, *gh.Response, error) {
	return c.inner.Repositories.Get(ctx, owner, repo)
}

func (c *realClient) IsOrgMember(ctx context.Context, org, user string) (bool, *gh.Response, error) {
	return c.inner.Organizations.IsMember(ctx, org, user)
}
