package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// runCommand is the name this command's messages and its gate are spelled with.
// `run` and `runs` sit one letter apart, which §9 calls a readability wart
// rather than a hazard — `run` requires an argument and `runs` accepts none
// positionally, so the typo in either direction is a usage error.
const runCommand = "run"

// RunRun implements `hyper run <procedure>` — the tracer bullet, and the first
// thing in the tool that writes to the record on its own account (issue #136).
//
// The name is the wart §9 already owns: every command in this package is
// `Run<Command>` and this command is `run`, so a reader looking for `hyper
// run`'s entry point finds it where they would look for any other's. Nothing
// else here is named for it.
//
// What it does is decide the occasion and render the answer, and the whole of
// what happens in between is internal/run's. The split is ADR-0026's, and it is
// the reason the engine can be exercised without a subprocess: the Trigger, the
// clock, the mint, the dialer and the streams are read and resolved here, and
// the engine reaches for none of them.
//
// The order past the gate is §9's *positional, then the Store*: `hyper run
// typo` is `2` on a repository with no Store at all, because a working-tree
// name needs nothing further to resolve and the typo is repaired before the
// Store is missed (§9, ADR-0060). Past that it is §6's fixed order, which
// internal/run states.
//
// **The positional is a Procedure's name and takes no second form**, unlike
// `review`'s, which takes a path beside one — the two commands differing rather
// than the two positionals (ADR-0090, unresolvedProcedure).
func RunRun(args []string, to destination, process Process, wd, binaryVersion string) int {
	// `--target` is refused by name rather than as an unknown flag, because
	// what it means is a decision rather than a typo: a Procedure is fully
	// bound and declares its own Target envelope, so a Target supplied at
	// invocation is either redundant with the artefact or it is authority
	// arriving after review (§5, §9, ADR-0008).
	if fault := refusedFlags(args); fault != "" {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, fault)
		return ExitUsage
	}

	// `--secret-out` is read off the argument list before the globals, for
	// splitInputs' own reason: the three globals are every command's and this
	// is one command's, and a parser that knew about both is one every other
	// command's signature would have to admit. `--` ends the flags, so a
	// positional spelled like a flag is still reachable (§9).
	sink, rest, fault := splitSecretOut(args)
	if fault != "" {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, fault)
		return ExitUsage
	}

	// `--dry-run` comes off next, and it is here rather than in parseArgs
	// for `--secret-out`'s reason: §9 gives it to `run` and to no other
	// command, so `hyper check --dry-run` stays the unknown flag it is.
	dryRun, rest := splitDryRun(rest)

	// No --limit: a Run reports what it just did rather than ranging over a
	// namespace, so there is no result set for a cap to cut (§9).
	// `--secret-out` and `--dry-run` are named here though they were taken
	// off the line above: the value states this command's flag namespace,
	// which is what a caller who typed a fourth flag is told (flags.go).
	parsed, to, code := parseArgs(runCommand, rest, parameters{limit: takesNoLimit, secretOut: true, dryRun: true}, process.LookupEnv, to)
	if code != 0 {
		return code
	}
	if len(parsed.positional) != 1 {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, arityFault(parsed.positional, "Procedure"))
		return ExitUsage
	}
	name := parsed.positional[0]

	repoRoot, code := resolveRepoRoot(runCommand, parsed.repoDir, process.LookupEnv, wd, to.narrate())
	if code != 0 {
		return code
	}
	// Where this Run's answer goes and whether the invocation was a
	// rehearsal, carried from here to every path that ends without an answer
	// being written the ordinary way (§8, §9, issue #172).
	rendering := runRendering{to: to, dryRun: dryRun}

	// The gate, and the first of the two paths on which a Run **declines**
	// before it has an id: the terminal row goes out regardless, carrying
	// the code the gate Refused under and no `run_id` (§8, §9, §10).
	//
	// Its usage arm is not one of them and returns unchanged: a hyper.yaml
	// that cannot be read at all is the command never starting, and a usage
	// error opens no stream (§9, ADR-0060).
	switch code, errorCode := gateOnVersionPin(runCommand, repoRoot, binaryVersion, to); {
	case code == ExitUsage:
		return code
	case code != 0:
		return rendering.terminate(store.OutcomeRefused, code, errorCode)
	}

	// The sink's path against the working tree, which needs the root and so
	// is judged here rather than where the flag was taken off the argument
	// list. It is after the gate for the reason every command's own work is:
	// the pin gate fires first everywhere (§9, ADR-0020).
	if fault := sinkInsideTheWorkingTree(sink, repoRoot, wd); fault != "" {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, fault)
		return ExitUsage
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, err)
		return ExitUsage
	}
	if fault := unresolvedProcedure(loaded, name); fault != "" {
		fmt.Fprint(to.narrate(), fault)
		return ExitUsage
	}

	// The lock, taken before the Store is reached at all and held for the
	// Run's duration. Which of the two it is was decided from the Kinds
	// `check` already computed, and it is taken here — before the sync,
	// before `run.json`, before Step 1 — because it is a lock on the Store
	// and everything past this line writes to one (§6, §7, issue #138).
	//
	// A Run that cannot take it is `failed` at 75 and is **not** a Refusal:
	// nothing declined, the other Run ends, and this invocation succeeds
	// verbatim five minutes later (§12, ADR-0061). It is also a Run that was
	// never attempted — the lock stands before the sync, before `run.json`
	// and before Step 1 — so it writes no terminal row either: §8 puts `run`
	// on the `outcome` side on every path a Run was attempted on and on the
	// two that decline before one is identified, and this is neither
	// (reportLockFault).
	mode := run.LockMode(loaded, name)
	lock, err := store.Acquire(repoRoot, mode)
	if err != nil {
		return reportLockFault(to.narrate(), err)
	}
	defer lock.Release()

	// The Store, located: the second thing §6's order does, and the one that
	// declines before the Run has an id to decline under. A Run never
	// creates the branch, read-only Runs included, because a fetch that
	// failed mid-flight and a branch that never existed look identical from
	// the inside (§6, §7).
	instant := process.Now()
	held, code := locateStore(repoRoot, instant, mode, rendering)
	if code != 0 {
		return code
	}

	// The watch for the two signals, installed here and not earlier: what
	// reads it is the engine, and a handler standing over the lock and the
	// sync would catch an interrupt nothing would then act on — leaving a
	// Ctrl-C during a fetch with no effect at all, which is worse than the
	// kernel's own answer. It comes down on the way out whichever way the
	// Run ended (signals.go).
	watch, release := watchForTheFirstInterrupt(process.Notify)
	defer release()

	answer := run.Perform(run.Request{
		Repository: loaded,
		RepoRoot:   repoRoot,
		Store:      held,
		Procedure:  name,
		// The rehearsal marker reaches two places and they must never
		// disagree: the entry carries it and the terminal row below
		// carries it, and §7's one exception to the absence rule is
		// written always, `false` included, precisely because a reader
		// that takes its absence for `false` cannot recover (§7, §8,
		// ADR-0001).
		DryRun:     dryRun,
		SecretSink: sink,
		Trigger:    readTrigger(process.LookupEnv, process.User, process.Hostname),
		Version:    binaryVersion,
		Now:        process.Now,
		Mint:       process.Mint,
		LookupEnv:  process.LookupEnv,
		Environ:    process.Environ,
		Dial:       process.Dial,
		Exec:       process.Exec,
		// The drain, as the engine asks it: *has this Run been stopped
		// by now*. Which signals are watched, what each exits with, when
		// a second one kills the process and what a cancelled call is
		// are all this side's (§6, §9, ADR-0015, ADR-0092).
		Interrupted: stopping(watch.interrupted, to.stopped),
		// Where the Steps go as they happen, which is the surface's and
		// not this command's: two stderr lines on the terminal, a
		// notification at the Step boundary on MCP, and nothing at all
		// where nobody is watching (§9, destination.go).
		Narrator: to.narrator(),
	})

	// The fault is narration's and stdout carries none of it: a failure
	// carries no `error_code`, and what tells two failures apart is the exit
	// code (§9, §12).
	if answer.Fault != nil {
		fmt.Fprintf(to.narrate(), "hyper %s: %s\n", runCommand, answer.Fault)
	}

	// What the signal, if one arrived, exits with. It is read here rather
	// than inside exitFor so that the mapping from an outcome to a code
	// stays a function of facts and not of a live watch (§12, ADR-0026).
	signalled, caught := watch.exit()
	terminal := outcomeRow{
		Type:    "outcome",
		Outcome: string(answer.Outcome),
		Code:    exitFor(answer, signalled, caught),
		DryRun:  dryRun,
	}
	if answer.Identified {
		terminal.RunID = answer.Run.String()
	}
	// The head of the Refusal's array, derived here and stored nowhere: a
	// stored head is a second representation of the array's first member and
	// the two can disagree (§7, §8). Where five checks declined, five rows go
	// out and this names the first.
	if len(answer.Refusal) > 0 {
		terminal.ErrorCode = answer.Refusal[0].ErrorCode
	}

	return rendering.write(runRows(answer, repoRoot), terminal, withheldStep(answer))
}

