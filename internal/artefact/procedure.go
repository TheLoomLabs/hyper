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
// already, in definition.go, needing no Step to exist.
//
// Issue #97 lands here too: the closed eleven-member operator set's own
// operand-type rules (predicate-type-mismatch), checked wherever a
// predicate stands — a selector, a condition, or a polling Pattern's
// `until:` in manifest.go, which calls back into this file's
// checkPredicateCore rather than duplicate it; the three `over:` forms
// themselves, `assets:`, `observations:` and `values:`, closed to a fourth
// and each checked against the Kind that may declare it; the two Record
// roots' own `field:` rule — one declared field name, resolved against the
// union of every Operation the bound Provider declares, since an
// `assets:`/`observations:` selector ranges over a (Definition, Target)
// series a different Operation of the same Provider may have written
// (reference-unresolvable), and refused again where that field is one the
// Manifest declares `secret:`; `skip-if-recorded-unreachable`, a
// `skip-if-recorded` Step expanding over `assets:`; and `bound-exceeded`,
// the offline half decided from an `over: values:` list's authored length
// alone, the run-time half over `assets:`/`observations:` left to a Run
// that needs the Store to count them.
//
// Issue #98 lands the host grant and the one Expansion-identity fault
// decidable with no Store: an Operation's host: template expanded at load
// into its finite candidate set — {from-target} to the bound Target's
// grant, an enumeration hole against the enumerations: entry it names,
// the cross-product where a template carries more than one hole — and
// compared against the grant, a member absent earning host-not-granted;
// the intersection deciding whether host-input: is required at all; an
// over: values: list wired {item: $} into the Operation's host-input:
// compared against the same grant under the same code, host-list-ness
// read off the wiring and never off a declaration; and
// record-identity-collision for an authored two-or-more-member values:
// list whose members can only ever project one identity, the Operation's
// identity: resolving before the call with no {item:} reference reaching
// the value that fills it — the same code the duplicate-member load site
// fires, found against the wiring rather than against the list.
package artefact

