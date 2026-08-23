package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/TheLoomLabs/hyper/internal/repository"
)

// commandArgs is one command's arguments, already read: the three globals §9
// closes configuration at, the --limit the listing commands carry, the --since
// the Inspection commands bound a window with, and whatever positionals were
// left over.
//
// One shape for every command rather than one per command, because the globals
// are the same three everywhere and a second copy of their parsing is where the
// day comes that one command spells --repo-dir= and another does not (§9,
// ADR-0014).
type commandArgs struct {
	json bool
	// noColor is resolved and has nothing to affect yet: no page any command
	// writes carries colour of its own to suppress, which is why the flag and
	// the variable already produce identical bytes. It is resolved anyway
	// rather than left for the milestone that colours something, so the
	// precedence is stated once beside the other two globals.
	noColor bool
	repoDir string
	// limit is the row cap in force: the value --limit named, or the
	// command's own default where it named none. It is 0 on a command that
	// takes no --limit at all.
	limit int
	// limitNamed says the cap came from the argument rather than from the
	// default, which is the difference between the two truncation lines a
	// command writes: a caller who typed --limit is told what their own
	// number cut, and one who typed nothing is told there is a default and
	// what would widen it.
	limitNamed bool
	// since is the lower bound --since named, in UTC. An offset names an
	// instant and the record spells one way (§7), so what is kept is the
	// instant and not the spelling it arrived in.
	since time.Time
	// sinceNamed says a window was asked for at all, and it is a second
	// member rather than a zero instant read as *no bound* for the reason
	// limitNamed is one: a limit left off has a default and a window left
	// off has none. The zero instant is a point in the year 1, and a
	// command reading it as a bound would be filtering rather than not
	// filtering.
	sinceNamed bool
	positional []string
}

// takesNoLimit is what a command passes where it has no --limit at all, and it
// is the absence of the flag rather than a default of zero: a cap of zero names
// no result set. §9 gives --limit to the listing commands, and `check` is not
// one of them — it reports every problem it found, and a truncated list of what
// is wrong with a repository is a repository that looks less wrong than it is.
const takesNoLimit = 0

// parameters is what a command takes past the three globals — §9's own word for
// them, the Inspection commands taking "typed, closed parameters and nothing
// else". It is one value rather than one argument each for the reason
// commandArgs is one shape rather than one per command: milestone 8 alone adds
// --since, --between, --procedure, --target, --outcome, --kind, --name,
// --definition and --history across four commands, and a parser that grew a
// positional flag per parameter is a signature every one of those tickets edits
// and a call site every command that takes none of them edits with it.
//
// It is the command's argument surface stated where the command is dispatched,
// so what a command accepts is read off its own line rather than out of the
// guards on each case of the loop below.
type parameters struct {
	// limit is the command's own default cap, or takesNoLimit where it has
	// no --limit at all. The default is the flag's value where the caller
	// names none, so one member carries both facts.
	limit int
	// since says the command takes --since. §9 gives it to three of the
	// four Inspection commands — `runs`, `records` and `changes` — and to
	// nothing else: a listing over a namespace has no time axis for a
	// window to cut, so offering it there would be a parameter that narrows
	// nothing.
	since bool
}

// defaultListLimit is the modest default §9 leaves to the implementation, and
// it is one number for every command that enumerates a namespace rather than
// one per command: a caller who has learnt what an unnamed --limit does on
// `providers` has learnt what it does on `targets`, and two constants holding
// one number is where the day comes that they stop holding it.
//
// It is deliberately not a fact any artefact, Record or check depends on:
// nothing reads it back, and a repository whose row count crosses it gets a
// truncation marker and a stderr line rather than a different answer.
const defaultListLimit = 50

