package github

import "encoding/gob"

func init() {
	gob.Register([]Repo{})
}

// Repo represents a GitHub repository with optional star count.
type Repo struct {
	Owner string
	Name  string
	Stars int
}

// FullName returns the "owner/name" form.
func (r Repo) FullName() string {
	return r.Owner + "/" + r.Name
}

// SearchOptions controls the behavior of repository search functions.
type SearchOptions struct {
	ShowFull  bool
	NoCache   bool
	DebugMode bool
}

// RepoInfo holds repository information for JSON export.
type RepoInfo struct {
	Date       string `json:"date"`
	Repository string `json:"repository"`
	Type       string `json:"type"`
	StarCount  int    `json:"starcount"`
}
