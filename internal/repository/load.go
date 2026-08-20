package repository

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
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
	// Manifests is the Provider namespace's other half: one entry per name
	// in Providers, carrying where that name's bytes came from and what
	// they were. Providers answers what a provider: resolves to and this
	// answers which Manifest it resolved to, and they are built from one
	// fold so the two can never mean different files by one name.
	Manifests map[string]LoadedManifest
	// TargetDeclarations is the Target namespace's other half, on the shape
	// Manifests gives the Provider one: one entry per name in Targets,
	// carrying the declaration that name was read from. Targets answers
	// what a targets: member resolves to — a membership set per name, which
	// is what a check asks — and this carries the declaration itself, which
	// is what a surface states a row off (issue #112). They are built from
	// one fold, so the two can never mean different files by one name.
	TargetDeclarations map[string]*yaml.Node
	// DefinitionDeclarations is the Definition namespace's other half, and
	// it is here for TargetDeclarations' own reason on the other end of one
	// relation: Definitions answers what a Step's definition: resolves to,
	// and this carries the file that name was read from, which is what
	// `review`'s AUTHORITY table states a row off (issue #121). The two are
	// built from one fold, so no surface can mean a different file by one
	// name than the check does.
	DefinitionDeclarations map[string]*yaml.Node
}

// DeclarationPath is where the Repository declaration sits: `hyper.yaml`, at
// the repository root rather than in a directory, and the one artefact keyed by
// its filename (§3, §12).
//
// It is spelled here because this is where the walk that finds it is, and
// because a command that reads a declared fact off it — `compact` reads
// `retention:` — must not have to know the name to ask (issue #131).
const DeclarationPath = "hyper.yaml"

// Declaration answers the Repository declaration's parsed root, and nil where
// the repository has none or its file would not parse.
//
// Nil is an answer rather than a fault, on the load's own rule: what a single
// file can do wrong is carried on that file's LoadedArtefact and stops nothing
// else (issue #88). Every caller reads nil the way a lookup into a nil mapping
// already reads — every key answers absent — so a command that reads a declared
// fact off it gets *the repository declared nothing* rather than an error it
// would have to invent a rendering for (ADR-0064).
func (l Loaded) Declaration() *yaml.Node {
	for _, a := range l.Artefacts {
		if a.Path == DeclarationPath {
			return a.Root
		}
	}
	return nil
}

// LoadedManifest is one member of the Provider namespace as the commands that
// report a Provider read it: the name the Manifest declares for itself, the
// origin §12 reads off where its bytes loaded from, those exact bytes, and what
// they parsed to.
//
// Bytes are here because manifest_digest is SHA-256 over a Manifest's exact
// bytes (§7) and Origin because the two facts are one question — which file did
// this name come from — answered once, at the fold, rather than by every caller
// re-deciding it off a path.
type LoadedManifest struct {
	Name   string
	Origin string
	Path   string
	Bytes  []byte
	Root   *yaml.Node
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

	declarations := targetDeclarationsByName(artefacts)
	targets := artefact.BuildTargetIndex(sortedByName(declarations))
	definitions := definitionDeclarationsByName(artefacts)
	manifests := manifestsByName(artefacts)
	return Loaded{
		Artefacts:              artefacts,
		Providers:              artefact.BuildProviderIndex(manifestRoots(sortedByName(manifests))),
		Targets:                targets,
		Definitions:            artefact.BuildDefinitionIndex(sortedByName(definitions), targets),
		Procedures:             artefact.BuildProcedureIndex(rootsUnder(artefacts, "procedures/")),
		Manifests:              manifests,
		TargetDeclarations:     declarations,
		DefinitionDeclarations: definitions,
	}, nil
}

// targetDeclarationsByName folds the loaded artefacts into the Target
// namespace's members: every targets/ file that parsed and named itself. A file
// that will not parse contributes nothing and neither does one whose target: is
// absent or is not a plain scalar — ADR-0064's rule, the same one
// manifestsByName folds the Provider namespace on.
//
// It is the one place a Target's name is decided to mean a declaration. Where
// two files declare one name the later one wins, which is not a precedence rule
// the tool is entitled to and which check will name once §11's collision code
// reaches this artefact; until then the fold has to answer something, and
// answering it once means the declaration `hyper targets` writes a row off and
// the declaration a Definition's targets: resolves to are the same file rather
// than two readings of one repository.
func targetDeclarationsByName(artefacts []LoadedArtefact) map[string]*yaml.Node {
	declarations := map[string]*yaml.Node{}
	for _, a := range artefacts {
		if !a.OK || !strings.HasPrefix(a.Path, "targets/") {
			continue
		}
		if name := artefact.TargetDeclarationName(a.Root); name != "" {
			declarations[name] = a.Root
		}
	}
	return declarations
}

