package cli_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// renderedGroup is one group as the page carries it — the title, and the
// commands under it with their positionals. It is the shape §9's table is
// transcribed in below and the shape the page is read back in, one type rather
// than two so that the transcription and the reading cannot drift into
// describing different things.
type renderedGroup struct {
	title    string
	commands []string
}

// sectionNineSignatures is §9's table transcribed from the specification
// rather than read from the package — the six groups in the spec's own order,
// each command as the table writes it, positional and all, and the three
// commands outside the tree as the seventh (issue #210).
//
// It is a second transcription beside tree_test.go's and not a duplicate of
// it: that one is the sixteen names the completion scripts offer, and this is
// what §9 says a caller types. The names being one list is exactly what the
// second case below asserts, and it can only assert it from two sources.
var sectionNineSignatures = []renderedGroup{
	{"Discovery", []string{"providers", "provider <name>", "operation <provider> <operation>"}},
	{"The repository", []string{"targets"}},
	{"Authoring", []string{"check [path...]", "review <artefact>"}},
	{"Execution", []string{"run <procedure>", "probe <provider> <operation>"}},
	{"Inspection", []string{"runs", "show <run-id>", "changes [procedure]", "records"}},
	{"Lifecycle", []string{"install <ref>", "project", "store init", "compact"}},
	{"Outside the tree", []string{"version", "completions <shell>", "mcp"}},
}

// sectionNineGlobals is §9's three configuration flags, transcribed from "The
// three configuration layers" rather than read from the package: *there are
// three globals and no more: `--json`, `--repo-dir` (`HYPER_REPO_DIR`), and
// `--no-color` (honouring `NO_COLOR`)*. Only the spelling a caller types is
// carried, that being what the page renders and what a completion offers
// (issue #210).
var sectionNineGlobals = []string{"--json", "--repo-dir <dir>", "--no-color"}

// TestUsage_TheBareInvocationWritesTheTree is the whole of issue #210: a binary
// asked what it is answers with §9's tree rather than with the word
// `<command>`.
//
// What it holds is the page against the specification's table, group by group
// and line by line, in the spec's own order. The observed failure was an agent
// that tried three ways to be told what the sixteen commands are, learned
// nothing from any of them, and went reading repositories outside the one it
// was standing in — so a page that named the groups and dropped a command
// would fail here for the same reason a page that names nothing does.
func TestUsage_TheBareInvocationWritesTheTree(t *testing.T) {
	page := bareInvocation(t)

	if got, want := firstLine(page), "usage: hyper <command> [args...]"; got != want {
		t.Errorf("the first line is %q, want %q", got, want)
	}

	rendered := usageGroups(page)
	if len(rendered) != len(sectionNineSignatures) {
		t.Fatalf("the page carries %d groups, want §9's %d:\n%s", len(rendered), len(sectionNineSignatures), page)
	}
	for i, want := range sectionNineSignatures {
		if got := rendered[i].title; got != want.title {
			t.Errorf("group %d is %q, want §9's %q", i+1, got, want.title)
		}
		if got := rendered[i].commands; !slices.Equal(got, want.commands) {
			t.Errorf("group %q is\n  %q,\nwant §9's\n  %q", want.title, got, want.commands)
		}
	}
}

// TestUsage_NamesTheThreeGlobals is the flags block: §9 closes configuration at
// three globals and no more, and a page that named the sixteen and left a
// caller to guess how to ask for JSON has ended one round of foraging and
// started another (§9, issue #210).
//
// **They are rendered apart from the commands, and the page says why**: the
// three govern the sixteen and none of the three outside the tree, which is the
// same distinction that stops a completion script offering `--json` after
// `version` (tree.go).
func TestUsage_NamesTheThreeGlobals(t *testing.T) {
	page := bareInvocation(t)

	flags := usageFlags(page)
	if !slices.Equal(flags.commands, sectionNineGlobals) {
		t.Errorf("the page's flags are\n  %q,\nwant §9's three\n  %q", flags.commands, sectionNineGlobals)
	}
	if !strings.Contains(flags.title, "sixteen") {
		t.Errorf("the flags are titled %q; the three govern the sixteen and none of the three outside the tree", flags.title)
	}

	// The same fence the commands have: the spellings the page renders are
	// the spellings a completion offers, read off the page rather than
	// transcribed a second time into the package (§9, issue #104).
	var spelled []string
	for _, flag := range flags.commands {
		spelled = append(spelled, strings.Fields(flag)[0])
	}
	if want := cli.Globals(); !slices.Equal(spelled, want) {
		t.Errorf("the page spells the flags\n  %q,\nwant the spellings a completion offers\n  %q", spelled, want)
	}
}

