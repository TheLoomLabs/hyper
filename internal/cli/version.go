package cli

import (
	"fmt"
	"io"

	"github.com/TheLoomLabs/hyper/internal/version"
)

// RunVersion implements `hyper version` — one of the two commands that stand
// outside the tree of sixteen, and the second way to read the version string
// the pin gate compares (§9, ADR-0020, issue #103). An operator whose `check`
// just Refused with *this binary is 1.4.0* has been told a version by the
// process whose identity is in question; this is where it is read on its own.
//
// It is a sibling of RunCheck and takes neither the environment nor a working
// directory, because neither is reachable from a command that reads no
// repository: the pin-gate exemption §9 grants it is stated in the signature
// rather than enforced by a branch inside it. facts is passed in for the same
// reason golden files exist at all — a page assembled from the running build
// changes with every commit made to the tree.
//
// It reaches no network on any path and never asks whether a newer version
// exists (ADR-0016, ADR-0019).
func RunVersion(args []string, stdout, stderr io.Writer, facts version.Facts) int {
	// Any argument at all is a usage error, `--json` included. The three
	// globals govern the sixteen; --repo-dir is meaningless on a command that
	// reads no repository, and a bespoke JSON object here would mint a second
	// document shape beside §8's one renderer. The accepted cost is that a
	// script wanting the bare version cuts the first line — and adding --json
	// later stays compatible, while removing it would not (issue #103).
	if len(args) > 0 {
		fmt.Fprintf(stderr, "hyper version: takes no arguments, got %s\n", args[0])
		return ExitUsage
	}

	if _, err := io.WriteString(stdout, facts.Page()); err != nil {
		fmt.Fprintf(stderr, "hyper version: %s\n", err)
		return ExitUsage
	}
	return ExitClean
}