// parseArgs reads one command's arguments: the three globals — --json,
// --repo-dir (also spelled --repo-dir=), --no-color — plus --limit and --since
// where the command takes them, and everything else a positional. Anything else
// beginning "--" is a usage error, and so is a --limit that is not a positive
// integer: a limit of none is the flag left off, and a limit of zero is a
// question with no answer in it. A --since that is not an RFC 3339 timestamp is
// readSince's own refusal below.
//
// Which of the two valued flags a command takes is named at its call site, in
// the parameters value above, so a command's argument surface is legible where
// the command is dispatched rather than by reading this loop for the guards on
// each case.
//
// "--" ends the flags, so a positional beginning with a hyphen is reachable.
// command names the command in every message, which is the whole reason it is a
// parameter: one parser, and a caller still reads `hyper providers: unknown
// flag --sicne`.
//
// lookupenv is here for --no-color's environment spelling, NO_COLOR, which any
// non-empty value sets: flags → environment → defaults is the precedence, so
// the flag alone can turn it on and the variable cannot turn it off (§9,
// ADR-0014). --repo-dir's spelling is resolved by resolveRepoRoot below, where
// the value it produces is a path and not a presentation flag.
//
// It answers a value *and whether the variable is set at all*, which is
// os.LookupEnv's shape rather than os.Getenv's, because `targets` reports
// whether the variable a credential slot names is present and an empty string
// is present (issue #112). Both globals here read the value and neither reads
// the presence — but one environment reader is threaded through the whole
// dispatch, so no two commands can disagree about what reading the environment
// means, exactly as one repository.Load is what reading a repository means.
func parseArgs(command string, args []string, takes parameters, lookupenv func(string) (string, bool), stderr io.Writer) (commandArgs, int) {
	noColor, _ := lookupenv("NO_COLOR")
	parsed := commandArgs{limit: takes.limit, noColor: noColor != ""}
	takesLimit := takes.limit != takesNoLimit
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			parsed.positional = append(parsed.positional, args[i+1:]...)
			i = len(args)
		case a == "--json":
			parsed.json = true
		case a == "--no-color":
			parsed.noColor = true
		case a == "--repo-dir":
			i++
			if i >= len(args) {
				fmt.Fprintf(stderr, "hyper %s: --repo-dir requires a value\n", command)
				return parsed, ExitUsage
			}
			parsed.repoDir = args[i]
		case strings.HasPrefix(a, "--repo-dir="):
			parsed.repoDir = strings.TrimPrefix(a, "--repo-dir=")
		case a == "--limit" && takesLimit:
			i++
			if i >= len(args) {
				fmt.Fprintf(stderr, "hyper %s: --limit requires a value\n", command)
				return parsed, ExitUsage
			}
			if code := parsed.readLimit(command, args[i], stderr); code != 0 {
				return parsed, code
			}
		case strings.HasPrefix(a, "--limit=") && takesLimit:
			if code := parsed.readLimit(command, strings.TrimPrefix(a, "--limit="), stderr); code != 0 {
				return parsed, code
			}
		case a == "--since" && takes.since:
			i++
			if i >= len(args) {
				fmt.Fprintf(stderr, "hyper %s: --since requires a value\n", command)
				return parsed, ExitUsage
			}
			if code := parsed.readSince(command, args[i], stderr); code != 0 {
				return parsed, code
			}
		case strings.HasPrefix(a, "--since=") && takes.since:
			if code := parsed.readSince(command, strings.TrimPrefix(a, "--since="), stderr); code != 0 {
				return parsed, code
			}
		case strings.HasPrefix(a, "--"):
			fmt.Fprintf(stderr, "hyper %s: unknown flag %s\n", command, a)
			return parsed, ExitUsage
		default:
			parsed.positional = append(parsed.positional, a)
		}
	}
	return parsed, 0
}

// readLimit reads --limit's value. A limit is a count of rows, so the only
// values it takes are the positive integers: zero and below name no result set
// at all, and a caller who wrote one meant something the flag cannot say.
func (c *commandArgs) readLimit(command, value string, stderr io.Writer) int {
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 {
		fmt.Fprintf(stderr, "hyper %s: --limit %s: want a positive integer\n", command, value)
		return ExitUsage
	}
	c.limit = limit
	c.limitNamed = true
	return 0
}

