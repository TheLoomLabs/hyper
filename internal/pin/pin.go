// Package pin implements the version pin gate every command compares itself
// against before reading a second file (§9, §11, ADR-0020). Milestone 1 needs
// only the version: member the gate reads — the Repository declaration's full
// schema, and its digest: and retention: members, land with the artefact
// schemas in a later ticket.
package pin

import "gopkg.in/yaml.v3"

// Codes the gate Refuses under (§12).
const (
	CodeMismatch = "version-pin-mismatch"
	CodeAbsent   = "version-pin-absent"
)

// Result is the gate's decision. A zero Result means the binary is cleared to
// proceed.
type Result struct {
	Refused bool
	Code    string
	Message string
}

// repositoryDeclaration reads only the one member this gate needs. The full
// schema — kind:, digest:, retention:, unknown-key rejection — belongs to the
// artefact-schema ticket that follows this one.
type repositoryDeclaration struct {
	Version string `yaml:"version"`
}

// Check compares binaryVersion against the pin in hyper.yaml's bytes. present
// is false where hyper.yaml does not exist at all. The comparison is exact —
// there is no range, no minimum, no compatible-release operator (§11).
func Check(present bool, data []byte, binaryVersion string) Result {
	if !present {
		return Result{
			Refused: true,
			Code:    CodeAbsent,
			Message: "hyper.yaml carries no version pin — run: hyper project",
		}
	}

	var decl repositoryDeclaration
	if err := yaml.Unmarshal(data, &decl); err != nil || decl.Version == "" {
		return Result{
			Refused: true,
			Code:    CodeAbsent,
			Message: "hyper.yaml carries no readable version pin — run: hyper project",
		}
	}

	if decl.Version != binaryVersion {
		return Result{
			Refused: true,
			Code:    CodeMismatch,
			Message: "this binary is " + binaryVersion + "; the repository pins " + decl.Version + " — run: hyper project, or install " + decl.Version,
		}
	}

	return Result{}
}
