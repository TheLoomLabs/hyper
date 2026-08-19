package artefact

import "gopkg.in/yaml.v3"

// The four rosters §8 fixes for the artefacts that are not a Procedure — what
// each one's gutter marks — read off the file under review and nothing else.
//
// A Procedure's roster is next door in procedure_marks.go, and it is apart for
// the reason its marks are: a Step's Kind is declared in a Manifest two
// directories away and its envelope quantifies over every Step's `target:` at
// once, so that reader is handed the repository's own namespaces. Every mark
// here is derived from the artefact's own lines, which is why these four take a
// root and nothing else — and why a Definition whose `provider:` resolves to
// nothing renders complete and unmarked: nothing on this screen is derived from
// a Manifest, so there is no name here for the gutter to fail to follow (§8,
// ADR-0064, issue #122).
//
// They read and judge nothing, on ReadProcedureMarks's own rule. A key an
// artefact never wrote has no line to mark; a key it wrote in a shape this
// cannot read has a line and nothing derived, which is a line the gutter leaves
// unmarked rather than one it marks empty. What is wrong with either is
// `check`'s to report and never a reader's to guess at (ADR-0064).

// KeyMark is one top-level key's own line and what `hyper` derived from what it
// carries: the line the key is written on, 0 where the artefact writes no such
// key, and the derived values in the artefact's own order.
//
// The values are the fact and not the marker: which of them is upper-cased,
// what stands in front of them and how they are separated is §8's vocabulary
// and the rendering surface's to compose, exactly as a StepMark's Kind is
// (§8).
//
// Values is empty on a key that is written and yields nothing — a `hosts:` that
// is not a list of scalars, an `auth:` naming neither of §12's two schemes.
// That is a line with no derived fact rather than a fact that is empty, and the
// two are one thing to every surface: nothing was derived, so nothing is
// marked.
type KeyMark struct {
	Line   int
	Values []string
}

// DefinitionMarks is §8's roster on a Definition: the Kinds it claims, the
// `destroy` Operations it names, and the Targets it may bind, each beside the
// line that makes the claim.
//
// All three are authored in the file being read, which is the whole of why a
// Definition's `provider:` carries no mark: this screen annotates what `hyper`
// derived from these lines, and nothing it derived reaches a Manifest (§8).
type DefinitionMarks struct {
	Kinds, Destroy, Targets KeyMark
}

// ReadDefinitionMarks reads those three off a Definition's own root. The values
// are ReadDefinitionFacts's, so the Kinds the gutter marks and the Kinds
// `AUTHORITY` states are one reading of one artefact rather than two that
// agree; what this adds is the line each of them is written on, which is the
// anchor a mark needs and a row does not (§8).
func ReadDefinitionMarks(root *yaml.Node) DefinitionMarks {
	facts := ReadDefinitionFacts(root)
	return DefinitionMarks{
		Kinds:   KeyMark{topLevelKeyLine(root, "kinds"), facts.Kinds},
		Destroy: KeyMark{topLevelKeyLine(root, "destroy"), facts.Destroy},
		Targets: KeyMark{topLevelKeyLine(root, "targets"), facts.Targets},
	}
}

// TargetDeclarationMarks is §8's roster on a Target declaration: the Kinds it
// accepts, the Capabilities it grants, the hosts it grants, the environment
// variable each credential slot resolves from, and the opt-in that admits an
// `opaque` `destroy` (§4, §8).
//
// Credentials is one entry per member of `auth:`, in the mapping's own order,
// each carrying the slot's own line and the variable it names. A slot whose
// value is not the mapping §4 fixes carries no value: the slot is still a line
// of the file and still renders, and what is wrong with it is
// `credential-slot-malformed` from `check` (ADR-0064).
//
// OpaqueDestroy is the opt-in's own line where the declaration grants it, and 0
// where it writes none or writes the default. The grant is the whole of what is
// derived there and there is no value beside it, so the line alone is the
// mark's supply — a declaration writing `opaque-destroy: false` admits exactly
// what one writing nothing admits, and a mark there would name a grant that was
// not made.
type TargetDeclarationMarks struct {
	Kinds, Capabilities, Hosts KeyMark
	Credentials                []KeyMark
	OpaqueDestroy              int
}

