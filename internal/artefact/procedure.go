// This file is the Procedure's own schema, the Step and the nested
// invocation that share `steps:`, the reference grammar a value position
// embeds a path in, and the three load-time rules issue #94 lands: the
// resolution positions `definition:`, `procedure:`, `operation:` and a
// reference's two halves add; the command, `hyper`'s own `shell` Provider
// knowing nothing of it so the argv arrives as the Step's own `args:`; and
// a repeated `over: values:` member, the Store's own
// `record-identity-collision` fired one Run earlier, against an artefact
// rather than a branch (§3, §4, §12).
//
// The two keys, the Bound and the opaque `destroy` opt-ins — the authority
// a Step's binding needs against its Definition's claim and its Target's
// grant — are issue #95's, landed here: kind-not-granted,
// operation-not-claimed, target-not-claimed, bound-missing, bound-illegal
// and opaque-destroy-unscoped. A Capability its Target grants is checked
// already, in definition.go, needing no Step to exist. A predicate's own
// operand-type rules, the three `over:` forms themselves, and
// bound-exceeded — the one Bound check a `values:` list's authored length
// decides — are #97's. This file admits `over:` and `when:` as open shapes
// beyond what opaque-destroy-unscoped and a repeated `values:` member read,
// leaving the rest for the ticket that grows this package next.
package artefact

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// CodeSeriesReference is the code a reference naming an earlier Step whose
// bound Operation's declared Record cardinality is series earns — pairing
// an expanding Step against a stored series is a join by identity between
// two Record series, and no such join is ever performed (§3, §4).
const CodeSeriesReference = "series-reference"

// CodeCommandMalformed is the code a shell Step's command: earns for being
// empty, there being no executable to name, or for naming its executable —
// the first member, the reach axis — by reference rather than by literal
// (§3, §4, ADR-0051). It is its own code and never a widening of
// hole-illegal: command: is never a template hole and could not be one
// (§3), and a reader handed hole-illegal here would go looking for one.
const CodeCommandMalformed = "command-malformed"

// CodeRecordIdentityCollision is the code two members of one `over: values:`
// list that are one identity under a case-insensitive fold earn at load —
// the Store's own check and the Store's own code, fired here one Run
// earlier, against an artefact rather than a branch (§3, §4, §8).
const CodeRecordIdentityCollision = "record-identity-collision"

// CodeKindNotGranted is the code a Step whose bound Operation's own Kind is
// not in the intersection of its Definition's claimed Kind and its bound
// Target's accepted Kinds earns — both authored, neither derived, so a
// claim of "never destroys" is a fact the reviewer can trust rather than
// the Manifest's word for it (§4, §5, issue #95).
const CodeKindNotGranted = "kind-not-granted"

// CodeOperationNotClaimed is the code a destroy Step whose Operation is not
// named among its Definition's destroy: claims earns — granularity follows
// severity, so read and mutate check at Kind level (kind-not-granted) and
// destroy checks by name (§4, §5, issue #95).
const CodeOperationNotClaimed = "operation-not-claimed"

// CodeTargetNotClaimed is the code a Step binding a Target its Definition's
// targets: list does not name earns — its own member rather than a
// widening of operation-not-claimed, since a reader handed that code on a
// target: line would go looking at destroy:, which is the wrong edit
// (§3, §4, issue #95).
const CodeTargetNotClaimed = "target-not-claimed"

// CodeBoundMissing is the code a destroy Step carrying no bound: earns — an
// absent Bound means unbounded, and unbounded is refused before anything
// runs (§4, §5, issue #95).
const CodeBoundMissing = "bound-missing"

// CodeBoundIllegal is the code an opaque destroy Step carrying a bound:
// earns — the one Step that carries no Bound, a count of the commands it
// ran saying nothing about what any of them did (§4, §5, issue #95).
const CodeBoundIllegal = "bound-illegal"

// CodeOpaqueDestroyUnscoped is the code an opaque destroy Step carrying no
// over: selector earns: without one it is invoked once, has no Expansion
// to write a Tombstone under and declares no identity, so it would reach
// the world and leave nothing in the record at all (§4, §5, ADR-0053,
// issue #95).
const CodeOpaqueDestroyUnscoped = "opaque-destroy-unscoped"