import (
	"fmt"
	"sort"
	"strconv"
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

// CodePredicateTypeMismatch is the code an operator handed a type it does
// not take earns — a timestamp under greater_than or less_than, an in:
// whose members are not all one type, exists: false, an in: of one member
// or none, an empty starts_with: or ends_with:, and a predicate against a
// field the Manifest declares secret, that field reaching the Store as a
// constant no comparison can read. It is the same code §6 carries for a
// stored value the same operator cannot compare, split on whether the fault
// is authored and knowable offline — this file's half — or found against a
// value only a Run has (§4, §6, §12, ADR-0035, issue #97).
const CodePredicateTypeMismatch = "predicate-type-mismatch"

// CodeSkipIfRecordedUnreachable is the code a skip-if-recorded Step
// expanding over assets: earns: an effectful Expansion reaches only Assets
// whose head stands, and the value's own test skips exactly while a head
// stands, so every member skips on every Run and no call can ever go out —
// a Step refused for what it can never do rather than for what it might
// (§4, §5, §12, ADR-0056, issue #97).
const CodeSkipIfRecordedUnreachable = "skip-if-recorded-unreachable"

// CodeBoundExceeded is the code an over: values: list longer than the
// Step's own declared bound: earns, decided offline: the list is authored
// in the Procedure, so its length is read off the file, and it is an upper
// bound on what the Expansion can reach — the Store only ever removes
// members from it. The run-time half of the same check, over assets: and
// observations:, needs the Store to count and is left to a Run (§4, §5,
// §6, §12, issue #97).
const CodeBoundExceeded = "bound-exceeded"

// CodeHostNotGranted is the one code over the two comparisons of a host
// set against the bound Target's grant (§3, §4, ADR-0024, ADR-0029,
// issue #98): a member of the candidate set an Operation's host: template
// expands to at load, and a member of an over: values: list the Step's
// wiring makes a host list, are the same comparison and carry the same
// name — a member absent from the grant, from either origin, is
// host-not-granted.
const CodeHostNotGranted = "host-not-granted"

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
// when: are Open here — a mapping keyed by a closed set (assets:/
// observations:/values:) rather than a fixed shape, and a single predicate
// respectively — so the generic engine stops at "is this a mapping" and
// checkOverForm and checkConditionPredicate read what is inside them, on
// checkAuth's own rule for a Target declaration's auth: (§4, §12, issue
// #97). bound: is a fixed Integer here regardless of a Step's own Kind —
// the schema stops at "is this an integer" and checkStepBound is what reads
// Kind into the question of whether one may stand at all (§4, §5, issue
// #95).
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

// ProcedureCadence is the recurrence a Procedure declares, exactly as it was
// written, and "" where it declares none. It is DeclaredName's reader under the
// key this one fixes — a caller asking for a Cadence is asking for a
// Procedure's recurrence rather than for a key's scalar, and a Cadence is a
// Procedure's alone: a key of that name on any other artefact is not one.
//
// It reads and does not judge. Whether the expression is one §10's grammar
// admits is the gloss's question to answer and `cadence-malformed`'s to refuse,
// and neither is this reader's (§10, §12).
func ProcedureCadence(root *yaml.Node) string {
	return DeclaredName(root, "cadence")
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
// Definition and Provider chain a second time. provider and haveProvider
// are the same walk's own Provider — the namespace a when: condition's
// field: resolves against once step: has named this entry (§4, §5, §12,
// issue #97) — haveProvider false where the Step's own definition: did not
// resolve or named a provider: nothing in this pass's ProviderIndex holds,
// in which case field: resolution is skipped rather than checked against an
// empty Provider that was never really named.
type stepRefInfo struct {
	op           OperationInfo
	provider     ProviderInfo
	haveProvider bool
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
// that Operation's input: schema, a repeated over: values: member, the
// three over: forms and the when: condition, and the authority a Step's
// binding needs — the two keys, the Bound and the opaque destroy opt-ins
// (§4, §5, issue #95, issue #97). It registers id: in stepIndex only once
// this entry's own args: have been checked against stepIndex as it stood
// before this entry — a Step may reference an id: written earlier in the
// same Procedure and never its own, "earlier" excluding the entry currently
// being read — and it registers id: whenever id: is legible whatever else
// about the entry failed to resolve, so a later reference naming this id:
// finds an empty OperationInfo and fails its own resolution once, rather
// than this entry's own fault being reported a second time under a
// different code.
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

	var provider ProviderInfo
	haveProvider := false
	var op OperationInfo
	haveOp := false
	if haveDef {
		provider, haveProvider = providers[defInfo.ProviderName]
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
		problems = append(problems, checkExpansionIdentity(file, field, fields, op)...)
	}
	problems = append(problems, checkOverValuesDuplicates(file, field+".over", fields["over"])...)
	problems = append(problems, checkOverForm(file, field+".over", fields["over"], op, haveOp, provider, haveProvider)...)
	problems = append(problems, checkBoundExceeded(file, field+".over.values", fields["bound"], fields["over"])...)
	problems = append(problems, checkConditionPredicate(file, field+".when", fields["when"], stepIndex)...)
	if haveDef {
		problems = append(problems, checkStepAuthority(file, field, entry, fields, provider, defInfo, op, haveOp)...)
	}
	problems = append(problems, checkStepEnvelope(file, field, entry, fields, declaredTargets, declaredKinds, op, haveOp)...)

	if idVal, idOK := resolveScalar(fields["id"]); idOK {
		stepIndex[idVal] = stepRefInfo{op: op, provider: provider, haveProvider: haveProvider}
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
// opaque destroy Step, the over: selector it must carry. The host grant is
// the same binding's reach half (§3, §4, issue #98): the candidate set the
// bound Operation's host: expands to, and any values: list the wiring makes
// a host list, each compared against the bound Target's hosts:. It is
// called only where the Definition itself resolved; an unresolved
// definition: has already earned artefact-absent and there is no claim
// here to check a binding against.
func checkStepAuthority(file, field string, entry *yaml.Node, fields map[string]*yaml.Node, provider ProviderInfo, defInfo DefinitionInfo, op OperationInfo, haveOp bool) []problem.Problem {
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

	if haveTarget {
		problems = append(problems, checkHostCandidateGrant(file, field, entry, provider, op, targetName, targetInfo)...)
		problems = append(problems, checkHostIntersection(file, field, entry, provider, op, targetName, targetInfo)...)
		problems = append(problems, checkValuesHostGrant(file, field, fields, op, targetName, targetInfo)...)
	}
	problems = append(problems, checkStepBound(file, field, entry, fields["bound"], fields["over"], op)...)
	problems = append(problems, checkSkipIfRecordedReachability(file, field, entry, fields["over"], op)...)
	return problems
}

// checkHostCandidateGrant compares the candidate set the bound Operation's
// host: template expands to at load against the bound Target's hosts:
// grant, a member absent from it earning host-not-granted (§3, §4,
// ADR-0024, ADR-0029, issue #98). The comparison is one of the two that
// share the code; the other — an over: values: list the Step's own wiring
// makes a host list — is checkValuesHostGrant's. Where a hole names
// neither from-target nor a declared enumerations: entry the Manifest has
// already earned hole-illegal on its own line, and nothing is added here.
func checkHostCandidateGrant(file, field string, entry *yaml.Node, provider ProviderInfo, op OperationInfo, targetName string, targetInfo TargetInfo) []problem.Problem {
	if op.HostTemplate == "" {
		return nil
	}
	candidates, ok := expandHostTemplate(op.HostTemplate, provider.Enumerations, targetInfo.Hosts)
	if !ok {
		return nil
	}
	var problems []problem.Problem
	line, column := position(entry)
	for _, candidate := range candidates {
		if targetInfo.Hosts[candidate] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeHostNotGranted,
			Message:   fmt.Sprintf("%q is a member of the candidate set the bound Operation's host: expands to, and is absent from %s's hosts: grant", candidate, targetName),
		})
	}
	return problems
}

// checkHostIntersection reads the third of ADR-0029's three steps off the
// same expansion (§3, issue #98): what a Run may reach is the candidates
// intersected with the grant — where that is one host hyper fills it and
// host-input: is not required, and where it is several the Operation's
// host-input: names which input carries one. A several-member
// intersection under an Operation declaring none leaves which host a
// request reaches undecidable — manifest-inconsistent, the one code a
// Manifest disagreeing with itself carries, found here at the binding
// that makes it decidable.
func checkHostIntersection(file, field string, entry *yaml.Node, provider ProviderInfo, op OperationInfo, targetName string, targetInfo TargetInfo) []problem.Problem {
	if op.HostTemplate == "" || op.HostInput != "" {
		return nil
	}
	candidates, ok := expandHostTemplate(op.HostTemplate, provider.Enumerations, targetInfo.Hosts)
	if !ok {
		return nil
	}
	granted := 0
	for _, candidate := range candidates {
		if targetInfo.Hosts[candidate] {
			granted++
		}
	}
	if granted <= 1 {
		return nil
	}
	line, column := position(entry)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field,
		ErrorCode: CodeManifestInconsistent,
		Message:   fmt.Sprintf("the candidate set and %s's hosts: grant intersect to several hosts, and the bound Operation declares no host-input:", targetName),
	}}
}

