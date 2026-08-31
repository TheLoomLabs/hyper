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
// EnvelopeTargets are the Targets that line declares, in the file's own order
// and as written, including one that resolves to no declaration. It is the
// envelope the mark is a verdict on, which is what an index of that verdict
// states rather than restating the verdict (§8, §12): the values are the fact
// and how they are punctuated is the rendering surface's.
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
	EnvelopeLine    int
	EnvelopeTargets []string
	EnvelopeHolds   bool
	Steps           []StepMark
}

// StepMark is one entry of a Procedure's steps: as the gutter marks it, in the
// order the file declares them.
//
// Line is the line the entry opens on, which is the line that binds the claim:
// a Step's own `- id:`, and a nested invocation's.
//
// ID is the id: the entry declares, "" where it declares none legibly. It is
// the coordinate a flag citing this line carries — a Step is named by its id:
// on every surface in the tool — and no part of the mark: the gutter stands
// beside the line and needs no name for it (§8, §12).
//
// Unresolved is §8's one mark for four absences: a definition:, an operation:,
// a bound Provider or a nested procedure: that names nothing. It is one name
// and not four because the gutter marks and does not classify — which name
// failed is FLAGS' text — and a mark that fired carries nothing else: no Kind
// to read, no opacity, and no contribution to the envelope check.
//
// Absent is which of those four it was, set exactly where Unresolved is: the
// key that carried the name, the name itself, and the Provider an Operation was
// looked for on. It is the classification the gutter does not make and the flag
// does, and it is carried as the facts rather than as a sentence — what a
// rendering says about them is the rendering surface's (§8, §12).
//
// Kind is the Operation's declared Kind, read from the Manifest and never
// inferred from the Operation's name (§12). It is "" on a nested invocation,
// which invokes no Operation and has no blast radius of its own to declare.
//
// Operation is the operation: the Step names, which resolved: a flag indexing
// this line names what is being invoked, where the marker beside it carries the
// Kind that invocation reaches. It is "" on a nested invocation for Kind's own
// reason.
//
// Bounded is whether the Step declares a bound: at all, whatever its Kind. It
// is the fact rather than the mark: a mutate Step with none is marked mutate!
// and a destroy Step with none is bound-missing, a static check's to report,
// so the rendering of this fact differs by Kind and the reading does not (§4).
//
// Bound is that Bound's value as the Step wrote it, "" where the Step declares
// none or declares one this cannot read. The two are one key read for two
// questions — whether a Bound stands behind the Step, and what it says — and a
// value that would not read is a bound: check's to report and a fact this
// states nothing about (ADR-0064).
//
// Opaque is whether the Operation's request uses an Opaque Capability, which is
// read off the request and declared beside no Operation anywhere (§12).
//
// Targets is what the mark binds: the one Target a Step's target: names, as
// written; or, on a nested invocation, the transitive envelope it reaches, in
// name order — the envelope §3 states, walked to any depth. It is empty on a
// Requirement, which reaches nothing.
//
// Requirement is whether the entry is one: a `require:` and an `id:`, binding
// nothing and invoking nothing. Every other member of the mark is empty on
// one, and that is the gutter's supply rule holding rather than a roster left
// short — a Requirement's whole content is authored on the line the reviewer
// is reading, so there is nothing `hyper` derived for a marker to carry (§8,
// ADR-0026, ADR-0116).
type StepMark struct {
	Line        int
	ID          string
	Requirement bool
	Unresolved  bool
	Absent      AbsentName
	Kind        string
	Operation   string
	Bounded     bool
	Bound       string
	Opaque      bool
	Targets     []string
}

// AbsentName is the name that resolved to nothing on a Step the gutter marks
// unresolved: the key it was written under — definition, operation, provider or
// procedure — the name it named, and, where the key is operation, the Provider
// whose Manifest it was looked for in.
//
// Name is "" where the key carried nothing legible to resolve at all, which is
// a different absence from a name that resolved to nothing and is why the name
// is carried rather than assumed present.
//
// The key is the authoring format's own, and provider is one of the four
// although no Step writes it: a Step's definition: resolves and the Definition
// it found names a Provider that does not, which is a name on the Step's own
// line failing one hop out (§3, §4).
type AbsentName struct {
	Key      string
	Name     string
	Provider string
}

