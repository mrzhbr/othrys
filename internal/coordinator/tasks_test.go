package coordinator

import (
	"testing"
)

// TestSlugify tests the slugify helper function.
func TestSlugify(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Hello World", "hello-world"},
		{"Implement Auth Module", "implement-auth-module"},
		{"Set up PostgreSQL DB!", "set-up-postgresql-db"},
		{"  leading spaces  ", "leading-spaces"},
		{"multiple---dashes", "multiple-dashes"},  // non-alphanums replaced, consecutive dashes preserved since they're already dashes
		{"UPPERCASE", "uppercase"},
		{"short", "short"},
	}

	for _, tc := range cases {
		got := slugify(tc.input)
		if got != tc.want {
			t.Errorf("slugify(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestSlugifyMaxLength tests that slugify truncates at 50 characters.
func TestSlugifyMaxLength(t *testing.T) {
	long := "this is a very long task title that exceeds fifty characters total for sure"
	got := slugify(long)
	if len(got) > 50 {
		t.Errorf("slugify length %d > 50 for long input", len(got))
	}
}

// TestBranchNameFormat tests the expected branch name format.
func TestBranchNameFormat(t *testing.T) {
	agentName := "omo-alice"
	taskTitle := "Implement Auth Module"
	expected := "othrys/omo-alice/implement-auth-module"

	got := "othrys/" + slugify(agentName) + "/" + slugify(taskTitle)
	if got != expected {
		t.Errorf("branch name = %q, want %q", got, expected)
	}
}