// checkValuesHostGrant is the second of the two comparisons
// CodeHostNotGranted covers (§3, §4, ADR-0024, issue #98): an over:
// values: list the Step wires into the bound Operation's host-input: —
// {item: $} at that input's args: position, and not otherwise — is a host
// list, and every member is compared against the bound Target's grant,
// the case where the Run-time membership check has nothing left to find.
// Which lists are host lists is read off the wiring rather than off an
// author's word for it: a list wired into any other input is a list of
// identifiers, compared against no grant, and a shell Operation has no
// host-input: at all, so a values: list on a shell Step is never one.
func checkValuesHostGrant(file, field string, fields map[string]*yaml.Node, op OperationInfo, targetName string, targetInfo TargetInfo) []problem.Problem {
	if op.HostInput == "" {
		return nil
	}
	argsVal := fields["args"]
	if argsVal == nil || argsVal.Kind != yaml.MappingNode {
		return nil
	}
	wired := parseReference(topLevelFields(argsVal, op.HostInput)[op.HostInput])
	if wired.kind != refItem || wired.path != "$" {
		return nil
	}
	valuesVal := overValuesList(fields["over"])
	if valuesVal == nil {
		return nil
	}

	var problems []problem.Problem
	for _, item := range valuesVal.Content {
		if item.Kind != yaml.ScalarNode || targetInfo.Hosts[item.Value] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: item.Line, Column: item.Column, Field: field + ".over.values",
			ErrorCode: CodeHostNotGranted,
			Message:   fmt.Sprintf("%q is wired into the bound Operation's host-input: and is absent from %s's hosts: grant", item.Value, targetName),
		})
	}
	return problems
}

// expandHostTemplate expands template's holes at load into the finite
// candidate set ADR-0029 names: {from-target} expands to the bound
// Target's granted host set, an enumeration hole against the
// enumerations: entry it names, and the cross-product where a template
// carries more than one hole. The set comes back deduplicated in first-
// expanded order, with from-target's own members sorted so the expansion
// does not depend on map iteration. ok is false where a hole names
// neither from-target nor a declared enumeration — a Manifest fault
// checkCapabilityHoles has already named, leaving nothing here to expand.
func expandHostTemplate(template string, enumerations map[string][]string, grant map[string]bool) ([]string, bool) {
	sets := []string{""}
	prev := 0
	for _, m := range holePattern.FindAllStringSubmatchIndex(template, -1) {
		var members []string
		switch name := template[m[2]:m[3]]; {
		case name == "from-target":
			for host := range grant {
				members = append(members, host)
			}
			sort.Strings(members)
		default:
			declared, ok := enumerations[name]
			if !ok {
				return nil, false
			}
			members = declared
		}
		literal := template[prev:m[0]]
		var next []string
		for _, prefix := range sets {
			for _, member := range members {
				next = append(next, prefix+literal+member)
			}
		}
		sets = next
		prev = m[1]
	}

	seen := map[string]bool{}
	var candidates []string
	for _, prefix := range sets {
		candidate := prefix + template[prev:]
		if !seen[candidate] {
			seen[candidate] = true
			candidates = append(candidates, candidate)
		}
	}
	return candidates, true
}

// checkSkipIfRecordedReachability reports skip-if-recorded-unreachable on a
// skip-if-recorded Step expanding over assets: (§4, §5, §12, ADR-0056,
// issue #97): an effectful Expansion reaches only Assets whose head stands,
// and the value's own test skips exactly while a head stands, so every
// member skips on every Run and no call can ever go out. The check needs no
// Store and no credential — the selector form and the Operation's declared
// Repeatability are both authored.
func checkSkipIfRecordedReachability(file, field string, entry, overVal *yaml.Node, op OperationInfo) []problem.Problem {
	if op.Repeatability != "skip-if-recorded" || overVal == nil || overVal.Kind != yaml.MappingNode {
		return nil
	}
	if topLevelFields(overVal, "assets")["assets"] == nil {
		return nil
	}
	line, column := position(entry)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field,
		ErrorCode: CodeSkipIfRecordedUnreachable,
		Message:   "a skip-if-recorded Step expands over assets: — every member skips on every Run and no call can ever go out",
	}}
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

