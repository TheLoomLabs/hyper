// Package pin is the version pin: the gate fifteen of §9's sixteen commands
// compare themselves against before reading a second file, and the edit the
// sixteenth makes to the file they read (§9, §11, ADR-0020).
//
// The two live together because they are one fact read from its two ends. Check
// answers *may this binary act on this repository*; Declaration answers *what
// does the file say once `hyper project` has written what it derived*. A gate
// reading one spelling of `version:` and a writer producing another is the day a
// `project` leaves behind a repository its own binary Refuses, so the schema of
// the thing is stated once, here.
//
// **Nothing here reads or writes a file.** Both doors take the declaration's
// bytes and answer a value: what a command does with the answer is
// internal/cli's, and where the bytes came from is internal/repository's. That
// is what lets the gate run before the repository is loaded at all, and what
// lets the writer be exercised over a byte string rather than a temp directory.
//
// The Repository declaration's full schema — its `kind:`, its `retention:`, and
// unknown-key rejection — is internal/artefact's, which is where every artefact's
// schema is (§3, §4).
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

// repositoryDeclaration reads only the one member this package needs: the pin
// itself, which the gate compares and the writer replaces. Everything else the
// declaration says is read where that member is checked or reported —
// artefact.RepositoryDeclaration for the schema, artefact.ReadRepositoryFacts
// for the two facts a command acts on.
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
