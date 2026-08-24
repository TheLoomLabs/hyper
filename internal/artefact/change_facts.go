package artefact

import (
	"slices"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// The code facts a review reads a direction off (§8, §12, issue #168).
//
// They are §12's nine `THE CODE MOVED` classes **less the digests**, which is
// Run-recorded and has no line in any artefact — one vocabulary across both
// surfaces, and the difference between them is that a review reads a file and a
// Comparison reads two Runs.
//
// Every one of them is authored in the artefact's own lines, which is what
// makes this reader take a root and nothing else: the baseline a review
// compares against is the same artefact at the revision the header names, and
// resolving a Definition or a Manifest at that revision would be the reach past
// the gutter ADR-0057 refuses. So the eight read here are the eight that can be
// read off one file, and that is not a shortfall — it is the same set (§8).
//
// It reads and judges nothing, on ReadProcedureMarks's own rule: a key an
// artefact never wrote is a fact with no line and the empty value, which is what
// §8's own `–` renders, and a key written in a shape this cannot read is a fact
// with a line and nothing derived. What is wrong with either is `check`'s to
// report (ADR-0064).

// FactShape is what a fact's value is, which decides both how a row renders it
// and what makes two of them differ (§8). It is the shape of the value at the
// row and never the class above it: the Target set class alone carries a set
// for a Procedure's envelope and a scalar for the `target:` a Step binds.
type FactShape int

const (
	// FactSet is a set of names, comparing by set equality: declared Kinds,
	// a Target set, required Capabilities, the Operations a Manifest
	// exposes, a Definition's `destroy:` claim. It is the shape a direction
	// is decidable on by inclusion, which is why the rule is quantified over
	// the shape rather than listed over the classes (§12).
	FactSet FactShape = iota
	// FactBound is a Step's `bound:`: a magnitude, comparing numerically,
	// and the one scalar a direction is decidable on.
	FactBound
	// FactScalar is a scalar no direction is available for — a Step's
	// `target:` and a credential slot's variable. It takes `changed` and its
	// full before-and-after text.
	FactScalar
	// FactCadence is a Procedure's declared recurrence: a scalar under the
	// shape rule and a stacked cell under the mandatory gloss, which are two
	// rules meeting in one place and neither is the other (§8, §10,
	// ADR-0005, ADR-0063).
	FactCadence
	// FactSelector is a Step's `over:`, in any of its three forms. It takes
	// `changed` however it moved: predicate subsumption is undecidable in
	// general, so a surface calling `equals: preview` → `starts_with:
	// preview-` a widening would be inventing the one thing it may not
	// invent (§12).
	FactSelector
)

// ChangeFact is one `(subject, fact)` pair as one artefact's own lines carry
// it: which key it is written at, which subject inside the artefact it belongs
// to, where it is written, and the value.
//
// A class emits one row per pair rather than one row, so a Definition's claimed
// Kinds and the `destroy` Operations it names are two facts under one class
// (§12). The class names are the grouping and reach no screen: what a row names
// is the key, which is what a reader greps for and what the gutter marks.
type ChangeFact struct {
	// Key is the key the fact is written at — `kinds`, `targets`,
	// `destroy`, `capabilities`, `operations`, `cadence`, `target`, `over`,
	// `bound` — or, for the credential source, the slot it belongs to
	// spelled as §8's own `credential <slot>`.
	Key string
	// Step is the id of the Step this fact belongs to, and "" where the
	// subject is the artefact itself. It is the coordinate a flag citing the
	// fact carries, and the identity the two sides of a range are paired on:
	// a Step on one side only has no before-and-after of its own.
	Step string
	// SubjectLine is the line the fact's subject opens on — a Step's own
	// `- id:`, or the artefact's first line where the subject is the
	// artefact. It is the anchor of last resort for a citation, and it is a
	// line the gutter marks on every Step there is (§8).
	SubjectLine int
	// Lines are the lines the fact is written across: the key's own, and
	// each line a member of the value stands on. They are what a citation is
	// chosen from — a flag cites the line carrying its subject, and where a
	// fact spans a block the line that moved is the one carrying it.
	//
	// It is empty where the artefact writes no such key, which is a fact
	// stated by omission and rendered `–` (§8).
	Lines []int
	// Shape is what the value is, which decides both what a row renders and
	// what makes two readings of it differ. It follows the value at the row
	// and never the class above it, which is why it is a member here rather
	// than a fact about the key: the Target set class alone carries a set
	// for a Procedure's envelope and a scalar for the `target:` a Step binds
	// (§8).
	Shape FactShape
	// Members are a set's members or a selector's, in the order a row
	// renders them: sorted by Unicode code point wherever the fact compares
	// as a set, and as authored for a `values:` selector, whose order *is*
	// the fact (§6, §8).
	Members []string
	// Value is a scalar's text as the artefact wrote it, and a selector's
	// form name — `values`, `assets` or `observations`. A cell dropping the
	// form could not tell an `assets` selector from an `observations` one,
	// which is the difference between ranging over what `hyper` built and
	// over what it read (§5, §8).
	Value string
}

// Written reports whether the artefact writes this fact at all. A fact with no
// line is one stated by omission — an absent `bound:` being unbounded, an
// absent `over:` a Step invoked once, an absent `cadence:` no recurrence — and
// what renders in its place is `–` rather than a naming of what the absence
// means, which is a claim and not a value (§8).
func (f ChangeFact) Written() bool { return len(f.Lines) > 0 }

// Same reports whether two readings of one fact are the same fact, which is
// what decides whether a row is emitted at all. **A fact that did not move
// emits no row, however its bytes moved**: a reordered set moves the file and
// moves nothing this reports, which is the comparison being by the fact's own
// equality and never by the text (§8, §12).
func (f ChangeFact) Same(other ChangeFact) bool {
	return f.Value == other.Value && slices.Equal(f.Members, other.Members)
}

// ReadChangeFacts is one artefact's facts, in the order its own lines carry
// them.
//
// A **Repository declaration** answers none, and that is the enumeration
// holding rather than a roster left short: its `version:` is the pin, which is
// `the digests`' `hyper_version` and has no class here, and its `retention:` is
// one of the lines §12's catch-all counts. Both move on a review's screen — the
// change column marks them like any other line — and neither is a fact this
// vocabulary names (§7, §12).
func ReadChangeFacts(kind string, root *yaml.Node) []ChangeFact {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	opens := rootLine(root)
	switch kind {
	case KindProcedure:
		return procedureChangeFacts(root, opens)
	case KindDefinition:
		return keyChangeFacts(root, opens, "kinds", "targets", "destroy")
	case KindTargetDeclaration:
		return append(keyChangeFacts(root, opens, "kinds"), credentialChangeFacts(root, opens)...)
	case KindProvider:
		return append(keyChangeFacts(root, opens, "capabilities"), operationSetFact(root, opens))
	}
	return nil
}

// procedureChangeFacts is a Procedure's: its declared envelope, its Cadence,
// and — on every Step that declares an id to be paired by — the Target it
// binds, the selector it expands and the Bound standing behind it.
//
// A nested invocation carries none of the three. It binds nothing, expands
// nothing and bounds nothing, and the envelope it reaches is the invoked
// Procedure's own fact on that Procedure's own review (§3, §8).
func procedureChangeFacts(root *yaml.Node, opens int) []ChangeFact {
	facts := keyChangeFacts(root, opens, "targets")
	cadence := ChangeFact{Key: "cadence", SubjectLine: opens, Shape: FactCadence}
	if line := TopLevelKeyLine(root, "cadence"); line > 0 {
		declared := topLevelFields(root, "cadence")["cadence"]
		cadence.Lines = scalarFactLines(line, declared)
		cadence.Value = scalarText(declared)
	}
	facts = append(facts, cadence)
	for _, step := range ReadProcedureSteps(root) {
		if step.ID == "" || step.IsInvocation() {
			continue
		}
		fields := stepFields(root, step)
		facts = append(facts,
			stepScalarFact(step, "target", fields, FactScalar),
			stepScalarFact(step, "bound", fields, FactBound),
			selectorFact(step, fields),
		)
	}
	return facts
}

// stepFields is one Step's own key lines, by key: the entry re-walked for the
// lines its keys are written on, which is what ReadProcedureSteps answers with
// values rather than positions for.
func stepFields(root *yaml.Node, step Step) []*yaml.Node {
	steps := topLevelFields(root, "steps")["steps"]
	if steps == nil {
		return nil
	}
	for _, entry := range steps.Content {
		if entry.Line == step.Line && entry.Kind == yaml.MappingNode {
			return entry.Content
		}
	}
	return nil
}

// stepScalarFact is one of a Step's scalar keys as a fact: its own line where
// the Step writes it, and the empty value where it does not.
func stepScalarFact(step Step, key string, entry []*yaml.Node, shape FactShape) ChangeFact {
	fact := ChangeFact{Key: key, Step: step.ID, SubjectLine: step.Line, Shape: shape}
	if keyNode, val := entryField(entry, key); keyNode != nil {
		fact.Lines = scalarFactLines(keyNode.Line, val)
		fact.Value = scalarText(val)
	}
	return fact
}

// selectorFact is a Step's `over:` as a fact: the form heading it, the members
// beneath, and every line the selector is written across.
//
// The whole subtree is the fact, which is why the lines are the span rather
// than the members': a conjunct's operand is as much the selector as the field
// it names, so a citation has to be able to land on the line that moved (§8).
func selectorFact(step Step, entry []*yaml.Node) ChangeFact {
	fact := ChangeFact{Key: "over", Step: step.ID, SubjectLine: step.Line, Shape: FactSelector}
	keyNode, val := entryField(entry, "over")
	if keyNode == nil {
		return fact
	}
	fact.Lines = subtreeLines(keyNode.Line, val)
	fact.Value, fact.Members = readSelector(val)
	return fact
}

// readSelector is a selector's form and its members in the notation §8 renders
// both surfaces in.
//
// A `values:` selector's members are the scalars **as authored**: its order is
// the fact, §6 ordering an Expansion by the artefact where the selector is a
// literal list, so a reordering moves which member a Run reaches first and
// sorting it would hide the whole of what changed. A predicate selector's are
// one `field operator operand` line per conjunct, sorted by Unicode code point
// on the rendered line: a predicate list is always AND and does not
// short-circuit, so conjunct order carries no meaning and two cells can only be
// differenced by eye where both run in the same order (§8, §12).
func readSelector(over *yaml.Node) (form string, members []string) {
	forms := topLevelFields(over, "assets", "observations", "values")
	if values := forms["values"]; values != nil {
		return "values", scalarSequence(values)
	}
	for _, named := range []string{"assets", "observations"} {
		list := forms[named]
		if list == nil {
			continue
		}
		for _, conjunct := range list.Content {
			if rendered := renderPredicate(conjunct); rendered != "" {
				members = append(members, rendered)
			}
		}
		sort.Strings(members)
		return named, members
	}
	return "", nil
}

// renderPredicate is one conjunct as §8 renders it: `field operator operand`,
// colons dropped. `exists` and `absent` render bare, their operand being the
// only one either takes.
//
// A conjunct this cannot read renders nothing at all rather than half a line:
// what is wrong with it is `check`'s to report, and a rendering that showed a
// field with no operator would be this surface asserting a predicate nobody
// wrote (ADR-0064).
func renderPredicate(conjunct *yaml.Node) string {
	if conjunct == nil || conjunct.Kind != yaml.MappingNode {
		return ""
	}
	fields := topLevelFields(conjunct, append([]string{"field"}, predicateOperators...)...)
	field := scalarText(fields["field"])
	if field == "" {
		return ""
	}
	for _, operator := range predicateOperators {
		operand, written := fields[operator]
		if !written {
			continue
		}
		switch operator {
		case "exists", "absent":
			return field + " " + operator
		}
		return strings.TrimRight(field+" "+operator+" "+renderOperand(operand), " ")
	}
	return ""
}

// renderOperand is a predicate's operand as a row carries it: the scalar as
// written, and a list in the flow notation the format writes one in — the one
// operator taking a list is `in:`, and a run of bare members would read as
// several operands rather than as one (§3, §12).
//
// **It diverges from the Step table's own conjunct cell, which renders nothing
// composed** (internal/cli/show.go, ADR-0059), and the divergence is the two
// rows being two things. That cell states one Run's selector, so an operand it
// cannot write is a value the reader goes to the file for; this row exists
// **because the fact moved**, and two sides rendering identically where an
// `in:` list is the whole of what changed would be the omission ADR-0026
// forbids — a row asserting a difference it does not show. §8 fixes no notation
// for a list operand either way, so what settles it here is the row it is in.
func renderOperand(operand *yaml.Node) string {
	if operand == nil {
		return ""
	}
	if operand.Kind == yaml.SequenceNode {
		return "[" + strings.Join(scalarSequence(operand), ", ") + "]"
	}
	return scalarText(operand)
}

// keyChangeFacts is the artefact's own top-level set-shaped keys as facts, in
// the order named.
func keyChangeFacts(root *yaml.Node, opens int, keys ...string) []ChangeFact {
	fields := topLevelFields(root, keys...)
	facts := make([]ChangeFact, 0, len(keys))
	for _, key := range keys {
		fact := ChangeFact{Key: key, SubjectLine: opens, Shape: FactSet}
		if line := TopLevelKeyLine(root, key); line > 0 {
			fact.Lines = memberLines(line, fields[key])
			fact.Members = sortedSet(scalarSequence(fields[key]))
		}
		facts = append(facts, fact)
	}
	return facts
}

// operationSetFact is the Operations a Manifest exposes: the keys of
// `operations:` and nothing beneath them. An Operation's own body is not this
// fact — what moved when a request changed is the digest a Run records, which
// is `the digests` and has no line here (§12).
func operationSetFact(root *yaml.Node, opens int) ChangeFact {
	fact := ChangeFact{Key: "operations", SubjectLine: opens, Shape: FactSet}
	line := TopLevelKeyLine(root, "operations")
	if line == 0 {
		return fact
	}
	fact.Lines = []int{line}
	operations := topLevelFields(root, "operations")["operations"]
	if operations == nil || operations.Kind != yaml.MappingNode {
		return fact
	}
	for i := 0; i+1 < len(operations.Content); i += 2 {
		if key := operations.Content[i]; key.Kind == yaml.ScalarNode {
			fact.Members = append(fact.Members, key.Value)
			fact.Lines = append(fact.Lines, key.Line)
		}
	}
	fact.Members = sortedSet(fact.Members)
	return fact
}

// credentialChangeFacts is the environment variable each of a Target
// declaration's credential slots resolves from — **names only, never values**:
// §6's whole reason for naming variables explicitly rather than deriving them
// is that `env: STAGING_TOKEN` → `env: PROD_TOKEN` is a visible one-line edit,
// and the secret behind an unchanged name rotating is a world fact `hyper`
// deliberately cannot see (§7, §12, ADR-0007).
//
// The pair is a **slot** rather than the declaration, so one carrying three of
// them answers three facts. The slot is named inside the key rather than beside
// it, which is §8's own `credential token`: a slot is a coordinate the wire
// carries no member for, only a Procedure's Step having one (§12).
//
// A slot on one side of a range only is a subject with no before-and-after,
// exactly as a Step is — which falls out of the pairing rather than being
// stated by it, a key neither side's roster fixes having nothing to pair with.
func credentialChangeFacts(root *yaml.Node, opens int) []ChangeFact {
	auth := topLevelFields(root, "auth")["auth"]
	if auth == nil || auth.Kind != yaml.MappingNode {
		return nil
	}
	var facts []ChangeFact
	for i := 0; i+1 < len(auth.Content); i += 2 {
		key, val := auth.Content[i], auth.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		facts = append(facts, ChangeFact{
			Key:         "credential " + key.Value,
			SubjectLine: opens,
			Lines:       subtreeLines(key.Line, val),
			Shape:       FactScalar,
			Value:       envVariable(val),
		})
	}
	return facts
}

// entryField is one key of a Step's own mapping content: the key node and its
// value, and two nils where the Step writes no such key.
func entryField(entry []*yaml.Node, name string) (key, val *yaml.Node) {
	for i := 0; i+1 < len(entry); i += 2 {
		if held := entry[i]; held.Kind == yaml.ScalarNode && held.Value == name {
			return held, entry[i+1]
		}
	}
	return nil, nil
}

// scalarFactLines is a scalar fact's lines: the key's own, and the value's
// where a block scalar puts it somewhere else.
func scalarFactLines(key int, val *yaml.Node) []int {
	if val == nil || val.Line == key {
		return []int{key}
	}
	return sortedLines([]int{key, val.Line})
}

// memberLines is a set fact's lines: the key's own, and the line each member
// stands on — which is the key's line again for a flow sequence and one line
// each for a block one.
func memberLines(key int, val *yaml.Node) []int {
	lines := []int{key}
	if val != nil {
		for _, member := range val.Content {
			lines = append(lines, member.Line)
		}
	}
	return sortedLines(lines)
}

// subtreeLines is every line a fact is written across: the key's own, and every
// line any node beneath it stands on.
func subtreeLines(key int, val *yaml.Node) []int {
	lines := []int{key}
	var walk func(node *yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil {
			return
		}
		lines = append(lines, node.Line)
		for _, child := range node.Content {
			walk(child)
		}
	}
	walk(val)
	return sortedLines(lines)
}

// sortedLines is a fact's lines in ascending order with the repeats taken out:
// one line carrying two members of one fact is one line to cite.
func sortedLines(lines []int) []int {
	slices.Sort(lines)
	return slices.Compact(lines)
}

// sortedSet is a set's members as a row renders and compares them: sorted by
// Unicode code point, with the repeats taken out. **Sorting where a fact
// compares as a set is what makes the two cells readable against each other** —
// a thirteen-name set beside a twelve-name one can be differenced by eye only
// where both sides run in the same order (§8).
func sortedSet(members []string) []string {
	if len(members) == 0 {
		return nil
	}
	sorted := slices.Clone(members)
	slices.Sort(sorted)
	return slices.Compact(sorted)
}