// overValuesList reads over:'s values: member where both are present and
// well-formed, nil where either is absent or misshapen — the one preamble
// every values:-reading check shares, each such fault being the shape
// checks' own to name rather than a reader's to repeat (§3, §12).
func overValuesList(overVal *yaml.Node) *yaml.Node {
	if overVal == nil || overVal.Kind != yaml.MappingNode {
		return nil
	}
	valuesVal := topLevelFields(overVal, "values")["values"]
	if valuesVal == nil || valuesVal.Kind != yaml.SequenceNode {
		return nil
	}
	return valuesVal
}

// checkOverValuesDuplicates reports record-identity-collision on two
// members of one over: values: list that are one identity under a
// case-insensitive fold — the Store's own check, fired here at load because
// the list is authored and needs no Store to compare against (§3, §4, §8).
// It says nothing where over: or over.values: is absent or not a sequence —
// the shape #97 checks — and skips a non-scalar member, which carries no
// identity of its own for this check to read.
func checkOverValuesDuplicates(file, field string, overVal *yaml.Node) []problem.Problem {
	valuesVal := overValuesList(overVal)
	if valuesVal == nil {
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

// checkExpansionIdentity reports record-identity-collision on the one
// Expansion-identity fault decidable with no Store (§4, issue #98): an
// over: values: list of two or more — the member count must be authored,
// a one-member Expansion having no sibling to collide with and an
// assets:/observations: selector's size being on no file — on an
// Operation whose identity: resolves before the call, with no {item:}
// reference reaching the value that fills it. A literal there, or a
// reference to another Step's output, is one value for the whole
// Expansion by construction, so every member projects one name however
// the Run goes. It is the same code the load site fires for a duplicate
// values: member — checkOverValuesDuplicates — found here against the
// wiring rather than against the list, and the same code §6 carries at
// Expansion over the identities that actually resolved. Where a filling
// input's args: entry is absent or malformed the Step has already earned
// its own fault there, and nothing is added here.
func checkExpansionIdentity(file, field string, fields map[string]*yaml.Node, op OperationInfo) []problem.Problem {
	valuesVal := overValuesList(fields["over"])
	if valuesVal == nil {
		return nil
	}
	members := 0
	for _, item := range valuesVal.Content {
		if item.Kind == yaml.ScalarNode {
			members++
		}
	}
	if members < 2 {
		return nil
	}
	fillers, ok := identityFillers(op)
	if !ok {
		return nil
	}

	var supplied map[string]*yaml.Node
	if argsVal := fields["args"]; argsVal != nil && argsVal.Kind == yaml.MappingNode {
		supplied = topLevelFields(argsVal, fillers...)
	}
	for _, filler := range fillers {
		entry := supplied[filler]
		if entry == nil || parseReference(entry).kind == refMalformed || containsItemReference(entry) {
			return nil
		}
	}

	line, column := position(valuesVal)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field + ".over.values",
		ErrorCode: CodeRecordIdentityCollision,
		Message:   "every member of this values: list resolves to one Record identity — the bound Operation's identity: resolves before the call and no {item:} reference reaches the value that fills it; wire the member into the input identity: reads, or write the calls out as Steps",
	}}
}

// identityFillers reports whether op's identity: resolves before the call
// — the property the offline collision check needs (§3, §4, issue #98) —
// and, where it does, the inputs whose args: entries fill the identity's
// value. A template hole resolves to an Operation input like any hole
// (§12), and $.command on a shell Operation sits in the response object
// precisely because it is a fact about the call rather than about the
// answer. Any other response path names a value that exists only once the
// call has gone out, and an Operation with no identity: at all — a
// destroy, projecting nothing — has the member as the name, so distinct
// members are distinct identities by construction.
func identityFillers(op OperationInfo) ([]string, bool) {
	switch {
	case op.Identity == "":
		return nil, false
	case strings.HasPrefix(op.Identity, "{"):
		var fillers []string
		for _, m := range holePattern.FindAllStringSubmatch(op.Identity, -1) {
			if _, declared := op.Inputs[m[1]]; !declared {
				// The Manifest's own hole-illegal has already named a
				// hole naming no input; there is no identity here left
				// to read.
				return nil, false
			}
			fillers = append(fillers, m[1])
		}
		return fillers, len(fillers) > 0
	case op.IsShell && op.Identity == "$.command":
		return []string{"command"}, true
	default:
		return nil, false
	}
}

// containsItemReference reports whether an {item:} reference stands
// anywhere in an args: entry — the entry itself for a scalar input, or
// any member of the argv for the array-typed command — the wiring that
// makes an Expansion's identity member-dependent (§4, issue #98).
func containsItemReference(node *yaml.Node) bool {
	switch node.Kind {
	case yaml.MappingNode:
		return parseReference(node).kind == refItem
	case yaml.SequenceNode:
		for _, item := range node.Content {
			if containsItemReference(item) {
				return true
			}
		}
	}
	return false
}

// overFormKeys is the closed three-member key set over: admits (§12,
// issue #97) — checkClosedKeySet's own comparand for checkOverForm, on
// checkPredicateCore's rule for a predicate's own key set.
var overFormKeys = map[string]bool{"assets": true, "observations": true, "values": true}