// withheldStep is the Step a rehearsal stopped at, named as the page names it,
// and "" where the Run withheld none (§9, issue #155).
//
// It asks the Answer rather than taking the first *never reached* row: a Run the
// world resisted leaves those rows too, and a page that inferred a rehearsal
// from them would put the sentence under a Run that failed. `runRows` asks the
// same predicate for the row's `withheld` member, which is what makes the
// sentence and the member two renderings of one fact (run.Answer.Withholds).
func withheldStep(answer run.Answer) string {
	for _, step := range answer.Steps {
		if answer.Withholds(step) {
			return namedStep(step.Path, step.ID)
		}
	}
	return ""
}

// exitFor maps a Run's answer onto the exit code §12 fixes for it. The code
// space is finer than the triple and never coarser, so `failed` is the one
// member that has to read more than the outcome: it takes four codes, and which
// one this Run earned is a fact about what stopped it (§9, §12).
//
// **A push that could not land is 75 and not 1.** Three rejections running is a
// Run that lost the Store, which is where the lock and the sync at Run start
// already are, and none of the three is the world resisting the *work*: what
// the Run did happened, its commits stand on the local branch, and the next Run
// that syncs sends them (§7, ADR-0061, ADR-0076). Everything else a Run failed
// on is 1.
//
// **So is an effectful Run whose `run.json` push did not land**, and it is the
// same sentence one moment earlier: that push *is* the Run's Store sync, so a
// Run that could not complete it lost the Store before it touched the world
// (§7, ADR-0083, run.ErrSyncFailed). The fetch half of the same sync is
// answered one function down, where the mode is in hand and no Run exists yet
// to carry a fault (locateStore).
//
// **The two signals' codes, 130 and 143, are decided where the signal was
// caught** and never derived from an outcome, which is why they arrive here as
// a parameter: a Run that drained is `failed` like any other stop, and what
// tells it from the world resisting is which signal arrived (§12, ADR-0015).
// signalled is that code and caught says one arrived at all — the pair rather
// than a code alone, because a `0` read as *no signal* would put `failed · exit
// 0` on the terminal line.
//
// The engine's own sentinel is read beside it rather than the code alone. Both
// operands are needed: the watch says a signal reached this process and the
// sentinel says the Run stopped because of it, and a Run the world resisted
// while somebody happened to hit Ctrl-C is `1`, which is what stopped it.
func exitFor(answer run.Answer, signalled int, caught bool) int {
	switch answer.Outcome {
	case store.OutcomeCompleted:
		return ExitClean
	case store.OutcomeRefused:
		return ExitRefused
	case store.OutcomeFailed:
		if caught && errors.Is(answer.Fault, run.ErrInterrupted) {
			return signalled
		}
		if errors.Is(answer.Fault, store.ErrPushExhausted) || errors.Is(answer.Fault, run.ErrSyncFailed) {
			return ExitStoreLost
		}
		return ExitProblems
	default:
		return ExitProblems
	}
}

// refusedFlags is the flag `run` refuses by name, and the reason it gives.
//
// It runs before parseArgs for splitInputs' own reason: the three globals are
// every command's and this is one command's, so a parser that knew about both
// is one every other command's signature would have to admit. `--` ends the
// flags, so a positional spelled like a flag is still reachable (§9).
func refusedFlags(args []string) string {
	for _, argument := range args {
		if argument == "--" {
			return ""
		}
		if argument == "--target" || strings.HasPrefix(argument, "--target=") {
			return "takes no --target: a Procedure is fully bound and declares its own Target envelope,\n" +
				"  so a Target named here is either redundant with the artefact or authority arriving after review"
		}
	}
	return ""
}

// splitDryRun takes `--dry-run` off the argument list and answers whether the
// invocation named it, beside what is left for parseArgs.
//
// **The flag is `run`'s and no other command's** (§9). It is spelled here
// rather than in parseArgs for exactly `--secret-out`'s reason: a parser that
// knew about it is one every other command's signature would have to admit, and
// `hyper compact --dry-run` would stop being the unknown flag it is. A `records
// --dry-run` or a `check --dry-run` would have to mean something, and neither
// does (ADR-0015).
//
// It carries no value, so `--dry-run=true` is not taken here and reaches
// parseArgs as the unknown flag it is: a marker is named or it is not, and a
// spelling that looks like it takes a value is a caller expecting one to be
// read. Naming it twice is naming it, the way a boolean flag always is.
//
// `--` ends the flags, so a positional spelled like one is still reachable —
// which is what keeps a Procedure named `--dry-run` runnable, however
// ill-advised the name.
func splitDryRun(args []string) (dryRun bool, rest []string) {
	for i, argument := range args {
		switch {
		case argument == "--":
			return dryRun, append(rest, args[i:]...)
		case argument == "--dry-run":
			dryRun = true
		default:
			rest = append(rest, argument)
		}
	}
	return dryRun, rest
}

