package artefact

import "gopkg.in/yaml.v3"

// A Procedure's Steps as a Run reads them (§3, §6, issue #136).
//
// procedure.go reads the same `steps:` sequence and asks a different question
// of it: whether what an author wrote resolves, binds and stays inside the
// envelope. This reads what the Steps *are*, for the engine that performs
// them — and it is a reader rather than a check, so it judges nothing and drops
// nothing: a Step whose `definition:` is absent comes back with an empty one,
// and what is wrong with it is `check`'s to report (ADR-0064).
//
// It is here rather than in internal/run because the shape of a Step is this
// package's fact. A second walk of `steps:` in the engine is a second reading
// of the artefact the checks were written against, and the day the two disagree
// is the day a Run performs something `check` never saw.

// Step is one member of a Procedure's `steps:` — a Step proper, the nested
// invocation that shares the sequence with it, or the Requirement that shares
// it with both, told apart by which of `procedure:`, `require:` and the three
// binding keys it carries (§3).
//
// Every member is held as it was authored and never as it resolves. `Kind` is
// not here: a Step's Kind is its Operation's, read off the Manifest, and a
// Procedure that named one would be declaring an authority its Provider owns
// (ADR-0025).
type Step struct {
	// ID is the Step's authored `id:` — what the Journal matches a Step by
	// across Runs, and what the Step table's second column renders (§7, §8).
	ID string
	// Definition, Operation and Target are the binding: the three names a
	// Step resolves against the repository's namespaces. All three are ""
	// on a nested invocation, which binds nothing.
	Definition, Operation, Target string
	// Invocation is the `procedure:` a nested invocation names, and "" on a
	// Step proper. An invocation is not a Step: it writes no Journal file
	// and reaches no Disposition, and its own Steps are Steps of the one
	// Run (§6, §7).
	Invocation string
	// Require is the `require:` a Requirement carries, and nil on both other
	// shapes. It is a Requirement's whole content beside its `id:`: the
	// predicate the Run must satisfy to go on, in the condition's own root —
	// a named earlier Step's Record, `step:` beside `field:` (§3, §12).
	//
	// A Requirement is not a Step either, and on the invocation's own three
	// grounds: it writes no Journal file, none of §12's Dispositions
	// describes one, and it takes no position in the sequence. What it does
	// instead is halt, which is how a Procedure that claims no effectful
	// authority stops a Run (§6, ADR-0116).
	Require *yaml.Node
	// Args are the Step's `args:`, by input name, as authored: a scalar
	// literal, or the mapping a reference is written as. What each may be
	// is §3's, and reading one against the Operation's declared input type
	// is the engine's at the moment the value is needed.
	Args map[string]*yaml.Node
	// Over, When and Bound are the three keys a Step may carry beside its
	// binding: the selector whose Expansion §6 resolves, the condition
	// §6 evaluates before it, and the Bound §5 makes mandatory on a
	// `destroy`. They are held as nodes because nothing here reads them —
	// their presence is what a milestone that has not built one declines
	// on, and their content belongs to the milestone that has.
	Over, When, Bound *yaml.Node
	// Line is where the Step's own entry begins, which is what a Refusal
	// citing this Step points a caret at (§8).
	Line int
}

// IsInvocation reports whether this member is a nested Procedure invocation
// rather than a Step: it names a `procedure:` and binds nothing.
func (s Step) IsInvocation() bool { return s.Invocation != "" }

// IsRequirement reports whether this member is a Requirement rather than a
// Step: it carries a `require:`, binds nothing and invokes nothing (§3,
// issue #236).
func (s Step) IsRequirement() bool { return s.Require != nil }

// ReadProcedureSteps reads a Procedure's `steps:` in written order, which is
// the order they run in and the order their `<nnnn>` is counted in (§6, §12).
//
// A root that is not a Procedure, or one carrying no legible `steps:`, reads as
// no Steps at all rather than as a fault — the reader's rule this package
// follows everywhere (ADR-0064).
func ReadProcedureSteps(root *yaml.Node) []Step {
	steps := topLevelFields(root, "steps")["steps"]
	if steps == nil || steps.Kind != yaml.SequenceNode {
		return nil
	}

	read := make([]Step, 0, len(steps.Content))
	for _, entry := range steps.Content {
		fields := topLevelFields(entry, "id", "definition", "operation", "target", "args", "over", "bound", "when", "procedure", "require")
		step := Step{
			ID:         scalarText(fields["id"]),
			Definition: scalarText(fields["definition"]),
			Operation:  scalarText(fields["operation"]),
			Target:     scalarText(fields["target"]),
			Invocation: scalarText(fields["procedure"]),
			Require:    fields["require"],
			Args:       argumentNodes(fields["args"]),
			Over:       fields["over"],
			When:       fields["when"],
			Bound:      fields["bound"],
			Line:       entry.Line,
		}
		read = append(read, step)
	}
	return read
}

// argumentNodes reads an `args:` mapping into its members by name, and answers
// nil where the Step writes none or writes something that is not a mapping. A
// nil answer reads the way a lookup into a nil mapping already does — every
// name absent — which is what a Step with no arguments means.
func argumentNodes(args *yaml.Node) map[string]*yaml.Node {
	if args == nil || args.Kind != yaml.MappingNode {
		return nil
	}
	read := make(map[string]*yaml.Node, len(args.Content)/2)
	for i := 0; i+1 < len(args.Content); i += 2 {
		if key := args.Content[i]; key.Kind == yaml.ScalarNode {
			read[key.Value] = args.Content[i+1]
		}
	}
	return read
}

// scalarText is a node's scalar text, and "" where the node is absent or is
// not a plain scalar. It is resolveScalar's answer without the second return:
// a reader has one reading for both, the absent key and the key carrying a
// mapping being equally *this Step names none*.
func scalarText(node *yaml.Node) string {
	value, ok := resolveScalar(node)
	if !ok {
		return ""
	}
	return value
}