// checkClosedKeySet reports unknown-key on every one of node's own
// top-level keys absent from allowed — the one walk checkOverForm and
// checkPredicateCore both need over a mapping keyed by a closed set rather
// than a fixed schema, so this file has it once (§3, §4, §12, issue #97).
// messageFor builds each problem's own message from the offending key,
// since the two callers point a reader at different remedies for the same
// shape of fault.
func checkClosedKeySet(file, field string, node *yaml.Node, allowed map[string]bool, messageFor func(key string) string) []problem.Problem {
	var problems []problem.Problem
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := node.Content[i]
		if key.Kind != yaml.ScalarNode || allowed[key.Value] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: key.Line, Column: key.Column, Field: field + "." + key.Value,
			ErrorCode: schema.CodeUnknownKey,
			Message:   messageFor(key.Value),
		})
	}
	return problems
}

// checkOverForm validates over: itself (§3, §4, §5, §12, issue #97): the
// closed three-form set — assets:, observations: and values:, and no
// fourth — each form's own structural shape, observations: legal only on a
// read Step (Expansion is scoped by Kind rather than by Record type), and
// each assets:/observations: predicate's own field: against the union of
// every Operation the bound Provider declares. It says nothing where
// overVal is absent or not a mapping — a Step declaring no over: is invoked
// once and draws no code here, and a malformed over: has already earned
// schema-mismatch from stepDeclaration's own schema check, which this does
// not repeat. haveProvider is false wherever the Definition or its
// provider: did not resolve, in which case field: resolution is skipped
// rather than checked against a Provider that was never really named.
func checkOverForm(file, field string, overVal *yaml.Node, op OperationInfo, haveOp bool, provider ProviderInfo, haveProvider bool) []problem.Problem {
	if overVal == nil || overVal.Kind != yaml.MappingNode {
		return nil
	}
	problems := checkExactlyOneOf(file, field, overVal, []string{"assets", "observations", "values"})
	problems = append(problems, checkClosedKeySet(file, field, overVal, overFormKeys, func(key string) string {
		return fmt.Sprintf("%q is not a key over: admits — assets:, observations: and values: are the whole of it", key)
	})...)

	forms := topLevelFields(overVal, "assets", "observations", "values")
	if assetsVal := forms["assets"]; assetsVal != nil {
		problems = append(problems, checkPredicateList(file, field+".assets", assetsVal, provider, haveProvider)...)
	}
	if obsVal := forms["observations"]; obsVal != nil {
		problems = append(problems, checkPredicateList(file, field+".observations", obsVal, provider, haveProvider)...)
		if haveOp && op.Kind != "read" {
			line, column := position(obsVal)
			problems = append(problems, problem.Problem{
				File: file, Line: line, Column: column, Field: field + ".observations",
				ErrorCode: schema.CodeMismatch,
				Message:   "observations: is legal only on a read Step — Expansion is scoped by Kind rather than by Record type",
			})
		}
	}
	if valuesVal := forms["values"]; valuesVal != nil {
		problems = append(problems, checkOverValuesShape(file, field+".values", valuesVal)...)
	}
	return problems
}

// checkPredicateList validates one assets: or observations: block: a
// sequence of predicates, each read by checkRecordPredicate against the
// Provider's own declared field set (§3, §4, §12, issue #97). A predicate
// list is always AND; there is no disjunction key anywhere in this format
// for a non-sequence to be mistaken for.
func checkPredicateList(file, field string, node *yaml.Node, provider ProviderInfo, haveProvider bool) []problem.Problem {
	if node.Kind != yaml.SequenceNode {
		line, column := position(node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: schema.CodeMismatch,
			Message:   "a selector's predicate list is a sequence of predicates, always AND",
		}}
	}
	var problems []problem.Problem
	for i, item := range node.Content {
		problems = append(problems, checkRecordPredicate(file, fmt.Sprintf("%s[%d]", field, i), item, provider, haveProvider)...)
	}
	return problems
}

// checkOverValuesShape validates a values: block's own shape: a sequence
// of bare scalars (§3, §4, §12, issue #97). A member is not a mapping — a
// mapping in a scalar position means a reference elsewhere in this format,
// and a compound identity needs none, the shared half of it being an
// argument and only the varying half a population.
func checkOverValuesShape(file, field string, node *yaml.Node) []problem.Problem {
	if node.Kind != yaml.SequenceNode {
		line, column := position(node)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: schema.CodeMismatch,
			Message:   "values: is a list of bare scalars",
		}}
	}
	var problems []problem.Problem
	for i, item := range node.Content {
		if item.Kind != yaml.ScalarNode {
			problems = append(problems, problem.Problem{
				File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("%s[%d]", field, i),
				ErrorCode: schema.CodeMismatch,
				Message:   "a values: member is a bare scalar — a mapping there means a reference, and a compound identity needs none",
			})
		}
	}
	return problems
}