// splitSecretOut takes `--secret-out <path>` off the argument list and answers
// the path, what is left for parseArgs, and the one fault decidable from the
// argument list alone.
//
// **`-` is not accepted.** stdout is exclusively the answer, and a secret
// written there lands in the same pipe a CI job logs (§9). It is refused by
// name rather than treated as a filename, because a caller who typed it meant
// the stream and would otherwise get a file called `-` in the working
// directory.
//
// A path named twice is the second one, which is the shell's own rule for a
// value flag and the one every command in the tree already follows for
// `--repo-dir`.
//
// The flag is `run`'s and no other command's: §9 gives it to `run` alone, and
// it is spelled here rather than in parseArgs so that `hyper check
// --secret-out x` stays the unknown flag it is.
//
// **What the path is accepted for is not yet written to.** §9 says the sink
// "is written `0600`", and nothing here or in the engine creates it: what a
// Run puts in a Secret sink has no stated format, and #133 flags an ADR as the
// prerequisite for one. Issue #137 carries that deferral in as many words —
// *the file's contents are out of scope* — so what this milestone owes the
// flag is that it is accepted, that its two faults are refused, and that its
// presence is what the sink gate reads (internal/run/gates.go). The `0600`
// creation lands with the format, and neither before the other: a file created
// empty at Run start would be a sink an operator could read as filled.
func splitSecretOut(args []string) (path string, rest []string, fault string) {
	take := func(value string) string {
		if value == "-" {
			return "--secret-out -: stdout is exclusively the answer, and a secret written there\n" +
				"  lands in the same pipe a CI job logs — name a path outside the repository"
		}
		// A value spelled like a flag is the next flag with the path left
		// out, and taking it would write a secret to a file called
		// `--json`. `--secret-out=--odd` still reaches a path beginning
		// with two hyphens, and so does one past a `--`.
		if value == "" || strings.HasPrefix(value, "--") {
			return "--secret-out requires a path"
		}
		path = value
		return ""
	}

	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--":
			return path, append(rest, args[i:]...), ""
		case argument == "--secret-out":
			i++
			if i >= len(args) {
				return "", nil, "--secret-out requires a path"
			}
			if fault := take(args[i]); fault != "" {
				return "", nil, fault
			}
		case strings.HasPrefix(argument, "--secret-out="):
			if fault := take(strings.TrimPrefix(argument, "--secret-out=")); fault != "" {
				return "", nil, fault
			}
		default:
			rest = append(rest, argument)
		}
	}
	return path, rest, ""
}

// sinkInsideTheWorkingTree is the second half of what `--secret-out` accepts:
// the path is refused where it resolves **inside the repository working tree**
// (§9). It answers the fault where it does, and "" where the invocation named
// no sink or named one outside.
//
// It is named for the state it detects rather than for the state it permits,
// so that its one call site — `if fault := sinkInsideTheWorkingTree(...)` —
// reads as the thing being refused.
//
// The reason is the whole reason the sink exists. A secret written into the
// tree is a secret one `git add .` away from a reviewed artefact and from the
// remote the Store pushes to — and `hyper` is a tool whose entire claim is that
// no artefact ever holds one (§7, ADR-0007). Nothing about the file's mode
// would help: it is where it sits that is the fault.
//
// Both faults `--secret-out` can have are **usage errors carrying no
// error_code**, and deliberately so: an `error_code` names a check that
// declined an artefact, and a path typed at a command line is not one (§9,
// §12, ADR-0060). Withholding the flag Refuses; misspelling it does not.
//
// The comparison is lexical over cleaned absolute paths. It does not resolve
// symlinks, and that is stated rather than hidden: a path that reaches the tree
// through one is not caught here, and what stands against it is the same thing
// that stands against a `--secret-out /dev/stdout` — the operator's own reading
// of where they are writing a secret.
func sinkInsideTheWorkingTree(sink, repoRoot, wd string) string {
	if sink == "" {
		return ""
	}
	resolved, err := filepath.Abs(absPath(wd, sink))
	if err != nil {
		return fmt.Sprintf("--secret-out %s: %s", sink, err)
	}
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return fmt.Sprintf("--secret-out %s: %s", sink, err)
	}
	inside, err := filepath.Rel(root, resolved)
	if err != nil || inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) {
		return ""
	}
	return fmt.Sprintf("--secret-out %s: the path resolves inside the repository working tree\n"+
		"  a secret written there is one `git add` away from the record — name a path outside the repository", sink)
}

// unresolvedProcedure is the message a positional that is not a Procedure
// earns, and "" where the name resolves.
//
// All three are usage errors at `2` carrying no `error_code`: a Refusal is the
// artefacts declining an act and a usage error is there being no act to decline
// (§9, ADR-0060). They are three messages because the remedies differ —
// one caller mistyped a name, one named the wrong kind of artefact, and one
// wrote the positional in the wrong grammar — and each is worth its own
// sentence: every Run is a Run of a Procedure, and what a caller reaching for a
// Definition wants is a Procedure of one Step (ADR-0036).
//
// **The path arm chooses a sentence and never a namespace** (ADR-0090). This
// command's positional has one form, a Procedure's name, so a path-shaped
// positional is a name that resolved to nothing like any other and the shape is
// read *after* the Procedure lookup rather than before it. `review` reads the same shape
// before its lookup because it has two namespaces to pick between and a name
// tried as a path would make which artefact it renders depend on what else is
// in the repository (review.go, ADR-0060); here there is one namespace, so
// reading the shape first would cost the Procedure whose own name happens to
// end `.yaml` and buy nothing. It is isPathForm either way — what a path looks
// like is one fact, and the two commands differ in what they do with one.
func unresolvedProcedure(loaded repository.Loaded, name string) string {
	if loaded.Procedures[name] {
		return ""
	}
	if isPathForm(name) {
		return fmt.Sprintf("hyper %s: %q is a path, and %s takes a Procedure's name\n"+
			"  the name is that file's basename without the .yaml; `review` is the command whose positional takes either form\n",
			runCommand, name, runCommand)
	}
	if _, isDefinition := loaded.Definitions[name]; isDefinition {
		return fmt.Sprintf("hyper %s: %q is a Definition, and every Run is a Run of a Procedure\n"+
			"  what runs a single Operation is a Procedure of one Step: a definition:, an operation:, a target: and its args:\n",
			runCommand, name)
	}
	return fmt.Sprintf("hyper %s: no Procedure named %q in this repository's Procedure namespace\n"+
		"  procedures/ is that namespace, and no command in the tree enumerates it\n", runCommand, name)
}

