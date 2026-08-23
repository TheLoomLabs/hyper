// Package git is the one rule two packages that start a git subprocess must
// agree on: which of the process's environment such a subprocess may inherit
// (§7, issue #136).
//
// It is deliberately not the git layer. internal/store reads and writes the
// record branch, internal/revision reads the code branch, and each keeps its
// own calls: their subjects are different, their failures mean different
// things, and store's environment carries a commit identity and two dates that
// a reader has no commit to give one to. What they cannot keep separately is
// this list — a variable added to one copy and not the other would leave one
// package's git acting on the repository the caller named and the other's
// acting on whatever an ambient variable points at, silently.
//
// Extracting the calls themselves is milestone 8's, where reading code-branch
// objects at a revision earns a layer with more than one caller; what is here
// is the half that is already shared and already dangerous.
package git

import (
	"slices"
	"strings"
)

// redirecting names the environment variables that decide **which** repository
// git acts on, rather than how it behaves while acting.
//
// This is not hypothetical: git sets GIT_DIR and GIT_INDEX_FILE for every hook
// it runs, so a `hyper store init` or a `hyper run` invoked from a pre-commit
// hook would inherit them and act through whatever they point at — silently,
// and on a repository the caller never named.
var redirecting = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_COMMON_DIR",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

// NoLazyFetch is git's own switch for lazy fetching, set: with it in a
// subprocess's environment an object read on a partial clone answers *missing*
// instead of going to the promisor remote for it.
//
// It is here rather than in either caller for this file's own reason. Both
// packages that start a git subprocess read objects, ADR-0071 requires every
// such read to run with lazy fetching off, and the line that decision draws —
// between *an object read that silently becomes a fetch* and *a branch sync
// `hyper` chose* — is a rule about git and not about either package's subject.
// A switch spelled in one copy and not the other would leave one package's
// reads reaching the network and the other's not, silently, which is exactly
// what the list above exists to prevent.
//
// It is **appended** to an inherited environment rather than substituted into
// it. os/exec keeps the final value of a repeated name, so an entry written
// last wins over whatever the caller's environment happened to carry — which is
// the same rule internal/store's commit identity and dates already rest on.
const NoLazyFetch = "GIT_NO_LAZY_FETCH=1"

// Inheritable is an environment with those dropped and nothing else touched.
//
// The rest is inherited rather than replaced, and that is the one place this
// layer is deliberately not hermetic: the git hyper shells out to is the same
// git that resolves the credential a checkout left behind (§7, §11), so its
// configuration, its credential helpers and its SSH agent are all reached the
// way the operator already set them up.
//
// A variable is matched on the name before its first "=", which is what an
// environment entry is; anything malformed enough to have none is passed
// through untouched rather than guessed at.
func Inheritable(env []string) []string {
	kept := make([]string, 0, len(env))
	for _, entry := range env {
		name, _, named := strings.Cut(entry, "=")
		if named && slices.Contains(redirecting, name) {
			continue
		}
		kept = append(kept, entry)
	}
	return kept
}
