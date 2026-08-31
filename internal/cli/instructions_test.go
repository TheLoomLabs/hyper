package cli_test

import (
	"bytes"
	"encoding/json"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/artefact"
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

// TestInstructions_TheBoundRuleIsTheOneCheckHolds is the fence issue #218 asks
// for, and the thing it exists to catch has already happened once.
//
// The orientation stated the Bound rule without its exception — *on a `destroy`
// it is **mandatory*** — while the binary holds two rules, one of them the
// opposite of that: a Bound is mandatory on a `destroy` and **refused** on an
// opaque one, an opaque `destroy` being every `destroy` whose request is
// `shell:` (`bound-missing`, `bound-illegal`, §5, ADR-0053). The sealed
// acceptance run of 2026-08-29 walked into it — the orientation taught the
// Bound, `check` declined it, and the session recovered the real rule from
// `review`'s `UNBOUNDED` flag rather than from the text whose whole job is to
// spare it that (ADR-0100, ADR-0101).
//
// **So the claim is held to the checker and not to a reader.** The four
// combinations below are the rule as `check` actually holds it, and each one
// `check` declines names the word the orientation has to carry for an agent to
// have avoided authoring it. A rule that moves in the binary and not in the text
// fails here, which is the direction it moved last time.
func TestInstructions_TheBoundRuleIsTheOneCheckHolds(t *testing.T) {
	sentence := boundSentence(t, mcp.Instructions("1.4.0"))

	for _, c := range []struct {
		name   string
		repo   func(*testing.T, bool) map[string]string
		bound  bool
		code   string
		stated string
	}{
		{name: "a destroy Step carrying its Bound", repo: exampleRepository, bound: true},
		{name: "a destroy Step carrying none", repo: exampleRepository, code: artefact.CodeBoundMissing, stated: "mandatory"},
		{name: "an opaque destroy Step carrying none", repo: opaqueRepository},
		{name: "an opaque destroy Step carrying one", repo: opaqueRepository, bound: true, code: artefact.CodeBoundIllegal, stated: "refused"},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			for path, content := range c.repo(t, c.bound) {
				writeFile(t, filepath.Join(root, path), content)
			}

			var stdout, stderr bytes.Buffer
			exit := cli.RunCheck([]string{"--repo-dir", root, "--json"}, cli.Streams(&stdout, &stderr), emptyEnvironment, t.TempDir(), "1.4.0")
			codes := errorCodesIn(t, stdout.String())

			switch {
			case c.code == "" && exit != cli.ExitClean:
				t.Fatalf("check declines it with %v (exit %d); the orientation teaches an artefact the checker refuses\n%s", codes, exit, stderr.String())
			case c.code != "" && !slices.Contains(codes, c.code):
				t.Fatalf("check answers %v and not %s (exit %d); the Bound rule the checker holds is not the one this case reads", codes, c.code, exit)
			}
			// The pairing, and the whole of why this is a fence rather
			// than two tests: a combination `check` declines with no word
			// for it in the orientation is a repair loop an agent is sent
			// on by the text it was oriented with.
			if c.stated != "" && !strings.Contains(sentence, c.stated) {
				t.Errorf("check declines this with %s and the orientation's Bound sentence never says %q: %q", c.code, c.stated, sentence)
			}
		})
	}
}

// TestInstructions_TheBoundRuleNamesWhatMakesADestroyOpaque is the other half of
// the exception, and the half a word like *refused* does not carry on its own.
//
// An agent told that some `destroy` Steps refuse a Bound and not which ones is
// an agent that guesses, and every `destroy` on the built-in `shell` Provider is
// one of them — which is every `destroy` a fresh repository can author at all,
// `providers/` being empty until somebody writes one (§9, ADR-0093).
//
// The Capability is read off the binary rather than spelled here: it is the one
// member of §12's reserved half, and the predicate that makes a `destroy` opaque
// is its Kind and that request block (`IsOpaqueDestroy`, ADR-0039).
func TestInstructions_TheBoundRuleNamesWhatMakesADestroyOpaque(t *testing.T) {
	sentence := boundSentence(t, mcp.Instructions("1.4.0"))

	for _, named := range []string{"destroy", "opaque", artefact.ReservedCapability} {
		if !strings.Contains(sentence, named) {
			t.Errorf("the orientation's Bound sentence never names %q: %q", named, sentence)
		}
	}
}