// checkBoundExceeded reports bound-exceeded on an over: values: list longer
// than the Step's own declared bound: — the offline half of the check,
// decided from the list's authored length alone (§4, §5, §6, §12, issue
// #97). It says nothing where bound: or over: values: is absent or
// illegible, or where over: is assets:/observations: — no file can count
// what an Expansion over the Store resolves to, and that half is a Run's.
func checkBoundExceeded(file, field string, boundVal, overVal *yaml.Node) []problem.Problem {
	if boundVal == nil || boundVal.Kind != yaml.ScalarNode {
		return nil
	}
	bound, err := strconv.Atoi(boundVal.Value)
	if err != nil {
		return nil
	}
	valuesVal := overValuesList(overVal)
	if valuesVal == nil || len(valuesVal.Content) <= bound {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: valuesVal.Line, Column: valuesVal.Column, Field: field,
		ErrorCode: CodeBoundExceeded,
		Message:   fmt.Sprintf("over: values: carries %d members, which exceeds bound: %d — the authored length is an upper bound the Expansion can only shrink", len(valuesVal.Content), bound),
	}}
}

// checkPredicateCore validates the shape and operand types every predicate
// carries regardless of its root — a selector, a condition, or a polling
// Pattern's until: in manifest.go, which calls back into this function
// rather than duplicate it (§3, §4, §5, §12, issue #97): a field: is
// present, exactly one of the closed eleven-member operator set is present,
// each present operator's own operand reads as one of the types §12 fixes
// for it, and no key beyond field:, the operator set and — where allowStep
// is true — step: is written at all, a predicate list being always AND
// with no disjunction key anywhere in it to admit. It returns the resolved
// field: and step: value nodes, nil wherever either is absent or not a
// plain scalar, leaving their own further rules — resolution, the root's
// own path-or-name shape — to each root's own caller.
func checkPredicateCore(file, field string, node *yaml.Node, allowStep bool) (problems []problem.Problem, fieldNameVal, stepVal *yaml.Node) {
	problems = schema.CheckAt(node, schema.Schema{Type: schema.Object, Open: true}, field, file)
	if node == nil || node.Kind != yaml.MappingNode {
		return problems, nil, nil
	}

	allowed := map[string]bool{"field": true}
	for _, opName := range predicateOperators {
		allowed[opName] = true
	}
	if allowStep {
		allowed["step"] = true
	}
	problems = append(problems, checkClosedKeySet(file, field, node, allowed, func(key string) string {
		if key == "step" {
			return "step: is declared here, and only a condition (when:) carries step: beside field: — a selector and a polling Pattern's until: root elsewhere and carry no step:"
		}
		return fmt.Sprintf("%q is not a key a predicate admits — a predicate list is always AND, and there is no disjunction key", key)
	})...)

	if val := topLevelFields(node, "field")["field"]; val != nil && val.Kind == yaml.ScalarNode {
		fieldNameVal = val
	}
	if allowStep {
		if val := topLevelFields(node, "step")["step"]; val != nil && val.Kind == yaml.ScalarNode {
			stepVal = val
		}
	}

	if fieldNameVal == nil {
		line, column := position(node)
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field + ".field",
			ErrorCode: schema.CodeMismatch,
			Message:   `the schema at this position declares "field", and this file does not supply it`,
		})
	}
	problems = append(problems, checkExactlyOneOf(file, field, node, predicateOperators)...)
	problems = append(problems, checkPredicateOperand(file, field, node)...)
	return problems, fieldNameVal, stepVal
}

// checkRecordPredicate validates one assets:/observations: entry: the
// shape and operand types checkPredicateCore reads regardless of root, and
// this root's own two rules — field: is one declared field name and never
// a path, and it carries no step:, a selector rooting at the Record being
// filtered rather than at an earlier Step's (§3, §4, §12, issue #97).
func checkRecordPredicate(file, field string, node *yaml.Node, provider ProviderInfo, haveProvider bool) []problem.Problem {
	problems, fieldNameVal, _ := checkPredicateCore(file, field, node, false)
	if fieldNameVal != nil {
		problems = append(problems, checkRecordFieldName(file, field+".field", fieldNameVal, provider, haveProvider)...)
	}
	return problems
}

// checkConditionPredicate validates a when: condition: the shape and
// operand types checkPredicateCore reads regardless of root, step:'s own
// resolution against an id: this Procedure declares earlier, and field:'s
// own two rules read against that earlier Step's own bound Provider — one
// declared field name and never a path, resolved against the union of
// every Operation that Provider declares (§3, §4, §12, issue #97). It says
// nothing where whenVal is absent — a Step declaring no when: carries no
// condition and draws no code here.
func checkConditionPredicate(file, field string, whenVal *yaml.Node, stepIndex map[string]stepRefInfo) []problem.Problem {
	if whenVal == nil {
		return nil
	}
	problems, fieldNameVal, stepVal := checkPredicateCore(file, field, whenVal, true)

	var provider ProviderInfo
	haveProvider := false
	if stepVal == nil {
		if whenVal.Kind == yaml.MappingNode {
			line, column := position(whenVal)
			problems = append(problems, problem.Problem{
				File: file, Line: line, Column: column, Field: field + ".step",
				ErrorCode: schema.CodeMismatch,
				Message:   `the schema at this position declares "step", and this file does not supply it — a condition roots at a named earlier Step's Record and carries step: beside field:`,
			})
		}
	} else if ref, ok := stepIndex[stepVal.Value]; !ok {
		problems = append(problems, problem.Problem{
			File: file, Line: stepVal.Line, Column: stepVal.Column, Field: field + ".step",
			ErrorCode: CodeReferenceUnresolvable,
			Message:   fmt.Sprintf("step: %s names no id: this Procedure declares earlier", stepVal.Value),
		})
	} else {
		provider, haveProvider = ref.provider, ref.haveProvider
	}

	if fieldNameVal != nil {
		problems = append(problems, checkRecordFieldName(file, field+".field", fieldNameVal, provider, haveProvider)...)
	}
	return problems
}

