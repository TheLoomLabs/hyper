// Command hyper is the CLI surface over hyper's core (§9). This binary
// implements three of them so far: check, the command issue #88 cuts the whole
// path for, and version and completions, the two that stand outside the tree
// of sixteen (issues #103 and #104).
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: hyper <command> [args...]")
		return cli.ExitUsage
	}

	switch args[0] {
	case "check":
		// The working directory is read here rather than above the switch,
		// so a command that reads no repository never depends on there being
		// one to read (issue #103).
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "hyper: %s\n", err)
			return 1
		}
		return cli.RunCheck(args[1:], stdout, stderr, os.Getenv, wd, version.Version)
	case "version":
		// Neither the environment nor a working directory is passed, and no
		// repository root is resolved: `version` is one of the two commands
		// outside the tree of sixteen and exempt from the pin gate (§9,
		// ADR-0020). The facts are read once, here, from Go's own build
		// stamping.
		return cli.RunVersion(args[1:], stdout, stderr, version.Current())
	case "completions":
		// The other command outside the tree, and exempt for the same
		// reason: it reads no repository, so shell setup in a dotfiles
		// bootstrap works before one exists (§9, ADR-0020, issue #104).
		return cli.RunCompletions(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "hyper: unknown command %q\n", args[0])
		return cli.ExitUsage
	}
}
