package cli

import "slices"

// The command surface, as one list. §9 fixes a tree of sixteen commands —
// flat, one noun group, no aliases and no hidden commands — beside three more
// that stand outside it because none of them reads a repository and none says
// anything about hyper's domain. That surface is stated here once, in the
// order §9's table states it, so a command added by a later milestone reaches
// every reader of it by one string (issue #104).
//
// The first reader is the completion scripts, and it is the reason the list
// exists at all: three shells describing one surface from three copies of it
// would drift the day the seventeenth command lands. Nothing here is a
// dispatch table — Main's switch is what runs a command, and a name in
// this list is a name the spec fixes, not a claim that the binary implements
// it yet.
//
// The lists are unexported and reached through the functions below, each
// returning a copy. Go has no constant slice, and an exported one is a
// package-level variable any caller can sort, truncate or clobber for every
// other caller; a surface the specification fixes should not be editable by
// the code that reads it.

// tree is §9's tree of sixteen, in the spec's own group order. A name is
// added here when the spec fixes it, not when the milestone that builds it
// lands: the completion script describes the surface the spec states, and
// `hyper run deploy` answering `unknown command` until milestone 5 is the
// accepted, stated cost of that (issue #104).
var tree = []string{
	// Discovery
	"providers",
	"provider",
	"operation",
	// The repository
	"targets",
	// Authoring
	"check",
	"review",
	// Execution
	"run",
	"probe",
	// Inspection
	"runs",
	"show",
	"changes",
	"records",
	// Lifecycle
	"install",
	"project",
	"store",
	"compact",
}

// outsideTree is the three commands §9 states are not among the sixteen:
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
//
// Nothing dispatches it yet. The server is built by a later ticket; the name
// is fixed here so that ticket inherits one rather than choosing it, and until
// then `hyper mcp` answers `unknown command` — the same accepted cost the
// tree's own comment states one list up, arriving for the first time at a name
// outside the tree.
var outsideTree = []string{
	"version",
	"completions",
	"mcp",
}

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
// outside the tree.
func Commands() []string { return append(slices.Clone(tree), outsideTree...) }