// locateStore puts the Store in hand, at the rhythm the Run's own Kinds already
// decided, and answers the exit code where it could not.
//
// **A read-only Run attempts the sync and tolerates its failure** (§7,
// ADR-0083). It proceeds against whatever branch the clone holds; it Refuses
// `store-absent` only where no branch is in hand *after* the attempt — which is
// Open's answer and not the sync's; and **it is never 75 for a sync it could
// not complete**, that code being the effectful Run's.
//
// That is a resolution rather than a reading, and the ADR is where it is argued
// out. §7 said only that *a read-only Run proceeds offline and pushes when it
// can*, which read literally is a Run that never reaches the remote — and a
// runner's clone holds no Store until a fetch brings one, so that reading
// Refuses `store-absent` on every scheduled monitoring Run. The section now
// states the resolution, and this comment stands beside the code that performs
// it rather than in place of either.
//
// A read-only Run that tolerated a failure **says so**, on stderr, before its
// first Step, and that half is syncForReading's (inspect.go): a Run that reads
// and an Inspection command that reads tolerate the same failure for the same
// reason, and the only thing the two do not share is which of them the line
// names.
//
// An effectful Run keeps §7's rule exactly. Its sync **is** the push of its own
// open Journal entry — the earliest moment it can know it will be able to
// record what it does — so a sync it could not complete is a Run that lost the
// Store before it touched the world, and it is `failed` at 75 (§7, ADR-0061).
// **This is the fetch half of that sync**, and the push half is the engine's,
// where the entry it sends exists (run.ErrSyncFailed). It lands beside the
// shared arm rather than after it, for the reason both lock modes land together
// — which sync a Run performs is read off the one value that already decided
// which lock it took.
//
// ErrAbsent is the one answer here that is a Refusal, and it is 77 from either
// call: a branch neither side holds is cleared by an act of somebody's, `hyper
// store init`, and never by waiting (§12, ADR-0061).
func locateStore(repoRoot string, instant time.Time, mode store.LockMode, rendering runRendering) (*store.Store, int) {
	if mode == store.Exclusive {
		if err := store.Sync(repoRoot, instant); err != nil {
			return nil, reportRunStoreFault(rendering, err)
		}
	} else {
		syncForReading(runCommand, "this Run", repoRoot, instant, rendering.to.narrate())
	}
	held, err := store.Open(repoRoot, instant)
	if err != nil {
		return nil, reportRunStoreFault(rendering, err)
	}
	return held, 0
}

// runRendering is where a Run's answer goes, and whether the invocation was a
// rehearsal. It is not the answer — that is `run.Answer`, which the engine
// returns — it is where one goes.
//
// It exists because **a Run answers on stdout at every exit on which a Run was
// attempted** (§9, issue #172), and the two paths that decline before the Run
// has an id are the ones furthest from the renderer that knows how — the pin
// gate, which fires before the positional is even resolved, and the bootstrap
// `store-absent`, three calls down inside the Store's own lookup. Threading the
// pair together is what lets each of them write §8's terminal line, or its
// `outcome` row, where each of them used to write nothing at all.
//
// It is two members rather than four because where an answer goes and in which
// form is one value now (destination.go). What is left beside it is the
// rehearsal marker, which is a fact about this invocation rather than about its
// destination.
type runRendering struct {
	to     destination
	dryRun bool
}

// write is a Run's answer in whichever mode was asked for, and the exit code
// that follows it: the rows and §8's terminal row on the wire, the Step table
// and §8's terminal line on the page.
//
// It is one function rather than two call sites because the ordinary path and
// the two declines write the same thing — a stream terminated by an `outcome`
// row, and a page ending in the line that row also carries — and what differs
// between them is only how much stands in front of it (ADR-0026).
//
// withheld is the Step a rehearsal stopped at, "" where it withheld none: a
// page fact no row carries (runPage).
func (r runRendering) write(rows []render.Row, terminal outcomeRow, withheld string) int {
	if fault := writeAnswer(runCommand, r.to, rows, terminal, runPage(terminal, withheld)); fault != 0 {
		return fault
	}
	return terminal.Code
}

// terminate is a Run's whole answer on a path that ended **before a Run was
// identified**: §8's terminal line on the page, the `outcome` row on the wire,
// and no rows in front of either.
//
// **The row is emitted regardless and what is missing is its `run_id`** (§8,
// §9, §10). `run` is on the `outcome` side on every path it takes, the two that
// decline before a Run is identified included — the version pin gate and the
// bootstrap `store-absent` — and a terminal type that flipped to `result`
// according to how early the tool declined would be one fact arriving under two
// contracts. Where no entry was written the id is absent and its absence is the
// fact: the line says what happened and names nothing to look up, there being
// nothing to look up (ADR-0047).
//
// **The 75s do not reach it**, and the rule that keeps them out is the one
// above read backwards: a lock another Run holds and a Store this one could not
// reach are paths on which no Run was attempted at all — both stand before
// `run.json` — so neither declined and neither answers. §9 says `run_id` is
// absent *exactly* where the two declines are, and a third place carrying no id
// would make that sentence false (reportLockFault, reportRunStoreFault).
//
// The narration is the caller's and has already gone to stderr. This is the
// answer beside it, and the two say different halves of one fact on different
// streams, exactly as they do on every other path a Run takes (§9).
func (r runRendering) terminate(outcome store.Outcome, code int, errorCode string) int {
	return r.write(nil, outcomeRow{
		Type:      "outcome",
		Outcome:   string(outcome),
		Code:      code,
		ErrorCode: errorCode,
		DryRun:    r.dryRun,
	}, "")
}

// reportLockFault renders a lock the Run could not take, and answers the exit
// code.
//
// Both arms are 75 and neither is a Refusal. Contention is another Run being
// alive, which nobody has to act on and time alone clears; a repository root
// holding no git repository is the invocation being wrong, and it reaches here
// only because the lock is taken before anything else touches git — it carried
// the same code from the sync before this ticket moved the order, and moving
// the order is not a licence to change what it means.
//
// **stdout is silent, and that is the rule rather than a deferral.** §8 puts
// `run` on the `outcome` side on every path on which a Run was **attempted**,
// the two that decline before a Run is identified included — and the lock is
// neither. Nothing was attempted: the lock is taken before the sync, before
// `run.json` and before Step 1, so no Run began here and none declined, which
// is why 75 carries no `error_code` either. §9 says `run_id` is absent exactly
// where the two declines are, and a terminal row emitted here would be a third
// place (§9, runAnswer.terminate).
func reportLockFault(stderr io.Writer, err error) int {
	fmt.Fprintf(stderr, "hyper %s: %s\n", runCommand, err)
	return ExitStoreLost
}

