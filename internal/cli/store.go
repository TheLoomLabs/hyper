package cli

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// RunStore implements `hyper store` — the one noun group in §9's tree, and the
// first command in the tool that writes anything at all (issue #126).
//
// The noun is what makes *this creates a git branch* legible at the point of
// use: a bare `init` would read as initialising a repository, which is the one
// thing this command does not do. The group is also where a second verb goes if
// one is ever earned, and `init` is the whole of it today — the sub-verbs are
// tree.go's list, so the surface the completion scripts describe and the
// surface the dispatch accepts are one statement.
//
// It is the first command handed the clock, which is why its signature carries
// one where its six neighbours do not: every commit `hyper` writes takes both
// its dates from it, so a fixture's branch is reproducible and `git log` on the
// Store is honest (§7, issue #125).
func RunStore(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string, now func() time.Time) int {
	// No --limit: `store` names no namespace and ranges over nothing, so
	// there is no result set for a cap to cut (§9). The name here is the
	// noun alone and not the two words a caller typed, because at this point
	// there is no verb yet: `hyper store init --force` answers `hyper store:
	// unknown flag --force`, the flags being read before the argument that
	// would say which verb the fault belongs to.
	parsed, code := parseArgs("store", args, takesNoLimit, lookupenv, stderr)
	if code != 0 {
		return code
	}

	// The verb, and its faults, decided from the argument list alone and
	// before any repository is resolved. That is `provider`'s and
	// `completions`' reading of their own arity: a word that is not one of
	// the sub-verbs is not a name resolved against a namespace the repository
	// holds — the set is compiled in — so there is nothing to load before the
	// invocation can be judged wrong (§9, ADR-0060).
	if len(parsed.positional) == 0 {
		fmt.Fprintf(stderr, "hyper store: %s\n  known verbs: %s\n", arityFault(nil, "verb"), strings.Join(storeSubVerbs, ", "))
		return ExitUsage
	}
	verb := parsed.positional[0]
	run, built := storeVerbs[verb]
	if !built {
		fmt.Fprintf(stderr, "hyper store: unknown verb %q\n  known verbs: %s\n", verb, strings.Join(storeSubVerbs, ", "))
		return ExitUsage
	}
	return run(parsed, stdout, stderr, lookupenv, wd, binaryVersion, now)
}

// storeVerb is the shape of one of the group's verbs: the arguments already
// read, the streams, the three reads of the process, and the version the gate
// compares. Its own positionals are its business rather than the group's —
// `init` takes none, and a verb that took one would be judging it against
// something only it knows.
type storeVerb func(parsed commandArgs, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string, now func() time.Time) int

// storeVerbs is the group's dispatch: which verb runs. It stands to
// storeSubVerbs exactly as repositoryCommands stands to tree.go's tree — the
// list is the surface §9 fixes and this is the subset the binary implements, so
// a verb the spec names and no milestone has built answers `unknown verb` while
// still being offered by the completion scripts (§9, issue #104).
//
// It is a table rather than a comparison against the list because the two
// questions are different ones. *Is this a verb the spec fixes* is what the
// message answers; *which function runs* is what this answers, and a second
// entry in the list with no arm here would otherwise have silently run `init`.
var storeVerbs = map[string]storeVerb{
	"init": runStoreInit,
}

// storeInit is the command name this verb's own messages and its gate are
// spelled with. It is the two words a caller typed rather than the noun alone:
// `hyper store init: ...` is what they can retype, and the group has no
// behaviour of its own for a message to belong to.
//
// The group's messages are `hyper store:`, and the split is where the verb is
// known: a flag fault and an unknown verb are read before there is a verb to
// name, and everything past the dispatch above belongs to `init`.
const storeInit = "store init"

