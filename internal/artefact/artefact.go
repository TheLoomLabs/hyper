// Package artefact holds each of the five artefacts' own schema, the checks
// that read one artefact against itself — kind: against its directory, an
// artefact's own name against its file's basename, the credential slot's
// shape, the Target declaration's own two cross-field rules, and the
// Manifest's request, input-schema and path grammars (§3, §4, §12) — and,
// starting with the Definition, the first checks that read more than one
// artefact at a time: whether a name an artefact writes for another
// resolves, and the two checks a (Definition, Target) binding decides that
// neither artefact alone can (§4, §5, issue #93). The Repository
// declaration, the Target declaration, the Manifest and the Definition's
// schemas exist so far — a Procedure arrives in its own ticket and grows
// this package the same way (issues #89, #90, #91, #93).
package artefact

import (
	"fmt"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// CodeKindMismatch is the code a kind: disagreeing with its directory or
// filename is refused under (§4, §12).
const CodeKindMismatch = "kind-mismatch"

// CodeNameMismatch is the code an artefact's own name disagreeing with its
// file's basename is refused under — a Target declaration's target:, a
// Manifest's provider:, a Definition's definition:, and, as that artefact
// arrives, a Procedure's procedure: (§4, §12). It is never widened into
// kind-mismatch: the two disagreements send a reader to different edits.
const CodeNameMismatch = "name-mismatch"

// CodeCredentialSlotMalformed is the code a credential slot's value earns
// wherever it is not exactly a mapping whose sole key is env:, and the code
// env: itself earns wherever it is written outside a credential slot, in
// any artefact (§4).
const CodeCredentialSlotMalformed = "credential-slot-malformed"

// CodeLocalReserved is the code a declaration named local earns for
// carrying an auth: block or a class: other than local — the two things
// the reserved name changes about an otherwise ordinary Target declaration
// (§4).
const CodeLocalReserved = "local-reserved"

// CodeTargetInconsistent is the code hosts: earns for disagreeing with
// whether capabilities: grants http — present without the grant, or the
// grant without it (§4).
const CodeTargetInconsistent = "target-inconsistent"

// KindRepositoryDeclaration is the one kind: value hyper.yaml may carry —
// the one artefact whose file agrees with its filename rather than a
// directory (§12's kind table).
const KindRepositoryDeclaration = "repository-declaration"

// RepositoryDeclaration is hyper's own schema for hyper.yaml (§3): kind,
// version and its digest, written only by hyper project, and the retention
// policy that bounds Compaction, omitted meaning nothing is ever removed.
// Nothing else is admitted — additionalProperties: false is forced rather
// than authored (§12), so a fifth key is unknown-key wherever it appears.
// hyper.yaml carries no name key: one repository has one Repository
// declaration, and there is nothing to tell it apart from (§3).
var RepositoryDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "version", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "digest", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "retention", Required: false, Schema: schema.Schema{Type: schema.Duration}},
	},
}

// CheckRepositoryDeclaration validates hyper.yaml's already-parsed root
// against RepositoryDeclaration and its kind:. root is nil where the file
// parsed to no document at all (yamlsubset.Parse's ok=true, root=nil case);
// the schema check still runs and reports every required key hyper.yaml
// never supplied.
func CheckRepositoryDeclaration(file string, root *yaml.Node) []problem.Problem {
	problems := withCredentialSlots(schema.Check(root, RepositoryDeclaration, file), file, root)
	problems = append(problems, checkKind(file, root, KindRepositoryDeclaration)...)
	return problems
}

// topLevelFields reads root's own top-level keys, where root is a mapping,
// and returns whichever of names it finds, keyed by name. It is the one
// walk every check below shares — the schema has already read the rest of
// root's shape, so a check that only cares about a handful of root's own
// keys reads them once here rather than re-scanning root.Content itself.
// It returns nil where root is not a mapping, which every caller reads the
// same way a lookup into a nil map already does: every key answers absent.
func topLevelFields(root *yaml.Node, names ...string) map[string]*yaml.Node {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	found := make(map[string]*yaml.Node, len(names))
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, val := root.Content[i], root.Content[i+1]
		if key.Kind == yaml.ScalarNode && want[key.Value] {
			found[key.Value] = val
		}
	}
	return found
}