// CodeEnvelopeExceeded is the code a Step outside its own Procedure's
// declared Target and Kind envelope earns, and the code an invoked
// Procedure's transitive envelope reaching outside its caller's declared one
// earns too — one code for both shapes, checked before the first Step of
// either runs so composition cannot widen blast radius by accident (§4, §5,
// issue #96).
const CodeEnvelopeExceeded = "envelope-exceeded"

// KindProcedure is the one kind: value a file in procedures/ may carry
// (§12's kind table).
const KindProcedure = "procedure"

// ProcedureDeclaration is a Procedure's own top-level schema (§3): the
// procedure: this file's name is checked against, the targets: envelope
// authored rather than derived, an optional cadence: whose grammar is not
// validated in this milestone (§10), and the ordered steps: list. Each
// steps: entry is read at its own position, against whichever of
// stepDeclaration or invocationDeclaration it turns out to be — an Open
// object here is deliberately coarse, catching only "is this an object" so
// checkSteps's own dispatch is what reports a mismatched shape, never this
// schema reporting one first under the wrong key. additionalProperties:
// false is forced rather than authored (§12), so a sixth top-level key is
// unknown-key wherever it appears.
var ProcedureDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "procedure", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "targets", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String},
		}},
		{Name: "cadence", Required: false, Schema: schema.Schema{Type: schema.String}},
		{Name: "steps", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.Object, Open: true},
		}},
	},
}

// stepDeclaration is a Step's own schema — the shape checkSteps validates
// an entry against wherever it carries no procedure: (§3, §4). over: and
// when: are Open here: a predicate's own operand-type rules and the three
// over: forms themselves are #97's, and this file reads only what a
// repeated over: values: member and, since issue #95, an opaque destroy
// Step's over: presence need. bound: is a fixed Integer here regardless of
// a Step's own Kind — the schema stops at "is this an integer" and
// checkStepBound is what reads Kind into the question of whether one may
// stand at all (§4, §5, issue #95).
var stepDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "id", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "definition", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "operation", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "target", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "args", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "over", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
		{Name: "bound", Required: false, Schema: schema.Schema{Type: schema.Integer}},
		{Name: "when", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
	},
}

// invocationDeclaration is a nested Procedure invocation's own schema — id:
// and procedure: in place of definition:/operation:/target: (§3, §4).
var invocationDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "id", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "procedure", Required: true, Schema: schema.Schema{Type: schema.String}},
	},
}

// ProcedureIndex maps a procedures/ file's own procedure: to whether it
// exists — the namespace a nested invocation's procedure: resolves against
// (§3, §4, issue #94).
type ProcedureIndex map[string]bool

