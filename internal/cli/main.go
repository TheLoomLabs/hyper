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
// Getwd is called on the repository commands' arm and nowhere else, because that
// exemption is a property of this dispatch and not of the commands behind it.
// `version`, `completions` and `mcp` are the three cases that resolve no working
// directory and call no gate — an exemption expressed as a path not taken (§9,
// ADR-0020) — so a working directory that cannot be read does not stop `hyper
// version`. `mcp` is the one of the three that is not exempt from the gate at
// all: it never reaches a repository because starting a server is not an act on
// one, and every tool it carries gates exactly as its command does (ADR-0088).
// `project` calls no gate either and is not one of them: it resolves a
// repository like every other command in the table below and declines to compare
// itself against the pin it is about to write, which is an exemption the command
// states rather than one this dispatch grants (§11, issue #178).
//
// facts is threaded whole rather than the bare version string it carries:
// RunVersion needs all of it, the gate needs the version out of it, and passing
// the value keeps Main deterministic under test.
//
// The two writers are assembled into the CLI's destination here and threaded
// down as one value: every command behind the dispatch is handed where its
// answer goes rather than the streams that answer is made of, which is what
// leaves room for a second destination that is not a pair of streams at all
// (destination.go, issue #194). Which form it takes is not decided here — the
// flag that names one is read by the shared parser, which is the only reader of
// it there is (flags.go).
func Main(args []string, stdout, stderr io.Writer, process Process, facts version.Facts) int {
	to := Streams(stdout, stderr)
	if len(args) == 0 {
		// The one invocation that asks what this binary is, answered
		// with §9's tree rather than with the word `<command>`
		// (usage.go, issue #210).
		fmt.Fprint(to.narrate(), usage())
		return ExitUsage
	}

	// The commands inside §9's tree that read a repository, and the whole of
	// what they share: the complete argv past the command name, the process,
	// and the version the gate compares. They are dispatched off one table
	// rather than a branch each, so the working directory — resolved for them
	// and for nobody else — is read in one place however many of the sixteen
	// land (issue #103, issue #111).
	if code, dispatched := runRepositoryCommand(args, to, process, facts); dispatched {
		return code
	}

	switch args[0] {
	case "version":
		// Neither the environment nor a working directory is passed, and no
		// repository root is resolved: `version` is one of the three
		// commands outside the tree of sixteen and exempt from the pin gate
		// (§9, ADR-0020). The destination is the third thing not passed,
		// and for the same reason: `version` and `completions` write no
		// answer through one — a version block and a shell script are the
		// whole of what each has to say, in one form, with no `--json` to
		// name a second (destination.go).
		return RunVersion(args[1:], stdout, stderr, facts)
	case "completions":
		// The second command outside the tree, and exempt for the same
		// reason: it reads no repository, so shell setup in a dotfiles
		// bootstrap works before one exists (§9, ADR-0020, issue #104).
		return RunCompletions(args[1:], stdout, stderr)
	case "mcp":
		// The third command outside the tree, and the one that starts §9's
		// second surface. It is exempt from the gate for neither of the
		// other two's reasons and needs no exemption at all: the
		// invocation is not the act, so the gate fires per tool at the
		// moment a tool resolves a repository, exactly as it fires for the
		// command that tool carries (§9, ADR-0088).
		//
		// **stdout is not passed**, and that is the arm's whole shape: the
		// server's stdout belongs to the protocol, so the one thing this
		// command is handed is the narration a usage error goes to. The
		// frames are the transport's, which reaches the process's own
		// streams and hands a writer to nothing behind it (mcp.go, issue
		// #195).
		return RunMCP(args[1:], stderr, process, facts)
	default:
		// §9's rule for a positional that matches nothing, applied to
		// the command name: what was typed, the namespace it was
		// resolved against, and the command that enumerates that
		// namespace. `help` and `--help` are correctly unknown — no
		// command is added here — but a message saying only *unknown*
		// leaves a caller exactly where the empty usage line did, and
		// both agents that went foraging reached this branch before
		// they reached the bare invocation (usage.go, issue #210).
		//
		// It writes the pointer and not the page: the tree is what a
		// bare `hyper` answers with, and twenty-eight lines in front
		// of somebody who missed a keystroke is narration nobody
		// asked for. It suggests no near miss either, for the reason
		// nothing here ever does (§9, ADR-0047).
		fmt.Fprintf(stderr, "hyper: unknown command %q\n%s", args[0], whereTheCommandsAre)
		return ExitUsage
	}
}

// runRepositoryCommand runs one of §9's sixteen where args names one, and
// answers whether it did.
//
// It is the dispatch's own arm, lifted out of Main so that a second surface can
// stand on it: the MCP server's tools build the command line their commands
// would have received and hand it here, so the tool and the terminal reach one
// command through one table — *ergonomics is the whole of the difference
// between the two* (§9, mcp.go, issue #195).
//
// **Getwd is called here and nowhere else**, which is the exemption stated as a
// path not taken: the commands outside the tree never reach this function, so a
// working directory that cannot be read does not stop `hyper version` (§9,
// ADR-0020, issue #103). It is called per dispatch rather than once, which is
// what makes the working directory a fact about the process for both surfaces
// alike: the server's tools resolve the repository the way every command does,
// against the directory the client started the process in.
func runRepositoryCommand(args []string, to destination, process Process, facts version.Facts) (int, bool) {
	run, gated := repositoryCommands[args[0]]
	if !gated {
		return 0, false
	}

	wd, err := process.Getwd()
	if err != nil {
		// The code cmd/hyper returned before the dispatch moved,
		// unchanged: #107 moves the decision about which command runs and
		// nothing a command prints or exits with. It is spelled with the
		// name §12's closed set already fixes for 1 rather than as a bare
		// literal, on exit.go's own rule that a milestone reaching a code
		// inherits the name instead of minting a second spelling for the
		// number (issue #102).
		fmt.Fprintf(to.narrate(), "hyper: %s\n", err)
		return ExitProblems, true
	}
	return run(args[1:], to, process, wd, facts.Version), true
}

