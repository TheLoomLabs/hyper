package cli_test

import (
	"bytes"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
)

// **`--expansion` is `show`'s and no other command's** (§9, issue #163).
//
// The positive half is the corpus's: `an-expansion-in-expansion-order` and
// `a-halted-destroy-under-expansion` drive the flag over a whole entry and hold
// the selector, the sequence and the Bound it renders. What no case directory
// can state is the negative half over the **whole surface**, because that is a
// claim about every command at once rather than about one of them — so it is
// asserted here, over §9's own list of names rather than a list this file
// keeps, which is `--dry-run`'s own arrangement one file over.

// TestExpansion_IsShowsFlagAndNoOtherCommands holds the flag to the one command
// §9 gives it to. Every other name in the surface answers a usage error with
// stdout completely silent: a `runs --expansion` or a `changes --expansion`
// would have to mean something, and neither does.
func TestExpansion_IsShowsFlagAndNoOtherCommands(t *testing.T) {
	for _, command := range cli.Commands() {
		if command == "show" {
			continue
		}
		t.Run(command, func(t *testing.T) {
			p := &process{wd: t.TempDir()}
			var stdout, stderr bytes.Buffer

			exit := cli.Main([]string{command, "--expansion"}, &stdout, &stderr, p.value(), testFacts)

			if exit != cli.ExitUsage {
				t.Errorf("hyper %s --expansion exit = %d, want %d — the flag is not this command's", command, exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("hyper %s --expansion wrote %q to stdout, want it silent: a usage error opens no row stream", command, stdout.String())
			}
			if stderr.Len() == 0 {
				t.Errorf("hyper %s --expansion said nothing on stderr, want the fault named", command)
			}
		})
	}
}

// TestExpansion_TakesNoValue is the spelling the flag does not have. It is a
// marker rather than a setting, so `--expansion=false` is a caller expecting a
// value to be read — and reading it as *the flag was named* would answer a
// question nobody asked with five hundred members of an Expansion.
func TestExpansion_TakesNoValue(t *testing.T) {
	for _, spelling := range []string{"--expansion=true", "--expansion=false"} {
		t.Run(spelling, func(t *testing.T) {
			p := &process{wd: t.TempDir()}
			var stdout, stderr bytes.Buffer

			exit := cli.Main([]string{"show", spelling}, &stdout, &stderr, p.value(), testFacts)

			if exit != cli.ExitUsage {
				t.Errorf("hyper show %s exit = %d, want %d — the flag is a marker and takes no value", spelling, exit, cli.ExitUsage)
			}
			if stdout.Len() != 0 {
				t.Errorf("hyper show %s wrote %q to stdout, want it silent", spelling, stdout.String())
			}
		})
	}
}
