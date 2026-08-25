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
	"github.com/TheLoomLabs/hyper/internal/workflow"
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
	// Workflows is every file in the namespace `hyper project` owns that
	// the working tree holds, in path order: `hyper-*.yml` directly under
	// `.github/workflows/`, as bytes and nothing else.
	//
	// It is not a namespace and its members are not artefacts. Nothing here
	// is parsed as YAML, nothing declares a name, and no schema check reads
	// one — a generated file is derived from the reviewed artefacts rather
	// than being one, and the only question anything asks of it is whether
	// its bytes are the bytes a fresh projection would write (§10, §12,
	// issue #179).
	//
	// It is on the load because that comparison has two callers and neither
	// is entitled to a walk of its own: `check` and a Run's pre-flight get
	// the rule through verify.Repository, whose signature this leaves where
	// it is, and `project` reads the same list to know which standing files
	// no Procedure asks for any more.
	//
	// A repository read through LoadFrom holds none. That door is the
	// reaper's and `changes`', which read artefacts out of a revision and
	// verify nothing, so there is no directory to walk and nothing that
	// would ask (§7, issue #154).
	Workflows []LoadedWorkflow
}

// LoadedWorkflow is one generated workflow as the load found it: where it sits,
// relative to the repository root with forward slashes, and its exact bytes.
//
// What it does not carry is the whole of what separates it from a
// LoadedArtefact: no Root, because nothing parses it; no Problems and no OK,
// because the load has nothing to say about it. Reading it is opening a file,
// and the one rule that reads it compares bytes (§10).
type LoadedWorkflow struct {
	Path  string
	Bytes []byte
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

// DeclarationBytes is the Repository declaration's exact bytes, and false where
// the repository holds no `hyper.yaml` at all.
//
// It stands beside Declaration for the reason LoadedArtefact keeps Bytes at all:
// the one command that **writes** the declaration edits it rather than
// regenerating it, so what it needs is the file as it stands and not a parse
// tree that has already thrown the comments and the layout away (§11, issue
// #178). The parse says where a scalar sits; these bytes are what it sits in.
//
// false is the answer a repository that has never been projected gives, and it
// is what `project` reads to know it is creating the file rather than editing
// one. It is not an error: nothing about the walk failed, and the pin gate is
// what has an opinion about a repository with no declaration (§9, ADR-0020).
func (l Loaded) DeclarationBytes() ([]byte, bool) {
	for _, a := range l.Artefacts {
		if a.Path == DeclarationPath {
			return a.Bytes, true
		}
	}
	return nil, false
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

	artefacts := make([]LoadedArtefact, 0, len(files))
	for _, rel := range files {
		artefacts = append(artefacts, loadFile(repoRoot, rel))
	}

	// The namespace beside the five artefact locations, and after them: it
	// is read on the same walk so that the rule holding a working tree to
	// its projection reaches every caller of this load, and so that no
	// caller has to be taught where a generated file sits (§10, issue #179).
	generated, err := generatedWorkflows(repoRoot)
	if err != nil {
		return Loaded{}, err
	}

	loaded := build(artefacts)
	loaded.Workflows = generated
	return loaded, nil
}

// generatedWorkflows reads every file in the namespace `hyper project` owns, in
// path order.
//
// **The namespace is the generator's own** and is asked rather than spelled
// again: workflow.ProcedureOf is Path read backwards, so which files are
// `project`'s to speak for has one answer here, at the command that writes them
// and at the check that holds them to their derivation (§10).
//
// A repository with no `.github/workflows/` holds none, which is an answer
// rather than a fault: the directory is created where a file is written into it
// and is not a thing a repository has to have first. Nothing here recurses — a
// directory whose name is in the namespace is not a file in it.
//
// A file in the namespace hyper cannot read is the walk's error, on the
// division Load's own error return states: what a single file can do wrong is
// carried on that file's LoadedArtefact, and this is not one — there is nothing
// to hang a problem on, and bytes read as absent would report a projection
// wanted and not held about a file that is sitting there.
func generatedWorkflows(repoRoot string) ([]LoadedWorkflow, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(workflow.Dir)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var held []LoadedWorkflow
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := workflow.Dir + "/" + entry.Name()
		if _, inside := workflow.ProcedureOf(path); !inside {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(path)))
		if err != nil {
			return nil, err
		}
		held = append(held, LoadedWorkflow{Path: path, Bytes: data})
	}
	return held, nil
}