// readSince reads --since's value: an RFC 3339 timestamp, and one place where
// that is decided (§9, issue #162). Three of §9's Inspection commands take a
// window's lower bound, and three readings of one flag are three chances to
// disagree about what a caller typed.
//
// It is the spelling and nothing looser. There is no relative form, no bare
// date and no duration: *yesterday* and *7d* are questions about a clock the
// caller is not holding — the Store's instants are written by two environments
// — and a bare date is a point only once somebody decides which second of it.
// A caller who wants the start of a day says so to the second.
//
// An offset is read as the instant it names and kept in UTC. RFC 3339 admits
// one, comparing instants is offset-agnostic, and §7's *UTC, Z mandatory* is
// how the record writes a timestamp down rather than how a command line has to
// spell one. What Go's parser does refuse is the lowercase t and z RFC 3339
// tolerates, and that is left refused: every timestamp hyper renders is
// uppercase, so the value a caller pastes back out of a page is one this reads.
//
// The refusal is a usage error carrying **no error_code**. An error_code names
// a check that declined an artefact, and a value typed at a command line is not
// one (§9, §12, ADR-0060) — the same reading `probe`'s inputs already take.
// The message names the flag, what it wanted, and an instant in the shape it
// wanted it, because a caller who mistyped a timestamp is one example away from
// the right one.
func (c *commandArgs) readSince(command, value string, stderr io.Writer) int {
	instant, err := time.Parse(time.RFC3339, value)
	if err != nil {
		fmt.Fprintf(stderr, "hyper %s: --since %s: want an RFC 3339 timestamp, like 2026-08-04T09:12:00Z\n", command, value)
		return ExitUsage
	}
	c.since = instant.UTC()
	c.sinceNamed = true
	return 0
}

// resolveRepoRoot applies flags → environment → defaults (ADR-0014):
// --repo-dir, then HYPER_REPO_DIR, then walking up from wd bounded by the
// git root. command names the command in its two messages, both of which are
// usage errors: there is no repository to refuse on behalf of.
func resolveRepoRoot(command, repoDirFlag string, lookupenv func(string) (string, bool), wd string, stderr io.Writer) (string, int) {
	repoDir := repoDirFlag
	if repoDir == "" {
		repoDir, _ = lookupenv("HYPER_REPO_DIR")
	}
	if repoDir != "" {
		resolved := absPath(wd, repoDir)
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(stderr, "hyper %s: --repo-dir %s: not a directory\n", command, repoDir)
			return "", ExitUsage
		}
		return resolved, 0
	}

	root, ok := repository.FindGitRoot(wd)
	if !ok {
		fmt.Fprintf(stderr, "hyper %s: not inside a git repository; pass --repo-dir or set HYPER_REPO_DIR\n", command)
		return "", ExitUsage
	}
	return root, 0
}

// arityFault says which of the two arity faults happened where a command takes
// a fixed number of positionals. Both are usage errors decided from the
// argument list alone, and the difference is worth a clause because the
// remedies differ: one caller forgot an argument and the other named a thing
// twice or slipped a flag past it (ADR-0060).
//
// nouns are what the command takes one of each of, in positional order and
// spelled as its own messages spell them — a shell for `completions`, a
// Provider for `provider`, a Provider and an Operation for `operation`. They
// are parameters for the reason parseArgs's own command is one: the fault is
// spelled in a single place, and a caller still reads a message in their own
// command's words.
func arityFault(args []string, nouns ...string) string {
	if len(args) == 0 {
		return "names no " + strings.Join(nouns, " and no ")
	}
	wanted := make([]string, len(nouns))
	for i, noun := range nouns {
		wanted[i] = "one " + noun
	}
	return fmt.Sprintf("takes %s, got %d", strings.Join(wanted, " and "), len(args))
}