// boundSentence is the one sentence of the orientation that states the Bound
// rule, unwrapped the way a reader takes it in.
//
// **Exactly one, and that is an assertion rather than a lookup.** The
// orientation is a budget paid on every session in every harness (ADR-0093), and
// the repair issue #218 asked for is a rule stated whole in the sentence that
// already stood — not a paragraph about Bounds added beside it. Two sentences
// here is the manual the text may not become; none is the rule gone.
//
// A sentence states the rule where it names the key or the term `CONTEXT.md`
// fixes for it, which is the pair a second sentence about Bounds would have to
// evade both of to slip past this. The Step's own `bound: 5` inside the worked
// example matches neither — it is the key unquoted, in a fenced block — and
// neither does the `bound Target` of the sentence above it, which is the English
// word.
func boundSentence(t *testing.T, instructions string) string {
	t.Helper()

	var carried []string
	for _, sentence := range strings.SplitAfter(strings.Join(strings.Fields(instructions), " "), ". ") {
		if strings.Contains(sentence, "`bound:`") || strings.Contains(sentence, "Bound") {
			carried = append(carried, strings.TrimSpace(sentence))
		}
	}
	if len(carried) != 1 {
		t.Fatalf("the orientation states the Bound rule in %d sentences, want exactly one: %q", len(carried), carried)
	}
	return carried[0]
}

// exampleRepository is the orientation's own worked example, whose `destroy`
// Step is an `http` one and carries a Bound, with that Bound taken away where
// bound is false.
//
// It is the shipped text rather than a fixture beside it, so that the
// non-opaque half of the rule is read off the artefact an agent transcribes.
func exampleRepository(t *testing.T, bound bool) map[string]string {
	t.Helper()

	written := workedExample(t, mcp.Instructions("1.4.0"))

	// The whole map rather than the first match, on the sibling case's own
	// footing: which artefact this reads may not depend on map order, and a
	// second Procedure carrying a Bound is a second thing this case would
	// have to say which of.
	var declaring []string
	for path, content := range written {
		if kindOf(content) == "procedure" && strings.Contains(content, "bound:") {
			declaring = append(declaring, path)
		}
	}
	if len(declaring) != 1 {
		t.Fatalf("%d of the worked example's artefacts declare a Bound, want exactly one: %v", len(declaring), slices.Sorted(maps.Keys(written)))
	}
	if bound {
		return written
	}

	at := declaring[0]
	var kept []string
	for _, line := range strings.Split(written[at], "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "bound:") {
			kept = append(kept, line)
		}
	}
	written[at] = strings.Join(kept, "\n")
	return written
}

// opaqueRepository is the smallest repository holding an opaque `destroy` Step:
// the built-in `shell` Provider, a `local` Target opted into `opaque-destroy:`,
// and one Step over two paths — with a Bound on it where bound is true.
//
// There is no `providers/` file because there is nothing to write one from: the
// built-in Manifest is compiled in, which is also why this shape is the one a
// fresh repository's first `destroy` takes (§12, ADR-0039).
func opaqueRepository(t *testing.T, bound bool) map[string]string {
	t.Helper()

	declared := ""
	if bound {
		declared = "    bound: 1\n"
	}
	return map[string]string{
		"hyper.yaml":         validHyperYAML,
		"targets/local.yaml": "kind: target-declaration\ntarget: local\nclass: local\nkinds: [read, mutate, destroy]\ncapabilities: [shell]\nopaque-destroy: true\n",
		"definitions/host-ops.yaml": "kind: definition\ndefinition: host-ops\nprovider: shell\n" +
			"kinds: [mutate]\ndestroy: [destroy]\ntargets: [local]\n",
		"procedures/purge-releases.yaml": "kind: procedure\nprocedure: purge-releases\ntargets: [local]\nsteps:\n" +
			"  - id: purge\n    definition: host-ops\n    operation: destroy\n    target: local\n" +
			"    over:\n      values: [/srv/app/releases/r41, /srv/app/releases/r42]\n" +
			declared +
			"    args:\n      command: [rm, -rf, {item: $}]\n",
	}
}

// errorCodesIn is every `error_code` the `--json` stream carries, in the order
// the rows arrive.
//
// It reads the machine-readable half rather than the table because a code is a
// member of a closed set on that channel and a column of a rendering on the
// other, and what these cases assert is membership (§8, ADR-0026).
func errorCodesIn(t *testing.T, stream string) []string {
	t.Helper()

	var codes []string
	for _, line := range strings.Split(strings.TrimSpace(stream), "\n") {
		if line == "" {
			continue
		}
		var row struct {
			ErrorCode string `json:"error_code"`
		}
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("the --json stream carries a line that is not a row: %q (%v)", line, err)
		}
		if row.ErrorCode != "" {
			codes = append(codes, row.ErrorCode)
		}
	}
	return codes
}

