package artefact

import (
	"gopkg.in/yaml.v3"
)

// ProcedureMarks is what a review derives about a Procedure's own lines — the
// supply behind §8's marker column, and the facts a reviewer with the file open
// in a diff cannot see because none of them are in the file: an Operation's
// Kind is declared in a Manifest two directories away, its opacity is declared
// nowhere at all, and the envelope check quantifies over every Step's target:
// at once (§8, issue #120).
//
// It reads and does not judge, on ReadManifestFacts's own rule: a name that
// resolves to nothing is marked as the absence it is rather than refused, a
// review not running check and not declining (ADR-0064). Nothing an author
// wrote is in here either — a comment is source and is read for nothing (§8).
//
// EnvelopeLine is the targets: line the envelope mark stands beside, and 0
// where the Procedure declares no targets: at all — there being no line to mark
// and the missing key being check's to report.
//
// EnvelopeHolds is whether every Step and every nested invocation this reader
// could derive an envelope contribution from stays inside the declared
// envelope. The comparisons are envelope-exceeded's — a bound Target outside
// the declared targets:, and a Kind outside the union those targets accept —
// together with the transitive half CheckProcedureGraph runs, over the graph
// that check walks rather than a second derivation of a Procedure's reach.
//
// What it quantifies over is not what check quantifies over, and the difference
// is a rendering rule rather than a disagreement. checkStepEnvelope reports an
// out-of-envelope target: whatever else about the Step failed to resolve; this
// mark reads nothing off a Step it derived nothing else about, §8 fixing that
// an unresolved Step carries no Kind, no opacity and no envelope contribution.
// So a Step whose definition: names nothing and whose target: is outside the
// envelope renders `unresolved` beside `envelope ✓` here and earns
// envelope-exceeded from check — which is the two surfaces doing their own
// jobs: check reports what is wrong with the artefact, and a review annotates
// what hyper derived from these lines (ADR-0026, ADR-0064).
type ProcedureMarks struct {
	EnvelopeLine  int
	EnvelopeHolds bool
	Steps         []StepMark
}

// StepMark is one entry of a Procedure's steps: as the gutter marks it, in the
// order the file declares them.
//
// Line is the line the entry opens on, which is the line that binds the claim:
// a Step's own `- id:`, and a nested invocation's.
//
// Unresolved is §8's one mark for four absences: a definition:, an operation:,
// a bound Provider or a nested procedure: that names nothing. It is one name
// and not four because the gutter marks and does not classify — which name
// failed is FLAGS' text — and a mark that fired carries nothing else: no Kind
// to read, no opacity, and no contribution to the envelope check.
//
// Kind is the Operation's declared Kind, read from the Manifest and never
// inferred from the Operation's name (§12). It is "" on a nested invocation,
// which invokes no Operation and has no blast radius of its own to declare.
//
// Bounded is whether the Step declares a bound: at all, whatever its Kind. It
// is the fact rather than the mark: a mutate Step with none is marked mutate!
// and a destroy Step with none is bound-missing, a static check's to report,
// so the rendering of this fact differs by Kind and the reading does not (§4).
//
// Opaque is whether the Operation's request uses an Opaque Capability, which is
// read off the request and declared beside no Operation anywhere (§12).
//
// Targets is what the mark binds: the one Target a Step's target: names, as
// written; or, on a nested invocation, the transitive envelope it reaches, in
// name order — the envelope §3 states, walked to any depth.
type StepMark struct {
	Line       int
	Unresolved bool
	Kind       string
	Bounded    bool
	Opaque     bool
	Targets    []string
}

