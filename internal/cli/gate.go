package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

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
// repository root; binaryVersion is what the running binary claims to be.
//
// It answers two things. The first is the exit code: 0 clears the caller to
// proceed, and anything else is what the caller returns, this having already
// rendered its own reason. The second is the `error_code` it Refused under, ""
// on the two answers that are not Refusals — the clear, and the unreadable
// hyper.yaml, which is a usage error and opens no stream (§9, ADR-0060).
//
// **Fifteen callers discard the name and one does not**, which is `run`: the
// gate is one of the two paths on which a Run declines *before it has an id*,
// and §8 puts `run` on the `outcome` side on both — what is missing there is
// the row's `run_id` and never the row (§9, §10, issue #172). That row carries
// the code of the check that declined it, so the one caller that writes a
// terminal row reads it here rather than deriving a second name for what this
// already knows.
func gateOnVersionPin(command, repoRoot, binaryVersion string, stderr io.Writer) (int, string) {
	data, err := os.ReadFile(filepath.Join(repoRoot, "hyper.yaml"))
	present := err == nil
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "hyper %s: %s\n", command, err)
		return ExitUsage, ""
	}

	if result := pin.Check(present, data, binaryVersion); result.Refused {
		return refuse(stderr, result.Code, result.Message), result.Code
	}

	return 0, ""
}

// refuse renders a Refusal and answers the exit code its caller returns.
//
// It is two lines on stderr with stdout left silent, whichever mode the caller
// was invoked in: a Refusal is not a row, so --json opens no stream to carry it
// (§9). What a Run writes to stdout beside it is the Run's own and is written
// there (run.go's runRendering.terminate), on the same footing as every other
// path a Run takes: the answer on one stream and the narration on the other.
//
// **It stands beside §8's caret form rather than being replaced by it**, and
// what sorts the two is whether the check has a line to point at. Every Refusal
// rendered here is a fact about the **invocation** — a binary the Repository
// declaration does not pin, a Store branch neither side holds — and none of
// them cites an artefact coordinate at all. A caret excerpt needs a file and a
// line, an `EDIT ONE OF` table needs a field, and inventing one to reach a
// richer rendering would point a reader at an edit that is not the remedy: what
// clears these is a different binary or `hyper store init`, which the message
// already names (§8, refusal.go).
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
// code its caller returns: §8's caret excerpt, its `=` notes and its `EDIT ONE
// OF` table, on stderr, with stdout silent in both modes.
//
// It stands beside refuse rather than replacing it because the two are one
// rendering split by what the fault has to point at. A Refusal with no artefact
// coordinate in it — the pin gate's two codes, `store init`'s absent Store — is
// a fact about the invocation, and the two-line form is the whole of what there
// is to say. A Refusal a Probe makes cites a file, a line and a field, and the
// remedy is an edit there: the excerpt is what carries it, and it is the same
// excerpt a Run's Refusal draws because a reader learns one shape (§8, §9).
//
// **It carries no phase note and no gloss**, and both absences are the same
// fact: a Probe is not a Run. There is no Step for a phase to have preceded,
// and no Run start for a relative operand to resolve against — a gloss is
// derived arithmetic and renders where its supply is (ADR-0034, ADR-0063).
//
// repoRoot is what the excerpt is read against, and the caller has already
// resolved it: an excerpt is the working tree's lines, and the working tree is
// where the edit is made.
func refuseProblems(stderr io.Writer, repoRoot string, problems []problem.Problem) int {
	problem.Sort(problems)
	rows := make([]render.Row, 0, 2*len(problems))
	for _, found := range problems {
		member := excerpted(refusalRow{
			Type:      "refusal",
			ErrorCode: found.ErrorCode,
			File:      found.File,
			Line:      found.Line,
			Field:     found.Field,
			Message:   found.Message,
		}, repoRoot, time.Time{})
		rows = append(rows, member)
		rows = append(rows, remediationsFor(member, nil, time.Time{})...)
	}
	if err := writeRefusal(stderr, rows, false); err != nil {
		fmt.Fprintf(stderr, "hyper: %s\n", err)
	}
	return ExitRefused
}
