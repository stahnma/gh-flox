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
func FindManifestRepos(ctx context.Context, client Client, c *cache.Cache, mc *MembershipCache, showFull, verbose, noCache, debugMode bool) ([]Repo, int, error) {
	cacheKey := fmt.Sprintf("floxManifestRepos:%t:%t", showFull, verbose)
	if !noCache {
		if val, found := c.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			if repos, ok := val.([]Repo); ok {
				return repos, sumStars(repos), nil
			}
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	seen := make(map[string]bool)
	var repositories []Repo
	var totalStars int

	query := ".flox/env/manifest.toml in:path"
	options := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}

	for {
		results, response, err := client.SearchCode(ctx, query, options)
		if err != nil {
			return nil, 0, err
		}

		for _, item := range results.CodeResults {
			owner := item.Repository.GetOwner().GetLogin()
			name := item.Repository.GetName()
			fullName := owner + "/" + name

			if seen[fullName] {
				continue
			}
			seen[fullName] = true

			if !showFull {
				isMember, e := mc.Check(ctx, client, owner, "flox")
				if e != nil {
					log.Printf("Error checking membership: %v", e)
				}
				if isMember {
					continue
				}
				if excludedOrgs[owner] {
					continue
				}
			}

			repo := Repo{Owner: owner, Name: name}
			if verbose {
				stars, err := GetStarCount(ctx, client, c, owner, name, noCache, debugMode)
				if err == nil {
					repo.Stars = stars
					totalStars += stars
				}
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
	if !noCache {
		c.Set(cacheKey, repositories)
	}
	return repositories, totalStars, nil
}

// FindReadmeRepos searches for repositories containing "flox install" in their README.
func FindReadmeRepos(ctx context.Context, client Client, c *cache.Cache, mc *MembershipCache, showFull, verbose, noCache, debugMode bool) ([]Repo, int, error) {
	cacheKey := fmt.Sprintf("floxReadmeRepos:%t:%t", showFull, verbose)
	if !noCache {
		if val, found := c.Get(cacheKey); found {
			if debugMode {
				log.Printf("Cache hit for key: %s", cacheKey)
			}
			if repos, ok := val.([]Repo); ok {
				return repos, sumStars(repos), nil
			}
		}
		if debugMode {
			log.Printf("Cache miss for key: %s", cacheKey)
		}
	}

	seen := make(map[string]bool)
	var repositories []Repo
	var totalStars int

	query := "\"flox install\" in:file filename:README"
	options := &gh.SearchOptions{ListOptions: gh.ListOptions{PerPage: 100}}

	for {
		results, response, err := client.SearchCode(ctx, query, options)
		if err != nil {
			return nil, 0, err
		}

		for _, item := range results.CodeResults {
			owner := item.Repository.GetOwner().GetLogin()
			name := item.Repository.GetName()
			fullName := owner + "/" + name

			if seen[fullName] {
				continue
			}
			seen[fullName] = true

			if !showFull {
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
			if verbose {
				stars, err := GetStarCount(ctx, client, c, owner, name, noCache, debugMode)
				if err == nil {
					repo.Stars = stars
					totalStars += stars
				}
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
	if !noCache {
		c.Set(cacheKey, repositories)
	}
	return repositories, totalStars, nil
}

func sumStars(repos []Repo) int {
	total := 0
	for _, r := range repos {
		total += r.Stars
	}
	return total
}
