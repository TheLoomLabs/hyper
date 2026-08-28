package cli

import "strings"

// usage is the page a bare `hyper` writes: the one line it has always written,
// and under it §9's tree — the six groups, the sixteen names with the
// positionals the table states beside them, and the three commands that stand
// outside the tree named as such (issue #210).
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
		b.WriteString("  " + g.title + "\n")
		for _, command := range g.commands {
			b.WriteString("    " + command.signature() + "\n")
		}
	}
	return b.String()
}
