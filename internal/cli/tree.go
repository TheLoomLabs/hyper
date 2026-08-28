package cli

import "slices"

// The command surface, in one place. §9 fixes a tree of sixteen commands —
// flat, one noun group, no aliases and no hidden commands — beside three more
// that stand outside it because none of them reads a repository and none says
// anything about hyper's domain. That surface is stated here once, in the
// groups and the order §9's table states it, so a command added by a later
// milestone reaches every reader of it by one edit (issue #104).
//
// The first reader is the completion scripts, and it is the reason the list
// exists at all: three shells describing one surface from three copies of it
// would drift the day the seventeenth command lands. The second is the usage
// page a bare `hyper` writes, which is why the table's groups and positionals
// are here beside the names rather than transcribed a fourth time (usage.go,
// issue #210). Nothing here is a dispatch table — Main's switch is what runs
// a command, and a name here is a name the spec fixes, not a claim that the
// binary implements it yet.
//
// The lists are unexported and reached through the functions below, each
// returning a copy. Go has no constant slice, and an exported one is a
// package-level variable any caller can sort, truncate or clobber for every
// other caller; a surface the specification fixes should not be editable by
// the code that reads it.

// entry is one command in the surface: the name a caller types, and the
// positional signature §9's table states beside it — `<name>` on `provider`,
// `[path...]` on `check`, and the empty string on the seven that take none.
//
// The signature rides beside the name rather than in a list of its own because
// the two are one fact about a command, and the page that carries the surface
// to a caller wants both: the bare names are what end the foraging, and the
// positionals are what stop the next guess being wrong (usage.go, issue #210).
type entry struct {
	name       string
	positional string
}

// signature is the entry as §9's table writes it, which is what a caller reads
// and types back.
func (e entry) signature() string {
	if e.positional == "" {
		return e.name
	}
	return e.name + " " + e.positional
}

// group is one row of §9's table: the name the specification teaches a run of
// commands under, and those commands in the order it states them.
//
// The groups are how §9 teaches the tree, and they are carried rather than
// flattened away at the source because the surface has one shape and this is
// it. Every reader wanting the flat list is handed one derived from these,
// which is what keeps the usage page and the three completion scripts
// describing one surface instead of two copies of it (issue #210).
type group struct {
	title    string
	commands []entry
}

// treeGroups is §9's tree of sixteen, in the spec's own group order and
// carrying the spec's own positionals. A name is added here when the spec
// fixes it, not when the milestone that builds it lands: the completion script
// describes the surface the spec states, and `hyper run deploy` answering
// `unknown command` until milestone 5 is the accepted, stated cost of that
// (issue #104).
var treeGroups = []group{
	{"Discovery", []entry{
		{name: "providers"},
		{name: "provider", positional: "<name>"},
		{name: "operation", positional: "<provider> <operation>"},
	}},
	{"The repository", []entry{
		{name: "targets"},
	}},
	{"Authoring", []entry{
		{name: "check", positional: "[path...]"},
		{name: "review", positional: "<artefact>"},
	}},
	{"Execution", []entry{
		{name: "run", positional: "<procedure>"},
		{name: "probe", positional: "<provider> <operation>"},
	}},
	{"Inspection", []entry{
		{name: "runs"},
		{name: "show", positional: "<run-id>"},
		{name: "changes", positional: "[procedure]"},
		{name: "records"},
	}},
	{"Lifecycle", []entry{
		{name: "install", positional: "<ref>"},
		{name: "project"},
		// `store`'s positional is its one sub-verb rather than a name
		// the caller supplies, which is the whole of the tree's
		// nesting written where the tree is written. storeSubVerbs
		// below is that same fact as the namespace a completion offers
		// at position two, and a case holds the two to each other (§9,
		// issue #126, usage_test.go).
		{name: "store", positional: "init"},
		{name: "compact"},
	}},
}

// tree is those sixteen as names alone, flattened out of the groups above. It
// is what every reader that renders no page wants — the three completion
// scripts, and the membership questions asked of the surface — and it is
// derived rather than transcribed a second time so that a command §9 adds to
// its table reaches all of them by one edit (issue #104, issue #210).
var tree = names(treeGroups...)