// The four keys an AbsentName is written under, which are the authoring
// format's own and not §12's kind: values — three of them spell a kind and the
// fourth names a key inside a Manifest. They are stated here because the reader
// that sets one and the surface that renders it are in two packages, and a
// fifth absence arriving should not be two literals nothing ties together.
const (
	KeyDefinition = "definition"
	KeyOperation  = "operation"
	KeyProvider   = "provider"
	KeyProcedure  = "procedure"
)

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
	marks := ProcedureMarks{
		EnvelopeLine:    TopLevelKeyLine(root, "targets"),
		EnvelopeTargets: scalarSequence(topLevelFields(root, "targets")["targets"]),
		EnvelopeHolds:   true,
	}

	stepsVal := topLevelFields(root, "steps")["steps"]
	if stepsVal == nil || stepsVal.Kind != yaml.SequenceNode {
		return marks
	}

	memo := map[string]walkedReach{}
	for _, entry := range stepsVal.Content {
		line, _ := position(entry)
		fields := topLevelFields(entry, "id", "definition", "operation", "target", "bound", "procedure", "require")

		id, _ := resolveScalar(fields["id"])

		var mark StepMark
		if fields["require"] != nil {
			// A Requirement contributes to neither half of the envelope:
			// it declares no Kind, so reading its empty one against the
			// union would put every Procedure that halts on one outside
			// its own envelope, and it binds no Target to compare.
			marks.Steps = append(marks.Steps, StepMark{Line: line, ID: id, Requirement: true})
			continue
		}
		if fields["procedure"] != nil {
			mark = invocationMark(line, id, fields["procedure"], graph, memo)
		} else {
			mark = stepMark(line, id, fields, providers, definitions)
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
func stepMark(line int, id string, fields map[string]*yaml.Node, providers ProviderIndex, definitions DefinitionIndex) StepMark {
	unresolved := func(absent AbsentName) StepMark {
		return StepMark{Line: line, ID: id, Unresolved: true, Absent: absent}
	}

	defName, ok := resolveScalar(fields["definition"])
	if !ok {
		return unresolved(AbsentName{Key: KeyDefinition})
	}
	defInfo, haveDef := definitions[defName]
	if !haveDef {
		return unresolved(AbsentName{Key: KeyDefinition, Name: defName})
	}
	provider, haveProvider := providers[defInfo.ProviderName]
	if !haveProvider {
		return unresolved(AbsentName{Key: KeyProvider, Name: defInfo.ProviderName})
	}
	opName, ok := resolveScalar(fields["operation"])
	if !ok {
		return unresolved(AbsentName{Key: KeyOperation, Provider: defInfo.ProviderName})
	}
	op, haveOp := provider.Operations[opName]
	if !haveOp {
		return unresolved(AbsentName{Key: KeyOperation, Name: opName, Provider: defInfo.ProviderName})
	}

	bound, _ := resolveScalar(fields["bound"])
	mark := StepMark{
		Line:      line,
		ID:        id,
		Kind:      op.Kind,
		Operation: opName,
		Bounded:   fields["bound"] != nil,
		Bound:     bound,
		Opaque:    op.IsShell,
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
func invocationMark(line int, id string, procVal *yaml.Node, graph ProcedureGraph, memo map[string]walkedReach) StepMark {
	name, ok := resolveScalar(procVal)
	if !ok {
		return StepMark{Line: line, ID: id, Unresolved: true, Absent: AbsentName{Key: KeyProcedure}}
	}
	if _, known := graph[name]; !known {
		return StepMark{Line: line, ID: id, Unresolved: true, Absent: AbsentName{Key: KeyProcedure, Name: name}}
	}
	return StepMark{Line: line, ID: id, Targets: sortedNames(walkProcedure(name, graph, memo, map[string]bool{}).targets)}
}
