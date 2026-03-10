package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

// TestAuthCommandsRegistered verifies that the auth command and its three
// subcommands (add, remove, list) are registered on the root command.
func TestAuthCommandsRegistered(t *testing.T) {
	t.Parallel()

	// Find the auth command on the root.
	var authCmd *cobra.Command
	for _, sub := range rootCmd.Commands() {
		if sub.Use == "auth" {
			authCmd = sub
			break
		}
	}
	if authCmd == nil {
		t.Fatal("auth command is not registered on rootCmd")
	}

	wantSubcommands := []string{"add", "remove", "list"}
	for _, want := range wantSubcommands {
		found := false
		for _, sub := range authCmd.Commands() {
			if sub.Use == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("auth subcommand %q is not registered", want)
		}
	}
}

// TestMaskToken verifies that maskToken produces the expected output for a
// range of inputs covering the normal case, the boundary case (exactly 4
// chars), and the short case (fewer than 4 chars).
func TestMaskToken(t *testing.T) {
	t.Parallel()

	cases := []struct {
		token string
		want  string
	}{
		// Normal token: first 4 chars preserved, rest replaced.
		{"abcd1234xyz", "abcd****"},
		// Exactly 4 characters: should return "****" (not longer than 4 means <=4).
		{"abcd", "****"},
		// Shorter than 4 characters.
		{"ab", "****"},
		// Empty string.
		{"", "****"},
		// Token of exactly 5 characters: first 4 visible.
		{"abcde", "abcd****"},
	}

	for _, tc := range cases {
		tc := tc // capture range var
		t.Run(tc.token, func(t *testing.T) {
			t.Parallel()
			got := maskToken(tc.token)
			if got != tc.want {
				t.Errorf("maskToken(%q) = %q, want %q", tc.token, got, tc.want)
			}
		})
	}
}