// reportRunStoreFault renders whichever way the record stopped a Run before it
// began, and answers the exit code.
//
// The two differ by what it would take to clear them, which is exactly what
// sorts 77 from 75 (§12, ADR-0061). A branch neither side holds Refuses
// `store-absent` naming `hyper store init` — the remedy is an act of
// somebody's. Everything else is a Run that **lost** the Store: a remote it
// could not reach, a fetch that would not land, a git that would not answer,
// and past those lies time rather than an act, so each is `failed` at 75 and
// never a Refusal.
//
// It renders both of locateStore's calls, which is why it says the Store could
// not be *reached* rather than that a sync failed: a read-only Run tolerates
// its sync outright and only ever arrives here off Open.
//
// `store-schema-unsupported` is not answered here and could not be: locating
// the Store opens no file, and the test over the files a Run will read is a
// gate of its own, one place further down §6's order. It arrives with that gate
// — inside the engine, into an entry that exists, rendered like every other
// Refusal a Run makes (internal/run/gates.go).
//
// **The two differ on stdout as well, and on the same axis.** §8 puts `run` on
// the `outcome` side "on every path on which a Run was attempted, the two that
// decline before a Run is identified included" — what is missing there is the
// row's `run_id` and never the row — and **the bootstrap `store-absent` is one
// of those two**, the version pin gate being the other (gate.go). So it writes
// §8's terminal line, or its `outcome` row, carrying the code this Refused
// under and no id, there being no entry to name.
//
// The 75 beside it writes nothing to stdout, and it is the same sentence read
// the other way: nothing declined and no Run was attempted, the sync standing
// before `run.json` and long before Step 1. §9 says `run_id` is absent exactly
// where the two declines are, so a terminal row here would be a third place —
// and a `failed` one carrying no `error_code`, a failure having no check to
// name (§12, runRendering.terminate).
func reportRunStoreFault(rendering runRendering, err error) int {
	if errors.Is(err, store.ErrAbsent) {
		refuse(rendering.to, storeAbsentCode, "no "+store.BranchName+" branch in this repository — hyper store init")
		return rendering.terminate(store.OutcomeRefused, ExitRefused, storeAbsentCode)
	}
	fmt.Fprintf(rendering.to.narrate(), "hyper %s: the Store could not be reached: %s\n", runCommand, err)
	return ExitStoreLost
}

// narration is §9's stderr narration: the Run naming itself before its first
// Step, and one line per Step boundary, in both modes, always on.
//
// It carries no machine contract and has no `--json` variant — a consumer
// wanting Step-level structure reads the Journal, which §7 writes per Step as
// that Step reaches its Disposition. With the outcome arriving last, a silent
// twenty minutes is otherwise indistinguishable from a hang.
type narration struct{ stderr io.Writer }

// Began writes the Run's id before its first Step. The id renders here as well
// as on the terminal line, and that repetition is deliberate: the terminal line
// is not always reached — a process killed outright renders nothing and leaves
// the open entry §7 states — and this is the one Run whose identity its own
// output would otherwise never carry (§9, ADR-0047).
func (n narration) Began(id store.RunID) {
	fmt.Fprintf(n.stderr, "run %s\n", id)
}

// Reached writes one Step boundary: which Step, of how many, and its authored
// id.
func (n narration) Reached(position, of int, id string) {
	fmt.Fprintf(n.stderr, "step %d/%d %s\n", position, of, id)
}

// notifications is §9's narration as the MCP surface carries it: one
// `notifications/progress` per Step boundary, at the same boundary §7 writes a
// Journal entry at and §9's stderr line above renders (§9, ADR-0092, issue
// #202).
//
// It is the Narrator's **second** implementation and the engine declares only
// the two events because §9's narration is two lines. What differs here is that
// only one of them is a line at all.
type notifications struct {
	progress mcp.Progress
}

// Began sends nothing.
//
// On the CLI the event exists because the terminal line is not always reached,
// and the Run that dies before it is the one Run whose identity its own output
// would otherwise never carry. Here the id arrives in the summary line and in
// `run_id`, and a client that gives up gets no delivery at all — so a
// notification naming the id would be narration with no reader on the one path
// it was invented for (§9, ADR-0047).
func (notifications) Began(store.RunID) {}

// Reached sends one Step boundary: the position, the total and the Step. It is
// narration, so it carries no machine contract and no row of its own — what
// happened is read off the envelope when the call returns, and this is what
// stands between now and then on a surface with no scrollback (§9).
func (n notifications) Reached(position, of int, id string) {
	n.progress(position, of, id)
}

// stopping is the drain the engine asks, composed from the ways one invocation
// can be stopped: the terminal's signal, watched beside the Run, and the
// client's cancelled call, read off the context the server handed the dispatch
// (§6, §9, ADR-0015, ADR-0092).
//
// **A surface supplies one of the two and never both.** The server installs no
// signal watch, so a Run there is one nobody interrupts by signal; the terminal
// has no call to cancel, so its destination never goes away. What this composes
// is therefore one live source and one that is inert, which is why it is an
// `or` rather than a decision about which surface is running.
//
// It is composed **here** and not in the engine, and that is the whole of what
// keeps the mapping honest: internal/run asks one question at one place — *has
// this Run been stopped by now*, asked where the next Step would start — and a
// second stopping mechanism inside it would be a second place a Run can stop
// (run.Request.Interrupted).
func stopping(interrupted, cancelled func() bool) func() bool {
	return func() bool { return interrupted() || cancelled() }
}

// runColumns is the Step table's header (§8).
var runColumns = []string{"STEP", "ID", "KIND", "DISPOSITION", "RECORDS"}

// noRecords is what the RECORDS column renders where **no set exists**, rather
// than a set with nothing in it. *refused*, *skipped by condition*, *never
// reached* and *attempted, world untouched* conclude about nothing, and the
// dash is what tells them from the *ran* Step whose Expansion resolved to
// nothing, whose set is written empty and whose cell reads `0` (§8, ADR-0030).
const noRecords = "–"

// runRows is the Run's answer as rows: one `step` row per Step that reached a
// Disposition, then the Run's `provenance` in both its scopes — the Run-wide
// row and one per Step file written — and the terminal `outcome` row is
// writeAnswer's (§8, §9).
//
// The order is the page's, and here it is a contract rather than a consequence:
// a row goes out on its own line, there is no cursor behind the stream, and a
// consumer cannot re-sort what it has already printed (§8).
func runRows(answer run.Answer, repoRoot string) []render.Row {
	rows := make([]render.Row, 0, 2*len(answer.Steps)+len(answer.Refusal)+1)
	for _, step := range answer.Steps {
		// The withheld Step's row is written here rather than in
		// stepRowOf because the fact is the Answer's and not the Step's,
		// and it is asked through the predicate the page's sentence is
		// asked through (§8, ADR-0091, run.Answer.Withholds).
		row := stepRowOf(step)
		row.Withheld = answer.Withholds(step)
		rows = append(rows, row)
	}
	// One `refusal` row per problem, in the array's order, and never one row
	// carrying an array: a consumer's `select(.type=="refusal")` returns one
	// problem per line, which it would stop doing the day this became the one
	// stream in the tool that nests (§8).
	//
	// **A member's `remediation` rows ride between it and the next member**,
	// which is the pairing the page draws and the only one the stream can
	// state: there is no cursor behind it and a row cannot point backwards,
	// so a row's place in the order is what says which member it is about
	// (§8, refusal.go).
	for _, refusal := range answer.Refusal {
		member := excerpted(refusalRowOf(refusal.RefusalMember, refusal.Operation, refusal.Target), repoRoot, answer.Started)
		rows = append(rows, member)
		rows = append(rows, remediationsFor(member, refusal.Narrowed, answer.Started)...)
	}
	if answer.Identified {
		rows = append(rows, runProvenanceRow(answer.Provenance))
	}
	for _, step := range answer.Steps {
		// A Step that was **never reached** wrote no file, so there is
		// no Step-scoped Provenance for it to carry: §7 splits
		// Provenance by where each member has exactly one value, and a
		// Step with no file has no Definition revision it read (§7,
		// ADR-0043). It has a `step` row all the same — the Step has a
		// cell — and this is the row it does not have.
		if step.Disposition == store.DispositionNeverReached {
			continue
		}
		rows = append(rows, stepProvenanceRow(step.Position, step.Provenance))
	}
	return rows
}