// ReadTargetDeclarationMarks reads that roster off a Target declaration's own
// root. The three list-shaped facts are ReadTargetFacts's, on
// ReadDefinitionMarks's own rule; the credential slots are walked here because
// what a mark needs off a slot is the line its key is written on, which is the
// one thing a row reporting a Target's credentials has never asked for (§9).
func ReadTargetDeclarationMarks(root *yaml.Node) TargetDeclarationMarks {
	facts := ReadTargetFacts(root)
	marks := TargetDeclarationMarks{
		Kinds:        KeyMark{topLevelKeyLine(root, "kinds"), facts.Kinds},
		Capabilities: KeyMark{topLevelKeyLine(root, "capabilities"), facts.Capabilities},
		Hosts:        KeyMark{topLevelKeyLine(root, "hosts"), facts.Hosts},
		Credentials:  credentialMarks(topLevelFields(root, "auth")["auth"]),
	}
	if grantsOpaqueDestroy(root) {
		marks.OpaqueDestroy = topLevelKeyLine(root, "opaque-destroy")
	}
	return marks
}

// credentialMarks is the `auth:` mapping's members in the mapping's own order,
// each paired with the variable its own `env:` names. The variable's name and
// never its value — nothing `hyper` writes has ever held a secret, and this
// surface resolves no credential at all (§7, §9, ADR-0007).
func credentialMarks(authVal *yaml.Node) []KeyMark {
	if authVal == nil || authVal.Kind != yaml.MappingNode {
		return nil
	}
	var marks []KeyMark
	for i := 0; i+1 < len(authVal.Content); i += 2 {
		key, val := authVal.Content[i], authVal.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		mark := KeyMark{Line: key.Line}
		if env := envVariable(val); env != "" {
			mark.Values = []string{env}
		}
		marks = append(marks, mark)
	}
	return marks
}

// ManifestMarks is §8's roster on a Manifest: the Auth scheme it names, the
// Capabilities its Operations require, and one entry per Operation.
type ManifestMarks struct {
	Auth, Capabilities KeyMark
	Operations         []OperationMark
}

// OperationMark is one member of `operations:` as the gutter marks it, beside
// the line its key is written on — which is the line that binds the claim, an
// Operation's body being everything indented beneath its name.
//
// Kind is what the Manifest declares and is never inferred from the Operation's
// name (§12), and it is "" where the entry declares none legibly.
//
// Repeatability is the effective value and not the declared one: an Operation
// whose Manifest omits `repeatability:` is run-once where it effects and
// repeatable where it reads (§12, ADR-0037). It is the same derivation `hyper
// operation` renders, so `run-once` — a word no artefact may write — reaches
// this column exactly as `opaque` does. It is "" where `kind:` is not one of
// the three, there being no default to read off a Kind the Manifest never
// stated.
//
// Opaque is whether the Operation's request uses an Opaque Capability, read off
// the request block because no artefact anywhere declares it (§12).
type OperationMark struct {
	Line          int
	Kind          string
	Repeatability string
	Opaque        bool
}

// ReadManifestMarks reads that roster off a Manifest's own root, the built-in's
// compiled-in bytes included: what the load handed this reader is a parsed
// root, and where those bytes came from is the header's fact and not a mark's
// (§8, §11, ADR-0039).
//
// Every Operation is read through operationInfoFromNode, which is the reading
// every check makes of the same entry, so the Kind the gutter marks and the
// Kind a Step is refused against are one read of one artefact (§4, §9).
func ReadManifestMarks(root *yaml.Node) ManifestMarks {
	fields := topLevelFields(root, "capabilities", "operations")
	return ManifestMarks{
		Auth:         KeyMark{topLevelKeyLine(root, "auth"), values(authSchemeNamed(root))},
		Capabilities: KeyMark{topLevelKeyLine(root, "capabilities"), scalarSequence(fields["capabilities"])},
		Operations:   operationMarks(fields["operations"]),
	}
}