// outsideGroup is the three commands §9 states are not among the sixteen:
// `version` prints the version of the binary that would act, `completions
// <shell>` writes a shell completion script, and `mcp` starts the MCP server
// §9's second half states. The first two are exempt from the version pin gate
// for reading no repository — `project` is exempt too and is inside the tree,
// being the pin's only writer, which is why this list is *what the globals
// govern* rather than *what the gate skips* — and the distinction is
// load-bearing here rather than decorative: the three globals govern the
// sixteen, so a completion script may not offer `--json` after any of these
// (§9, §11, ADR-0020).
//
// `mcp` is here rather than in the tree because it passes the same test those
// two pass and fails the tree's own: it reads no repository — the pin gate
// fires per tool, at the moment a tool resolves one, so the process compares
// nothing at startup — and `mcp` is the protocol's name rather than a word
// CONTEXT.md defines, where every name in the tree is one. It is a third
// command outside the tree and not a seventeenth command in it — not a third
// exemption from anything, the gate's three being ADR-0020's and unmoved,
// which is the distinction the paragraph above exists to carry (§9, ADR-0088,
// issue #193).
var outsideGroup = group{"Outside the tree", []entry{
	{name: "version"},
	{name: "completions", positional: "<shell>"},
	{name: "mcp"},
}}

// outsideTree is those three as names alone, on the same footing as `tree`
// above and for the same reason.
var outsideTree = names(outsideGroup)

// surface is the whole of §9's command line in the shape the section teaches
// it: the tree's six groups, and then the three that stand outside it as a
// seventh. The usage page renders it and Commands flattens it, which is what
// makes the page and the completion scripts two readings of one list rather
// than two lists (usage.go, issue #210).
var surface = append(slices.Clone(treeGroups), outsideGroup)

// storeSubVerbs is `store init`'s one sub-verb, and the whole of the tree's
// nesting: §9 admits one noun group and no other. It is a list rather than a
// constant because the shape of the thing is *a command's sub-verbs*, and a
// single member is a fact about `store` today rather than a rule about the
// grammar.
var storeSubVerbs = []string{
	"init",
}

// globals is the three configuration flags §9 closes at three and no more:
// --json, --repo-dir (HYPER_REPO_DIR) and --no-color (honouring NO_COLOR).
// They are named here in the spelling a caller types, which is what a
// completion offers; internal/cli's own parsing of them is check.go's.
var globals = []string{
	"--json",
	"--repo-dir",
	"--no-color",
}

// shells is the closed set `hyper completions` writes a script for, in the
// order its usage error names them, which is alphabetical. It belongs beside
// the rest of the surface rather than in completions.go: it is the namespace
// one of the nineteen resolves its positional against, and all three scripts
// interpolate it. A fourth shell is a fourth compiled-in script and a fourth
// member here, and nothing else.
var shells = []string{"bash", "fish", "zsh"}

// names is the commands of a run of groups, flattened in the order those
// groups state them. It is the one place the surface goes from the shape §9
// teaches the tree in to the flat list its other readers ask for — the three
// completion scripts, and every membership question asked of the surface
// (§9, issue #210).
//
// It is variadic so that the one group outside the tree is named as one group
// rather than wrapped in a slice to be unwrapped again.
func names(groups ...group) []string {
	var flat []string
	for _, g := range groups {
		for _, command := range g.commands {
			flat = append(flat, command.name)
		}
	}
	return flat
}

// Tree is §9's sixteen.
func Tree() []string { return slices.Clone(tree) }

// OutsideTree is the three commands that stand outside the sixteen.
func OutsideTree() []string { return slices.Clone(outsideTree) }

// StoreSubVerbs is `store`'s sub-verbs, the tree's only nesting.
func StoreSubVerbs() []string { return slices.Clone(storeSubVerbs) }

// Globals is the three configuration flags, in the spelling a caller types.
func Globals() []string { return slices.Clone(globals) }

// Shells is the set `hyper completions` writes a script for, alphabetically.
func Shells() []string { return slices.Clone(shells) }

// Commands is the whole surface as one list: §9's sixteen, then the three
// outside the tree. It flattens the same groups the usage page renders, which
// is what makes the two one list read two ways (usage.go).
func Commands() []string { return names(surface...) }
