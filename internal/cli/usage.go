package cli

import "strings"

// usage is the page a bare `hyper` writes: the one line it has always written,
// and under it §9's tree — the six groups, the sixteen names with the
// positionals the table states beside them, the three commands that stand
// outside the tree named as such, and §9's three configuration flags in a
// block of their own (issue #210).
//
// **It adds no command, no alias and no seventeenth entry.** `help` and
// `--help` stay unknown, because §9's tree is sixteen names with no aliases and
// no hidden commands and neither of those is one of them; the exit code stays
// `2`; and the page goes to stderr like every other narration. Printing the
// list is the opposite of hiding one.
//
// What it exists for is a caller who has no other way to ask. The MCP surface
// answers the same question in its handshake (§9, ADR-0093), and until this
// page the terminal answered it nowhere: `hyper completions bash` emits every
// name, but it emits them as a shell script, which is not a thing anybody runs
// to find out what a binary does. An agent that asked and was told `<command>`
// went reading the directories around it instead, which in a customer's
// checkout is their source.
//
// The groups are §9's own, and they are kept because they are how §9 teaches
// the tree — *which Provider, which Operation, how do I call it* is a run of
// three questions and not three unrelated names. The positionals are kept for
// the reason the names alone are not enough: the names end the foraging, and
// the signature is what stops the next invocation being a guess.
//
// Nothing is assembled from a second list. The groups, the names and the
// positionals are all tree.go's, so a command §9 adds reaches this page and
// the three completion scripts by the same edit.
func usage() string {
	var b strings.Builder
	b.WriteString("usage: hyper <command> [args...]\n\n")
	// The tree's six groups, then the three outside it as a seventh — a
	// caller who cannot tell `version` from `records` has learnt the names
	// and not the surface (tree.go).
	for _, g := range surface {
		writeGroup(&b, g)
	}
	// The globals, in a block of their own past a blank line. They are not
	// commands and are not rendered as if they were: what closes the last
	// guess a caller has left is *these three and no fourth*, and the block
	// is what says the sixteen take them and the three above do not.
	b.WriteString("\n")
	writeGroup(&b, globalFlags)
	return b.String()
}

// writeGroup writes one group the one way a group is written: the title at two
// spaces, and each of its entries at four. The page has one shape, so a block
// added to it is a block a reader already knows how to read.
func writeGroup(b *strings.Builder, g group) {
	b.WriteString("  " + g.title + "\n")
	for _, command := range g.commands {
		b.WriteString("    " + command.signature() + "\n")
	}
}

// whereTheCommandsAre is the second line of an unknown command: the namespace
// the name was resolved against, and the invocation that enumerates it.
//
// It lives beside the page rather than beside the dispatch because it is a
// claim about the page — that a bare `hyper` lists them — and a claim about
// one thing belongs where that thing is. Its shape is `show`'s and `changes`'s,
// which answer an unresolved id the same way: the fault on the first line, and
// the namespace with its enumerating command indented under it (§9, show.go).
const whereTheCommandsAre = "  the nineteen commands are that namespace, and hyper with no arguments lists them\n"