// authSchemeNamed is the scheme the Manifest names: `header` or `basic`, §12
// closing the set at two.
//
// It is the scheme's name rather than the header it composes. The composition
// is what `hyper provider` renders, an answer to *what does a call to this
// Provider send*; the gutter is annotating a line, and what the `auth:` line
// itself does not say is which of the two closed members this Manifest names
// (§8, §9, §13).
//
// It carries nothing where `auth:` is absent — there is no line to mark — and
// nothing where the block names neither scheme, which is `schema-mismatch` and
// `check`'s to report (ADR-0064). Absence is not a third member: `none` is what
// a row renders for a Provider that sends no credential, and a line that is not
// there has no cell for it (§13).
func authSchemeNamed(root *yaml.Node) string {
	authVal := topLevelFields(root, "auth")["auth"]
	named := topLevelFields(authVal, authSchemes...)
	for _, scheme := range authSchemes {
		if named[scheme] != nil {
			return scheme
		}
	}
	return ""
}

// operationMarks is one mark per member of `operations:`, in the Manifest's own
// order, which is the order the file renders in — the normative order a listing
// is ranged over in is a listing's rule, and this surface writes the artefact's
// lines back where they stand (§9).
//
// An entry whose key is not a plain scalar has no line the gutter can anchor to
// and contributes nothing.
func operationMarks(opsVal *yaml.Node) []OperationMark {
	if opsVal == nil || opsVal.Kind != yaml.MappingNode {
		return nil
	}
	var marks []OperationMark
	for i := 0; i+1 < len(opsVal.Content); i += 2 {
		key, op := opsVal.Content[i], opsVal.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		info := operationInfoFromNode(op)
		marks = append(marks, OperationMark{
			Line:          key.Line,
			Kind:          info.Kind,
			Repeatability: effectiveRepeatability(info),
			Opaque:        info.IsShell,
		})
	}
	return marks
}

// RepositoryDeclarationMarks is §8's roster on a Repository declaration: the
// `hyper` version every Run in this repository is gated on, and the retention
// policy that bounds Compaction (§3, §11).
//
// The digest beside the pin carries no mark, and the roster is what says so:
// the two members here are the facts that govern every Run in this repository —
// what `hyper` it is pinned to, and how long its Records are kept — where the
// digest is `hyper project`'s own writing about the pin rather than a third
// thing the repository declares (§11, §12).
type RepositoryDeclarationMarks struct {
	Version, Retention KeyMark
}

// ReadRepositoryDeclarationMarks reads those two off `hyper.yaml`'s own root.
// A repository declaring no `retention:` has no line to mark and therefore no
// cell, which is different from a line rendering a blank one (§8).
func ReadRepositoryDeclarationMarks(root *yaml.Node) RepositoryDeclarationMarks {
	fields := topLevelFields(root, "version", "retention")
	return RepositoryDeclarationMarks{
		Version:   KeyMark{topLevelKeyLine(root, "version"), scalarValues(fields["version"])},
		Retention: KeyMark{topLevelKeyLine(root, "retention"), scalarValues(fields["retention"])},
	}
}

// scalarValues is a plain scalar as a mark's values — the one value it carries,
// or none where the key holds something this cannot read.
func scalarValues(val *yaml.Node) []string {
	return values(scalarValue(val))
}

// values is one derived fact as a mark carries it, and nothing at all where
// there is none: "" is what every reader in this file answers with where it
// could read nothing, and a mark carrying it would be a cell asserting a fact
// nobody stated (ADR-0064).
func values(text string) []string {
	if text == "" {
		return nil
	}
	return []string{text}
}