// BuildProcedureIndex adds one entry per procedures/ root whose procedure:
// is a legible scalar, on BuildProviderIndex's own rule.
func BuildProcedureIndex(procedureRoots []*yaml.Node) ProcedureIndex {
	idx := ProcedureIndex{}
	for _, root := range procedureRoots {
		nameVal := topLevelFields(root, "procedure")["procedure"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		idx[nameVal.Value] = true
	}
	return idx
}

// CheckProcedure validates a procedures/ file's already-parsed root against
// ProcedureDeclaration and every check that reads a Procedure against itself
// and the repository (§3, §4, issue #94): kind: against procedures/,
// procedure: against the file's basename, and each steps: entry against
// whichever shape it turns out to be. providers, definitions and procedures
// are the repository-wide namespaces a Step's operation: and definition:,
// and a nested invocation's procedure:, resolve against; targets is the
// namespace this Procedure's own declared targets: list resolves against,
// read here into the declared Target and Kind envelope every Step directly
// in this file is checked against (envelope-exceeded, §4, §5, issue #96).
// The transitive half of that same walk — an invoked Procedure's own
// envelope against its caller's, and the two Cadence rules that ride the
// same walk — needs every procedures/ file at once and is CheckProcedureGraph's
// (issue #96). root is nil where the file parsed to no document at all; the
// schema check still runs and reports every required key the file never
// supplied.
func CheckProcedure(file string, root *yaml.Node, providers ProviderIndex, definitions DefinitionIndex, targets TargetIndex, procedures ProcedureIndex) []problem.Problem {
	problems := schema.Check(root, ProcedureDeclaration, file)
	problems = append(problems, checkKind(file, root, KindProcedure)...)
	problems = append(problems, checkName(file, root, "procedure")...)

	declaredTargets, declaredKinds := procedureEnvelope(root, targets)
	stepsVal := topLevelFields(root, "steps")["steps"]
	if stepsVal != nil && stepsVal.Kind == yaml.SequenceNode {
		problems = append(problems, checkSteps(file, stepsVal, providers, definitions, procedures, declaredTargets, declaredKinds)...)
	}
	return problems
}

// procedureEnvelope reads a Procedure's own top-level targets: list — the
// full set the Procedure and everything it invokes may touch, authored
// rather than derived (§3, §4) — into the two sets a Step directly in this
// file is checked against: declaredTargets, the raw names as written,
// whether or not each one resolves; and declaredKinds, the Kind envelope
// those names imply for free, the union of every resolved Target's own
// accepted Kinds. A name that does not resolve against targets contributes
// nothing to declaredKinds — it names no accepted Kinds for a union to
// read — the same way an unresolved reference contributes nothing
// elsewhere in this package (issue #96). It says nothing about whether
// targets: itself is present or well-formed — the schema check above has
// already named that fault — and returns two empty sets where it is
// absent or not a sequence.
func procedureEnvelope(root *yaml.Node, targets TargetIndex) (declaredTargets, declaredKinds map[string]bool) {
	declaredTargets = readDeclaredTargetNames(root)
	declaredKinds = map[string]bool{}
	for name := range declaredTargets {
		if info, found := targets[name]; found {
			for kind := range info.Kinds {
				declaredKinds[kind] = true
			}
		}
	}
	return declaredTargets, declaredKinds
}

// readDeclaredTargetNames reads root's own top-level targets: list into the
// raw names it names, whether or not each one resolves — the one walk
// procedureEnvelope and buildProcedureGraphInfo both need off a
// procedures/ file's targets: and would otherwise duplicate (§4, §5,
// issue #96). It returns an empty, non-nil set where targets: is absent or
// not a sequence.
func readDeclaredTargetNames(root *yaml.Node) map[string]bool {
	names := map[string]bool{}
	targetsVal := topLevelFields(root, "targets")["targets"]
	if targetsVal == nil || targetsVal.Kind != yaml.SequenceNode {
		return names
	}
	for _, item := range targetsVal.Content {
		if name, ok := resolveScalar(item); ok {
			names[name] = true
		}
	}
	return names
}

// stepRefInfo is what a later reference's step: half resolves an earlier
// id: to: the OperationInfo its binding read, so a reference's path: and
// cardinality check have something to read against without re-walking the
// Definition and Provider chain a second time.
type stepRefInfo struct {
	op OperationInfo
}

// checkSteps walks stepsVal in order, dispatching each entry to the Step or
// nested-invocation shape its own keys imply and threading a step: index
// forward so a reference may resolve only against an id: written earlier in
// the same list — never against one written later, and never across a
// nested invocation's own boundary, Procedures composing by invoking one
// another rather than by sharing a Step namespace (ADR-0002, §3, §4).
func checkSteps(file string, stepsVal *yaml.Node, providers ProviderIndex, definitions DefinitionIndex, procedures ProcedureIndex, declaredTargets, declaredKinds map[string]bool) []problem.Problem {
	var problems []problem.Problem
	stepIndex := map[string]stepRefInfo{}

	for i, entry := range stepsVal.Content {
		field := fmt.Sprintf("steps[%d]", i)
		fields := topLevelFields(entry, "id", "definition", "operation", "target", "args", "over", "bound", "when", "procedure")
		hasProcedure := fields["procedure"] != nil
		hasStepKeys := fields["definition"] != nil || fields["operation"] != nil || fields["target"] != nil

		switch {
		case hasProcedure && hasStepKeys:
			line, column := position(entry)
			problems = append(problems, problem.Problem{
				File: file, Line: line, Column: column, Field: field,
				ErrorCode: schema.CodeMismatch,
				Message:   "carries both procedure: and one of definition:/operation:/target: — a nested invocation names id: and procedure: in place of them",
			})
		case hasProcedure:
			problems = append(problems, schema.CheckAt(entry, invocationDeclaration, field, file)...)
			problems = append(problems, checkInvocationResolution(file, field, fields["procedure"], procedures)...)
		default:
			problems = append(problems, checkStepEntry(file, field, entry, fields, providers, definitions, stepIndex, declaredTargets, declaredKinds)...)
		}
	}
	return problems
}

// checkInvocationResolution reports artefact-absent on a nested invocation's
// procedure: naming no file in procedures/ (§3, §4).
func checkInvocationResolution(file, field string, procVal *yaml.Node, procedures ProcedureIndex) []problem.Problem {
	name, ok := resolveScalar(procVal)
	if !ok || procedures[name] {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: procVal.Line, Column: procVal.Column, Field: field + ".procedure",
		ErrorCode: CodeArtefactAbsent,
		Message:   fmt.Sprintf("procedure: %s resolves to nothing — no procedures/%s.yaml", name, name),
	}}
}

