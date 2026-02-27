package github

import "testing"

func TestRepoFullName(t *testing.T) {
	tests := []struct {
		owner, name, want string
	}{
		{"alice", "myrepo", "alice/myrepo"},
		{"org", "project", "org/project"},
		{"", "", "/"},
	}
	for _, tt := range tests {
		r := Repo{Owner: tt.owner, Name: tt.name}
		if got := r.FullName(); got != tt.want {
			t.Errorf("Repo{%q,%q}.FullName() = %q, want %q", tt.owner, tt.name, got, tt.want)
		}
	}
}