// TopLevelKeyLine is the line root's own top-level key of that name is
// written on, and 0 where it writes none. It is topLevelFields' other half:
// that walk answers with a key's value and this one with the key itself,
// which is what a surface annotating a *line* needs — §8 marks the
// targets: line, and a block sequence's first member is a line below the
// key that names it.
//
// It is exported for the one surface that reports a fault against an
// artefact it did not check: a Probe's host is refused against the hosts:
// the Target named local declares, and a Refusal names the line to edit
// (§9, ADR-0042). Every other caller of it here is a mark or a check that
// already holds the node.
func TopLevelKeyLine(root *yaml.Node, name string) int {
	if root == nil || root.Kind != yaml.MappingNode {
		return 0
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		if key := root.Content[i]; key.Kind == yaml.ScalarNode && key.Value == name {
			return key.Line
		}
	}
	return 0
}

// DeclaredName is the name an artefact declares for itself: the scalar under
// the top-level key its kind names itself with — definition:, procedure:,
// provider: or target: — and "" where that key is absent, carries something
// other than a plain scalar, or the file did not parse at all.
//
// It is one reader for the four because the four kinds name themselves by one
// rule and differ only in the word (§3, §12). The rule it holds is the one every
// name in the repository resolves by: a name is matched against what the
// artefact declares rather than against its filename, so a case-insensitive
// filesystem cannot decide what resolves and what does not (§9, ADR-0060). A
// key the artefact never wrote answers nothing rather than guessing, which is
// ADR-0064's rule — what is wrong with an artefact is check's to report and
// never a reader's to substitute for. Only top-level keys are read: a Step's
// own definition: is a Step's, and no artefact is named by a key nested inside
// it.
//
// The two named readers beside it — ManifestProviderName and
// TargetDeclarationName — are this reader under the key each of them fixes, and
// they keep their names because a caller folding the Provider namespace is
// asking for a Provider's name rather than for a key's scalar (issue #118).
func DeclaredName(root *yaml.Node, key string) string {
	named := topLevelFields(root, key)[key]
	if named == nil || named.Kind != yaml.ScalarNode {
		return ""
	}
	return named.Value
}

// position reports where to point a problem that has no node of its own to
// point at. (1, 1) is the file itself, the same fallback schema.Check's own
// position uses for the same reason.
func position(n *yaml.Node) (int, int) {
	if n == nil {
		return 1, 1
	}
	return n.Line, n.Column
}

// checkKind reads the top-level kind: scalar, where one is present and
// legible, and reports a disagreement with want under kind-mismatch. It
// says nothing where kind: is absent or is not a plain scalar: the schema
// check above has already named that fault under schema-mismatch, and a
// reader does not need two rows pointing at one line for one cause.
func checkKind(file string, root *yaml.Node, want string) []problem.Problem {
	val := topLevelFields(root, "kind")["kind"]
	if val == nil || val.Kind != yaml.ScalarNode || val.Value == want {
		return nil
	}
	return []problem.Problem{{
		File:      file,
		Line:      val.Line,
		Column:    val.Column,
		Field:     "kind",
		ErrorCode: CodeKindMismatch,
		Message:   fmt.Sprintf("kind: %s does not agree with %s — want %s", val.Value, file, want),
	}}
}

// KindTargetDeclaration is the one kind: value a file in targets/ may
// carry (§12's kind table).
const KindTargetDeclaration = "target-declaration"

// TargetDeclaration is the reviewed half of a Target (§3, §4): the kinds:
// it accepts and the capabilities: it grants, both closed sets (§12); the
// hosts: it grants — one list rather than a mapping keyed by Capability,
// since http is the only member that ever reaches one; its class:, open
// rather than enumerated here, a class only ever rejecting a mismatch
// against a Provider's own; whether it opts into opaque-destroy:; and an
// auth: mapping naming the environment variable each credential slot
// resolves from. auth: is Open: its members are credential slots the
// repository names itself, and checkCredentialSlots reads their insides —
// there is no fixed Properties list for a mapping keyed by name to enumerate.
var TargetDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "target", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "class", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "kinds", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String, Enum: []string{"read", "mutate", "destroy"}},
		}},
		{Name: "capabilities", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String, Enum: []string{"http", "shell"}},
		}},
		{Name: "hosts", Required: false, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String},
		}},
		{Name: "opaque-destroy", Required: false, Schema: schema.Schema{Type: schema.Boolean}},
		{Name: "auth", Required: false, Schema: schema.Schema{Type: schema.Object, Open: true}},
	},
}