// refusalRow is one check that declined the Run, on §8's wire: what a `check`
// problem row carries — minus the `column`, which rides on `check`'s stream
// alone and is read back out of no file — plus what a Run adds.
//
// `step` and `step_id` are written where the check cites a Step, and `step` is
// an **artefact coordinate and never an execution fact**: the Step it names may
// have no file in the entry at all, every Refusal in this milestone declining
// before Step 1 (§7, ADR-0061). `operation` and `target` are what that Step was
// bound to, and they are the two members a row carries that its Store
// counterpart does not — the entry holds them on the Step's own file wherever
// one exists, and a Refusal before Step 1 writes none.
//
// `declared` and `observed` are what a check that compared two values wrote,
// and `bound-exceeded` is the one member of the closed set that does: the Bound
// its author declared against the count its Expansion resolved to (§5, §7,
// issue #149). Every other code writes neither, on the absence rule — nothing
// is invented to fill a member that does not apply — so the two are absent from
// most rows this stream ever carries.
//
// They ride as the canonical bytes the entry holds for them rather than as a
// second reading of the same fact: what `outcome.json` writes for a `declared`
// of 5 is `5`, and one renderer over one value is what keeps the wire and the
// file from disagreeing about a number (§7, §8, ADR-0026).
//
// `resolved` maps a relative operand to the instant it resolved to, and it
// rides **where the text this row renders holds one** — the `=` note beneath
// the caret excerpt, and nowhere else (§8, ADR-0034). It is derived from
// `run.json`'s `started_at` and stored nowhere, which is why it rides on the
// rows that render it rather than on the `provenance` row beside them: a Run
// does evaluate a relative predicate, at an Expansion, against one instant, and
// what that instant was is arithmetic a reader would otherwise redo.
type refusalRow struct {
	Type      string            `json:"type"`
	ErrorCode string            `json:"error_code"`
	Step      *int              `json:"step,omitempty"`
	StepID    string            `json:"step_id,omitempty"`
	Operation string            `json:"operation,omitempty"`
	Target    string            `json:"target,omitempty"`
	Declared  json.RawMessage   `json:"declared,omitempty"`
	Observed  json.RawMessage   `json:"observed,omitempty"`
	File      string            `json:"file,omitempty"`
	Line      int               `json:"line,omitempty"`
	Field     string            `json:"field,omitempty"`
	Message   string            `json:"message,omitempty"`
	Resolved  map[string]string `json:"resolved,omitempty"`
	// excerpt is the caret excerpt the page draws for this member, read
	// from the working tree when the row was built, and resolvedOperands
	// the glosses that excerpt earned. Neither is on the wire: the excerpt
	// is a file a consumer already has, and the glosses go out as Resolved
	// above. They are held here so that the block the page draws and the
	// member the stream carries are one reading of one Refusal (ADR-0026).
	excerpt          caretExcerpt
	resolvedOperands []gloss
}

// refusalRowOf is one member of the Refusal's array as a row, with the binding
// of the Step it cites.
//
// It is the shape the array takes wherever it is rendered — as a Run reports it
// and as `hyper show` reads it back — so the two are one constructor over the
// Store's own member and never two that have to agree (ADR-0026). The binding
// is the pair a row carries that its Store counterpart does not: the entry
// holds it on the cited Step's own file wherever one exists, and a Refusal that
// reached no Step has none (§8).
//
// Every member the check did not have is absent rather than written empty,
// which is §7's absence rule holding on the wire: `store-schema-unsupported`
// cites a Store file and no line, and a Refusal that reached no Step carries
// neither `step` nor the binding beside it.
func refusalRowOf(member store.RefusalMember, operation, target string) refusalRow {
	row := refusalRow{
		Type:      "refusal",
		ErrorCode: member.ErrorCode,
		StepID:    member.StepID,
		Operation: operation,
		Target:    target,
		Declared:  wireValue(member.Declared),
		Observed:  wireValue(member.Observed),
		File:      member.File,
		Line:      member.Line,
		Field:     member.Field,
		Message:   member.Message,
	}
	if member.Step != 0 {
		step := member.Step
		row.Step = &step
	}
	return row
}

// wireValue is a Store value on §8's wire, and nothing at all where the check
// wrote none — which is the absence rule arriving here as an omitted key rather
// than as a `null`.
//
// The bytes are the Store's own (store.Unframed), which is the one door between
// §7's encoding and this one: what a row states about a value and what the
// entry states about it are then one reading rather than two that have to agree
// (ADR-0026).
func wireValue(held store.Value) json.RawMessage {
	if held == nil {
		return nil
	}
	return json.RawMessage(store.Unframed(held))
}

// Cells is empty: a Refusal's line on the page is §8's caret excerpt and its
// `=` notes, which writeRefusal draws as a block rather than as a line of a
// table (refusal.go). It is the shape outcomeRow already has for the same
// reason — a row the page renders some other way contributes no line.
func (r refusalRow) Cells() []string { return nil }

