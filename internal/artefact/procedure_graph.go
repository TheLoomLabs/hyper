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
//
// A fourth rule is here because it is about the graph rather than about
// anything read through it: procedure-cycle, an invocation graph that closes
// on itself (issue #146). The three above tolerate a cycle — a name already
// being walked contributes nothing further — which is right for them and
// silent, and §6 states the graph is acyclic rather than hoping it is.
package artefact

import (
	"fmt"
	"sort"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// CodeCadenceRunOnce is the code a Procedure declaring a Cadence earns for
// reaching a run-once Step at any depth: a Cadence over a run-once Step
// declares a recurrence with a lifespan of one occurrence, which is not a
// thing an author can have meant (§4, §5, ADR-0038, issue #96).
const CodeCadenceRunOnce = "cadence-run-once"

// CodeCadenceSecretOutput is the code a Procedure declaring a Cadence earns
// for reaching a Step whose Operation declares secret: output at any
// depth: such a Step Refuses on any Run at all while no hyper writes a
// Secret sink, and once one does the workflow project generates will still
// supply none, so where a Cadence carries it the Refusal lands at every
// occurrence and the Procedure works never (§4, §5, ADR-0077, ADR-0146,
// issues #96, #266). It is its own code rather than a
// second cause folded into cadence-run-once: a reader handed
// cadence-run-once on a secret: clash edits a repeatability: that is
// correct.
const CodeCadenceSecretOutput = "cadence-secret-output"

// CodeProcedureCycle is the code an invocation graph that closes on itself
// earns: a Procedure invoking one it is already inside of, directly or
// through a chain of any length. §6 states the graph is static and that a
// cycle is rejected before the first Step, and this is the walk that can
// state it — every procedures/ file at once, and the chain in hand as it
// recurses. It is cited at the invocation entry that closes the loop rather
// than at the Procedure the walk entered at, that entry being the line an
// author edits to break it (§4, §6, ADR-0002, issue #146).
const CodeProcedureCycle = "procedure-cycle"

// ProcedureRoot pairs one procedures/ file's own relative path with its
// already-parsed root — what BuildProcedureGraph reads the transitive
// walk's own per-file facts off of, needing every procedures/ file's own
// name to know where a citation belongs (§4, §5, issue #96).
type ProcedureRoot struct {
	File string
	Root *yaml.Node
}

// procedureGraphStep is what this walk reads off a Step declared directly in
// a procedures/ file, read once while building the graph rather than
// re-resolved on every walk (§4, §5, issues #96, #176): whether its
// Operation is run-once and whether it declares secret: output, which are
// the two Cadence rules'; the pair it binds and whether its Operation's Kind
// is read, which are the projection's.
type procedureGraphStep struct {
	runOnce   bool
	hasSecret bool
	// pair is the (Definition, Target) pair the Step makes, as authored,
	// and the zero Pair where the Step names neither — the pair the
	// generated workflow's env: block derives from (§10).
	pair store.Pair
	// effects is whether the Step's Operation declares a Kind other than
	// read. A Step whose binding does not resolve carries false, on this
	// walk's own rule for every fact it reads: an Operation it could not
	// resolve declares nothing for it to read, the resolution fault is
	// already reported, and `project` writes nothing where `check`
	// reports anything (ADR-0064).
	effects bool
}

// ProcedureReach is what one Procedure reaches, to any depth, that nothing
// about a single procedures/ file can answer: the pairs its Steps bind, and
// whether every Step it reaches is a read (§10, issue #176).
//
// Both are the projection's, and both come off the walk the two Cadence
// rules already ride rather than a traversal of their own: a Procedure's
// reach is one question, and asking it twice is where the day comes that the
// env: block and the concurrency group disagree about what a Procedure runs.
type ProcedureReach struct {
	// Pairs is each (Definition, Target) pair once, in the Steps' own
	// order and a nested invocation's after its caller's — deterministic,
	// so one repository answers one way. The env: block orders itself by
	// variable name and does not read this order (§10); what it needs is
	// that a walk of one repository twice is a walk of one repository.
	//
	// They are store.Pair because they are §6's own noun, and the same
	// pairs a Run's credential pass and the Store's schema test are
	// quantified over — one concept, and the walk that finds them early
	// answers in the type the walk that binds them uses.
	Pairs []store.Pair
	// EveryStepReads is whether every Step reachable from this Procedure
	// declares kind: read — the fact deciding whether its workflow takes
	// the Store's concurrency group. Reachability decides it and not the
	// Procedure's own declared Steps: a read-looking Procedure that
	// invokes an effectful one effects (§10).
	EveryStepReads bool
}

// Reaches answers name's own ProcedureReach, walking every procedures/ file
// the graph holds to whatever depth name's invocations run to.
//
// A name the graph does not hold reaches nothing and reads everything it
// reaches, which is the same answer a Procedure with no Steps gives and the
// same answer this walk gives every name it cannot resolve: there is no Step
// under it that effects, the fault is already reported where it was authored,
// and `project` writes nothing where `check` reports anything (ADR-0064).
func (g ProcedureGraph) Reaches(name string) ProcedureReach {
	reach := walkProcedure(name, g, map[string]walkedReach{}, map[string]bool{})
	return ProcedureReach{Pairs: reach.pairs, EveryStepReads: !reach.effects}
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

// ProcedureGraph is every procedures/ file's own per-file facts, keyed by the
// name it declares: the namespace the transitive walk recurses through, and
// what a review's own gutter reads a nested invocation's transitive envelope
// off of (§4, §5, §8, issues #96, #120). Its members are unexported because
// what a caller does with the graph is walk it, never read one file's entry.
type ProcedureGraph map[string]procedureGraphInfo

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
func BuildProcedureGraph(roots []ProcedureRoot, providers ProviderIndex, definitions DefinitionIndex) ProcedureGraph {
	graph := ProcedureGraph{}
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
		entryFields := topLevelFields(entry, "definition", "operation", "target", "procedure", "require")
		// A Requirement binds nothing, invokes nothing and calls nothing,
		// so it contributes to neither half of this walk: no pair, no
		// Kind, no Repeatability and no secret output. It is a Step of
		// neither the graph nor the sequence (§3, ADR-0116).
		if entryFields["require"] != nil {
			continue
		}
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
	step := procedureGraphStep{pair: pairOf(fields)}

	defName, defOK := resolveScalar(fields["definition"])
	if !defOK {
		return step
	}
	defInfo, haveDef := definitions[defName]
	if !haveDef {
		return step
	}
	opName, opOK := resolveScalar(fields["operation"])
	if !opOK {
		return step
	}
	op, haveOp := providers[defInfo.ProviderName].Operations[opName]
	if !haveOp {
		return step
	}
	step.runOnce = op.IsRunOnce()
	step.hasSecret = op.HasSecret
	step.effects = op.Kind != "read"
	return step
}

// pairOf is the pair one Step entry names, as authored and unresolved, and
// the zero Pair where either half is absent or illegible — a Step that does
// not name both names no pair, and what is wrong with it is `check`'s to
// report rather than this walk's to repeat (ADR-0064).
func pairOf(fields map[string]*yaml.Node) store.Pair {
	definition, defOK := resolveScalar(fields["definition"])
	target, targetOK := resolveScalar(fields["target"])
	if !defOK || !targetOK || definition == "" || target == "" {
		return store.Pair{}
	}
	return store.Pair{Definition: definition, Target: target}
}

// walkedReach is what walkProcedure accumulates for one procedure, to
// any depth: every target name reachable through its own declared
// targets: and everything it invokes; whether that same reach touches a
// run-once Step or a Step whose Operation declares secret: output — the two
// facts the Cadence rules read; and the pairs it binds and whether anything
// it reaches effects — the two the projection reads (§4, §5, §10, issues
// #96, #176).
type walkedReach struct {
	targets   map[string]bool
	runOnce   bool
	hasSecret bool
	// pairs is each reachable (Definition, Target) pair once, in the
	// Steps' own order.
	pairs []store.Pair
	// effects is whether any reachable Step declares a Kind other than
	// read.
	//
	// It is the negation Reaches answers with, and it is accumulated in
	// this direction because that is the direction reach composes in: a
	// caller effects where anything it reaches does, and an empty walk
	// carries the identity rather than a claim — which is the same shape
	// runOnce and hasSecret compose in, and the same reading each of the
	// three gives a name it could not resolve.
	effects bool
}

// walkProcedure computes name's own walkedReach, memoized in memo so a
// Procedure invoked from several places is walked once per repository pass
// rather than once per caller. visiting guards the one recursion in
// progress against a cycle: a name already being walked contributes an
// empty walkedReach rather than recursing forever, since there is
// nothing further this walk could learn from a Procedure it is already
// inside of. A name absent from graph — an invocation naming nothing,
// already reported by procedure.go's own artefact-absent — contributes an
// empty walkedReach the same way (§4, §5, issue #96).
func walkProcedure(name string, graph ProcedureGraph, memo map[string]walkedReach, visiting map[string]bool) walkedReach {
	if r, ok := memo[name]; ok {
		return r
	}
	info, ok := graph[name]
	if !ok || visiting[name] {
		return walkedReach{targets: map[string]bool{}}
	}

	visiting[name] = true
	r := walkedReach{targets: map[string]bool{}}
	held := map[store.Pair]bool{}
	for t := range info.targets {
		r.targets[t] = true
	}
	for _, s := range info.steps {
		r.runOnce = r.runOnce || s.runOnce
		r.hasSecret = r.hasSecret || s.hasSecret
		r.effects = r.effects || s.effects
		r.pairs = appendPair(r.pairs, held, s.pair)
	}
	for _, inv := range info.invocations {
		child := walkProcedure(inv.procedureName, graph, memo, visiting)
		for t := range child.targets {
			r.targets[t] = true
		}
		r.runOnce = r.runOnce || child.runOnce
		r.hasSecret = r.hasSecret || child.hasSecret
		r.effects = r.effects || child.effects
		for _, pair := range child.pairs {
			r.pairs = appendPair(r.pairs, held, pair)
		}
	}
	delete(visiting, name)

	memo[name] = r
	return r
}

// checkProcedureCycles reports every invocation entry that closes a cycle.
//
// It is its own depth-first walk rather than a report bolted onto
// walkProcedure, and for a reason that outlives this issue: walkProcedure is
// memoized and answers one question per name, so it visits a Procedure once
// per repository pass and cannot be relied on to have arrived through the
// caller a cycle closes at. This walk carries `visiting` as a **stack** — the
// chain it is inside of right now — and `walked` as everything already
// explored, so each edge of the graph is examined exactly once: a back edge
// into the chain is the cycle and is reported there, and an edge into a
// Procedure already explored is a diamond and is not.
//
// A name absent from graph contributes nothing, on walkProcedure's own rule:
// an invocation naming nothing is procedure.go's artefact-absent, and a name
// that resolves to no file closes no loop.
func checkProcedureCycles(graph ProcedureGraph) []problem.Problem {
	var problems []problem.Problem
	walked := map[string]bool{}
	visiting := map[string]bool{}

	var walk func(name string)
	walk = func(name string) {
		info, ok := graph[name]
		if !ok {
			return
		}
		visiting[name] = true
		for _, inv := range info.invocations {
			if visiting[inv.procedureName] {
				problems = append(problems, problem.Problem{
					File: info.file, Line: inv.line, Column: inv.column, Field: inv.field,
					ErrorCode: CodeProcedureCycle,
					Message:   fmt.Sprintf("procedure: %s already reaches %s, so this invocation closes a cycle — the invocation graph is static and no Run performs one", inv.procedureName, name),
				})
				continue
			}
			if walked[inv.procedureName] {
				continue
			}
			walk(inv.procedureName)
		}
		delete(visiting, name)
		walked[name] = true
	}

	for _, name := range sortedProcedureNames(graph) {
		if !walked[name] {
			walk(name)
		}
	}
	return problems
}

// CheckProcedureGraph walks graph and reports the rules that need every
// procedures/ file at once (§4, §5, issues #96, #146): the graph closing on
// itself — a Procedure invoking one it is already inside of, which is
// procedure-cycle and is collected first, being the one fault about the shape
// of the graph rather than about anything reachable through it (what order a
// surface renders it in is problem.Sort's, over the file and line every row
// carries); an invoked
// Procedure's own transitive envelope reaching outside its caller's declared
// targets: — the composition half of envelope-exceeded, cited at the
// invocation that makes the composition, procedure.go's own file-local
// checks having already covered a Step directly in a file reaching past its
// own Procedure's declared envelope — and, on a Procedure declaring a Cadence,
// a reachable run-once Step (cadence-run-once) or a reachable Step whose
// Operation declares secret: output (cadence-secret-output), each cited at
// the cadence: line of the Procedure declaring the recurrence rather than
// wherever in the graph the fact was read, since that line is the one an
// author can act on: narrow the Cadence away, or edit the Step.
func CheckProcedureGraph(graph ProcedureGraph) []problem.Problem {
	memo := map[string]walkedReach{}
	problems := checkProcedureCycles(graph)

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
func sortedProcedureNames(graph ProcedureGraph) []string {
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

// appendPair adds pair to pairs where it is one not already held, and
// answers what to keep. The zero Pair is no pair and is dropped: a Step
// naming neither a Definition nor a Target binds nothing.
//
// **Each once** is the same rule internal/run's own pairsOf folds over a
// Run's sequenced Steps, for the same reason — a Procedure of ten Steps
// against one pair makes one pair — and the two are two folds rather than
// one because they run over different things at different times: this over
// procedures/ files' nodes before a Run exists, that over the flattened
// sequence a Run performs.
//
// It appends into a slice of the caller's own rather than into a memoized
// one — walkProcedure builds each walkedReach's pairs fresh, so a Procedure
// invoked from two places contributes its pairs to both callers without
// either being able to write into the answer the memo holds.
func appendPair(pairs []store.Pair, held map[store.Pair]bool, pair store.Pair) []store.Pair {
	if pair == (store.Pair{}) || held[pair] {
		return pairs
	}
	held[pair] = true
	return append(pairs, pair)
}
