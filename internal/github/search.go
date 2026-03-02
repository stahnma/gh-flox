package github

import (
	"context"
	"fmt"
	"log"
	"sort"

	gh "github.com/google/go-github/v68/github"
	"github.com/stahnma/gh-flox/internal/cache"
)

// FindManifestRepos searches for repositories containing .flox/env/manifest.toml.
func FindManifestRepos(ctx context.Context, client Client, c *cache.Cache, mc *MembershipCache, opts SearchOptions) ([]Repo, error) {
	return findRepos(ctx, client, c, mc, ".flox/env/manifest.toml in:path", "floxManifestRepos", opts)
}

// FindReadmeRepos searches for repositories containing "flox install" in their README.
func FindReadmeRepos(ctx context.Context, client Client, c *cache.Cache, mc *MembershipCache, opts SearchOptions) ([]Repo, error) {
	return findRepos(ctx, client, c, mc, "\"flox install\" in:file filename:README", "floxReadmeRepos", opts)
}

func findRepos(ctx context.Context, client Client, c *cache.Cache, mc *MembershipCache, query, cacheKeyPrefix string, opts SearchOptions) ([]Repo, error) {
	cacheKey := fmt.Sprintf("%s:v2:%t", cacheKeyPrefix, opts.ShowFull)
	if !opts.NoCache {
		if val, found := c.Get(cacheKey); found {
			if opts.DebugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			if repos, ok := val.([]Repo); ok {
				return repos, nil
			}
		}
		if opts.DebugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	seen := make(map[string]bool)
	var repositories []Repo

	options := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}

	for {
		results, response, err := client.SearchCode(ctx, query, options)
		if err != nil {
			return nil, err
		}

		for _, item := range results.CodeResults {
			owner := item.Repository.GetOwner().GetLogin()
			name := item.Repository.GetName()
			fullName := owner + "/" + name

			if seen[fullName] {
				continue
			}
			seen[fullName] = true

			if !opts.ShowFull {
				isMember, err := mc.Check(ctx, client, owner, "flox")
				if err != nil {
					log.Printf("Error checking membership: %v", err)
					continue
				}
				if isMember {
					continue
				}
				if excludedOrgs[owner] {
					continue
				}
			}

			repo := Repo{Owner: owner, Name: name}
			stars, err := GetStarCount(ctx, client, c, owner, name, opts.NoCache, opts.DebugMode)
			if err == nil {
				repo.Stars = stars
			}
			repositories = append(repositories, repo)
		}

		if response.NextPage == 0 {
			break
		}
		options.Page = response.NextPage
	}

	sort.Slice(repositories, func(i, j int) bool {
		return repositories[i].FullName() < repositories[j].FullName()
	})
	if !opts.NoCache {
		c.Set(cacheKey, repositories)
	}
	return repositories, nil
}

func sumStars(repos []Repo) int {
	total := 0
	for _, r := range repos {
		total += r.Stars
	}
	return total
}
