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
// property #100 established and this must not lose: the arguments, the reads
// process.go states, and the facts the build stamped. Nothing in the body
// below reaches the process for itself, which is what makes the whole dispatch
// exercisable without a subprocess.
//
// process is those reads as one value rather than as one parameter each, which is
// the only thing issue #134 changes: the clock reached this signature loose one
// milestone earlier, three more reads land in this one, and six of them threaded
// singly is a parameter list a reader counts instead of a type they open. What
// each member is and why it is threaded is process.go's to say; that a command
// reaches for none of them itself is this dispatch's.
//
// Getwd is called on the repository commands' arm and nowhere else, because the
// exemption is a property of this dispatch and not of the commands behind it.
// `version` and `completions` are the two cases that resolve no working
// directory and call no gate — an exemption expressed as a path not taken (§9,
// ADR-0020) — so a working directory that cannot be read does not stop `hyper
// version`.
//
// facts is threaded whole rather than the bare version string it carries:
// RunVersion needs all of it, the gate needs the version out of it, and passing
// the value keeps Main deterministic under test.
func Main(args []string, stdout, stderr io.Writer, process Process, facts version.Facts) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: hyper <command> [args...]")
		return ExitUsage
	}

	// The commands inside §9's tree that read a repository, and the whole of
	// what they share: the complete argv past the command name, the process,
	// and the version the gate compares. They are dispatched off one table
	// rather than a branch each, so the working directory — resolved for them
	// and for nobody else — is read in one place however many of the sixteen
	// land (issue #103, issue #111).
	if run, gated := repositoryCommands[args[0]]; gated {
		// The working directory is read here rather than above, so a
		// command that reads no repository never depends on there being one
		// to read (issue #103).
		wd, err := process.Getwd()
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
		return run(args[1:], stdout, stderr, process, wd, facts.Version)
	}

	switch args[0] {
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

// repositoryCommand is the shape of a command that reads a repository: the
// arguments past its own name, the two streams, the process it is handed rather
// than reaches for, the working directory already resolved out of it, and the
// version the pin gate compares. Every one of §9's sixteen takes it — the gate
// is stated once for all of them, and nothing else about a command is Main's
// business.
//
// The working directory rides beside the process rather than being left to
// Getwd because resolving it is the dispatch's act and not a command's: it is
// resolved once, on this arm, so that the two commands on the other arm never
// resolve one at all (§9, ADR-0020). A command behind the dispatch holds a
// Getwd it could call and has no reason to — the answer is already in its
// hand — which is a rule this signature states rather than a shape it enforces,
// the enforcement being that the exempt arm never gets here.
type repositoryCommand func(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int

// environmentOnly adapts a command that reads nothing of the process but the
// environment to the dispatch's one shape.
//
// It is what is left of clockless once the reads became one value, and it
// exists for the reason clockless did: a command's signature says what it
// reads, and #100's property is that what a command reads from the process is
// visible in what it takes rather than discharged by reading its body. Six of
// §9's sixteen landed before the Store, and the environment is the whole of
// what they read — HYPER_REPO_DIR and NO_COLOR — so they take a lookup and say
// by their shape that they read no clock, mint no id, dial nothing and start no
// child. `store` and `compact` write, and take the whole value.
func environmentOnly(run func(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int) repositoryCommand {
	return func(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
		return run(args, stdout, stderr, process.LookupEnv, wd, binaryVersion)
	}
}

// repositoryCommands is the dispatch: which command runs, for the commands that
// stand behind the pin gate. It is not §9's tree — tree.go holds that, and a
// name the spec fixes is there whether or not a milestone has built it yet;
// this is the subset the binary implements, and a name absent from it is the
// `unknown command` below.
var repositoryCommands = map[string]repositoryCommand{
	"check":     environmentOnly(RunCheck),
	"review":    environmentOnly(RunReview),
	"providers": environmentOnly(RunProviders),
	"provider":  environmentOnly(RunProvider),
	"operation": environmentOnly(RunOperation),
	"targets":   environmentOnly(RunTargets),
	// The one noun group, dispatched on its noun: `init` is the verb and
	// RunStore reads it, so the group's grammar is stated in one place
	// rather than split between the table and the command (§9, issue #126).
	"store": RunStore,
	// The second command that reads the record, and the first thing in the
	// tool that removes anything. It takes the whole value for `store`'s
	// reason and for one of its own: every commit hyper writes takes both
	// its dates from the clock, and retention is an age (§7, issue #131).
	"compact": RunCompact,
	// The first command that touches the world. It reads no record at all
	// and still takes the whole value: it dials, and it reads the clock its
	// tls.days_left counts from (§9, §12, issue #135).
	"probe": RunProbe,
	// The tracer bullet, and the only command in the tree that is a Run:
	// artefact, check, call, projection, Record, Store. It reads every
	// member of the process there is — the environment for its Trigger, the
	// working directory the dispatch resolved, the machine's name, the
	// clock, the mint, the dialer and the launcher — which is what makes it
	// the command the whole value was assembled for (§6, §9, issue #136).
	"run": RunRun,
	// The first command a person can type that reads the record back. It
	// takes the whole value for the clock alone — the Store's handle is
	// opened at one instant, as `compact`'s is — and it writes nothing:
	// four of §9's sixteen exist to read the Journal and the Records back,
	// and this is the first of them (§9, issue #163).
	"show": RunShow,
}