// checkStepEntry validates one Step-shaped steps: entry: its own schema,
// its definition:'s resolution against definitions/, its operation:'s
// resolution against the bound Definition's Provider, its args: against
// that Operation's input: schema, a repeated over: values: member, and the
// authority a Step's binding needs — the two keys, the Bound and the
// opaque destroy opt-ins (§4, §5, issue #95). It registers id: in
// stepIndex only once this entry's own args: have been checked against
// stepIndex as it stood before this entry — a Step may reference an id:
// written earlier in the same Procedure and never its own, "earlier"
// excluding the entry currently being read — and it registers id: whenever
// id: is legible whatever else about the entry failed to resolve, so a
// later reference naming this id: finds an empty OperationInfo and fails
// its own resolution once, rather than this entry's own fault being
// reported a second time under a different code.
func checkStepEntry(file, field string, entry *yaml.Node, fields map[string]*yaml.Node, providers ProviderIndex, definitions DefinitionIndex, stepIndex map[string]stepRefInfo, declaredTargets, declaredKinds map[string]bool) []problem.Problem {
	problems := schema.CheckAt(entry, stepDeclaration, field, file)

	defName, defOK := resolveScalar(fields["definition"])
	var defInfo DefinitionInfo
	haveDef := false
	if defOK {
		defInfo, haveDef = definitions[defName]
		if !haveDef {
			problems = append(problems, problem.Problem{
				File: file, Line: fields["definition"].Line, Column: fields["definition"].Column, Field: field + ".definition",
				ErrorCode: CodeArtefactAbsent,
				Message:   fmt.Sprintf("definition: %s resolves to nothing — no definitions/%s.yaml", defName, defName),
			})
		}
	}

	var op OperationInfo
	haveOp := false
	if haveDef {
		provider := providers[defInfo.ProviderName]
		opName, opOK := resolveScalar(fields["operation"])
		if opOK {
			op, haveOp = provider.Operations[opName]
			if !haveOp {
				problems = append(problems, problem.Problem{
					File: file, Line: fields["operation"].Line, Column: fields["operation"].Column, Field: field + ".operation",
					ErrorCode: CodeReferenceUnresolvable,
					Message:   fmt.Sprintf("operation: %s names no Operation the Definition's Provider declares", opName),
				})
			}
		}
	}

	if haveOp {
		problems = append(problems, checkStepArgs(file, field+".args", entry, fields["args"], op, stepIndex)...)
	}
	problems = append(problems, checkOverValuesDuplicates(file, field+".over", fields["over"])...)
	if haveDef {
		problems = append(problems, checkStepAuthority(file, field, entry, fields, defInfo, op, haveOp)...)
	}
	problems = append(problems, checkStepEnvelope(file, field, entry, fields, declaredTargets, declaredKinds, op, haveOp)...)

	if idVal, idOK := resolveScalar(fields["id"]); idOK {
		stepIndex[idVal] = stepRefInfo{op: op}
	}
	return problems
}

