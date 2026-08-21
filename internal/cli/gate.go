package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TheLoomLabs/hyper/internal/pin"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
)

// gateOnVersionPin is the version pin gate, and it is one function because
// every command stands behind it rather than each carrying its own copy: all
// sixteen compare themselves against the pin in the Repository declaration and
// Refuse on mismatch, on a laptop and on a runner alike, with `version` and
// `completions` the only exemptions (§9, ADR-0020). It lives in internal/cli
// rather than in cmd/hyper/main.go deliberately — main.go has no golden
// coverage, and the tool's most load-bearing guardrail must not sit outside the
// only harness that proves it byte for byte (issue #102).
//
// Every gated entry point calls it as its first act after resolving a
// repository root, and before its own positionals and work. That ordering is
// the contract: `hyper check definitions/typo.yaml` against a mismatched pin
// exits 77 and not 2, because the gate fires first everywhere (§9). It also
// fires before the repository is loaded, which is why a hyper.yaml carrying an
// unknown-key still gates — its schema fault is check's to report, once the
// binary is cleared to read it at all (§4).
//
// command names the command for the one message that is not a Refusal: a
// hyper.yaml that exists and cannot be read at all. repoRoot is the resolved
// repository root; binaryVersion is what the running binary claims to be. A
// return of 0 clears the caller to proceed; anything else is the exit code the
// caller returns unchanged, having already rendered its own reason.
func gateOnVersionPin(command, repoRoot, binaryVersion string, stderr io.Writer) int {
	data, err := os.ReadFile(filepath.Join(repoRoot, "hyper.yaml"))
	present := err == nil
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "hyper %s: %s\n", command, err)
		return ExitUsage
	}

	if result := pin.Check(present, data, binaryVersion); result.Refused {
		return refuse(stderr, result.Code, result.Message)
	}

	return 0
}

// refuse renders a Refusal and answers the exit code its caller returns.
//
// It is two lines on stderr with stdout left silent, whichever mode the caller
// was invoked in: a Refusal is not a row, so --json opens no stream to carry it
// (§9). §8's caret form is milestone 8's renderer, and this is the form every
// Refusal a command makes takes until it lands.
//
// It is one function rather than the same two lines at each site because the
// milestone that reaches a second code is the one that would otherwise mint a
// second rendering of the first: the pin gate's two codes and `compact`'s
// store-absent and store-schema-unsupported are four Refusals and one shape
// (§12, issue #131).
func refuse(stderr io.Writer, code, message string) int {
	fmt.Fprintf(stderr, "refused: %s\n  %s\n", code, message)
	return ExitRefused
}

// refuseProblems renders a Refusal that has a position, and answers the exit
// code its caller returns. It is the milestone-5 Refusal rendering: the problem
// table `check` already renders, on stderr, with stdout silent in both modes.
//
// It stands beside refuse rather than replacing it because the two are one
// rendering split by what the fault has to point at. Every Refusal reached
// before this milestone — the pin gate's two codes, `store init`'s absent
// Store — is a fact about the invocation with no artefact coordinate in it, and
// the two-line form is the whole of what there is to say. A Refusal a Run or a
// Probe makes cites a file, a line and a field, and the remedy is an edit
// there: the columns are what carry it, and they are `check`'s own columns
// because a reader has already learnt to read them (§8, §9).
//
// §8's caret excerpt, its `=` notes and its `EDIT ONE OF` table are milestone
// 8's, which reads §8 whole. What is deferred is the shape; every fact §8
// requires — file, line, field, code, message — is on the page already, and
// milestone 8 replaces the table with the excerpt without changing what is
// reported.
func refuseProblems(stderr io.Writer, problems []problem.Problem) int {
	problem.Sort(problems)
	rows := checkRows(problems)
	if err := render.WriteTable(stderr, checkColumns, rows); err != nil {
		fmt.Fprintf(stderr, "hyper: %s\n", err)
	}
	return ExitRefused
}