// repositoryCommand is the shape of a command that reads a repository: the
// arguments past its own name, the destination its answer goes to, the process
// it is handed rather than reaches for, the working directory already resolved
// out of it, and the version the pin gate compares. Every one of §9's sixteen
// takes it — the gate is stated once for all of them, and nothing else about a
// command is Main's business.
//
// **The destination stands where the two writers stood**, and it carries a
// third thing they never could: which form the answer takes. A command is
// handed one place to put its answer, one place to put a Refusal and one place
// to narrate, and none of the three is a stream it chose (destination.go, issue
// #194).
//
// The working directory rides beside the process rather than being left to
// Getwd because resolving it is the dispatch's act and not a command's: it is
// resolved once, on this arm, so that the commands on the other arm never
// resolve one at all (§9, ADR-0020). A command behind the dispatch holds a
// Getwd it could call and has no reason to — the answer is already in its
// hand — which is a rule this signature states rather than a shape it enforces,
// the enforcement being that the exempt arm never gets here.
type repositoryCommand func(args []string, to destination, process Process, wd, binaryVersion string) int

// environmentOnly adapts a command that reads nothing of the process but the
// environment to the dispatch's one shape.
//
// It is what is left of clockless once the reads became one value, and it
// exists for the reason clockless did: a command's signature says what it
// reads, and #100's property is that what a command reads from the process is
// visible in what it takes rather than discharged by reading its body. Five of
// §9's sixteen read the environment and nothing else — HYPER_REPO_DIR and
// NO_COLOR — so they take a lookup and say by their shape that they read no
// clock, mint no id, dial nothing and start no child. `store` and `compact`
// write, and take the whole value; `review` grew a clock and left this arm for
// the table below, which is the property doing its work rather than failing it
// (issue #164).
func environmentOnly(run func(args []string, to destination, lookupenv func(string) (string, bool), wd, binaryVersion string) int) repositoryCommand {
	return func(args []string, to destination, process Process, wd, binaryVersion string) int {
		return run(args, to, process.LookupEnv, wd, binaryVersion)
	}
}

// repositoryCommands is the dispatch: which command runs, for the commands that
// stand behind the pin gate. It is not §9's tree — tree.go holds that, and a
// name the spec fixes is there whether or not a milestone has built it yet;
// this is the subset the binary implements, and a name absent from it is the
// `unknown command` below.
var repositoryCommands = map[string]repositoryCommand{
	"check":     environmentOnly(RunCheck),
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
	// The first of the six that took a lookup alone to outgrow it. A review
	// opens a range against the last Run that read the artefact (§8), so it
	// reads the Journal for that entry and the clock for the age it renders
	// beside the gloss — and it takes the whole value to say so, exactly as
	// the commands that read the record do. What it still does not do is
	// sync: it reads whatever branch this clone holds, no credential
	// resolves and no network is touched (§9, issue #164).
	"review": RunReview,
	// The first command a person can type that reads the record back. It
	// takes the whole value for the clock alone — the Store's handle is
	// opened at one instant, as `compact`'s is — and it writes nothing:
	// four of §9's sixteen exist to read the Journal and the Records back,
	// and this is the first of them (§9, issue #163).
	"show": RunShow,
	// The second of the four, and the surface that enumerates the namespace
	// `show`'s own unresolved-id message points at. It takes the whole
	// value for the same clock and writes nothing either (§9, issue #165).
	"runs": RunRuns,
	// The fourth of the four, and the one §1's second claim rests on:
	// *what changed between one run and the next* is a window over the
	// Journal, and this is the surface that opens one. It takes the whole
	// value for the clock the Store's handle is opened at, and writes
	// nothing (§8, §9, issue #167).
	"changes": RunChanges,
	// The third of the four, and the one whose job is finding a version.
	// It reads the record for the versions and the working tree for one
	// column of them — an Asset whose Definition no longer exists is
	// Orphaned, and what exists is a fact about the repository rather than
	// about the branch (§7, §9, issue #166).
	"records": RunRecords,
	// The fifteenth of the sixteen, and the first thing in the tool that
	// writes a file into the working tree. It took a lookup alone until it
	// grew the one thing the version pin ever reaches the network for: the
	// checksum published for the version it is about to pin, resolved once,
	// attended, and only where that version differs from the pin already in
	// the declaration (§11, issue #178). Everything else about it is still a
	// function of the reviewed artefacts and of the binary's own version —
	// no clock, no id, no child — and the signature says which of those two
	// it is.
	//
	// It is also the one name in this table that calls no gate, and it is
	// the command itself that does not call one rather than the dispatch
	// exempting it: the exemption belongs to the pin's only writer, and
	// stating it here would put it a layer above the reason for it (§9,
	// ADR-0020).
	"project": RunProject,
	// The sixteenth and last of §9's sixteen, and the single point at which
	// third-party data enters the repository. It takes the whole value for
	// `project`'s reason and for one of `probe`'s: it dials, and Dial is the
	// member that says so. It reads no clock, mints no id and starts no
	// child — the fetch is hyper's own and not an Operation — and the file it
	// writes is a tracked one a human reads in a diff (§9, §11, ADR-0087,
	// issue #187).
	"install": RunInstall,
}