// checkRecordFieldName validates a Record root's own field: value against
// §12's rule: one declared field name and nothing else — no descent, no
// brackets, no path (§3, §4, §12, issue #97). A value written as a path —
// told apart by its first character, the way every scalar in this grammar
// is — is schema-mismatch; a bare name resolving to nothing an Operation of
// the bound Provider projects is reference-unresolvable, the same code and
// the same check a reference already carries; and a bare name that
// resolves but is one the Manifest declares secret: is
// predicate-type-mismatch, that field reaching the Store as a constant no
// comparison can read. It says nothing about resolution where haveProvider
// is false — the Provider this field: would resolve against was never
// really named, and that fault is reported once, where the name itself was
// written.
func checkRecordFieldName(file, field string, node *yaml.Node, provider ProviderInfo, haveProvider bool) []problem.Problem {
	if strings.HasPrefix(node.Value, "$") {
		return []problem.Problem{{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: schema.CodeMismatch,
			Message:   "field: at a Record root is one declared field name — no descent, no brackets, no path",
		}}
	}
	if !haveProvider {
		return nil
	}
	if !provider.RecordFields[node.Value] {
		return []problem.Problem{{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: CodeReferenceUnresolvable,
			Message:   fmt.Sprintf("field: %s names what no Operation of the Provider projects", node.Value),
		}}
	}
	if provider.SecretFields[node.Value] {
		return []problem.Problem{{
			File: file, Line: node.Line, Column: node.Column, Field: field,
			ErrorCode: CodePredicateTypeMismatch,
			Message:   fmt.Sprintf("field: %s is declared secret: — it reaches the Store as a constant no comparison can read", node.Value),
		}}
	}
	return nil
}

// checkPredicateOperand validates every operator key node carries against
// the closed operand-type table (§12, issue #97) — every key, not merely
// the one checkExactlyOneOf would have picked, so a malformed entry
// carrying two operators still names every operand fault it carries rather
// than only the first.
func checkPredicateOperand(file, field string, node *yaml.Node) []problem.Problem {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var problems []problem.Problem
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, val := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		problems = append(problems, checkOneOperandType(file, field+"."+key.Value, key.Value, val)...)
	}
	return problems
}

// checkOneOperandType dispatches operator to the operand rule §12 fixes for
// it; a name outside the closed set draws nothing here, checkExactlyOneOf
// already having named that fault under unknown-key.
func checkOneOperandType(file, field, operator string, val *yaml.Node) []problem.Problem {
	switch operator {
	case "equals", "not_equals":
		return checkScalarOperandType(file, field, val, []schema.Type{schema.String})
	case "in":
		return checkInOperand(file, field, val)
	case "exists", "absent":
		return checkBooleanTrueOperand(file, field, operator, val)
	case "starts_with", "ends_with":
		return checkNonEmptyStringOperand(file, field, operator, val)
	case "greater_than", "less_than":
		return checkScalarOperandType(file, field, val, []schema.Type{schema.Integer, schema.Number, schema.Duration})
	case "older_than", "newer_than":
		return checkScalarOperandType(file, field, val, []schema.Type{schema.Duration, schema.Timestamp})
	default:
		return nil
	}
}

// checkScalarOperandType reports predicate-type-mismatch on val being
// neither a scalar nor readable as any member of allowed — equals and
// not_equals pass String, which reads any scalar's characters unconditionally
// and so never refuses one; greater_than, less_than, older_than and
// newer_than each pass their own narrower pair, which is what refuses a
// timestamp under the first two and everything but a duration or a
// timestamp under the last two (§12, issue #97).
func checkScalarOperandType(file, field string, val *yaml.Node, allowed []schema.Type) []problem.Problem {
	if val == nil || val.Kind != yaml.ScalarNode {
		line, column := position(val)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodePredicateTypeMismatch,
			Message:   "this operator's operand is a scalar, and this is neither one nor a reference — a predicate operand is always a literal",
		}}
	}
	for _, t := range allowed {
		if operandReadsAs(t, val) {
			return nil
		}
	}
	return []problem.Problem{{
		File: file, Line: val.Line, Column: val.Column, Field: field,
		ErrorCode: CodePredicateTypeMismatch,
		Message:   fmt.Sprintf("%q is not one of the operand types this operator takes", val.Value),
	}}
}

