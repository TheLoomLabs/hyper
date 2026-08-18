package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/repository"
)

// commandArgs is one command's arguments, already read: the three globals §9
// closes configuration at, the --limit the listing commands carry, and whatever
// positionals were left over.
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
	positional []string
}

// takesNoLimit is what a command passes where it has no --limit at all, and it
// is the absence of the flag rather than a default of zero: a cap of zero names
// no result set. §9 gives --limit to the listing commands, and `check` is not
// one of them — it reports every problem it found, and a truncated list of what
// is wrong with a repository is a repository that looks less wrong than it is.
const takesNoLimit = 0

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
// --repo-dir (also spelled --repo-dir=), --no-color — plus --limit where
// the command takes one, and everything else a positional. Anything else
// beginning "--" is a usage error, and so is a --limit that is not a positive
// integer: a limit of none is the flag left off, and a limit of zero is a
// question with no answer in it.
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
func parseArgs(command string, args []string, defaultLimit int, lookupenv func(string) (string, bool), stderr io.Writer) (commandArgs, int) {
	noColor, _ := lookupenv("NO_COLOR")
	parsed := commandArgs{limit: defaultLimit, noColor: noColor != ""}
	takesLimit := defaultLimit != takesNoLimit
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
