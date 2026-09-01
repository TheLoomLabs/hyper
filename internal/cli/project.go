package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/mcp"
	"github.com/TheLoomLabs/hyper/internal/pin"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/release"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/verify"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// RunProject implements `hyper project` — the fifteenth of §9's sixteen, and
// the first thing in the tool that writes a file into the working tree (§10,
// §11, issue #177).
//
// **It takes no arguments at all.** It is repo-wide and all-or-nothing: there is
// no `project <procedure>`, since per-Procedure projection would let two
// Procedures pin two versions against one Store, so a positional is a usage
// error at `2`. There is no `--dry-run` either — the diff it writes is the
// rehearsal, and `git` is where it is read — and the three globals apply with no
// fourth (§9, ADR-0015).
//
// **What it does, in one act**: one workflow file per Procedure declaring a
// Cadence, and every file in the namespace no Procedure asks for any more,
// removed. Generation is whole-file and always overwriting, never merging: a
// hand-edit to a projected file does not survive, which is correct rather than
// regrettable, being authority living outside every reviewed artefact (§9, §10).
//
// **It writes nothing where `check` would report anything.** The projection is
// derived from reviewed artefacts, and deriving from a repository that does not
// check is deriving from something nobody could review. It runs the same static
// pass every other reader does, prints `check`'s own problem table and exits
// `1`; the rule is load-bearing rather than defensive in one case above all,
// `cadence-malformed` existing precisely so that an expression no grammar admits
// never reaches an executor's clock (§4, §10, issue #174).
//
// **One code is excluded from that pass and only one**: `projection-stale`, the
// drift this command is the repair for. Including it would make the command
// that repairs the drift refuse to run because of the drift, which is a state
// with no way out of it (§10, issue #179).
//
// **It writes the pin, and nothing else in the tool does.** The version is the
// binary's own, derived rather than authored, and the digest beside it is the
// checksum published for that version — so changing the version is *install a
// binary, run one command, read the diff*, three acts in the open, each leaving
// something behind (§11, ADR-0020).
//
// **`hyper.yaml` is edited, never regenerated**, and created where the
// repository holds none. What that costs and why it is not the whole-file rule
// the workflows are written under is internal/pin's to state; what is this
// command's is that the edit happens in the same act the workflows do.
//
// **And it writes `AGENTS.md` where the repository holds none**, which is the
// third path in its namespace and the only one that is not derived from an
// artefact. The bytes are the orientation internal/mcp already states, at this
// binary's version, because two orientations disagree the first time either is
// edited. ADR-0093 refused this and is amended rather than reversed: what moved
// is that `project` is the documented first act on a new repository, so the
// cold start the ADR said `project` could not reach is exactly the moment
// `project` runs (§9, §11, ADR-0095, issue #211).
//
// **It is create-if-absent and never an overwrite**, which is the one place
// this command departs from whole-file regeneration. `AGENTS.md` is a shared
// file most repositories already hold for reasons having nothing to do with
// `hyper`, and ADR-0093 is right that always-overwriting semantics are correct
// for a generated workflow and wrong for a note addressed to a reader. Where
// one stands, `hyper` leaves it, and the orientation's own closing paragraph is
// what covers the gap: the agent offers to add a section, and the human decides.
//
// **And this one stands outside the pin gate**, the only command in §9's tree
// that does. It is exempt not for being read-only, which §11 refuses as a ground
// for anything, but for being **the pin's only writer**: a gated `project` on an
// unpinned repository would Refuse naming itself, and a gated `project` under a
// newer binary would Refuse naming itself, which makes the upgrade ritual
// unperformable at step two — a writer gated on what it writes is a bootstrap
// with no bootstrap. ADR-0001 is untouched: `project` does not proceed under a
// pin it disagrees with, it replaces the pin and writes the replacement into a
// tracked file whose diff is the review (§9, §11, ADR-0020).
//
// That sentence is this command's, and the surfaces that assert the exemption
// point back at it rather than restating it (golden_test.go). **The MCP tool is
// one of them**: it inherits the exemption by going through the same dispatch
// rather than declaring one, which is what makes *ergonomics is the whole of the
// difference between the two* true of the one command that stands outside the
// gate (§9, internal/mcp's projectTool, issue #203).
func RunProject(args []string, to destination, process Process, wd, binaryVersion string) int {
	// No --limit: `project` names no namespace to range over — it writes
	// what the repository asks for, all of it, and a cap on that would be a
	// projection nobody could review against the artefacts (§9).
	parsed, to, code := parseArgs("project", args, parameters{limit: takesNoLimit}, process.LookupEnv, to)
	if code != 0 {
		return code
	}
	// The whole grammar is the bare command. A positional here would have
	// to name a Procedure, and there is no per-Procedure projection for it
	// to name (§9, ADR-0060).
	if len(parsed.positional) > 0 {
		fmt.Fprintf(to.narrate(), "hyper project: takes no positional argument, got %s — projection is repo-wide and all-or-nothing\n", parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("project", parsed.repoDir, process.LookupEnv, wd, to.narrate())
	if code != 0 {
		return code
	}

	// No gate. Every other command in the tree compares itself against the
	// pin here, before it reads a second file; this one is about to write
	// that pin, and a repository with no pin at all is exactly the
	// repository it exists to give one to (§9, §11, ADR-0020).
	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper project: %s\n", err)
		return ExitUsage
	}

	if problems := blockingProblems(verify.Repository(loaded)); len(problems) > 0 {
		problem.Sort(problems)
		rows := checkRows(problems)
		page := func(w io.Writer, rows []render.Row) error { return render.WriteTable(w, checkColumns, rows) }
		if code := writeAnswer("project", to, rows, render.NewResultRow(false), page); code != 0 {
			return code
		}
		return ExitProblems
	}

	// The declaration as it stands, read once: the bytes about to be edited
	// and the two facts about to be replaced.
	declared := readDeclaration(loaded)

	// The one network read, where there is one at all: the version this
	// invocation is about to pin, and the checksum published for it. A
	// Refusal or a failure here happens before anything is computed and long
	// before anything is written (§11).
	digest, code := frozenDigest(process.Dial, declared, binaryVersion, to)
	if code != 0 {
		return code
	}

	// Nothing is written until everything is computed: the declaration's new
	// bytes, every wanted file's, and every standing file the namespace
	// holds, all in hand before the first byte lands. A refusal or a failure
	// before this point leaves the tree untouched (§10).
	//
	// The two derived facts go to the projection as arguments rather than
	// through the file: `loaded` still carries the pin being replaced, so a
	// workflow generated off it would install the version this invocation is
	// upgrading away from.
	//
	// The pass above projected too, at the pin the declaration carries
	// rather than at the one being written — which is why an upgrade is not
	// a repository that fails its own pre-write check: the files standing
	// were written at the declared version and match a regeneration at it,
	// and what moves them is this write (§11, verify.Projection).
	pinned := pin.Written(declared.bytes, declared.present, binaryVersion, digest)
	wanted := verify.Projection(loaded, binaryVersion, digest)
	unwanted := verify.UnwantedWorkflows(loaded, wanted)
	note := mcp.Instructions(binaryVersion)

	if path, err := writeDerived(repoRoot, pinned, wanted, unwanted, note); err != nil {
		// The file it died on, named, and the tree left as it stands.
		// git is the undo, the tree is under review, and a rollback path
		// is code that runs only when something has already gone wrong
		// and is therefore the least-tested thing in the command (§10).
		fmt.Fprintf(to.narrate(), "hyper project: %s: %s\n", path, reasonFor(err))
		return ExitProblems
	}

	rows := projectionRows(loaded, wanted, unwanted)
	if code := writeAnswer("project", to, rows, render.NewResultRow(false), writeProjectionTable); code != 0 {
		return code
	}
	return ExitClean
}

// blockingProblems is what stops a projection: everything `check` would report
// except the one thing this command exists to repair.
//
// **`projection-stale` is excluded and nothing else is.** Every other problem
// still stops it, on the rule above — a projection derived from a repository
// that does not check is derived from something nobody could review — and this
// one is the exception that rule cannot survive without: a command that refused
// to run because the working tree's workflows are stale is the repair refusing
// on the ground that there is something to repair, and the state it leaves a
// reader in has no exit (§4, §10, issue #179).
func blockingProblems(problems []problem.Problem) []problem.Problem {
	kept := make([]problem.Problem, 0, len(problems))
	for _, found := range problems {
		if found.ErrorCode != verify.CodeProjectionStale {
			kept = append(kept, found)
		}
	}
	return kept
}

// writeDerived is the one act: the Repository declaration, then every wanted
// file written whole, then every unwanted one removed, then the note where the
// repository holds none. It answers the path it died on and the reason, and the
// empty path where it wrote everything.
//
// It is named for what it writes rather than for the namespace, because the
// declaration is not part of the projection: it is a reviewed artefact carrying
// two derived facts, and what these two writes have in common is that `hyper`
// derived both (§9, §11).
//
// **The order is write-then-remove and it matters at one point only**: nothing
// is both, `removed` having subtracted one set from the other, so what the order
// buys is that a tree interrupted half way carries a file too many rather than a
// Cadence with nothing behind it.
//
// The directory is created where a file goes into it and never otherwise: a
// repository that projects nothing gets no empty `.github/workflows/`, an empty
// directory being a thing git will not carry anyway.
//
// **The note goes last, and that is the same choice read once more.** It is the
// only thing written here that no artefact derives, so a tree interrupted
// before it is a repository that projected correctly and lacks a note — which
// is the reading that costs a reader least, and the one running this again
// repairs (§10, ADR-0095).
func writeDerived(repoRoot string, pinned []byte, wanted []verify.ProjectedWorkflow, unwanted []string, note string) (string, error) {
	// The declaration first, because it is the fact the rest of this act
	// derives from: a tree interrupted after it names the version its
	// workflows are converging on, where the other order leaves a repository
	// pinning a binary its own files no longer install. Both are
	// `projection-stale` and both are repaired by running this again, which
	// is why the choice is about which half-written tree reads truthfully
	// rather than about which one is safe (§10, §11).
	if err := os.WriteFile(filepath.Join(repoRoot, repository.DeclarationPath), pinned, 0o644); err != nil {
		return repository.DeclarationPath, err
	}
	if len(wanted) > 0 {
		if err := os.MkdirAll(filepath.Join(repoRoot, filepath.FromSlash(workflow.Dir)), 0o755); err != nil {
			return workflow.Dir, err
		}
	}
	for _, file := range wanted {
		if err := os.WriteFile(filepath.Join(repoRoot, filepath.FromSlash(file.Path)), file.Bytes, 0o644); err != nil {
			return file.Path, err
		}
	}
	for _, path := range unwanted {
		if err := os.Remove(filepath.Join(repoRoot, filepath.FromSlash(path))); err != nil {
			return path, err
		}
	}
	if err := writeNote(repoRoot, note); err != nil {
		return notePath, err
	}
	return "", nil
}

// notePath is where the orientation lands. It is the convention's own name and
// not `hyper`'s: harnesses read `AGENTS.md` up front, unprompted, which is the
// whole of why the file is worth writing (ADR-0095).
//
// It is a literal here rather than a constant somewhere shared because nothing
// but this command and the cases over it has any business naming it — no
// artefact resolves through it, `check` does not count it, and no Run reads it
// (§9, ADR-0093).
const notePath = "AGENTS.md"

// writeNote writes the orientation where the repository holds no `AGENTS.md`,
// and leaves any file that stands exactly as it is.
//
// **The absence is tested by the write rather than before it.** `O_EXCL` is the
// one form in which *is it there* and *claim it* are a single syscall; a `Stat`
// followed by a `WriteFile` is a window in which a file appears and is then
// taken, and the file this is about is one somebody may be authoring in the
// next pane. An `EEXIST` is therefore the ordinary outcome rather than a
// failure, and it is the only error swallowed here.
func writeNote(repoRoot, note string) error {
	file, err := os.OpenFile(filepath.Join(repoRoot, notePath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if os.IsExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := file.WriteString(note); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

// standingDeclaration is the Repository declaration as it stands before
// `project` writes: the bytes it will edit, whether the repository holds the
// file at all, and the two derived facts already in it.
//
// It is named for standing as the files verify.UnwantedWorkflows reads are:
// what this command does is compare what a repository asks for against what is
// already there, and both halves of *what is already there* read alike.
//
// It is a type rather than four values threaded singly because they are one
// read of one file, and because two of them decide whether the third is
// replaced: a pin equal to the binary's version keeps the digest beside it, and
// any other pin resolves a new one. What separates it from
// artefact.RepositoryFacts is `retention:` — the declaration's one authored
// fact, which `project` neither reads nor writes (§3, §11).
type standingDeclaration struct {
	bytes   []byte
	present bool
	version string
	digest  string
}

// readDeclaration reads all four off one loaded repository.
//
// **The two facts come through two doors and that is the pin's own shape.** The
// version is internal/pin's, because the gate reads it off `hyper.yaml`'s bytes
// before a repository is loaded at all and there must not be a second reading of
// it; the digest is internal/artefact's, because it is one of the things the
// declaration *says* and every such fact is read there (§9, §11, ADR-0020). What
// this does is put the two answers in one value, so that nothing downstream has
// to know there were two doors.
func readDeclaration(loaded repository.Loaded) standingDeclaration {
	bytes, present := loaded.DeclarationBytes()
	return standingDeclaration{
		bytes:   bytes,
		present: present,
		version: pin.Declared(bytes),
		digest:  artefact.ReadRepositoryFacts(loaded.Declaration()).Digest,
	}
}

// frozenDigest is the checksum the declaration this invocation writes will
// carry, and the exit code where there is none to write.
//
// **Where the pin already equals the binary's version, nothing is resolved at
// all.** Re-projection reaches no network: the digest already in the declaration
// is a reviewed fact, and it is copied into every workflow exactly as it stands.
// Only a version change fetches, which is what makes the fetch a thing that
// happens on the upgrade a human is performing rather than on every invocation
// (§11).
//
// **What it fetches is a few hundred bytes, attended, at review time**, and what
// that buys is internal/release's to say.
//
// The two ways it can fail are two exits, and what sorts them is whether the
// answer is *that there is nothing to pin*. A release naming no artefact for
// this version is a check declining, and it Refuses at `77` — rendered in the
// two-line form with no caret, the fault having no artefact coordinate and the
// remedy being a released binary rather than an edit (§8, §12). Everything else
// is the world resisting and exits `1`: a host that did not respond, a
// resolution that timed out, and equally a release host that answered a rate
// limit — all of them can differ between two invocations of an identical command
// line, which is exactly what `77` promises they cannot, and `1` is where
// `install` already puts them (§11, §12, ADR-0060). Which statuses fall on which
// side is internal/release's, and which remedy the Refusal names is remedyFor's.
// Nothing is written on either path.
func frozenDigest(dial capability.Dial, declared standingDeclaration, binaryVersion string, to destination) (string, int) {
	if declared.version == binaryVersion {
		return declared.digest, 0
	}

	digest, err := release.Digest(context.Background(), dial, binaryVersion)
	var absent *release.Absent
	switch {
	case errors.As(err, &absent):
		// An unreleased binary runs and checks and cannot project, which
		// is the same statement as: every pin in every repository names a
		// version somebody can download (§11).
		return "", refuse(to, release.CodeArtefactAbsent, absent.Error()+" — "+remedyFor(absent, binaryVersion))
	case err != nil:
		fmt.Fprintf(to.narrate(), "hyper project: the checksum for %s did not arrive: %s\n", binaryVersion, err)
		return "", ExitProblems
	}
	return digest, 0
}

// remedyFor is the sentence after the em dash on a `release-artefact-absent`
// Refusal, and there are two of them because the two absences are of different
// standing (#254, ADR-0127).
//
// **A remedy may not assert what the answer could not establish.** Where the
// checksums file arrived and named no artefact, the release is readable and its
// contents were seen: it does not hold this version's artefact, and the two
// routes out are publishing one and running a binary some release does name.
// Where nothing arrived, the status was `404` or `410` — and this read carries no
// credential (ADR-0007), so a release published where nothing unauthenticated
// may read it answers exactly as an absent one does. Telling that operator to
// publish a release they have already published, and to install a released
// binary they are already running, is a remedy that is false twice over and
// leaves the true route out — make it readable — unnamed. So the sentence names
// the third possibility rather than resolving it, which is the one honest thing
// to say about an answer that cannot tell them apart.
//
// **The Refusal and the exit code are unchanged**, and that is the point: `77`
// promises a verbatim retry Refuses identically, which a private release keeps —
// it is `hyper`'s reading of the fact that is corrected here and never its
// verdict (§11, §12).
func remedyFor(absent *release.Absent, binaryVersion string) string {
	if absent.Arrived() {
		return "publish a release for " + binaryVersion + ", or install a released hyper"
	}
	return "publish a release for " + binaryVersion + ", make an existing one readable unauthenticated, or install a released hyper"
}

// reasonFor is what a file operation failed with, less the path it already
// carries.
//
// os names the file in every *os.PathError, absolutely and as the process
// resolved it — and the message above has already named it in the repository's
// own vocabulary, which is the vocabulary every path this command reports is
// spelled in. Naming it twice, once relative and once absolute, is one fault
// reported as two files.
//
// Anything that is not a *os.PathError is written whole: the unwrapping is
// about a duplicate this command introduced and not about shortening what the
// world said.
func reasonFor(err error) error {
	var path *os.PathError
	if errors.As(err, &path) {
		return path.Err
	}
	return err
}

// workflowRow is `project`'s row, and the only one it writes: §9 fixes it at
// `{ type: "workflow", path, procedure, cadence, phrase, rate }` — the gloss's
// parts rather than the composed line, one per Procedure, all of them.
//
// It carries no `outcome` key and terminates in `result`, `project` not being a
// Run; and it carries no last Journal entry, which is the one absence §10 states
// outright: a surface with no artefact under review has no side to hang one on
// (§8, §9, §10).
//
// **A removed file's row is the same type carrying `path` and `procedure` and no
// gloss at all** — the absence is the fact, and it is the ordinary absence rule
// rather than a widening of the shape. Where the Procedure no longer exists,
// `procedure` is absent too and only `path` stands (§7, §10).
type workflowRow struct {
	Type      string `json:"type"`
	Path      string `json:"path"`
	Procedure string `json:"procedure,omitempty"`
	// The gloss's parts, embedded so that `cadence`, `phrase` and `rate`
	// stand at this row's own top level after `procedure` — the same value
	// a review's `artefact` row carries, read once for both
	// (cadence_gloss.go, §9, §10).
	cadenceGloss
}

// Cells is the row's line: where the file is, which Procedure asked for it, and
// the Cadence column §10 fixes the content of.
func (r workflowRow) Cells() []string {
	return []string{r.Path, r.Procedure, r.cadenceCell()}
}

// removedCell is what the Cadence column renders for a file that is no longer
// asked for. It is a word rather than a blank because the row is the record of
// an act: a cell left empty would read as *this file declares no recurrence*,
// which is true of it and is not what happened to it (§10).
const removedCell = "removed"

// cadenceCell is §10's own arrangement, in a table cell: the expression, the
// phrase and the rate **stack**, the gloss under the cron it glosses, and the
// two facts close it — the same shape `THE CODE MOVED`'s `cadence` row draws,
// because a reader learns one (§8, §10).
//
// **The arrangement is written here rather than shared with that row**, and that
// is §10's own division: *how the three parts are arranged is the surface's, and
// what they are is not*. What they are is the value this row embeds, read once;
// how they are laid out differs surface by surface — a review's header joins
// them with `·` and hangs *last ran* off the end — so a page that borrowed
// another's arrangement would be a surface with no say in its own layout.
//
// The column is derived from the row and adds no member: everything on it comes
// off `cadence`, `phrase` and `rate`, which is what lets a consumer of the wire
// render the same cell (§9, ADR-0026).
func (r workflowRow) cadenceCell() string {
	if r.Cadence == "" {
		return removedCell
	}
	return strings.Join(append([]string{r.Cadence, r.Phrase, r.rateText}, r.facts...), "\n")
}

// projectionRows is the whole answer, ordered: one row per file written, one per
// file removed, by Procedure name by Unicode code point, page and wire alike
// (§9, §10).
//
// **The name a row is ordered by is the one its own path carries**, which for a
// written file is the Procedure's name and is the only name a removed file that
// names no Procedure has. One key rather than two is what keeps the order total
// over a list that holds both kinds.
func projectionRows(loaded repository.Loaded, wanted []verify.ProjectedWorkflow, unwanted []string) []render.Row {
	reported := make([]workflowRow, 0, len(wanted)+len(unwanted))
	for _, file := range wanted {
		reported = append(reported, writtenRow(file))
	}
	for _, path := range unwanted {
		reported = append(reported, removedRow(loaded, path))
	}
	slices.SortFunc(reported, func(a, b workflowRow) int {
		return strings.Compare(a.orderedBy(), b.orderedBy())
	})

	rows := make([]render.Row, 0, len(reported))
	for _, row := range reported {
		rows = append(rows, row)
	}
	return rows
}

// orderedBy is the name this row is ordered by: the one its path carries.
func (r workflowRow) orderedBy() string {
	name, _ := workflow.ProcedureOf(r.Path)
	return name
}

// writtenRow is the row for a file this invocation wrote, glossed. A Cadence
// that reached here is one §10's grammar admits — `cadence-malformed` is a
// problem, and `project` writes nothing where `check` reports one — so the gloss
// is always there to render (§4, §10).
func writtenRow(file verify.ProjectedWorkflow) workflowRow {
	row := workflowRow{Type: "workflow", Path: file.Path, Procedure: file.Procedure}
	row.read(file.Cadence)
	return row
}

// removedRow is the row for a file this invocation took away: the path, and the
// Procedure where one of that name still stands.
//
// The Procedure is looked up rather than assumed, and the lookup is what tells
// §10's two removals apart: a Procedure that dropped its `cadence:` is still a
// Procedure and is named, and a `hyper-*.yml` naming nothing in the repository
// leaves only its path (§10).
func removedRow(loaded repository.Loaded, path string) workflowRow {
	row := workflowRow{Type: "workflow", Path: path}
	if name, inside := workflow.ProcedureOf(path); inside && loaded.Procedures[name] {
		row.Procedure = name
	}
	return row
}

// projectColumns is `project`'s header. The Cadence column is the widest thing
// on the page and stands last for it: a stacked cell opens out downwards and
// takes the columns beside it with it, so a column that wraps belongs where
// nothing has to be aligned past it (§8, render.WriteAligned).
var projectColumns = []string{"PATH", "PROCEDURE", "CADENCE"}

// writeProjectionTable is `project`'s page: its three columns, and the sentence
// that stands where there is nothing to report.
//
// The sentence states both halves of what an empty answer means, because the
// page has no rows to say either: no Procedure asked for a file, and none stood
// to be taken away. A header over no rows would say less than that (§9, issue
// #99).
func writeProjectionTable(w io.Writer, rows []render.Row) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no Procedure declares a Cadence, and no generated workflow stands")
		return err
	}
	return render.WriteTable(w, projectColumns, rows)
}
