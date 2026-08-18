package cli

import (
	"fmt"
	"io"

	"github.com/TheLoomLabs/hyper/internal/version"
)

// Main is hyper's one entry point: it takes the complete argv and returns the
// exit code, and which command runs is decided here rather than in
// cmd/hyper/main.go (issue #107).
//
// The gate's own reasoning is why the dispatch followed it in — gate.go states
// it, and it reaches the thing that decides which commands call the gate at
// all. With four commands landing in milestone 2 and eleven more to come,
// dispatch is not a detail of `main`; it is the surface §9 fixes, and it
// belongs on this side of the package boundary where the golden harness can
// reach it. That the harness does not yet drive it is #108's, which collapses
// the corpora onto this entry point; #107 is the half that makes the change
// easy.
//
// Everything a command reads from the process is a parameter, which is the
// property #100 established and this must not lose: the arguments, the
// environment, the working directory, and the facts the build stamped. Nothing
// in the body below reaches the process for itself, which is what makes the
// whole dispatch exercisable without a subprocess.
//
// getwd is a function rather than a resolved path because the exemption is a
// property of this dispatch and not of the commands behind it. `version` and
// `completions` are the two cases that resolve no working directory and call no
// gate — an exemption expressed as a branch not taken (§9, ADR-0020) — so the
// working directory is resolved inside the branches that need one, and a
// working directory that cannot be read does not stop `hyper version`.
//
// facts is threaded whole rather than the bare version string it carries:
// RunVersion needs all of it, the gate needs the version out of it, and passing
// the value keeps Main deterministic under test.
func Main(args []string, stdout, stderr io.Writer, getenv func(string) string, getwd func() (string, error), facts version.Facts) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: hyper <command> [args...]")
		return ExitUsage
	}

	switch args[0] {
	case "check":
		// The working directory is read here rather than above the switch,
		// so a command that reads no repository never depends on there being
		// one to read (issue #103).
		wd, err := getwd()
		if err != nil {
			// The code cmd/hyper returned before the dispatch moved,
			// unchanged: #107 moves the decision about which command runs
			// and nothing a command prints or exits with. It is spelled
			// with the name §12's closed set already fixes for 1 rather
			// than as a bare literal, on exit.go's own rule that a
			// milestone reaching a code inherits the name instead of
			// minting a second spelling for the number (issue #102).
			fmt.Fprintf(stderr, "hyper: %s\n", err)
			return ExitProblems
		}
		return RunCheck(args[1:], stdout, stderr, getenv, wd, facts.Version)
	case "version":
		// Neither the environment nor a working directory is passed, and no
		// repository root is resolved: `version` is one of the two commands
		// outside the tree of sixteen and exempt from the pin gate (§9,
		// ADR-0020).
		return RunVersion(args[1:], stdout, stderr, facts)
	case "completions":
		// The other command outside the tree, and exempt for the same
		// reason: it reads no repository, so shell setup in a dotfiles
		// bootstrap works before one exists (§9, ADR-0020, issue #104).
		return RunCompletions(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "hyper: unknown command %q\n", args[0])
		return ExitUsage
	}
}