// runPage is the Run's page: §8's Step table, and beneath it §8's terminal
// line, which is the answer's last line and goes to stdout like the rest of it
// — which is what puts the Run id on a job summary (§9, §10).
//
// The terminal row is captured rather than found among the rows, because the
// two forms are one fact rendered twice: what the row carries the line carries,
// and a fact the stream states and the page does not is the two surfaces
// disagreeing about what happened rather than differing in shape (ADR-0026).
//
// withheld is captured for the same reason and is the Step a rehearsal stopped
// at, "" where it withheld none. It is a **page** fact and no row carries it:
// the withheld Step's row is the *never reached* row it shares with every Step
// after it, and what §9 puts here is the sentence that says which of them the
// Run stopped at and why (§9, ADR-0010, issue #155).
func runPage(terminal outcomeRow, withheld string) func(io.Writer, []render.Row) error {
	return func(w io.Writer, rows []render.Row) error {
		steps := rowsOf[stepRow](rows)
		refusals := rowsOf[refusalRow](rows)

		blocks := []func(io.Writer) error{}
		switch {
		case len(steps) > 0:
			blocks = append(blocks, func(w io.Writer) error { return render.WriteTable(w, runColumns, steps) })
		case len(refusals) > 0:
			// **The Step table is omitted entirely** where no Step was
			// reached, and this stands in its place. An empty table
			// asserts *we looked at the Steps*, which is false, and
			// the fact that no Step was reached is the most important
			// thing on the page — an absence cannot carry it (§8).
			//
			// It is written on the refused path and on no other. A Run
			// the world resisted before its first Step reached one and
			// was stopped inside it, so *no step was reached* would be
			// a sentence that is not true there.
			blocks = append(blocks, func(w io.Writer) error { _, err := fmt.Fprintln(w, nothingRan); return err })
		}
		if len(refusals) > 0 {
			// §8's Refusal: every member of the array, each with its
			// own caret excerpt and its own remediation table where
			// it has one (refusal.go).
			//
			// Each member carries the phase its own citation earns,
			// which is refusal.go's; a Run is what supplies one at
			// all, and this is a Run.
			blocks = append(blocks, func(w io.Writer) error { return writeRefusal(w, rows, true) })
			// **The sentence beneath names no count.** What it
			// carries is that the Steps before the refused one ran
			// and that nothing rewinds them, which is a different
			// fact from how many Records they touched and the one a
			// reader of this surface needs; the counts are in the
			// column directly above it (§6, §8).
			if sentence := refusedStepSentence(steps); sentence != "" {
				blocks = append(blocks, func(w io.Writer) error { _, err := fmt.Fprintln(w, sentence); return err })
			}
		}
		// **A rehearsal that stopped says so, and names the Step it
		// withheld** — the one the table renders *never reached* first,
		// which without this line is indistinguishable from a Step the
		// Run simply ended before (§9).
		//
		// It stands between the table and the terminal line because it
		// is what the table's silence means: the rows say which Steps
		// have no account, and this says which of them the Run stopped
		// at and why it did.
		if withheld != "" {
			blocks = append(blocks, func(w io.Writer) error { _, err := fmt.Fprintln(w, stopped(withheld)); return err })
		}
		// **The terminal line's pointer is earned by truncation, not by
		// the outcome** (§8). It stands where the page is incomplete —
		// where the caret reported a **count** standing for members it
		// could not fit — and falls back to the completed form
		// everywhere else. Where the page is complete, pointing at
		// `show` would send a reader to look up what they are already
		// looking at.
		pointer := ""
		if len(refusals) > 0 {
			pointer = expansionPointer(terminal.RunID, refusals[0].(refusalRow))
		}
		blocks = append(blocks, func(w io.Writer) error { _, err := fmt.Fprintln(w, terminal.line(pointer)); return err })

		for i, block := range blocks {
			if i > 0 {
				if _, err := fmt.Fprintln(w); err != nil {
					return err
				}
			}
			if err := block(w); err != nil {
				return err
			}
		}
		return nil
	}
}

// nothingRan is what stands where the Step table would be on a Run that
// Refused before Step 1 (§8).
const nothingRan = "nothing ran. no step was reached."

// stopped is what a rehearsal that stopped writes beneath its Step table: that
// it stopped, the Step it withheld, and why there is nothing beneath that Step
// (§9, ADR-0010).
//
// **It describes nothing the Step would have done.** `hyper` has no plan, and a
// line here saying what the `mutate` would have changed would be the prospective
// counterpart the Comparison deliberately is not — arriving on the one surface
// whose whole purpose is to have touched nothing. What it says instead is what
// the tool did: it read up to here, and it stopped.
//
// The Step is named as the table names it — the invocation chain where it was
// reached through a nested Procedure, the authored id where it sits at the top
// level — so the sentence and the row point at one Step rather than at two
// spellings of one (§8).
func stopped(withheld string) string {
	return "stopped at " + withheld + ". a rehearsal performs the reads it reaches and withholds the first effectful step rather than simulating it."
}

// rowsOf narrows a Run's rows to one of the four types on its stream, in the
// order they were built.
//
// The page reads its blocks off the rows rather than off the Answer, which is
// ADR-0026's own rule taken one step further than writeAnswer takes it: the two
// renderings are one list of rows written twice, so a block the page draws and
// a row the stream emits cannot come from two different readings of what
// happened.
func rowsOf[T render.Row](rows []render.Row) []render.Row {
	kept := make([]render.Row, 0, len(rows))
	for _, row := range rows {
		if _, is := row.(T); is {
			kept = append(kept, row)
		}
	}
	return kept
}

// stepRow is one Step of the Run: its position, its authored id, its Kind, its
// Disposition and how many Records it concluded about (§8).
//
// records is a pointer because it is a member the row must be able to *not*
// carry and whose zero value is a value it must be able to state: a *ran* Step
// whose Expansion resolved to nothing concluded about nothing and writes `0`,
// and a *refused* Step has no set at all and writes no key.
//
// expanded is a pointer for a reason of its own, and it is not the one above.
// §8 puts it on the row **where the column renders `n of m`** — a Step that
// stopped short of its Expansion — and there only where it differs from
// `records`, which on such a Step it always does. Where the Step accounted for
// its whole Expansion there is nothing for a second number to say (§8, issue
// #140).
//
// path is the invocation chain a Step reached through a nested Procedure was
// reached under, beside its own id and absent on a top-level Step — the same
// pair the Step file carries (§7, issue #141).
//
// withheld is the Step a rehearsal stopped at, and §7's absence rule is the
// whole of its semantics: written `true` on the one Step and carrying no key
// anywhere else, so the member is itself the discriminator. It has to be one,
// because no Disposition is — the withheld Step's is *never reached*, which is
// what it shares with every Step behind it (§8, ADR-0091, issue #206). It is
// the one member here written from the Answer rather than the Step, and
// `runRows` is what writes it.
type stepRow struct {
	Type        string `json:"type"`
	Step        int    `json:"step"`
	ID          string `json:"id"`
	Path        string `json:"path,omitempty"`
	Kind        string `json:"kind"`
	Disposition string `json:"disposition"`
	Records     *int   `json:"records,omitempty"`
	Expanded    *int   `json:"expanded,omitempty"`
	Withheld    bool   `json:"withheld,omitempty"`
}

// stepRowOf is one Step of the answer as a row. Records is written where the
// Step concluded about anything at all and absent where it concluded about
// nothing, which is the distinction §8's dash renders (ADR-0030).
//
// Expanded rides beside it where the Step **stopped short of its Expansion** —
// a `read` Expansion that drained and then halted (§6) — and nowhere else. The
// engine says which Steps those are by writing the count on them and on no
// other, so what this reads is *did anything stand unaccounted for* rather than
// *do two counts differ*, which is a question the arithmetic alone cannot
// answer (run.Step).
//
// A Disposition carrying no set never carries it either, whatever its Expansion
// resolved to. *attempted, world untouched* is the case §8 names: a `destroy`
// whose Expansion resolved to five and whose first request never left did name
// five Assets and touch none of them, and it renders the dash all the same —
// rendering the safest state in the tool through the column form reserved for
// doubt would invert it (ADR-0062).
func stepRowOf(step run.Step) stepRow {
	row := stepRow{
		Type:        "step",
		Step:        step.Position,
		ID:          step.ID,
		Path:        step.Path,
		Kind:        string(step.Kind),
		Disposition: string(step.Disposition),
	}
	if step.Concluded {
		records := step.Records
		row.Records = &records
		if step.Expanded != 0 {
			expanded := step.Expanded
			row.Expanded = &expanded
		}
	}
	return row
}