// definitionDeclarationsByName folds the loaded artefacts into the Definition
// namespace's members, on targetDeclarationsByName's own rule: every
// definitions/ file that parsed and named itself, and nothing from one that did
// neither (ADR-0064).
//
// It is what the Definition namespace is built from as well as what carries it,
// which is the whole point of folding first: the index sees one member per name,
// so the file a Step's definition: resolves to and the file `review`'s
// AUTHORITY table reads a row off are the same file rather than two walks
// agreeing about one repository (issue #121).
func definitionDeclarationsByName(artefacts []LoadedArtefact) map[string]*yaml.Node {
	declarations := map[string]*yaml.Node{}
	for _, a := range artefacts {
		if !a.OK || !strings.HasPrefix(a.Path, "definitions/") {
			continue
		}
		if name := artefact.DeclaredName(a.Root, artefact.KindDefinition); name != "" {
			declarations[name] = a.Root
		}
	}
	return declarations
}

// manifestsByName folds the loaded artefacts into the Provider namespace's
// members: the built-in, whose bytes are the compiled-in constant, and every
// providers/ file that parsed and named itself. A file that will not parse
// contributes nothing and neither does one whose provider: is absent or is not
// a plain scalar — ADR-0064's rule, that an authored name resolving to nothing
// is a check rather than a load failure.
//
// It is the one place a name is decided to mean a file. Where two Manifests
// declare one name the later one wins, which is not a precedence rule the tool
// is entitled to — an Extension may never shadow a built-in Provider's name,
// and a collision is a load failure §11 fixes under provider-name-collision.
// Until that check exists the fold has to answer something, and answering it
// once means the Manifest `hyper providers` reports for a name and the Manifest
// a Definition's provider: resolves to are the same file rather than two
// readings of one repository.
func manifestsByName(artefacts []LoadedArtefact) map[string]LoadedManifest {
	manifests := map[string]LoadedManifest{}
	for _, a := range artefacts {
		if !a.OK || !isManifest(a.Path) {
			continue
		}
		name := artefact.ManifestProviderName(a.Root)
		if name == "" {
			continue
		}
		manifests[name] = LoadedManifest{
			Name:   name,
			Origin: artefact.ProviderOrigin(a.Path),
			Path:   a.Path,
			Bytes:  a.Bytes,
			Root:   a.Root,
		}
	}
	return manifests
}

// isManifest says whether a loaded artefact is a Provider's Manifest: the
// pseudo-path the built-in carries, or a file in providers/ — the two places
// §12's origin set says a Manifest's bytes can be.
func isManifest(path string) bool {
	return path == artefact.BuiltinShellProviderPath || strings.HasPrefix(path, "providers/")
}

// sortedByName is a fold's members in name order, which is how both namespaces
// are handed to the index built from them. Passing a fold's own output rather
// than the directory's roots is what makes the two views one: the index sees
// one member per name, so its own fold decides nothing the load has not already
// decided, and the order it walks them in cannot matter — which is also why
// sorting here is a courtesy to a reader of a failure rather than a rule
// anything depends on.
func sortedByName[T any](members map[string]T) []T {
	sorted := make([]T, 0, len(members))
	for _, name := range slices.Sorted(maps.Keys(members)) {
		sorted = append(sorted, members[name])
	}
	return sorted
}

// manifestRoots is what the Provider namespace is built from: the roots of the
// folded Manifests, whose other three members are the bytes and the origin no
// index carries.
func manifestRoots(manifests []LoadedManifest) []*yaml.Node {
	roots := make([]*yaml.Node, 0, len(manifests))
	for _, manifest := range manifests {
		roots = append(roots, manifest.Root)
	}
	return roots
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
// The three namespaces built this way are the Target, Definition and Procedure
// ones. The Provider namespace is not: its members carry the bytes their digest
// covers, so they are folded by manifestsByName above, which reaches the
// built-in — whose <built-in>/shell path matches no prefix here — as well as
// the providers/ files.
func rootsUnder(artefacts []LoadedArtefact, prefix string) []*yaml.Node {
	var roots []*yaml.Node
	for _, a := range artefacts {
		if a.OK && strings.HasPrefix(a.Path, prefix) {
			roots = append(roots, a.Root)
		}
	}
	return roots
}
