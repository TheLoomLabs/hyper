package cli_test

import (
	"bytes"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **`--dry-run` is `run`'s and no other command's** (§9, ADR-0015, issue #145).
//
// The positive half is the corpus's: `a-rehearsal-performs-the-reads-it-reaches`
// drives the flag through a whole Run and holds the entry, the Observations and
// the terminal line it produced. What no case directory can state is the
// negative half over the **whole surface**, because that is a claim about every
// command at once rather than about one of them — so it is asserted here, over
// §9's own list of names rather than a list this file keeps.
//
// Reading the surface from `cli.Commands()` is what makes it stay true. A
// command the spec fixes but the binary has not built yet answers `unknown
// command`, which is a usage error like any other; the day it is built it
// inherits this case without anybody remembering to add a line, and the day
// somebody wires the flag into parseArgs — where it would reach every command
// at once — this is what fails.

// TestDryRun_IsRunsFlagAndNoOtherCommands holds the flag to the one command §9
// gives it to. Every other name in the surface answers a usage error with
// stdout completely silent: a `records --dry-run` or a `check --dry-run` would
// have to mean something, and neither does.
func TestDryRun_IsRunsFlagAndNoOtherCommands(t *testing.T) {
	for _, command := range cli.Commands() {
		if command == "run" {
			continue
		}
		t.Run(command, func(t *testing.T) {
			p := &process{wd: t.TempDir()}
			var stdout, stderr bytes.Buffer

			exit := cli.Main([]string{command, "--dry-run"}, &stdout, &stderr, p.value(), testFacts)

			if exit != cli.ExitUsage {
				t.Errorf("hyper %s --dry-run exit = %d, want %d — the flag is not this command's", command, exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("hyper %s --dry-run wrote %q to stdout, want it silent: a usage error opens no row stream", command, stdout.String())
			}
			if stderr.Len() == 0 {
				t.Errorf("hyper %s --dry-run said nothing on stderr, want the fault named", command)
			}
		})
	}
}

// TestDryRun_TakesNoValue is the spelling the flag does not have. It is a
// marker rather than a setting, so `--dry-run=false` is a caller expecting a
// value to be read — and reading it as *the flag was named* would turn an
// invocation that meant to rehearse nothing into a rehearsal.
func TestDryRun_TakesNoValue(t *testing.T) {
	for _, spelling := range []string{"--dry-run=true", "--dry-run=false"} {
		t.Run(spelling, func(t *testing.T) {
			p := &process{wd: t.TempDir()}
			var stdout, stderr bytes.Buffer

			exit := cli.Main([]string{"run", spelling, "anything"}, &stdout, &stderr, p.value(), testFacts)

			if exit != cli.ExitUsage {
				t.Errorf("hyper run %s exit = %d, want %d — the marker carries no value", spelling, exit, cli.ExitUsage)
			}
			if want := "unknown flag " + spelling; !bytes.Contains(stderr.Bytes(), []byte(want)) {
				t.Errorf("stderr = %q, want it to name %q", stderr.String(), want)
			}
		})
	}
}
