package artefact

import "gopkg.in/yaml.v3"

// DefinitionFacts is what a Definition claims, in the shape a surface reports
// it: ordered lists rather than the membership sets DefinitionInfo carries.
//
// It is ReadTargetFacts's other end. The two readings are one artefact read for
// two questions: a check asks *does this Definition claim that Kind*, which is
// a set, and a row states *what does this Definition claim*, which is an
// enumeration answered in the file's own order — a claim silently reduced or
// re-sorted is not the claim the reviewer has open beside it (§3, §5, §8).
//
// Targets is every member of targets: as written, including one that resolves
// to no declaration. DefinitionInfo.Targets drops those, a check having nothing
// to check them against; a row may not, a Definition claiming three Targets and
// rendering two rows saying the third was never claimed (§8, ADR-0064).
//
// Each list is nil where its key is absent, which is the ordinary absence rule
// a reader reads off it (§7): a Definition claiming no `destroy` Operation
// carries no destroy: at all.
type DefinitionFacts struct {
	Kinds   []string
	Destroy []string
	Targets []string
}

// ReadDefinitionFacts reads those three facts off a Definition's own root. It
// judges none of them, on ReadTargetFacts's own rule: a Definition naming a
// Kind outside the closed set, or a Target that is not there, states what it
// states here and earns its problem from check (ADR-0064). What it drops is
// what it cannot read — a list member that is not a plain scalar has no value
// to report.
func ReadDefinitionFacts(root *yaml.Node) DefinitionFacts {
	fields := topLevelFields(root, "kinds", "destroy", "targets")
	return DefinitionFacts{
		Kinds:   scalarSequence(fields["kinds"]),
		Destroy: scalarSequence(fields["destroy"]),
		Targets: scalarSequence(fields["targets"]),
	}
}
