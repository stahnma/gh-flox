package github

import (
	"context"
	"fmt"
	"log"

	gh "github.com/google/go-github/v68/github"
	"github.com/stahnma/gh-flox/internal/cache"
)

// excludedOrgs is the set of organizations excluded from non-full results.
var excludedOrgs = map[string]bool{"flox": true, "flox-examples": true}

// MembershipCache caches GitHub org membership lookups.
type MembershipCache struct {
	entries map[string]bool
}

// NewMembershipCache creates a new MembershipCache.
func NewMembershipCache() *MembershipCache {
	return &MembershipCache{entries: make(map[string]bool)}
}

// Check returns whether the user is a member of the org, using the cache.
func (mc *MembershipCache) Check(ctx context.Context, client Client, username, org string) (bool, error) {
	key := org + "/" + username
	if member, ok := mc.entries[key]; ok {
		return member, nil
	}
	member, _, err := client.IsOrgMember(ctx, org, username)
	if err != nil {
		fmt.Println("Error during membership check:", err)
		return false, err
	}
	mc.entries[key] = member
	return member, nil
}

// GetStarCount retrieves the star count for a repository, using the cache.
func GetStarCount(ctx context.Context, client Client, c *cache.Cache, owner, repo string, noCache, debugMode bool) (int, error) {
	cacheKey := fmt.Sprintf("starCount:%s/%s", owner, repo)
	if !noCache {
		if val, found := c.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			if stars, ok := val.(int); ok {
				return stars, nil
			}
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	repository, _, err := client.GetRepository(ctx, owner, repo)
	if err != nil {
		return 0, err
	}
	starCount := repository.GetStargazersCount()
	if !noCache {
		c.Set(cacheKey, starCount)
	}
	return starCount, nil
}

// FetchManifestPath searches for the manifest.toml path in a repository.
func FetchManifestPath(ctx context.Context, client Client, owner, repo string) (string, error) {
	query := fmt.Sprintf("manifest.toml repo:%s/%s path:.flox/env", owner, repo)
	options := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 1}}
	results, _, err := client.SearchCode(ctx, query, options)
	if err != nil {
		return "", err
	}
	if len(results.CodeResults) == 0 {
		return "", nil
	}
	return results.CodeResults[0].GetPath(), nil
}
