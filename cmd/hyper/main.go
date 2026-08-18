// Command hyper is the CLI surface over hyper's core (§9). This file is the
// whole of what the binary adds to the library behind it: it reads the process
// — the arguments, the environment, the working directory, the facts Go's build
// stamped — and hands them to cli.Main, which decides which command runs.
//
// No command name appears here, and no dispatch does. This file has no golden
// coverage, which is the argument gate.go makes for the gate and cli.Main
// makes for the dispatch behind it (issues #102 and #107).
//
// os.Getwd is handed over uncalled. `version` and `completions` stand outside
// §9's tree of sixteen and resolve no repository, so a working directory that
// cannot be read must not stop them (§9, ADR-0020).
package main

import (
	"os"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/version"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr, os.Getenv, os.Getwd, version.Current()))
}
