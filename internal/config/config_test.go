package config

import (
	"os"
	"testing"
)

func TestFromEnvironment_Defaults(t *testing.T) {
	os.Unsetenv("GITHUB_TOKEN")
	os.Unsetenv("SLACK_MODE")
	os.Unsetenv("DEBUG")

	cfg := FromEnvironment()
	if cfg.GitHubToken != "" {
		t.Errorf("expected empty token, got %q", cfg.GitHubToken)
	}
	if cfg.SlackMode {
		t.Error("expected SlackMode false by default")
	}
	if cfg.DebugMode {
		t.Error("expected DebugMode false by default")
	}
	if cfg.CacheFile == "" {
		t.Error("expected non-empty CacheFile")
	}
}

func TestFromEnvironment_GitHubToken(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "ghp_test123")

	cfg := FromEnvironment()
	if cfg.GitHubToken != "ghp_test123" {
		t.Errorf("got %q, want ghp_test123", cfg.GitHubToken)
	}
}

func TestFromEnvironment_SlackMode(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"FALSE", false},
		{"0", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run("SLACK_MODE="+tt.val, func(t *testing.T) {
			t.Setenv("SLACK_MODE", tt.val)
			cfg := FromEnvironment()
			if cfg.SlackMode != tt.want {
				t.Errorf("SLACK_MODE=%q → SlackMode=%v, want %v", tt.val, cfg.SlackMode, tt.want)
			}
		})
	}
}

func TestFromEnvironment_DebugMode(t *testing.T) {
	tests := []struct {
		val  string
		want bool
	}{
		{"true", true},
		{"1", true},
		{"yes", true},
		{"false", false},
		{"0", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run("DEBUG="+tt.val, func(t *testing.T) {
			t.Setenv("DEBUG", tt.val)
			cfg := FromEnvironment()
			if cfg.DebugMode != tt.want {
				t.Errorf("DEBUG=%q → DebugMode=%v, want %v", tt.val, cfg.DebugMode, tt.want)
			}
		})
	}
}
