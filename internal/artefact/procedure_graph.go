// This file is the transitive walk itself — the one traversal issue #96's
// three rules ride: an invoked Procedure's own declared envelope reaching
// outside its caller's (the composition half of envelope-exceeded,
// procedure.go carrying the file-local half), and the two Cadence rules,
// cadence-run-once and cadence-secret-output, which cost this walk a rule
// each rather than a traversal of their own. All three read every
// procedures/ file at once — a nested invocation's own file, to any
// depth — which is what sets this apart from procedure.go's per-file
// checks and is why it lives in its own file with its own entry points,
// BuildProcedureGraph and CheckProcedureGraph.
package artefact

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// CodeCadenceRunOnce is the code a Procedure declaring a Cadence earns for
// reaching a run-once Step at any depth: a Cadence over a run-once Step
// declares a recurrence with a lifespan of one occurrence, which is not a
// thing an author can have meant (§4, §5, ADR-0038, issue #96).
const CodeCadenceRunOnce = "cadence-run-once"

// CodeCadenceSecretOutput is the code a Procedure declaring a Cadence earns
// for reaching a Step whose Operation declares secret: output at any
// depth: such a Step Refuses where the invocation supplied no Secret sink,
// and the workflow project generates supplies none, so where a Cadence
// carries it the Refusal lands at every occurrence and the Procedure works
// never (§4, §5, ADR-0077, issue #96). It is its own code rather than a
// second cause folded into cadence-run-once: a reader handed
// cadence-run-once on a secret: clash edits a repeatability: that is
// correct.
const CodeCadenceSecretOutput = "cadence-secret-output"

// ProcedureRoot pairs one procedures/ file's own relative path with its
// already-parsed root — what BuildProcedureGraph reads the transitive
// walk's own per-file facts off of, needing every procedures/ file's own
// name to know where a citation belongs (§4, §5, issue #96).
type ProcedureRoot struct {
	File string
	Root *yaml.Node
}

// procedureGraphStep is the one fact the two Cadence rules read off a Step
// declared directly in a procedures/ file — whether its Operation is
// run-once, and whether its Operation declares secret: output — read once
// while building the graph rather than re-resolved on every walk (§4, §5,
// issue #96).
type procedureGraphStep struct {
	runOnce   bool
	hasSecret bool
}

// procedureGraphInvocation is one nested invocation entry a procedures/
// file declares directly: the Procedure it names, and the position a
// composition fault found through it is cited at (§4, §5, issue #96).
type procedureGraphInvocation struct {
	procedureName string
	line, column  int
	field         string
}

// procedureGraphInfo is what the transitive walk needs off one
// procedures/ file, read once per repository pass rather than reparsed per
// walk (§4, §5, issue #96): its own file, for citation; its own declared
// targets: envelope, the raw names as authored on procedureEnvelope's own
// rule; every Step it declares directly, whichever facts the two Cadence
// rules read off it; every nested invocation it declares directly; and its
// own cadence: line, where it declares one.
type procedureGraphInfo struct {
	file          string
	targets       map[string]bool
	steps         []procedureGraphStep
	invocations   []procedureGraphInvocation
	hasCadence    bool
	cadenceLine   int
	cadenceColumn int
}

