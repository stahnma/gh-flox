package config

import (
	"os"
	"strings"
)

// Config holds application configuration loaded from environment variables.
type Config struct {
	GitHubToken string
	SlackMode   bool
	DebugMode   bool
	CacheFile   string
	NoCache     bool
}

// FromEnvironment creates a Config from environment variables.
func FromEnvironment() Config {
	slack := os.Getenv("SLACK_MODE")
	slackMode := slack != "" && strings.ToLower(slack) != "false" && slack != "0"

	debug := os.Getenv("DEBUG")
	debugMode := debug != "" && debug != "0" && strings.ToLower(debug) != "false"

	return Config{
		GitHubToken: os.Getenv("GITHUB_TOKEN"),
		SlackMode:   slackMode,
		DebugMode:   debugMode,
		CacheFile:   "/tmp/cache.gob",
	}
}
