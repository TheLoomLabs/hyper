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

// The worked example the orientation carries, held to the only standard that
// matters for it: **it checks clean** (issues #209, #211 and #212).
//
// It is carried twice over — as the `instructions` field of the handshake, and
// as the `AGENTS.md` `project` writes where a repository holds none — and this
// case is about the artefacts in it rather than about either channel.
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
// **A fenced block that names no path is not an artefact and is not written.**
// The multi-host `read` shape is a fragment — three keys of an Operation whose
// Manifest does not exist — and the parser below picks up only the blocks a
// repository path stands over, which is the same line a reader reads them by
// (§3).
//
// The version is the one the pin gate compares, threaded into the text the same
// way the server threads its own: an example carrying any other version would
// Refuse the gate before a single artefact was read, which would say nothing
// about the artefacts (§11, ADR-0020).
func TestInstructions_TheWorkedExampleChecksClean(t *testing.T) {
	root := t.TempDir()
	written := workedExample(t, mcp.Instructions("1.4.0"))

	// **The five kinds §2 counts, not a file count.** An example that
	// dropped a kind is one an agent cannot author that kind against, and
	// it would otherwise check clean by being smaller. Which shape the whole
	// Manifest carries is free to move and has moved twice, one request
	// shape being a fragment throughout: this case reads whatever stands
	// under a path, and the case below says which one that must be
	// (issues #211, #212).
	carried := map[string]bool{}
	for _, content := range written {
		carried[kindOf(content)] = true
	}
	for _, kind := range []string{"provider", "target-declaration", "definition", "procedure", "repository-declaration"} {
		if !carried[kind] {
			t.Errorf("the orientation carries no %s; §2 counts five artefacts and an agent authors four of them", kind)
		}
	}
	if t.Failed() {
		t.Fatalf("the example is %v", slices.Sorted(maps.Keys(written)))
	}
	for path, content := range written {
		writeFile(t, filepath.Join(root, path), content)
	}

	var stdout, stderr bytes.Buffer
	exit := cli.RunCheck([]string{"--repo-dir", root}, cli.Streams(&stdout, &stderr), emptyEnvironment, t.TempDir(), "1.4.0")
	if exit != cli.ExitClean {
		t.Fatalf("the worked example the orientation teaches does not check clean (exit %d):\n%s%s", exit, stdout.String(), stderr.String())
	}
}

// TestInstructions_TheShapeCarriedWholeIsTheEffectfulOne is the swap issue #212
// made, and the thing a rewrite would undo without noticing.
//
// **The worked example may not be the acceptance task.** The multi-host `read`
// is what a fresh repository's first agent is asked for — it is #209's canonical
// task and the criterion #211 shipped against — and while that shape was the
// whole Manifest, the transcript that met the criterion met it by transcription:
// twenty-two of twenty-two lines identical, and the run said nothing about
// whether the text was sufficient. The shape carried whole is therefore the
// effectful one, whose rules are the ones no tool call and no format prose
// state: a `record:` fixed by Kind, an `identity:` that must resolve before the
// call, a `destroy:` claim naming Operations, a selector, a Bound.
//
// The `read` keeps a fragment beside it, and that is the whole of what it keeps.
// **Both shapes whole is the 13,191 characters #211 cut, re-acquired** — the
// length is paid on every session in every harness — so this case holds the
// fragment down by its one distinguishing key: `host-input:` is taught, and no
// artefact in the example carries it.
func TestInstructions_TheShapeCarriedWholeIsTheEffectfulOne(t *testing.T) {
	orientation := mcp.Instructions("1.4.0")
	written := workedExample(t, orientation)

	manifest := ""
	for _, content := range written {
		if kindOf(content) == "provider" {
			manifest = content
		}
	}
	if manifest == "" {
		t.Fatalf("the orientation writes out no Manifest at all; the example is %v", slices.Sorted(maps.Keys(written)))
	}
	for _, line := range []string{"kind: mutate", "kind: destroy", "repeatability: skip-if-recorded"} {
		if !strings.Contains(manifest, line) {
			t.Errorf("the Manifest carried whole declares no %q; the effectful shape is the one carrying rules an agent cannot infer", line)
		}
	}
	// The identity **beside that declaration**, and not one anywhere in the
	// file: a Manifest carrying a `read` as well would carry a path as
	// readily, and the rule is this Operation's. Its `record:` is mandatory
	// (§3), so the first one below the line is the one it declares.
	if at := strings.Index(manifest, "repeatability: skip-if-recorded"); at >= 0 {
		identity := ""
		for _, line := range strings.Split(manifest[at:], "\n") {
			if declared, ok := strings.CutPrefix(strings.TrimSpace(line), "identity: "); ok {
				identity = declared
				break
			}
		}
		if !strings.HasPrefix(identity, `"{`) {
			t.Errorf("the example's `skip-if-recorded` Operation takes its identity from %q; the test reads the head of the series before the call, so it is a hole and never a response path", identity)
		}
	}
	// The scheme, and the whole of what a Manifest says about a credential.
	if !strings.Contains(manifest, "auth:") {
		t.Error("the Manifest carried whole names no auth scheme; a header-authenticated request is the shape an effectful Operation is asked for in")
	}

	// The other half of the swap. The `read` shape is taught and is not a
	// second whole Manifest, which is what the reduction #211 made a design
	// constraint buys back.
	if !strings.Contains(orientation, "host-input:") {
		t.Error("the orientation never names `host-input:`; the multi-host shape is what #209 bought and an agent without it disassembles the binary")
	}
	for path, content := range written {
		if strings.Contains(content, "host-input:") {
			t.Errorf("%s carries the multi-host shape as an artefact; it is a fragment beside the example, and two whole Manifests is the length #211 cut", path)
		}
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

// kindOf is an artefact's `kind:`, which every one of the five declares on a
// line of its own — the discriminator §3 reads them by, read here the same way.
func kindOf(content string) string {
	for _, line := range strings.Split(content, "\n") {
		if after, ok := strings.CutPrefix(line, "kind: "); ok {
			return strings.TrimSpace(after)
		}
	}
	return ""
}