// BuildProcedureGraph reads roots into the namespace CheckProcedureGraph
// walks: one procedureGraphInfo per procedures/ file whose procedure: is a
// legible scalar, on BuildProcedureIndex's own rule — a root whose
// procedure: is absent or illegible contributes no entry, having already
// earned its own schema-mismatch or name-mismatch and being no name a
// nested invocation could ever resolve to (§4, §5, issue #96). providers
// and definitions are the repository-wide namespaces a Step's definition:
// and operation: resolve against, needed here only to read a Step's own
// Kind, Repeatability and secret-output fact — every other fact about a
// Step is procedure.go's own file-local checks' business, not this walk's.
func BuildProcedureGraph(roots []ProcedureRoot, providers ProviderIndex, definitions DefinitionIndex) map[string]procedureGraphInfo {
	graph := map[string]procedureGraphInfo{}
	for _, r := range roots {
		nameVal := topLevelFields(r.Root, "procedure")["procedure"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		graph[nameVal.Value] = buildProcedureGraphInfo(r.File, r.Root, providers, definitions)
	}
	return graph
}

// buildProcedureGraphInfo reads one procedures/ file's own root into its
// procedureGraphInfo (§4, §5, issue #96).
func buildProcedureGraphInfo(file string, root *yaml.Node, providers ProviderIndex, definitions DefinitionIndex) procedureGraphInfo {
	info := procedureGraphInfo{file: file, targets: readDeclaredTargetNames(root)}

	fields := topLevelFields(root, "cadence", "steps")
	if cadenceVal := fields["cadence"]; cadenceVal != nil {
		info.hasCadence = true
		info.cadenceLine, info.cadenceColumn = position(cadenceVal)
	}

	stepsVal := fields["steps"]
	if stepsVal == nil || stepsVal.Kind != yaml.SequenceNode {
		return info
	}
	for i, entry := range stepsVal.Content {
		field := fmt.Sprintf("steps[%d]", i)
		entryFields := topLevelFields(entry, "definition", "operation", "procedure")
		if procVal := entryFields["procedure"]; procVal != nil {
			name, ok := resolveScalar(procVal)
			if !ok {
				continue
			}
			line, column := position(procVal)
			info.invocations = append(info.invocations, procedureGraphInvocation{
				procedureName: name, line: line, column: column, field: field + ".procedure",
			})
			continue
		}
		info.steps = append(info.steps, procedureGraphStepFacts(entryFields, providers, definitions))
	}
	return info
}

// procedureGraphStepFacts resolves one Step entry's definition: and
// operation: against definitions and providers and reads the Operation's
// own RunOnce and HasSecret facts off it. A Step whose definition: or
// operation: does not resolve contributes false for both — the resolution
// fault is already reported by procedure.go's own file-local checks, and an
// unresolved Operation carries no Kind or Repeatability for this walk to
// read (§4, §5, issue #96).
func procedureGraphStepFacts(fields map[string]*yaml.Node, providers ProviderIndex, definitions DefinitionIndex) procedureGraphStep {
	defName, defOK := resolveScalar(fields["definition"])
	if !defOK {
		return procedureGraphStep{}
	}
	defInfo, haveDef := definitions[defName]
	if !haveDef {
		return procedureGraphStep{}
	}
	opName, opOK := resolveScalar(fields["operation"])
	if !opOK {
		return procedureGraphStep{}
	}
	op, haveOp := providers[defInfo.ProviderName].Operations[opName]
	if !haveOp {
		return procedureGraphStep{}
	}
	return procedureGraphStep{runOnce: op.IsRunOnce(), hasSecret: op.HasSecret}
}

// procedureReach is what walkProcedure accumulates for one procedure, to
// any depth: every target name reachable through its own declared
// targets: and everything it invokes, and whether that same reach touches
// a run-once Step or a Step whose Operation declares secret: output — the
// two facts the Cadence rules read (§4, §5, issue #96).
type procedureReach struct {
	targets   map[string]bool
	runOnce   bool
	hasSecret bool
}

// walkProcedure computes name's own procedureReach, memoized in memo so a
// Procedure invoked from several places is walked once per repository pass
// rather than once per caller. visiting guards the one recursion in
// progress against a cycle: a name already being walked contributes an
// empty procedureReach rather than recursing forever, since there is
// nothing further this walk could learn from a Procedure it is already
// inside of. A name absent from graph — an invocation naming nothing,
// already reported by procedure.go's own artefact-absent — contributes an
// empty procedureReach the same way (§4, §5, issue #96).
func walkProcedure(name string, graph map[string]procedureGraphInfo, memo map[string]procedureReach, visiting map[string]bool) procedureReach {
	if r, ok := memo[name]; ok {
		return r
	}
	info, ok := graph[name]
	if !ok || visiting[name] {
		return procedureReach{targets: map[string]bool{}}
	}

	visiting[name] = true
	r := procedureReach{targets: map[string]bool{}}
	for t := range info.targets {
		r.targets[t] = true
	}
	for _, s := range info.steps {
		r.runOnce = r.runOnce || s.runOnce
		r.hasSecret = r.hasSecret || s.hasSecret
	}
	for _, inv := range info.invocations {
		child := walkProcedure(inv.procedureName, graph, memo, visiting)
		for t := range child.targets {
			r.targets[t] = true
		}
		r.runOnce = r.runOnce || child.runOnce
		r.hasSecret = r.hasSecret || child.hasSecret
	}
	delete(visiting, name)

	memo[name] = r
	return r
}

// CheckProcedureGraph walks graph and reports the two rules that need every
// procedures/ file at once (§4, §5, issue #96): an invoked Procedure's own
// transitive envelope reaching outside its caller's declared targets: —
// the composition half of envelope-exceeded, cited at the invocation that
// makes the composition, procedure.go's own file-local checks having
// already covered a Step directly in a file reaching past its own
// Procedure's declared envelope — and, on a Procedure declaring a Cadence,
// a reachable run-once Step (cadence-run-once) or a reachable Step whose
// Operation declares secret: output (cadence-secret-output), each cited at
// the cadence: line of the Procedure declaring the recurrence rather than
// wherever in the graph the fact was read, since that line is the one an
// author can act on: narrow the Cadence away, or edit the Step.
func CheckProcedureGraph(graph map[string]procedureGraphInfo) []problem.Problem {
	memo := map[string]procedureReach{}
	var problems []problem.Problem

	for _, name := range sortedProcedureNames(graph) {
		info := graph[name]
		for _, inv := range info.invocations {
			child := walkProcedure(inv.procedureName, graph, memo, map[string]bool{})
			for _, target := range sortedNames(child.targets) {
				if info.targets[target] {
					continue
				}
				problems = append(problems, problem.Problem{
					File: info.file, Line: inv.line, Column: inv.column, Field: inv.field,
					ErrorCode: CodeEnvelopeExceeded,
					Message:   fmt.Sprintf("procedure: %s's transitive envelope reaches %s, outside this Procedure's own declared targets:", inv.procedureName, target),
				})
			}
		}

		if !info.hasCadence {
			continue
		}
		reach := walkProcedure(name, graph, memo, map[string]bool{})
		if reach.runOnce {
			problems = append(problems, problem.Problem{
				File: info.file, Line: info.cadenceLine, Column: info.cadenceColumn, Field: "cadence",
				ErrorCode: CodeCadenceRunOnce,
				Message:   "cadence: reaches a run-once Step at some depth — a Cadence over a run-once Step declares a recurrence with a lifespan of one occurrence",
			})
		}
		if reach.hasSecret {
			problems = append(problems, problem.Problem{
				File: info.file, Line: info.cadenceLine, Column: info.cadenceColumn, Field: "cadence",
				ErrorCode: CodeCadenceSecretOutput,
				Message:   "cadence: reaches a Step whose Operation declares secret: output at some depth — the workflow project generates supplies no Secret sink, so the Refusal would land at every occurrence",
			})
		}
	}
	return problems
}

// sortedProcedureNames returns graph's own keys in byte order, so
// CheckProcedureGraph's own output is deterministic across a map's
// unordered iteration.
func sortedProcedureNames(graph map[string]procedureGraphInfo) []string {
	names := make([]string, 0, len(graph))
	for name := range graph {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// sortedNames returns set's own members in byte order, on
// sortedProcedureNames' own rule.
func sortedNames(set map[string]bool) []string {
	names := make([]string, 0, len(set))
	for name := range set {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
