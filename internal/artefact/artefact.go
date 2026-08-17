// Package artefact holds each of the five artefacts' own schema and the
// kind: check that reads a file's declared kind against where it lives
// (§3, §4, §12). Only the Repository declaration's schema exists yet — a
// Definition, a Procedure, a Target declaration and a Manifest each arrive
// in their own ticket and grow this package the same way, one artefact at a
// time (issue #89).
package artefact

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// CodeKindMismatch is the code a kind: disagreeing with its directory or
// filename is refused under (§4, §12).
const CodeKindMismatch = "kind-mismatch"

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
	problems := schema.Check(root, RepositoryDeclaration, file)
	problems = append(problems, checkKind(file, root, KindRepositoryDeclaration)...)
	return problems
}

// checkKind reads the top-level kind: scalar, where one is present and
// legible, and reports a disagreement with want under kind-mismatch. It
// says nothing where kind: is absent or is not a plain scalar: the schema
// check above has already named that fault under schema-mismatch, and a
// reader does not need two rows pointing at one line for one cause.
func checkKind(file string, root *yaml.Node, want string) []problem.Problem {
	if root == nil || root.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, val := root.Content[i], root.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Value != "kind" {
			continue
		}
		if val.Kind == yaml.ScalarNode && val.Value != want {
			return []problem.Problem{{
				File:      file,
				Line:      val.Line,
				Column:    val.Column,
				Field:     "kind",
				ErrorCode: CodeKindMismatch,
				Message:   fmt.Sprintf("kind: %s does not agree with %s — want %s", val.Value, file, want),
			}}
		}
		return nil
	}
	return nil
}