// checkBooleanTrueOperand reports predicate-type-mismatch on exists: or
// absent: carrying anything but true — negation lives in the operator's
// name and never in its operand, which is why exists: false is refused
// rather than read as a second spelling of absent: true (§12, issue #97).
func checkBooleanTrueOperand(file, field, operator string, val *yaml.Node) []problem.Problem {
	if val != nil && val.Kind == yaml.ScalarNode && val.Value == "true" {
		return nil
	}
	line, column := position(val)
	return []problem.Problem{{
		File: file, Line: line, Column: column, Field: field,
		ErrorCode: CodePredicateTypeMismatch,
		Message:   fmt.Sprintf("%s: takes only true — negation lives in the operator's name and never in its operand", operator),
	}}
}

// checkNonEmptyStringOperand reports predicate-type-mismatch on starts_with:
// or ends_with: carrying anything but a non-empty string — an empty one is
// refused as a predicate whose truth cannot depend on the value, a
// starts_with: "" being a destroy selector that reaches the whole series
// and reads like a filter (§12, issue #97).
func checkNonEmptyStringOperand(file, field, operator string, val *yaml.Node) []problem.Problem {
	if val == nil || val.Kind != yaml.ScalarNode {
		line, column := position(val)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodePredicateTypeMismatch,
			Message:   fmt.Sprintf("%s: takes a non-empty string", operator),
		}}
	}
	if val.Value != "" {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: val.Line, Column: val.Column, Field: field,
		ErrorCode: CodePredicateTypeMismatch,
		Message:   fmt.Sprintf("%s: \"\" reaches the whole series and reads like a filter — its truth cannot depend on the value", operator),
	}}
}

// checkInOperand reports predicate-type-mismatch on in: carrying anything
// but a list of two or more literals, all one type: empty, being a
// predicate whose truth cannot depend on the value; one member, being
// equals spelled twice; and a mixed-type list, which disjoins the values of
// one field over one population rather than reopening AND into OR (§12,
// issue #97).
func checkInOperand(file, field string, val *yaml.Node) []problem.Problem {
	if val == nil || val.Kind != yaml.SequenceNode {
		line, column := position(val)
		return []problem.Problem{{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodePredicateTypeMismatch,
			Message:   "in: is a list of two or more literals, all one type",
		}}
	}
	switch len(val.Content) {
	case 0:
		return []problem.Problem{{
			File: file, Line: val.Line, Column: val.Column, Field: field,
			ErrorCode: CodePredicateTypeMismatch,
			Message:   "an empty in: cannot depend on the value — its truth is fixed before any Record is read",
		}}
	case 1:
		item := val.Content[0]
		return []problem.Problem{{
			File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("%s[0]", field),
			ErrorCode: CodePredicateTypeMismatch,
			Message:   "a one-member in: is equals spelled twice — one filter, two ways to write it",
		}}
	}

	var problems []problem.Problem
	var firstCategory operandCategory
	for i, item := range val.Content {
		if item.Kind != yaml.ScalarNode {
			problems = append(problems, problem.Problem{
				File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("%s[%d]", field, i),
				ErrorCode: CodePredicateTypeMismatch,
				Message:   "an in: member is a bare literal",
			})
			continue
		}
		category := classifyOperand(item)
		if firstCategory == "" {
			firstCategory = category
			continue
		}
		if category != firstCategory {
			problems = append(problems, problem.Problem{
				File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("%s[%d]", field, i),
				ErrorCode: CodePredicateTypeMismatch,
				Message:   "in:'s members are not all one type",
			})
		}
	}
	return problems
}

// operandCategory is the type category classifyOperand reads a predicate
// operand's own characters into — its own small type rather than a bare
// string, on the rule that a domain concept with more than one caller earns
// one (§12, issue #97).
type operandCategory string

const (
	operandBoolean   operandCategory = "boolean"
	operandTimestamp operandCategory = "timestamp"
	operandDuration  operandCategory = "duration"
	// operandNumeric merges schema.Integer and schema.Number into one
	// domain — the rule that makes equals: 1 hold against 1.0, applied here
	// to in:'s own all-one-type check (§12, ADR-0081).
	operandNumeric operandCategory = "numeric"
	// operandString is the fallback every scalar reads as regardless of its
	// content — equals and not_equals' own operand rule, and in:'s own
	// category for anything the four above do not claim.
	operandString operandCategory = "string"
)

// classifyOperand reads node's own characters into the operandCategory
// checkInOperand compares an in: list's members by (§12, issue #97).
func classifyOperand(node *yaml.Node) operandCategory {
	switch {
	case operandReadsAs(schema.Boolean, node):
		return operandBoolean
	case operandReadsAs(schema.Timestamp, node):
		return operandTimestamp
	case operandReadsAs(schema.Duration, node):
		return operandDuration
	case operandReadsAs(schema.Integer, node) || operandReadsAs(schema.Number, node):
		return operandNumeric
	default:
		return operandString
	}
}

// operandReadsAs reports whether node's own characters read as t at an
// otherwise unconstrained position, reusing schema.CheckAt against a bare
// Schema{Type: t} rather than duplicating its own text-form rules here
// (§12, ADR-0081, issue #97).
func operandReadsAs(t schema.Type, node *yaml.Node) bool {
	return len(schema.CheckAt(node, schema.Schema{Type: t}, "", "")) == 0
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