// Source is one artefact's path and its exact bytes, as a caller that already
// holds them supplies them to LoadFrom.
//
// The path is where the file sits in the repository, relative to the root and
// with forward slashes — the same spelling a walk answers, because it is what
// every namespace, every `check` problem and every Provenance member is stated
// in terms of.
type Source struct {
	Path  string
	Bytes []byte
}

// LoadFrom builds a repository out of artefact bytes a caller already holds,
// which is what a reaper has: it reads the dead Run's artefacts out of the
// **revision** that Run named rather than off the working tree, so there is no
// directory to walk (§7, issue #154).
//
// It is the same load through another door and it judges nothing differently:
// the same parse, the same problems carried per file, the same four namespaces,
// and the same built-in Provider — which no walk found and no caller can hand
// over, its bytes being compiled in (§3, ADR-0039).
//
// **The order handed in is the order it folds**, so the sequence is the
// caller's to fix: a walk answers its directories' order and a revision answers
// git's, and both are one answer for two reads. What the order decides is not
// which Manifest a name means — that is answered by rule rather than by
// sequence, an Extension never taking a built-in's name (manifestsByName) — but
// the order the artefacts themselves are carried in, which is what every pass
// over them walks.
//
// It answers no error. Everything a single file can do wrong — bytes that will
// not parse, a name it does not declare — is carried on that file's own
// LoadedArtefact, and the walk's one error is the directory listing this door
// does not have (issue #88).
func LoadFrom(sources []Source) Loaded {
	artefacts := make([]LoadedArtefact, 0, len(sources))
	for _, source := range sources {
		artefacts = append(artefacts, parseFile(source.Path, source.Bytes))
	}
	return build(artefacts)
}

// IsArtefact says whether a repository path is one of the five artefact
// locations' files: a `.yaml` directly under one of the four directories, or
// `hyper.yaml` at the root.
//
// It is exported because the walk is no longer the only reader of that rule — a
// caller reading a revision out of git filters a listing by it where
// artefactFiles filters a directory — and the two must be one rule rather than
// two that happen to agree. The walk is what it is checked against
// (source_test.go).
func IsArtefact(path string) bool {
	if path == DeclarationPath {
		return true
	}
	dir, name, nested := strings.Cut(path, "/")
	return nested &&
		slices.Contains(artefactDirs, dir) &&
		!strings.Contains(name, "/") &&
		strings.HasSuffix(name, ".yaml")
}

// build is the load past the reading: the built-in Provider, and the four
// namespaces folded out of what parsed.
//
// It is where the two doors meet, and it is one function rather than two
// because a repository read from a revision and a repository read from the
// working tree must be one repository — a namespace built differently by the
// door its bytes came in is two readings of one name.
//
// The built-in shell Provider is loaded first and through the same shape as any
// other artefact: its bytes are the compiled-in constant and its path the
// pseudo-path §9 renders, and the one Provider with no blob in the repository
// is a member of the load on the same footing as the rest (§3, §11, ADR-0039).
// It is not read from disk and cannot fail to parse — hyper's own bytes are not
// reviewed text — so it carries no problem of its own; its schema is
// CheckBuiltinShellProvider's to state, exactly as a providers/ file's is
// CheckManifest's.
func build(read []LoadedArtefact) Loaded {
	artefacts := make([]LoadedArtefact, 0, len(read)+1)
	artefacts = append(artefacts, LoadedArtefact{
		Path:  artefact.BuiltinShellProviderPath,
		Bytes: []byte(artefact.BuiltinShellProviderYAML),
		Root:  artefact.BuiltinShellProviderRoot(),
		OK:    true,
	})
	artefacts = append(artefacts, read...)

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
	}
}