// checkStepEnvelope reports envelope-exceeded on a Step reaching outside its
// own Procedure's declared Target and Kind envelope (§4, §5, issue #96): a
// bound Target absent from declaredTargets — the raw targets: names this
// file itself wrote, whatever else about the Step failed to resolve — or,
// where the Operation resolved, a Kind absent from declaredKinds — the
// union of accepted Kinds every declaredTargets member's own Target
// declaration grants. It is the file-local half of the walk: composition
// cannot widen a Procedure's reach past what it declares, and neither can a
// Step written past the envelope its own file declares. The transitive
// half — an invoked Procedure's own envelope reaching outside its caller's —
// needs every procedures/ file at once and is CheckProcedureGraph's.
func checkStepEnvelope(file, field string, entry *yaml.Node, fields map[string]*yaml.Node, declaredTargets, declaredKinds map[string]bool, op OperationInfo, haveOp bool) []problem.Problem {
	var problems []problem.Problem

	if targetName, targetOK := resolveScalar(fields["target"]); targetOK && !declaredTargets[targetName] {
		targetVal := fields["target"]
		problems = append(problems, problem.Problem{
			File: file, Line: targetVal.Line, Column: targetVal.Column, Field: field + ".target",
			ErrorCode: CodeEnvelopeExceeded,
			Message:   fmt.Sprintf("target: %s is outside this Procedure's own declared targets: envelope", targetName),
		})
	}

	if haveOp && !declaredKinds[op.Kind] {
		line, column := position(entry)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeEnvelopeExceeded,
			Message:   fmt.Sprintf("%s is outside the Kind envelope this Procedure's own declared targets: imply", op.Kind),
		})
	}
	return problems
}

// checkStepAuthority runs the checks that need a Step's binding rather than
// either the Definition or the Operation alone (§4, §5, issue #95): its
// target: against the Definition's own targets: claim, the two keys — the
// Definition's claimed Kind against the bound Target's accepted Kinds, and,
// for a destroy Step, its Operation against the Definition's destroy:
// claim by name — the Bound a destroy Step carries or must not, and, for an
// opaque destroy Step, the over: selector it must carry. It is called only
// where the Definition itself resolved; an unresolved definition: has
// already earned artefact-absent and there is no claim here to check a
// binding against.
func checkStepAuthority(file, field string, entry *yaml.Node, fields map[string]*yaml.Node, defInfo DefinitionInfo, op OperationInfo, haveOp bool) []problem.Problem {
	var problems []problem.Problem

	targetName, targetOK := resolveScalar(fields["target"])
	targetInfo, haveTarget := TargetInfo{}, false
	if targetOK {
		targetInfo, haveTarget = defInfo.Targets[targetName]
		if !haveTarget {
			problems = append(problems, problem.Problem{
				File: file, Line: fields["target"].Line, Column: fields["target"].Column, Field: field + ".target",
				ErrorCode: CodeTargetNotClaimed,
				Message:   fmt.Sprintf("target: %s is not a member of the bound Definition's targets:", targetName),
			})
		}
	}

	if !haveOp {
		return problems
	}

	if haveTarget && !(defInfo.ClaimsKind(op.Kind) && targetInfo.Kinds[op.Kind]) {
		line, column := position(entry)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeKindNotGranted,
			Message:   fmt.Sprintf("%s is not in the intersection of the bound Definition's claimed Kind and %s's accepted Kinds", op.Kind, targetName),
		})
	}

	if op.Kind == "destroy" {
		opName, _ := resolveScalar(fields["operation"])
		if !defInfo.Destroy[opName] {
			problems = append(problems, problem.Problem{
				File: file, Line: fields["operation"].Line, Column: fields["operation"].Column, Field: field + ".operation",
				ErrorCode: CodeOperationNotClaimed,
				Message:   fmt.Sprintf("operation: %s is not named among the bound Definition's destroy: claims", opName),
			})
		}
	}

	problems = append(problems, checkStepBound(file, field, entry, fields["bound"], fields["over"], op)...)
	return problems
}