// Cells is the Step's line in §8's table: `n` where the set is all the Step
// reached, `n of m` where the Expansion reached `m` and the rest are
// unaccounted for, and the dash where no set exists at all.
//
// The `n of m` cell **names no member**. Which Records those are is
// `expanded_to`'s and nowhere else, and a column that named one would be the
// second place a reader could read it from — and the shorter one, on a Step
// that expanded to five hundred (§8).
func (r stepRow) Cells() []string {
	records := noRecords
	switch {
	case r.Records != nil && r.Expanded != nil:
		records = fmt.Sprintf("%d of %d", *r.Records, *r.Expanded)
	case r.Records != nil:
		records = fmt.Sprintf("%d", *r.Records)
	}
	return []string{fmt.Sprintf("%d", r.Step), r.named(), r.Kind, r.Disposition, records}
}

// named is what the `ID` column renders, which is the rule below over the two
// members this row carries separately.
func (r stepRow) named() string { return namedStep(r.Path, r.ID) }

// namedStep is what a Step is called on the page: the invocation chain where it
// was reached through a nested Procedure, and the authored id where it sits at
// the top level.
//
// **The table renders nested Steps in the order they ran, under their paths**
// (§6, §8). The path ends in that id, so the column reads as one name at both
// depths — `retire.probe` beside `probe` — and the row on the wire carries the
// two separately, as the Step file does.
//
// It is a function over the pair rather than the row's method alone because the
// page names a Step in two places now: this column, and the sentence a rehearsal
// that stopped writes beneath it. One rule, so the two cannot come to spell one
// Step differently (§9, issue #155).
func namedStep(path, id string) string {
	if path != "" {
		return path
	}
	return id
}

// outcomeRow is the terminal row of a Run's stream, and the only stream in the
// tool that ends in one: everything that is not a Run ends in `result` (§8,
// §9).
//
// `dry_run` is written always, `false` included — §7's one exception to the
// absence rule holding on the wire for the reason it holds in the Store: what a
// reader that takes its absence for `false` gets wrong is unrecoverable.
// `run_id` is written whole and is absent exactly where no entry was written.
// `error_code` names the check that declined a `refused` Run and is absent on
// the other two, a failure having no check to name (§12).
type outcomeRow struct {
	Type      string `json:"type"`
	Outcome   string `json:"outcome"`
	Code      int    `json:"code"`
	ErrorCode string `json:"error_code,omitempty"`
	DryRun    bool   `json:"dry_run"`
	RunID     string `json:"run_id,omitempty"`
}

// Cells is empty: the terminal row's line on the page is §8's terminal line,
// which runPage writes beneath the table rather than inside it.
func (r outcomeRow) Cells() []string { return nil }

// line is §8's terminal line: what happened, the exit code §9 fixes, and the
// entry that holds it — with the Run id **whole**, this being the one id an
// operator retypes (ADR-0047).
//
// A rehearsal carries the marker its entry carries. Without it the line a Run
// that reached the world writes and the line a rehearsal writes are the same
// bytes, on the one path where the difference is the whole point.
//
// Where no entry was written the id is absent and its absence is the fact: the
// line says what happened and names nothing to look up, there being nothing to
// look up.
// pointer is the remediation pointer a Refusal's line absorbs **in place of**
// the id beneath it, and "" on every line that names none. A Refusal is the one
// outcome with a next command to name, its rendering being the entire path back
// (ADR-0001); a completed Run names none, and saying what to look at next would
// editorialise on a surface that reports (ADR-0026).
func (r outcomeRow) line(pointer string) string {
	parts := []string{r.Outcome}
	if r.DryRun {
		parts = append(parts, "dry-run")
	}
	parts = append(parts, fmt.Sprintf("exit %d", r.Code))
	switch {
	case pointer != "":
		parts = append(parts, pointer)
	case r.RunID != "":
		parts = append(parts, "run "+r.RunID)
	}
	return strings.Join(parts, " · ")
}

// expansionPointer is where the whole expansion is read back, and "" where the
// page carrying it is already complete.
//
// **What earns it is the count**, and the check that reports one is
// `bound-exceeded` and no other member of the closed set (§7): the caret said
// twenty-three assets and named none of them, and this says where the
// twenty-three are. Every other Refusal names what it found — the two members
// that collided, the Record whose field would not compare, the Step a Journal
// entry already records — so its page is the whole of the fact and has nothing
// to point past itself for (§8).
//
// It is "" where no entry was written, there being nothing to look up. That is
// unreachable from a Refusal at a Step, `run.json` being written at Run start,
// and it is answered rather than assumed: a pointer naming an empty id is a
// command a reader would paste.
func expansionPointer(runID string, first refusalRow) string {
	if runID == "" || len(first.Observed) == 0 {
		return ""
	}
	return "hyper show " + runID + " --expansion   for all " + string(first.Observed)
}

// refusedStepSentence is what stands between the Step table and the terminal
// line on a Run a guardrail declined at a Step: that nothing rewinds what the
// Steps before it did, and that the refused Step wrote nothing (§6, §8).
//
// **It is written from the refused Step's position and from nothing else**,
// which is what §8 fixes its words as — and the words say *ran*, so a Step
// before the refused one that skipped is described by a verb it did not earn.
// That is the sentence's stated cost rather than a fault in this reading: what
// it carries is *nothing rewinds*, which is true of a skipped Step as much as
// of a Step that called, and which Disposition each of them reached is the
// column directly above, per Step and in the vocabulary §12 closes (§6, §8,
// ADR-0026). A sentence that re-derived the table's own cells would be a second
// rendering of them, and the second one can be wrong about the first.
//
// It is "" where the table holds no refused Step, which is every Refusal that
// declined before Step 1 — there the table is omitted entirely and *nothing
// ran. no step was reached.* stands in its place, already saying the whole of
// it (runPage above).
func refusedStepSentence(steps []render.Row) string {
	position := 0
	for _, row := range steps {
		if step, is := row.(stepRow); is && step.Disposition == string(store.DispositionRefused) {
			position = step.Step
			break
		}
	}
	switch {
	case position == 0:
		return ""
	case position == 1:
		return "step 1 wrote nothing."
	case position == 2:
		return "step 1 ran and what it did stands. step 2 wrote nothing."
	default:
		return fmt.Sprintf("steps 1-%d ran and what they did stands. step %d wrote nothing.", position-1, position)
	}
}
