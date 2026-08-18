package artefact

import (
	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// The three members of the Bound fact §9's derived block carries, and it is a
// three-member set rather than a boolean because §5 gives the fact three
// states: a `destroy` Step's Bound is mandatory, an opaque `destroy` Step is
// the one Step that carries no Bound at all — writing one there is refused
// (bound-illegal) — and on the rest none is mandatory, a `read` Step having
// nothing for one to guard and a `mutate` Step's being its author's to write
// or leave out.
//
// A boolean would carry *you need not write one* and *writing one is refused*
// under one value, on the most severe Operation the tool runs. The names are
// the wire's own and milestone 11's MCP tool carries the same set (§9, §12).
// They are unexported on the rule the Kind set is read under: every surface
// reads the value this package computed, and the day one needs the word itself
// is the day these are exported.
const (
	boundMandatory = "mandatory"
	boundIllegal   = "illegal"
	boundNone      = "none"
)

// The two members of the Record cardinality §9's derived block carries: a
// series where the Operation's record: carries an over:, and one where it does
// not. An Operation declaring no record: at all carries neither this nor an
// identity, both being absent together rather than written empty (§3,
// ADR-0037).
const (
	recordSeries = "series"
	recordOne    = "one"
)

// patternMembers is §12's closed three-member Pattern set in §12's own order,
// which is the order patterns_resolved lists them in: a set the tool defines
// renders in the order the tool states it, and not in the order an author
// happened to write two of them.
var patternMembers = []string{"pagination", "polling", "retry"}

// OperationDetail is the derived half of `hyper operation`'s answer: the facts
// the Manifest's own lines do not carry in that form, computed once here so
// that no caller re-derives what `hyper` already has (§9).
//
// It stands beside the source rather than instead of it. The source teaches the
// format a caller is expected to author Definitions in, and this states what
// reading that format would otherwise cost — which Capability the request is
// written under, whether a Bound is mandatory, illegal or moot, and the three
// facts an Operation states by omission.
//
// Capabilities carries the one Capability the Operation's request is written
// under — an Operation uses exactly one (§12) — and is empty where the request
// block names neither Capability or names both, which is schema-mismatch from
// `check` and never a Capability this reader may pick between (ADR-0064).
//
// Bound is one of the three members above, and "" where kind: is not one of the
// three Kinds: the fact is read off a declared Kind, and there is none to read
// it off.
//
// PatternsResolved is the members of §12's three-member set the Operation
// declares, and it is empty rather than nil where it declares none: a caller
// asking which Patterns run around this call is answered *none of them*, which
// is a fact, where an absent member would say the question was not asked.
//
// RecordCardinality and RecordIdentity are the record: block's two facts and
// are both "" where there is no record: — a destroy, which projects no Record
// of its own. The identity is the identity: scalar verbatim, a template hole
// and a response path alike.
//
// Repeatability is the effective value and not the declared one: an Operation
// whose Manifest omits repeatability: gets run-once where it effects and
// repeatable where it reads (§12, ADR-0037). run-once is rendered even though
// no artefact may write that word, which makes it exactly parallel to opaque —
// a fact no artefact declares and every surface renders.
//
// Deadline is the authored spelling, 30s, and DeadlineSeconds is that duration
// in seconds: §9 fixed the wire name and its unit with it, and the spelling is
// what a page standing beside the source renders, that being what the source
// says. DeadlineSeconds is a pointer because 0s is a duration an author can
// write, so 0 is a value this member must be able to state rather than the
// absence of one (§7).
//
// ConcurrencyLimit is the effective limit and is always present: the declared
// concurrency:, or 1 where absent, and 1 on every mutate and destroy, whose
// Expansion is serial and which may not declare the key at all. A caller asking
// *how many at once* gets a number for every Operation, and the rule about
// which Kinds may author the key stays in §3 where authoring rules live, rather
// than being inferred here from a field that came back empty (ADR-0045).
type OperationDetail struct {
	Capabilities      []string
	Bound             string
	PatternsResolved  []string
	RecordCardinality string
	RecordIdentity    string
	Repeatability     string
	Deadline          string
	DeadlineSeconds   *int
	ConcurrencyLimit  int
}

// ReadOperationDetail reads the derived block for one Operation of a Manifest's
// own root. Like ReadManifestFacts it judges nothing and drops what it cannot
// read: a member this returns empty is a member the Manifest did not legibly
// state, which is check's to report and never this reader's to guess at
// (ADR-0064).
//
// It answers with the empty detail where name is not a key of a legible
// operations: block — the nothing-to-read case of that same rule. There is no
// second return value saying so, because resolution is the surface's and has
// already happened where the Operation's own source came from: OperationSource
// and this read one mapping for one key, so neither can find what the other did
// not (§9, ADR-0060).
func ReadOperationDetail(root *yaml.Node, name string) OperationDetail {
	op := operationNode(root, name)
	capability, _ := operationCapability(op)

	var detail OperationDetail
	if capability != "" {
		detail.Capabilities = []string{capability}
	}
	info := operationInfoFromNode(op)
	detail.Bound = boundRule(info)
	detail.PatternsResolved = patternsResolved(op)
	detail.RecordCardinality, detail.RecordIdentity = recordProjection(info)
	detail.Repeatability = effectiveRepeatability(info)
	detail.Deadline = scalarValue(topLevelFields(op, "deadline")["deadline"])
	if seconds, authored := schema.DurationSeconds(detail.Deadline); authored {
		detail.DeadlineSeconds = &seconds
	}
	detail.ConcurrencyLimit = effectiveConcurrencyLimit(op, info)
	return detail
}

// boundRule is §5's Bound fact for one Operation, read off the Kind it declares
// and the Capability its request uses: mandatory on a `destroy`, illegal on the
// `destroy` whose request is opaque — a count of the commands it ran says
// nothing about what any of them did — and none on a `read` or a `mutate`,
// where a Step author is required to write no Bound at all.
func boundRule(info OperationInfo) string {
	switch {
	case info.IsOpaqueDestroy():
		return boundIllegal
	case info.Kind == "destroy":
		return boundMandatory
	case info.Kind == "read" || info.Kind == "mutate":
		return boundNone
	default:
		return ""
	}
}

// operationNode is the node one Operation of a Manifest is declared by, and nil
// where the name is not a key of a legible operations: block — the same lookup
// OperationSource performs over the same mapping, matching byte-exact as every
// name in the tool does (§9, ADR-0060).
func operationNode(root *yaml.Node, name string) *yaml.Node {
	operations := operationsMapping(root)
	if operations == nil {
		return nil
	}
	for i := 0; i+1 < len(operations.Content); i += 2 {
		if key := operations.Content[i]; key.Kind == yaml.ScalarNode && key.Value == name {
			return operations.Content[i+1]
		}
	}
	return nil
}

// patternsResolved is the members of §12's Pattern set this Operation declares,
// in §12's order. A key under patterns: that names none of the three is not a
// Pattern hyper performs — it is unknown-key from check — and contributes
// nothing here, exactly as a Kind outside the three does (ADR-0064).
func patternsResolved(op *yaml.Node) []string {
	patterns := topLevelFields(op, "patterns")["patterns"]
	resolved := []string{}
	if patterns == nil || patterns.Kind != yaml.MappingNode {
		return resolved
	}
	declared := topLevelFields(patterns, patternMembers...)
	for _, member := range patternMembers {
		if declared[member] != nil {
			resolved = append(resolved, member)
		}
	}
	return resolved
}

// recordProjection is the record: block's two derived facts, read through the
// same OperationInfo every check reads a projection off: the cardinality its
// over: decides, and the identity: it declares, verbatim. Both are "" where the
// Operation declares no record: — RecordFields being nil is that absence, the
// mapping being made wherever a block was there to read — which is what makes
// the pair absent together on a destroy (§3, §4).
//
// The absence is keyed on the block and not on the pair, so an Operation
// carrying a record: and no legible identity: states the cardinality it did
// declare and nothing else. That is the drop rule and not a half-written fact:
// what the block says is a fact, and a record: with no identity is
// identity-undeclared from check — a Record with no name being the fault, not
// this reader's to paper over by dropping the cardinality beside it (ADR-0064).
func recordProjection(info OperationInfo) (cardinality, identity string) {
	if info.RecordFields == nil {
		return "", ""
	}
	if info.HasSeries {
		return recordSeries, info.Identity
	}
	return recordOne, info.Identity
}

// effectiveConcurrencyLimit is ADR-0045's limit: the declared concurrency: on a
// read, and 1 everywhere else — a read that omits the key, and every mutate and
// destroy, whose Expansion is serial and which may not declare the key at all.
// A limit written where no Kind may author one is manifest-inconsistent from
// check and is not a number this reader may report as effective: what governs
// there is 1, whatever was written.
func effectiveConcurrencyLimit(op *yaml.Node, info OperationInfo) int {
	const serial = 1
	if info.Kind != "read" {
		return serial
	}
	declared := integerScalar(topLevelFields(op, "concurrency")["concurrency"])
	if declared == nil {
		return serial
	}
	return *declared
}

// operationsMapping is a Manifest's operations: block where it is legible, and
// nil where the Manifest has none to look in — the one place the two questions
// asked of that block, *which Operation* and *where are its lines*, agree on
// what they are looking at (§9, ADR-0064).
func operationsMapping(root *yaml.Node) *yaml.Node {
	operations := topLevelFields(root, "operations")["operations"]
	if operations == nil || operations.Kind != yaml.MappingNode {
		return nil
	}
	return operations
}
