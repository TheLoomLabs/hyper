package git_test

import (
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/git"
)

// TestInheritable_DropsWhatRedirectsGitAndNothingElse is the whole of what this
// package promises, and the reason it is a package: two of hyper's own packages
// start a git subprocess, and a variable that decided **which** repository git
// acts on would have one of them acting on the repository the caller named and
// the other on whatever an ambient variable points at (§7).
func TestInheritable_DropsWhatRedirectsGitAndNothingElse(t *testing.T) {
	kept := git.Inheritable([]string{
		"PATH=/usr/bin",
		"GIT_DIR=/elsewhere/.git",
		"HOME=/home/igor",
		"GIT_WORK_TREE=/elsewhere",
		"GIT_COMMON_DIR=/elsewhere/.git",
		"GIT_INDEX_FILE=/elsewhere/.git/index",
		"GIT_OBJECT_DIRECTORY=/elsewhere/.git/objects",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES=/elsewhere",
		"GIT_NAMESPACE=elsewhere",
		"GIT_AUTHOR_NAME=hyper",
		"SSH_AUTH_SOCK=/run/agent",
	})

	want := []string{
		"PATH=/usr/bin",
		"HOME=/home/igor",
		// Not dropped, and deliberately: it says how git behaves while
		// acting rather than what it acts on, and hyper writes its own
		// over it by writing it last.
		"GIT_AUTHOR_NAME=hyper",
		// The credential helper's own socket. The git hyper shells out
		// to is the git that resolves the credential a checkout left
		// behind (§7, §11), so the operator's setup is reached.
		"SSH_AUTH_SOCK=/run/agent",
	}
	if !slices.Equal(kept, want) {
		t.Errorf("kept %q, want %q", kept, want)
	}
}

// An entry with no "=" at all is not an environment entry. It is passed through
// untouched rather than guessed at: dropping it would be this package deciding
// something about a value it cannot read.
func TestInheritable_AMalformedEntryIsPassedThrough(t *testing.T) {
	if kept := git.Inheritable([]string{"GIT_DIR"}); !slices.Equal(kept, []string{"GIT_DIR"}) {
		t.Errorf("kept %q, want the entry passed through", kept)
	}
}