// checkStepBound validates bound: against op's own Kind (§4, §5,
// issue #95): a read Step carries no Bound at all, so one present is
// unknown-key; a destroy Step's Bound is mandatory unless op is opaque, in
// which case one present is bound-illegal and one absent is the correct
// combination; a mutate Step's Bound is optional either way and draws no
// code. Where op is an opaque destroy Operation this also fires
// opaque-destroy-unscoped for a Step carrying no over: selector — the third
// requirement the Bound's own place stands in for (§5, ADR-0053).
func checkStepBound(file, field string, entry, boundVal, overVal *yaml.Node, op OperationInfo) []problem.Problem {
	var problems []problem.Problem
	line, column := position(entry)
	if boundVal != nil {
		line, column = position(boundVal)
	}

	opaqueDestroy := op.IsOpaqueDestroy()
	switch {
	case op.Kind == "read" && boundVal != nil:
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".bound",
			ErrorCode: schema.CodeUnknownKey,
			Message:   "bound: is declared on a read Step — a read carries no Bound at all, having nothing for one to guard",
		})
	case opaqueDestroy && boundVal != nil:
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".bound",
			ErrorCode: CodeBoundIllegal,
			Message:   "bound: is declared on an opaque destroy Step — a count of the commands it ran says nothing about what any of them did",
		})
	case op.Kind == "destroy" && !opaqueDestroy && boundVal == nil:
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".bound",
			ErrorCode: CodeBoundMissing,
			Message:   "a destroy Step carries no bound: — an absent Bound means unbounded, and unbounded is refused before anything runs",
		})
	}

	if opaqueDestroy && overVal == nil {
		entryLine, entryColumn := position(entry)
		problems = append(problems, problem.Problem{
			File: file, Line: entryLine, Column: entryColumn, Field: field,
			ErrorCode: CodeOpaqueDestroyUnscoped,
			Message:   "an opaque destroy Step carries no over: selector — it would reach the world once and leave nothing in the record",
		})
	}
	return problems
}

// checkStepArgs validates args: against op's input: schema (§3, §4): every
// input op declares is supplied, an args: member op does not declare is
// unknown-key — which is what a Step whose Operation declares no input: at
// all draws for every member it writes, op.Inputs being empty there — a
// scalar value outside its input's enum is schema-mismatch, and every
// value is read as a literal or a reference by checkArgValue, except
// command: on a shell Operation, which draws checkShellCommand's own rules
// instead (§3, ADR-0051).
func checkStepArgs(file, field string, entry, argsVal *yaml.Node, op OperationInfo, stepIndex map[string]stepRefInfo) []problem.Problem {
	var problems []problem.Problem
	line, column := position(entry)
	if argsVal != nil {
		line, column = position(argsVal)
	}

	present := map[string]*yaml.Node{}
	if argsVal != nil && argsVal.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(argsVal.Content); i += 2 {
			key, val := argsVal.Content[i], argsVal.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			present[key.Value] = val
			if _, declared := op.Inputs[key.Value]; !declared {
				problems = append(problems, problem.Problem{
					File: file, Line: key.Line, Column: key.Column, Field: field + "." + key.Value,
					ErrorCode: schema.CodeUnknownKey,
					Message:   fmt.Sprintf("%q is not an input the bound Operation declares", key.Value),
				})
			}
		}
	}

	for name, input := range op.Inputs {
		val, ok := present[name]
		if !ok {
			problems = append(problems, problem.Problem{
				File: file, Line: line, Column: column, Field: field + "." + name,
				ErrorCode: schema.CodeMismatch,
				Message:   fmt.Sprintf("the bound Operation declares input %q, and args: does not supply it", name),
			})
			continue
		}
		if op.IsShell && name == "command" {
			problems = append(problems, checkShellCommand(file, field+".command", val, stepIndex)...)
			continue
		}
		problems = append(problems, checkArgValue(file, field+"."+name, val, input, stepIndex)...)
		if val.Kind == yaml.ScalarNode && len(input.Enum) > 0 && !enumHas(input.Enum, val.Value) {
			problems = append(problems, problem.Problem{
				File: file, Line: val.Line, Column: val.Column, Field: field + "." + name,
				ErrorCode: schema.CodeMismatch,
				Message:   fmt.Sprintf("%q is outside the enum at this position", val.Value),
			})
		}
	}
	return problems
}