// CheckTargetDeclaration validates a targets/ file's already-parsed root
// against TargetDeclaration and the four checks that read this artefact
// against itself: kind: against targets/, target: against the file's
// basename, every credential slot's shape, local's two reserved rules, and
// hosts: against capabilities: (§3, §4, issue #90).
func CheckTargetDeclaration(file string, root *yaml.Node) []problem.Problem {
	problems := withCredentialSlots(schema.Check(root, TargetDeclaration, file), file, root)
	problems = append(problems, checkKind(file, root, KindTargetDeclaration)...)
	problems = append(problems, checkName(file, root, "target")...)
	problems = append(problems, checkLocalReserved(file, root)...)
	problems = append(problems, checkHostsConsistency(file, root)...)
	return problems
}

// checkName reads the top-level nameKey scalar, where one is present and
// legible, and reports a disagreement with file's basename under
// name-mismatch — the argument ADR-0023 makes for kind: and the directory,
// applied to names rather than kinds (§4). It says nothing where nameKey is
// absent or is not a plain scalar, on checkKind's own rule: the schema
// check has already named that fault under schema-mismatch.
func checkName(file string, root *yaml.Node, nameKey string) []problem.Problem {
	val := topLevelFields(root, nameKey)[nameKey]
	if val == nil || val.Kind != yaml.ScalarNode {
		return nil
	}
	want := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	if val.Value == want {
		return nil
	}
	return []problem.Problem{{
		File:      file,
		Line:      val.Line,
		Column:    val.Column,
		Field:     nameKey,
		ErrorCode: CodeNameMismatch,
		Message:   fmt.Sprintf("%s: %s does not agree with %s's basename — want %s", nameKey, val.Value, file, want),
	}}
}

// withCredentialSlots layers the credential-slot check over a schema
// check's own problems: it drops the generic unknown-key report a
// misplaced env: key already earns from the schema, since
// checkCredentialSlots names that same fault under the one code
// credential-slot-malformed and a reader does not need two rows pointing
// at one line for one cause (§4).
func withCredentialSlots(problems []problem.Problem, file string, root *yaml.Node) []problem.Problem {
	problems = dropReservedEnvKey(problems)
	return append(problems, checkCredentialSlots(file, root)...)
}

// dropReservedEnvKey removes the schema package's own unknown-key report
// for a literal env: key — env: is reserved across every artefact and
// checkCredentialSlots is what names that fault (§4).
func dropReservedEnvKey(problems []problem.Problem) []problem.Problem {
	kept := problems[:0]
	for _, p := range problems {
		if p.ErrorCode == schema.CodeUnknownKey && lastFieldSegment(p.Field) == "env" {
			continue
		}
		kept = append(kept, p)
	}
	return kept
}

// lastFieldSegment returns the part of a dot-joined field path after its
// final ".", or field whole where it carries none.
func lastFieldSegment(field string) string {
	if i := strings.LastIndex(field, "."); i >= 0 {
		return field[i+1:]
	}
	return field
}

// checkCredentialSlots validates every member of a top-level auth: mapping
// against the credential slot's own shape, then walks the whole document
// for env: written anywhere else. auth: is Open in the schema above
// precisely so this is the one place that reads a slot's insides — the
// schema stops at "is this a mapping" and this function reads what is
// inside it (§4).
func checkCredentialSlots(file string, root *yaml.Node) []problem.Problem {
	var problems []problem.Problem
	slotValues := map[*yaml.Node]bool{}

	if authVal := topLevelFields(root, "auth")["auth"]; authVal != nil && authVal.Kind == yaml.MappingNode {
		for j := 0; j+1 < len(authVal.Content); j += 2 {
			slotKey, slotVal := authVal.Content[j], authVal.Content[j+1]
			if slotKey.Kind != yaml.ScalarNode {
				continue
			}
			slotValues[slotVal] = true
			problems = append(problems, checkOneSlot(file, "auth."+slotKey.Value, slotVal)...)
		}
	}

	problems = append(problems, findReservedEnvKeys(file, root, "", slotValues)...)
	return problems
}

// checkOneSlot reads one credential slot's value against the shape §4
// fixes: a mapping whose sole key is env:. A scalar there is always a load
// error with no exception, and a mapping carrying any other key, or more
// than one, is the same code.
func checkOneSlot(file, field string, val *yaml.Node) []problem.Problem {
	malformed := val == nil || val.Kind != yaml.MappingNode ||
		len(val.Content) != 2 || val.Content[0].Kind != yaml.ScalarNode || val.Content[0].Value != "env"
	if !malformed {
		return nil
	}
	line, column := position(val)
	return []problem.Problem{{
		File:      file,
		Line:      line,
		Column:    column,
		Field:     field,
		ErrorCode: CodeCredentialSlotMalformed,
		Message:   fmt.Sprintf("%s: a credential slot's value must be a mapping whose sole key is env:", field),
	}}
}