// TestUsage_AnUnknownCommandNamesWhereTheListIs is the path both observed
// agents actually took. `hyper help` and `hyper --help` are correctly unknown —
// neither is among §9's sixteen — but a message that says only *unknown* leaves
// a caller exactly where the empty usage line did, which is reading whatever is
// nearby (issue #210).
//
// What it writes is §9's own rule for a positional that matches nothing,
// applied to the command name: **the name that was typed, the namespace it was
// resolved against, and the command that enumerates that namespace**. It
// suggests no near miss, for the reason nothing here ever does — a suggestion
// is a name resolved on the caller's behalf (§9, ADR-0047).
//
// It stays a usage error at exit 2, and no command is added: `help` still names
// nothing, and being told where the list is is not being given a `help`
// command.
func TestUsage_AnUnknownCommandNamesWhereTheListIs(t *testing.T) {
	for _, typed := range []string{"help", "--help", "bogus"} {
		t.Run(typed, func(t *testing.T) {
			p := &process{wd: t.TempDir()}
			var stdout, stderr bytes.Buffer

			if exit := cli.Main([]string{typed}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitUsage {
				t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want it untouched", stdout.String())
			}
			p.untouched(t)

			got := stderr.String()
			if !strings.Contains(got, `"`+typed+`"`) {
				t.Errorf("stderr = %q, want it to name what was typed", got)
			}
			if !strings.Contains(got, "hyper with no arguments") {
				t.Errorf("stderr = %q, want it to name the command that enumerates the namespace", got)
			}
			// The pointer and not the page: the tree is what a bare
			// `hyper` writes, and printing it after every typo would
			// put twenty-eight lines in front of a caller who missed
			// a keystroke.
			if len(strings.Split(strings.TrimSuffix(got, "\n"), "\n")) != 2 {
				t.Errorf("stderr is\n%s\nwant two lines: the fault, and where the list is", got)
			}
		})
	}
}

// TestUsage_NamesTheSurfaceTheCompletionsDo is the fence against a second list,
// and it holds by construction today: `usage()` renders tree.go's groups and
// cli.Commands() flattens the same ones, so there is one list and this case
// cannot fail as the code stands.
//
// That is the point of it rather than an argument against it. `completions`
// already enumerates the nineteen, and the reason the surface is one
// compiled-in list is drift — three shells describing one surface from three
// transcriptions of it would disagree the day the seventeenth command lands
// (tree.go, issue #104). A usage page assembled from a fourth transcription
// would be that same defect with one more reader, and this is the case that
// stops being vacuous the moment somebody writes one.
func TestUsage_NamesTheSurfaceTheCompletionsDo(t *testing.T) {
	var named []string
	for _, g := range usageGroups(bareInvocation(t)) {
		for _, command := range g.commands {
			named = append(named, strings.Fields(command)[0])
		}
	}
	if want := cli.Commands(); !slices.Equal(named, want) {
		t.Errorf("the usage page names\n  %q,\nwant the surface the completions offer\n  %q", named, want)
	}
}

// TestUsage_StoreNamesTheSubVerbACompletionOffers holds the tree's one piece of
// nesting to itself. `store init` is written on §9's table as a positional and
// offered by the completion scripts as a namespace at position two, and those
// are one fact about one command stated for two readers — so a sub-verb added
// to `store` and left off its line would leave the page teaching a grammar the
// shell does not complete (§9, issue #126).
func TestUsage_StoreNamesTheSubVerbACompletionOffers(t *testing.T) {
	page := bareInvocation(t)

	var line string
	for _, g := range usageGroups(page) {
		for _, command := range g.commands {
			if strings.Fields(command)[0] == "store" {
				line = command
			}
		}
	}
	if line == "" {
		t.Fatalf("the usage page names no `store`:\n%s", page)
	}
	for _, verb := range cli.StoreSubVerbs() {
		if !slices.Contains(strings.Fields(line)[1:], verb) {
			t.Errorf("the page writes %q and a completion offers %q after `store`", line, verb)
		}
	}
}

// bareInvocation drives a bare `hyper` and hands back what it wrote on stderr,
// holding the invocation's own criteria on the way past: it is still a usage
// error, stdout is still untouched, and no part of the process is read.
//
// Those three belong here rather than in one case because they are true of the
// invocation and not of any one thing the page says — the tree is narration
// like every other rendering of an error, and a page that resolved a working
// directory or read the environment to render itself would be a command by
// another name (§9, ADR-0020). Holding them once leaves each case above about
// the page alone.
func bareInvocation(t *testing.T) string {
	t.Helper()

	p := &process{wd: t.TempDir()}
	var stdout, stderr bytes.Buffer

	if exit := cli.Main(nil, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitUsage {
		t.Errorf("exit = %d, want %d", exit, cli.ExitUsage)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it untouched; the tree is narration and stdout is the answer", stdout.String())
	}
	p.untouched(t)

	return stderr.String()
}

// usageGroups reads the page's command groups back: a line indented two spaces
// is a group's title and a line indented four is one command in it, which is
// the whole of the page's structure.
//
// It reads the first block alone. The page is blocks separated by blank lines —
// the usage line, the groups, then the flags — so the blank line after the last
// group is what ends the commands, and the flags below it are read by
// usageFlags rather than mistaken for a seventh group of commands.
func usageGroups(page string) []renderedGroup {
	return blocks(page)[1]
}

// usageFlags is the page's last block, which is the three globals under a title
// of their own.
func usageFlags(page string) renderedGroup {
	last := blocks(page)
	return last[len(last)-1][0]
}

// blocks reads the whole page as its blank-line-separated blocks, each parsed
// into the titled groups it holds. The usage line is block zero and holds none,
// which is what makes the groups block index one however the page grows.
func blocks(page string) [][]renderedGroup {
	parsed := [][]renderedGroup{{}}
	for _, line := range strings.Split(strings.TrimSuffix(page, "\n"), "\n") {
		at := &parsed[len(parsed)-1]
		switch {
		case strings.TrimSpace(line) == "":
			parsed = append(parsed, []renderedGroup{})
		case strings.HasPrefix(line, "    "):
			if len(*at) == 0 {
				continue
			}
			group := &(*at)[len(*at)-1]
			group.commands = append(group.commands, strings.TrimSpace(line))
		case strings.HasPrefix(line, "  "):
			*at = append(*at, renderedGroup{title: strings.TrimSpace(line)})
		}
	}
	return parsed
}

// firstLine is the page's opening line, which is the one line the message had
// before this issue and the one line it must not lose.
func firstLine(page string) string {
	line, _, _ := strings.Cut(page, "\n")
	return line
}