// runStoreInit is `hyper store init`: the pin gate, then the branch, then the
// two forms of one row.
//
// The order is §9's and is the same everywhere — the repository root, the gate,
// then the command's own work — so a mismatched pin exits 77 before a single
// git subprocess runs, and a repository with no pin at all Refuses naming
// `hyper project`. That makes the bootstrap sequence for a new repository
// `hyper project`, then `hyper store init`, then anything else (§11, ADR-0020).
func runStoreInit(parsed commandArgs, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string, now func() time.Time) int {
	// `init` takes its verb and nothing else. The arity is judged here rather
	// than by the group, a verb being the only thing that knows what it takes.
	if rest := parsed.positional[1:]; len(rest) > 0 {
		fmt.Fprintf(stderr, "hyper %s: takes no positional argument, got %s\n", storeInit, rest[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot(storeInit, parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	if code := gateOnVersionPin(storeInit, repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	done, err := store.Init(repoRoot, now())
	if err != nil {
		fmt.Fprintf(stderr, "hyper %s: %s\n", storeInit, err)
		// The two ways this command can stop, and the whole of what
		// separates them. A repository root holding no git repository is
		// the invocation being wrong — there is no branch to create and no
		// repository to refuse on behalf of — and everything else is the
		// world resisting: a push rejected, a remote unreachable, a git
		// object that would not write. Neither is 75 or 77: 75 is a Run
		// that lost the Store and this is not a Run, and 77 promises that
		// a verbatim retry Refuses identically, which is false the moment
		// the network comes back (§9, §12, ADR-0061).
		if errors.Is(err, store.ErrNoRepository) {
			return ExitUsage
		}
		return ExitProblems
	}

	row := branchRow{Type: "branch", Branch: store.BranchName, Created: done.Created}
	if done.Pushed {
		row.Pushed = &done.Pushed
	}

	if code := writeAnswer(storeInit, stdout, stderr, parsed.json, []render.Row{row}, render.NewResultRow(false), writeBranchPage); code != 0 {
		return code
	}
	return ExitClean
}

// branchRow is `store init`'s row, and the one row it writes: the branch, and
// what this invocation did to it. `store init` is not a Run, so the stream
// terminates in `result` rather than `outcome`, and `truncated` is always false
// — one branch is not a result set a limit could cut (§8, §9).
//
// created is written always, false included: it is the answer to the question
// the command was asked, and a second `init` that omitted it would be a stream
// that said nothing rather than one that said *there was already a Store*.
//
// pushed is a pointer because it is a member the row must be able to *not
// carry*. It is written where this invocation pushed and absent otherwise — a
// repository with no remote configured, or a branch fetched from a remote that
// already held it — which is the ordinary absence rule: a fact the command did
// not perform is not stated against a blank, and a `false` there would be an
// answer to a question nothing asked (§7). A push that could not complete
// writes no row at all, the command having exited 1 before there was one.
type branchRow struct {
	Type    string `json:"type"`
	Branch  string `json:"branch"`
	Created bool   `json:"created"`
	Pushed  *bool  `json:"pushed,omitempty"`
}

// Cells is empty: this row is a block of labelled values rather than a line in
// a table of like rows, and the page renders it as writeBranchPage writes it.
// A row contributing no line is the shape the terminal row already has
// (ADR-0026).
func (r branchRow) Cells() []string { return nil }

// writeBranchPage is `store init`'s page: three labelled values, each naming
// what the command did and what it did it to, and no line at all for a thing it
// did not do (§9, issue #124, issue #126).
//
// The three read as sentences with their verbs in the label — `created
// hyper-store`, `pushed origin`, `wrote STORE.md` — which is what lets the
// absence rule carry information here rather than merely omit a blank: a page
// with no `pushed` line says the branch went nowhere, and one with no `wrote`
// line says this invocation authored no file.
//
// All three come off the row's own members, which is why `created` means what
// it does. A minted root and the STORE.md written into it are one act, so
// `created` decides two of the three lines; a branch fetched from a remote that
// already held it created no Store and wrote no file, and says so by writing
// neither line (§7, ADR-0026).
//
// A run that did none of the three has a line of its own rather than an empty
// block, which is `targets`'s empty repository and `check`'s clean run read once
// more: a command that found the world already as its arguments asked states
// that, and a block over no values would state less than a sentence does (§9,
// issue #99). It is reached by the two invocations that changed nothing at all —
// the branch was already on both sides, or it came down from a remote that
// already held it — and not by the one that created nothing and pushed anyway,
// which has a `pushed` line and is the whole of its page.
func writeBranchPage(w io.Writer, rows []render.Row) error {
	if len(rows) == 0 {
		return nil
	}
	branch, written := rows[0].(branchRow)
	if !written {
		return nil
	}

	created, wrote := "", ""
	if branch.Created {
		created, wrote = branch.Branch, store.IntroductionPath
	}
	pushed := ""
	if branch.Pushed != nil && *branch.Pushed {
		pushed = store.RemoteName
	}

	values := []labelledValue{
		{"created", created},
		{"pushed", pushed},
		{"wrote", wrote},
	}
	if !slices.ContainsFunc(values, func(stated labelledValue) bool { return stated.value != "" }) {
		_, err := fmt.Fprintf(w, "%s is already there\n", branch.Branch)
		return err
	}
	return writeLabelledValues(w, values)
}