// findReservedEnvKeys walks the whole document for a mapping key literally
// env: outside a credential slot's own value. slotValues names every node
// checkCredentialSlots already read as one, whatever its own shape turned
// out to be, so this never re-reports what checkOneSlot already named (§4).
// field is the dotted path to n, "" at the document root, threaded down so
// a nested env: still points a reader at an editable position rather than
// a bare "env" that could be any of several lines.
func findReservedEnvKeys(file string, n *yaml.Node, field string, slotValues map[*yaml.Node]bool) []problem.Problem {
	if n == nil {
		return nil
	}
	var problems []problem.Problem
	switch n.Kind {
	case yaml.MappingNode:
		for i := 0; i+1 < len(n.Content); i += 2 {
			key, val := n.Content[i], n.Content[i+1]
			childField := joinField(field, key.Value)
			if key.Kind == yaml.ScalarNode && key.Value == "env" && !slotValues[n] {
				problems = append(problems, problem.Problem{
					File:      file,
					Line:      key.Line,
					Column:    key.Column,
					Field:     childField,
					ErrorCode: CodeCredentialSlotMalformed,
					Message:   "env: may appear only as a credential slot's sole key",
				})
			}
			problems = append(problems, findReservedEnvKeys(file, val, childField, slotValues)...)
		}
	case yaml.SequenceNode:
		for i, item := range n.Content {
			problems = append(problems, findReservedEnvKeys(file, item, fmt.Sprintf("%s[%d]", field, i), slotValues)...)
		}
	}
	return problems
}

// joinField appends name to field on the dot notation problem.Field uses
// ("auth.token"), or returns name alone at the document root.
func joinField(field, name string) string {
	if field == "" {
		return name
	}
	return field + "." + name
}

// checkLocalReserved reads target: local's two reserved rules: it declares
// class: local and it carries no auth: block, the second being what leaves
// a Probe no credential to resolve (§4). It says nothing about a
// declaration named anything else — more than one Target may claim
// class: local, each a name for the machine hyper runs on with its own
// grant (ADR-0041).
func checkLocalReserved(file string, root *yaml.Node) []problem.Problem {
	fields := topLevelFields(root, "target", "class", "auth")
	targetVal := fields["target"]
	if targetVal == nil || targetVal.Kind != yaml.ScalarNode || targetVal.Value != "local" {
		return nil
	}

	var problems []problem.Problem
	if authVal := fields["auth"]; authVal != nil {
		problems = append(problems, problem.Problem{
			File:      file,
			Line:      authVal.Line,
			Column:    authVal.Column,
			Field:     "auth",
			ErrorCode: CodeLocalReserved,
			Message:   "a declaration named local carries no auth: block — the reserved name leaves a Probe no credential to resolve",
		})
	}
	if classVal := fields["class"]; classVal != nil && classVal.Kind == yaml.ScalarNode && classVal.Value != "local" {
		problems = append(problems, problem.Problem{
			File:      file,
			Line:      classVal.Line,
			Column:    classVal.Column,
			Field:     "class",
			ErrorCode: CodeLocalReserved,
			Message:   "a declaration named local declares class: local",
		})
	}
	return problems
}

// checkHostsConsistency reads hosts: against whether capabilities: grants
// http (§4): present without the grant, or the grant without it, is
// target-inconsistent either way. It says nothing where capabilities: is
// absent or not a list of scalars — the schema check has already named
// that fault.
func checkHostsConsistency(file string, root *yaml.Node) []problem.Problem {
	fields := topLevelFields(root, "capabilities", "hosts")
	capsVal := fields["capabilities"]
	if capsVal == nil || capsVal.Kind != yaml.SequenceNode {
		return nil
	}
	hostsVal := fields["hosts"]
	grantsHTTP := false
	for _, item := range capsVal.Content {
		if item.Kind == yaml.ScalarNode && item.Value == "http" {
			grantsHTTP = true
		}
	}

	switch {
	case grantsHTTP && hostsVal == nil:
		return []problem.Problem{{
			File:      file,
			Line:      capsVal.Line,
			Column:    capsVal.Column,
			Field:     "hosts",
			ErrorCode: CodeTargetInconsistent,
			Message:   "capabilities: grants http and this declaration carries no hosts:",
		}}
	case !grantsHTTP && hostsVal != nil:
		return []problem.Problem{{
			File:      file,
			Line:      hostsVal.Line,
			Column:    hostsVal.Column,
			Field:     "hosts",
			ErrorCode: CodeTargetInconsistent,
			Message:   "hosts: is present and capabilities: does not grant http",
		}}
	}
	return nil
}
