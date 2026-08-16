// Command hyper is the CLI surface over hyper's core (§9). This binary
// implements the one command issue #88 cuts the whole path for: check.
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
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(stderr, "hyper: %s\n", err)
			return 1
		}
		return cli.RunCheck(args[1:], stdout, stderr, os.Getenv, wd, version.Version)
	default:
		fmt.Fprintf(stderr, "hyper: unknown command %q\n", args[0])
		return cli.ExitUsage
	}
}
