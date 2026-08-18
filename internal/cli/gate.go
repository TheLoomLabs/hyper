package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TheLoomLabs/hyper/internal/pin"
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

	// The Refusal renders as two lines on stderr with stdout left silent,
	// whichever mode the caller was invoked in: a Refusal is not a row, so
	// --json opens no stream to carry it (§9). §8's caret form is milestone
	// 8's renderer.
	if result := pin.Check(present, data, binaryVersion); result.Refused {
		fmt.Fprintf(stderr, "refused: %s\n  %s\n", result.Code, result.Message)
		return ExitRefused
	}

	return 0
}