// ReadProcedureMarks reads a Procedure's own root into the marks its gutter
// carries. providers, definitions and graph are the namespaces a Step's
// definition: and operation: and a nested invocation's procedure: resolve
// against — the same three CheckProcedure and CheckProcedureGraph resolve them
// against, so what the gutter marks and what check refuses are one reading of
// one repository. targets is the namespace this Procedure's own declared
// targets: resolve against, which is where the Kind half of the envelope comes
// from.
//
// It is a Procedure's reader and answers nothing on any other artefact: only a
// Procedure has Steps, so only a Procedure carries a Kind, a Target or an
// envelope mark (§8).
func ReadProcedureMarks(root *yaml.Node, providers ProviderIndex, definitions DefinitionIndex, targets TargetIndex, graph ProcedureGraph) ProcedureMarks {
	declaredTargets, declaredKinds := procedureEnvelope(root, targets)
	marks := ProcedureMarks{EnvelopeLine: topLevelKeyLine(root, "targets"), EnvelopeHolds: true}

	stepsVal := topLevelFields(root, "steps")["steps"]
	if stepsVal == nil || stepsVal.Kind != yaml.SequenceNode {
		return marks
	}

	memo := map[string]procedureReach{}
	for _, entry := range stepsVal.Content {
		line, _ := position(entry)
		fields := topLevelFields(entry, "definition", "operation", "target", "bound", "procedure")

		var mark StepMark
		if fields["procedure"] != nil {
			mark = invocationMark(line, fields["procedure"], graph, memo)
		} else {
			mark = stepMark(line, fields, providers, definitions)
			// The Kind half of the envelope, which only a Step has: a
			// nested invocation declares none, and reading its empty
			// Kind against the union would put every Procedure that
			// invokes one outside its own envelope.
			if !mark.Unresolved && !declaredKinds[mark.Kind] {
				marks.EnvelopeHolds = false
			}
		}
		// The Target half, off the mark's own Targets — which is the one it
		// binds on a Step, its transitive envelope on an invocation, and
		// empty on a mark that resolved to nothing and contributes none.
		for _, reached := range mark.Targets {
			if !declaredTargets[reached] {
				marks.EnvelopeHolds = false
			}
		}
		marks.Steps = append(marks.Steps, mark)
	}
	return marks
}

// stepMark reads one steps: entry that is a Step. A Step that resolved carries
// the Target it binds and one that did not carries none, which is also its
// envelope contribution: an unresolved Step contributes nothing for the reason
// it carries no Kind — the derivation the mark would have carried is the
// derivation the check would have read (§8).
func stepMark(line int, fields map[string]*yaml.Node, providers ProviderIndex, definitions DefinitionIndex) StepMark {
	unresolved := StepMark{Line: line, Unresolved: true}

	defName, ok := resolveScalar(fields["definition"])
	if !ok {
		return unresolved
	}
	defInfo, haveDef := definitions[defName]
	if !haveDef {
		return unresolved
	}
	provider, haveProvider := providers[defInfo.ProviderName]
	if !haveProvider {
		return unresolved
	}
	opName, ok := resolveScalar(fields["operation"])
	if !ok {
		return unresolved
	}
	op, haveOp := provider.Operations[opName]
	if !haveOp {
		return unresolved
	}

	mark := StepMark{
		Line:    line,
		Kind:    op.Kind,
		Bounded: fields["bound"] != nil,
		Opaque:  op.IsShell,
	}
	if targetName, named := resolveScalar(fields["target"]); named {
		mark.Targets = []string{targetName}
	}
	// A Step that names no Target binds none this surface can mark. The
	// missing key is schema-mismatch and check's to report; what stands here
	// is the Kind, which resolved perfectly well.
	return mark
}

// invocationMark reads one steps: entry that is a nested Procedure invocation.
// What a Procedure's own line derives is the transitive envelope §3 states —
// the Targets everything it invokes may touch, to any depth — and where its
// procedure: names nothing there is no walk to make and the mark is
// unresolved, which is the same absence one level up (§8).
//
// The walk is walkProcedure's, which is the one CheckProcedureGraph makes for
// envelope-exceeded: this makes it again over the same graph rather than
// deriving a Procedure's reach a second way, so the gutter and the check
// cannot disagree about what a Procedure reaches. memo is the caller's, so a
// Procedure invoked from several lines of one file is walked once.
func invocationMark(line int, procVal *yaml.Node, graph ProcedureGraph, memo map[string]procedureReach) StepMark {
	name, ok := resolveScalar(procVal)
	if !ok {
		return StepMark{Line: line, Unresolved: true}
	}
	if _, known := graph[name]; !known {
		return StepMark{Line: line, Unresolved: true}
	}
	return StepMark{Line: line, Targets: sortedNames(walkProcedure(name, graph, memo, map[string]bool{}).targets)}
}