// checkShellCommand validates a shell Step's command: (§3, ADR-0051): empty
// is command-malformed, there being no executable to name; a first member
// that is not a literal scalar is command-malformed, the reach axis being
// the one word a reference may never fill; every member after the first is
// read as a literal or a reference by checkArgValue and draws no code of
// its own for being one. A command: written whole as a reference is read by
// checkArgValue directly — the array-typed position already refuses a
// reference under the object/array rule every other input's does, and there
// is no literal list in hand for the empty or first-member rule to read.
func checkShellCommand(file, field string, node *yaml.Node, stepIndex map[string]stepRefInfo) []problem.Problem {
	if node.Kind == yaml.MappingNode {
		return checkArgValue(file, field, node, InputInfo{Type: "array"}, stepIndex)
	}
	if node.Kind != yaml.SequenceNode {
		line, column := position(node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeCommandMalformed,
			Message:   "command: is a list of argv words, and this is neither one nor a reference to one",
		}}
	}
	if len(node.Content) == 0 {
		line, column := position(node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeCommandMalformed,
			Message:   "a shell Step's command: is empty — there is no executable to name",
		}}
	}

	var problems []problem.Problem
	first := node.Content[0]
	if first.Kind != yaml.ScalarNode {
		problems = append(problems, problem.Problem{
			File: file, Line: first.Line, Column: first.Column, Field: field + "[0]",
			ErrorCode: CodeCommandMalformed,
			Message:   "command:'s first member is the reach axis and must be a literal, never a reference",
		})
	}
	for i, item := range node.Content[1:] {
		problems = append(problems, checkArgValue(file, fmt.Sprintf("%s[%d]", field, i+1), item, InputInfo{Type: "string"}, stepIndex)...)
	}
	return problems
}

// checkOverValuesDuplicates reports record-identity-collision on two
// members of one over: values: list that are one identity under a
// case-insensitive fold — the Store's own check, fired here at load because
// the list is authored and needs no Store to compare against (§3, §4, §8).
// It says nothing where over: or over.values: is absent or not a sequence —
// the shape #97 checks — and skips a non-scalar member, which carries no
// identity of its own for this check to read.
func checkOverValuesDuplicates(file, field string, overVal *yaml.Node) []problem.Problem {
	if overVal == nil || overVal.Kind != yaml.MappingNode {
		return nil
	}
	valuesVal := topLevelFields(overVal, "values")["values"]
	if valuesVal == nil || valuesVal.Kind != yaml.SequenceNode {
		return nil
	}

	var problems []problem.Problem
	seen := map[string]string{}
	for _, item := range valuesVal.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		fold := strings.ToLower(item.Value)
		if prior, dup := seen[fold]; dup {
			problems = append(problems, problem.Problem{
				File: file, Line: item.Line, Column: item.Column, Field: field + ".values",
				ErrorCode: CodeRecordIdentityCollision,
				Message:   fmt.Sprintf("%q is one identity with %q, already a member of this values: list", item.Value, prior),
			})
			continue
		}
		seen[fold] = item.Value
	}
	return problems
}

// reference is one parsed value-position mapping — the two legal forms a
// path may be embedded in, {step:, path:} and {item:}, and the one
// catch-all, malformed, that stands for every mapping written where neither
// shape fits: a third key, a missing half, or a value under step:/path:/
// item: that is not itself a plain scalar (§3, §4).
type reference struct {
	kind referenceKind
	step string
	path string
	node *yaml.Node
}

type referenceKind int

const (
	refNone referenceKind = iota
	refStep
	refItem
	refMalformed
)

// parseReference reads node as a reference. refNone is returned for a node
// that is not a mapping at all — a literal value, read no further here —
// and every mapping is read as an attempt at one of the two legal forms,
// malformed where it fits neither exactly.
func parseReference(node *yaml.Node) reference {
	if node == nil || node.Kind != yaml.MappingNode {
		return reference{kind: refNone}
	}
	fields := topLevelFields(node, "step", "path", "item")
	stepVal, pathVal, itemVal := fields["step"], fields["path"], fields["item"]

	switch {
	case stepVal != nil && pathVal != nil && itemVal == nil && len(node.Content) == 4:
		step, stepOK := resolveScalar(stepVal)
		path, pathOK := resolveScalar(pathVal)
		if !stepOK || !pathOK {
			return reference{kind: refMalformed, node: node}
		}
		return reference{kind: refStep, step: step, path: path, node: node}
	case itemVal != nil && stepVal == nil && pathVal == nil && len(node.Content) == 2:
		path, pathOK := resolveScalar(itemVal)
		if !pathOK {
			return reference{kind: refMalformed, node: node}
		}
		return reference{kind: refItem, path: path, node: node}
	default:
		return reference{kind: refMalformed, node: node}
	}
}