// targetDeclarationsByName folds the loaded artefacts into the Target
// namespace's members: every targets/ file that parsed and named itself. A file
// that will not parse contributes nothing and neither does one whose target: is
// absent or is not a plain scalar — ADR-0064's rule, the same one
// manifestsByName folds the Provider namespace on.
//
// It is the one place a Target's name is decided to mean a declaration, and
// there is no collision for it to answer: `hyper` ships no built-in Target, so
// §11's `provider-name-collision` has no half at this artefact, and the
// construction argument that leaves one collision rule for Providers leaves
// none here (internal/verify's collisionProblems). Deciding it once is what
// makes the declaration `hyper targets` writes a row off and the declaration a
// Definition's targets: resolves to the same file rather than two readings of
// one repository.
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
// It is the one place a name is decided to mean a file, and **it answers no
// precedence rule at all**. An Extension declaring a built-in Provider's name
// contributes nothing: the compiled-in Manifest is what the name means, so a
// Definition resolving through it resolves through the built-in, which is what
// it was reviewed against. That is `provider-name-collision`, a load failure
// §11 states and never a *the later one wins* — precedence being how a
// Definition reviewed as one thing runs as another. The declining is here and
// the row that explains it is in the pass whose subject is the namespace
// (internal/verify), one predicate answering both (artefact.IsBuiltinProviderName).
//
// Two Extensions colliding is impossible by construction, which is why one rule
// covers every collision this fold can see. That argument is written at the
// check rather than repeated here (internal/verify's collisionProblems).
//
// Deciding it once is what makes the Manifest `hyper providers` reports for a
// name and the Manifest a Definition's provider: resolves to the same file
// rather than two readings of one repository.
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
		if artefact.ProviderOrigin(a.Path) == artefact.OriginExtension && artefact.IsBuiltinProviderName(name) {
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

	return parseFile(rel, data)
}

// parseFile is the reading itself, over bytes that are already in hand: the
// grammar and then the subset's violations, which is what a caller that read
// them off a revision needs and what loadFile does once it has the bytes off
// disk.
func parseFile(rel string, data []byte) LoadedArtefact {
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

// The three artefact directories a name is looked up in (§3, §12). The key each
// kind declares its own name under travels beside the directory at every call
// site below, because the pairing is what makes a lookup a lookup: a directory
// searched under the wrong key answers *the repository holds none* about an
// artefact that is sitting there.
//
// Two of the three keys are internal/artefact's own kind: constants, a
// Definition and a Procedure naming themselves under the word their kind: also
// carries. A Target declaration does not — its kind: is `target-declaration`
// and its name key is `target` — which is why TargetDeclarationName exists
// there and why the key is written out here.
const (
	definitionsDir = "definitions"
	proceduresDir  = "procedures"
	targetsDir     = "targets"
)

// Procedure answers the artefact a Procedure name was read from: its path, its
// exact bytes and what they parsed to.
//
// It is the file rather than the parse tree, which is what separates it from
// the four namespaces above: a Run's Provenance names the git blob id of the
// **file** the Procedure was read from (§7), and a digest over a parse tree is
// a second representation of bytes nobody hashed. `Procedures` answers whether
// a name resolves and this answers what it resolved to.
func (l Loaded) Procedure(name string) (LoadedArtefact, bool) {
	return l.declared(proceduresDir, artefact.KindProcedure, name)
}

// Definition answers the artefact a Definition name was read from, on
// Procedure's own footing and for its own reason: `definition_revision` is the
// blob id of the Definition file (§7).
func (l Loaded) Definition(name string) (LoadedArtefact, bool) {
	return l.declared(definitionsDir, artefact.KindDefinition, name)
}

// TargetDeclaration answers the artefact a Target name was read from. Its
// caller wants the **path** rather than the bytes — a Refusal names the file
// and the line to edit (§8, ADR-0042) — and a path a surface names must be a
// path that exists, which is why it is found by walking the load rather than
// composed from the name: `name-mismatch` pins a basename to a declared name
// (§4), and this reports where the bytes came from.
func (l Loaded) TargetDeclaration(name string) (LoadedArtefact, bool) {
	return l.declared(targetsDir, "target", name)
}

// declared is the walk all three are: the artefact under dir that parsed and
// declares name under key.
//
// Where two files declare one name it answers the **last** of them, which is
// not a precedence rule this walk is entitled to either — it is the rule the
// folds above already have, a map assignment keeping the last write. It is
// matched here deliberately: a surface that named a different file than the
// namespace resolved to would report one artefact and act on another, which is
// the one thing a collision `check` has not yet named must not cost.
func (l Loaded) declared(dir, key, name string) (LoadedArtefact, bool) {
	found, ok := LoadedArtefact{}, false
	for _, a := range l.Artefacts {
		if a.OK && strings.HasPrefix(a.Path, dir+"/") && artefact.DeclaredName(a.Root, key) == name {
			found, ok = a, true
		}
	}
	return found, ok
}