// TestInstructions_TheSharedCheckItTeachesIsOneCheckAccepts is issue #236's own
// fence, on the Bound rule's footing above.
//
// The orientation now teaches the shape that ticket exists for: a shared check
// that halts on a `require:` of its own rather than claiming `mutate` in order
// to be able to fail (ADR-0111, ADR-0116). It is the highest-stakes sentence in
// the text — an agent that does not believe it authors an effectful Step on the
// one artefact whose point is that it writes nothing — so the fragment is
// written into a repository and checked rather than read.
//
// **And the wall it names is held too.** The paragraph tells an agent that no
// `when:` reaches across an invocation, which is the claim the sealed run of
// 2026-08-30 spent three Refusals discovering (ADR-0111). A text that said so
// while `check` had come to accept it would be teaching an artefact nobody can
// author, so both halves are held to the checker: the `require:` checks clean
// and the `when:` across the boundary does not.
func TestInstructions_TheSharedCheckItTeachesIsOneCheckAccepts(t *testing.T) {
	requirement := taughtRequirement(t, mcp.Instructions("1.4.0"))

	base := map[string]string{
		"hyper.yaml": validHyperYAML,
		"targets/archive.yaml": "kind: target-declaration\ntarget: archive\nclass: local\n" +
			"kinds: [read]\ncapabilities: [shell]\n",
		"targets/local.yaml": "kind: target-declaration\ntarget: local\nclass: local\n" +
			"kinds: [read, mutate]\ncapabilities: [shell]\n",
		"definitions/archive-audit.yaml": "kind: definition\ndefinition: archive-audit\nprovider: shell\n" +
			"kinds: [read]\ntargets: [archive]\n",
		"definitions/live-ops.yaml": "kind: definition\ndefinition: live-ops\nprovider: shell\n" +
			"kinds: [mutate]\ntargets: [local]\n",
		"procedures/verify-archive.yaml": "kind: procedure\nprocedure: verify-archive\ntargets: [archive]\nsteps:\n" + requirement,
	}
	promote := func(gate string) string {
		return "kind: procedure\nprocedure: promote\ntargets: [archive, local]\nsteps:\n" +
			"  - {id: verify, procedure: verify-archive}\n" +
			"  - id: swap\n    definition: live-ops\n    operation: mutate\n    target: local\n" +
			"    bound: 1\n" + gate +
			"    args:\n      command: [sh, -c, \"ln -sfn /srv/archive/wanted /srv/live/current\"]\n"
	}

	for _, c := range []struct {
		name string
		gate string
		code string
	}{
		{name: "the check halts on its own require:", gate: ""},
		{
			name: "the caller conditions on the invocation",
			gate: "    when: {step: verify, field: exit_code, equals: 0}\n",
			code: artefact.CodeReferenceUnresolvable,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			root := t.TempDir()
			for path, content := range base {
				writeFile(t, filepath.Join(root, path), content)
			}
			writeFile(t, filepath.Join(root, "procedures/promote.yaml"), promote(c.gate))

			var stdout, stderr bytes.Buffer
			exit := cli.RunCheck([]string{"--repo-dir", root, "--json"}, cli.Streams(&stdout, &stderr), emptyEnvironment, t.TempDir(), "1.4.0")
			codes := errorCodesIn(t, stdout.String())

			switch {
			case c.code == "" && exit != cli.ExitClean:
				t.Fatalf("check declines the shared check the orientation teaches with %v (exit %d)\n%s", codes, exit, stderr.String())
			case c.code != "" && !slices.Contains(codes, c.code):
				t.Fatalf("check answers %v and not %s (exit %d); the orientation says this boundary holds", codes, c.code, exit)
			}
		})
	}
}

// taughtRequirement is the `steps:` fragment the orientation's shared-check
// paragraph carries, indented as a `steps:` list already is there.
//
// **Exactly one such block, and that is an assertion.** The orientation is a
// budget paid on every session (ADR-0093), and what it may carry about this is
// one shape to transcribe — a second fenced `require:` is a manual growing
// inside the text, and none is the rule gone from it.
func taughtRequirement(t *testing.T, instructions string) string {
	t.Helper()

	var carried []string
	var block []string
	fenced := false
	for _, line := range strings.Split(instructions, "\n") {
		switch {
		case fenced && line == "```":
			if slices.ContainsFunc(block, func(l string) bool { return strings.Contains(l, "require:") }) {
				carried = append(carried, strings.Join(block, "\n")+"\n")
			}
			block, fenced = nil, false
		case fenced:
			block = append(block, line)
		case line == "```yaml":
			fenced = true
		}
	}
	if len(carried) != 1 {
		t.Fatalf("the orientation carries %d fenced blocks writing a require:, want exactly one", len(carried))
	}
	return carried[0]
}