// checkArgValue reads node as a literal or a reference at a position whose
// declared input is input (§3, §4). A literal — anything but a mapping —
// draws no code here: the input-schema subset's own type reading is not
// this milestone's job beyond the enum check checkStepArgs already applies.
// A reference is read by parseReference and checked against input's own
// type — object and array refuse a reference outright, a whole object never
// being referenceable — and, for the step: form, against stepIndex.
func checkArgValue(file, field string, node *yaml.Node, input InputInfo, stepIndex map[string]stepRefInfo) []problem.Problem {
	ref := parseReference(node)
	switch ref.kind {
	case refNone:
		return nil
	case refMalformed:
		line, column := position(ref.node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: schema.CodeMismatch,
			Message:   "a reference is a mapping and never a string — {step:, path:} and {item:} are the only two forms",
		}}
	case refStep:
		return checkStepReference(file, field, ref, input, stepIndex)
	default:
		return checkItemReference(file, field, ref, input)
	}
}

// checkReferenceType reports schema-mismatch on a reference written where
// input declares object or array — a hole may not fill either position for
// the same reason (ADR-0078), and a reference is refused there on the same
// ground: a whole object can never be referenced, only a scalar (§3, §4).
func checkReferenceType(file, field string, ref reference, input InputInfo) []problem.Problem {
	if input.Type != "object" && input.Type != "array" {
		return nil
	}
	line, column := position(ref.node)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field,
		ErrorCode: schema.CodeMismatch,
		Message:   fmt.Sprintf("a reference is written where the schema declares %s — a reference may appear only where a scalar is expected", input.Type),
	}}
}

// checkStepReference validates a {step:, path:} reference (§3, §4): step:
// names no id: this Procedure declares earlier, or path: names no field the
// Record it points at carries, is reference-unresolvable either way; step:
// naming an earlier Step whose Operation is series cardinality is
// series-reference, checked before path: resolution — a series carries no
// one field for path: to have named regardless of what it spells.
func checkStepReference(file, field string, ref reference, input InputInfo, stepIndex map[string]stepRefInfo) []problem.Problem {
	problems := checkReferenceType(file, field, ref, input)
	line, column := position(ref.node)

	target, ok := stepIndex[ref.step]
	if !ok {
		return append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".step",
			ErrorCode: CodeReferenceUnresolvable,
			Message:   fmt.Sprintf("step: %s names no id: this Procedure declares earlier", ref.step),
		})
	}
	if target.op.HasSeries {
		return append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeSeriesReference,
			Message:   fmt.Sprintf("step: %s is series cardinality — a reference names one Record and cannot join against a stored series", ref.step),
		})
	}
	if !target.op.RecordFields[pathField(ref.path)] {
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".path",
			ErrorCode: CodeReferenceUnresolvable,
			Message:   fmt.Sprintf("path: %s names no field the Record %s carries", ref.path, ref.step),
		})
	}
	return problems
}

// checkItemReference validates an {item:} reference (§3, §4): only the
// object/array refusal every reference draws and the path grammar itself —
// what field it names is decided by over:'s own form, #97's, so this
// milestone reads no further than that.
func checkItemReference(file, field string, ref reference, input InputInfo) []problem.Problem {
	problems := checkReferenceType(file, field, ref, input)
	if pathPattern.MatchString(ref.path) {
		return problems
	}
	line, column := position(ref.node)
	return append(problems, problem.Problem{
		File: file, Line: line, Column: column, Field: field + ".item",
		ErrorCode: schema.CodeMismatch,
		Message:   fmt.Sprintf("%q is not a well-formed path — the grammar is $, .member and [\"member\"]", ref.path),
	})
}

// pathField reads path as the one flat field name it names — a Record's
// field names being flat and authored, a reference's path: resolves against
// them the same way a predicate's field: does at either Record root (§3,
// §12) — and returns "" where path does not name exactly one such segment,
// which target.op.RecordFields then reads as absent like any other name
// that does not resolve. $["member"] is deliberately not read as a further
// grammar to spell — one segment is one field either way it is written.
func pathField(path string) string {
	rest := strings.TrimPrefix(path, "$")
	if rest == path || rest == "" {
		return ""
	}
	switch {
	case strings.HasPrefix(rest, "."):
		rest = rest[1:]
	case strings.HasPrefix(rest, `["`) && strings.HasSuffix(rest, `"]`):
		rest = rest[2 : len(rest)-2]
	default:
		return ""
	}
	if strings.ContainsAny(rest, `.["`) {
		return ""
	}
	return rest
}

// enumHas reports whether value is a member of list.
func enumHas(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
