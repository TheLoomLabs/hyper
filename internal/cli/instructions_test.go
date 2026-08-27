package cli_test

import (
	"bytes"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/mcp"
)

// The worked example the handshake carries, held to the only standard that
// matters for it: **it checks clean** (issue #209).
//
// It is asserted here rather than beside the text because a `check` is not
// reachable from internal/mcp and must not become so — the surface hands a Call
// to a dispatch and knows no command (server.go). internal/cli is where the
// commands are, and it already imports the surface to stand a server over one,
// so this is the corpus reading the orientation the way a client's agent will:
// as files to write down and then check.
//
// **What it defends is the failure mode a worked example has.** Prose that is
// wrong is read and discarded; a Manifest that is wrong is *copied*, and the
// agent that copied it spends its next three turns repairing an artefact the
// tool taught it. §4 counts thirty-two static codes over §3's format, and a
// hole where a `record.fields` wants a path is not a thing a reader spots.

// TestInstructions_TheWorkedExampleChecksClean writes every artefact the
// orientation names into a repository of its own and runs `check` over it.
//
// The version is the one the pin gate compares, threaded into the text the same
// way the server threads its own: an example carrying any other version would
// Refuse the gate before a single artefact was read, which would say nothing
// about the artefacts (§11, ADR-0020).
func TestInstructions_TheWorkedExampleChecksClean(t *testing.T) {
	root := t.TempDir()
	written := workedExample(t, mcp.Instructions("1.4.0"))

	// The five §2 counts, and the count is the assertion: an example that
	// dropped one is one an agent cannot author against, and it would
	// otherwise check clean by being smaller.
	if len(written) != 5 {
		t.Fatalf("the orientation carries %d artefacts, want the five §2 counts: %v", len(written), slices.Sorted(maps.Keys(written)))
	}
	for path, content := range written {
		writeFile(t, filepath.Join(root, path), content)
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root}, cli.Streams(&stdout, &stderr), emptyEnvironment, t.TempDir(), "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("the worked example the handshake teaches does not check clean (exit %d):\n%s%s", exit, stdout.String(), stderr.String())
	}
}

// workedExample is every artefact the orientation carries: each one a fenced YAML
// block under the repository path it belongs at, keyed by that path.
//
// **The path is part of the teaching, not scaffolding for this case.** §3
// resolves a Manifest by the directory it sits in, so an example that showed
// the bytes and not the file would leave an agent to guess `providers/` —
// which is a guess it makes wrong, and one `check` reports as an artefact that
// does not exist rather than as a file in the wrong place.
func workedExample(t *testing.T, instructions string) map[string]string {
	t.Helper()

	written := map[string]string{}
	var at string
	var block []string
	fenced := false

	for _, line := range strings.Split(instructions, "\n") {
		switch {
		case fenced && line == "```":
			written[at] = strings.Join(block, "\n") + "\n"
			at, block, fenced = "", nil, false
		case fenced:
			block = append(block, line)
		case at != "" && line == "```yaml":
			fenced = true
		case strings.HasPrefix(line, "`") && strings.HasSuffix(line, ".yaml`"):
			at = strings.Trim(line, "`")
		}
	}
	if fenced {
		t.Fatalf("the orientation opens a block for %s and never closes it", at)
	}
	return written
}
