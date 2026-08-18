package repository

import (
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// LoadedArtefact is one artefact hyper read, in the shape every command that
// asks a question about the repository reads it in (issue #109).
//
// Three of its members are the artefact itself and are why the load is a value
// rather than a sequence written inside one command:
//
//   - Path is where the bytes came from, relative to the repository root with
//     forward slashes — or the <built-in>/shell pseudo-path for the one
//     Provider that has no file (§9, ADR-0039).
//   - Bytes are exactly what was read, unmodified. manifest_digest is SHA-256
//     over a Manifest's exact bytes and never over a canonical form of what
//     they parse to (§7), and `hyper operation` writes a Manifest's declaring
//     lines back verbatim (§9) — neither fact is recoverable from a parse tree,
//     so the load keeps what a parse would throw away.
//   - Root is what those bytes parsed to, which every schema and resolution
//     check reads.
//
// Problems and OK are the load's own findings, on the rule that loading a file
// is the first check and failing it stops every check after it for that file —
// never for the repository (§4, issue #88). OK is false where the read failed
// or the file will not parse at all, which is the one case a caller's second
// pass skips entirely. OK is true and Root nil where the file is empty: zero
// documents is valid YAML, and whether it is a valid artefact is a schema
// question the load does not answer.
type LoadedArtefact struct {
	Path     string
	Bytes    []byte
	Root     *yaml.Node
	Problems []problem.Problem
	OK       bool
}

// Loaded is one read of a repository: every artefact hyper found, and the four
// namespaces built from them.
//
// The namespaces are part of the load because they are what every name in the
// repository resolves against, and building them is the second half of the
// two-pass rule Artefacts alone cannot express: every file is parsed before a
// single name is resolved, so a Definition's provider: and targets: resolve
// against the whole repository's names rather than against the files walked
// before it (issue #93).
//
// What is deliberately not here: no digest is computed, no line range is
// extracted, no schema is checked, no graph is walked. Those belong to the
// commands that report them — a load that judged its own artefacts would be
// `check` with a different name.
type Loaded struct {
	Artefacts   []LoadedArtefact
	Providers   artefact.ProviderIndex
	Targets     artefact.TargetIndex
	Definitions artefact.DefinitionIndex
	Procedures  artefact.ProcedureIndex
}

// Load reads the repository at repoRoot: it walks the five artefact locations,
// reads and parses each file, and builds the four namespaces from what parsed.
// It is the one call a command makes to get a repository, which is why the walk
// beneath it is not exported — two readers of one repository must not be able
// to disagree about what reading it means.
//
// The error return is the walk's alone: a directory hyper cannot list is not an
// artefact's problem to carry, having no artefact to carry it. Everything a
// single file can do wrong — an unreadable file, a file that will not parse — is
// carried on that file's own LoadedArtefact and stops nothing else.
func Load(repoRoot string) (Loaded, error) {
	files, err := artefactFiles(repoRoot)
	if err != nil {
		return Loaded{}, err
	}

	// The built-in shell Provider is loaded first and through the same shape
	// as any other artefact: its bytes are the compiled-in constant and its
	// path the pseudo-path §9 renders, and the one Provider with no blob in
	// the repository is a member of the load on the same footing as the rest
	// (§3, §11, ADR-0039). It is not read from disk and cannot fail to parse
	// — hyper's own bytes are not reviewed text — so it carries no problem of
	// its own; its schema is CheckBuiltinShellProvider's to state, exactly as
	// a providers/ file's is CheckManifest's.
	artefacts := make([]LoadedArtefact, 0, len(files)+1)
	artefacts = append(artefacts, LoadedArtefact{
		Path:  artefact.BuiltinShellProviderPath,
		Bytes: []byte(artefact.BuiltinShellProviderYAML),
		Root:  artefact.BuiltinShellProviderRoot(),
		OK:    true,
	})

	for _, rel := range files {
		artefacts = append(artefacts, loadFile(repoRoot, rel))
	}

	targets := artefact.BuildTargetIndex(rootsUnder(artefacts, "targets/"))
	return Loaded{
		Artefacts:   artefacts,
		Providers:   artefact.BuildProviderIndex(rootsUnder(artefacts, "providers/")),
		Targets:     targets,
		Definitions: artefact.BuildDefinitionIndex(rootsUnder(artefacts, "definitions/"), targets),
		Procedures:  artefact.BuildProcedureIndex(rootsUnder(artefacts, "procedures/")),
	}, nil
}

// loadFile reads and parses one artefact file. An artefact hyper cannot even
// read is judged the same way as one that will not parse: exactly one problem,
// positioned at the file's first character, on the loader's own error code —
// there being no line to cite in bytes that never arrived.
func loadFile(repoRoot, rel string) LoadedArtefact {
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(rel)))
	if err != nil {
		return LoadedArtefact{Path: rel, Problems: []problem.Problem{{
			File:      rel,
			Line:      1,
			Column:    1,
			ErrorCode: yamlsubset.ErrorCode,
			Message:   err.Error(),
		}}}
	}

	root, problems, ok := yamlsubset.Parse(rel, data)
	if ok && root != nil {
		problems = append(problems, yamlsubset.Violations(root, rel)...)
	}
	return LoadedArtefact{Path: rel, Bytes: data, Root: root, Problems: problems, OK: ok}
}

// rootsUnder returns the root of every loaded artefact whose path starts with
// prefix and that parsed at all — the roots each namespace is built off of. A
// file that failed to parse contributes no root and therefore no name to any
// namespace, on ADR-0064's own rule: an authored name that resolves to nothing
// is a check, and a file that will not parse is check's to report rather than
// the load's to guess at.
//
// The built-in Provider's path is <built-in>/shell and matches no prefix here,
// which is how it stays out of the providers/ roots: BuildProviderIndex seeds
// the Provider namespace from the built-ins itself, and a second entry from the
// same bytes would be one name declared twice.
func rootsUnder(artefacts []LoadedArtefact, prefix string) []*yaml.Node {
	var roots []*yaml.Node
	for _, a := range artefacts {
		if a.OK && strings.HasPrefix(a.Path, prefix) {
			roots = append(roots, a.Root)
		}
	}
	return roots
}
