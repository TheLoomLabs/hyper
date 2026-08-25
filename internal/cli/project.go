package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/cadence"
	"github.com/TheLoomLabs/hyper/internal/problem"
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
// **This one reads the pin and does not write it.** The version and the digest
// come from the binary and from `hyper.yaml`, which the gate has already proved
// is this binary's — so `project` is still gated here like every other command.
// Writing the pin, freezing the digest and standing outside the gate are the
// next ticket's (§11, ADR-0020).
func RunProject(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	// No --limit: `project` names no namespace to range over — it writes
	// what the repository asks for, all of it, and a cap on that would be a
	// projection nobody could review against the artefacts (§9).
	parsed, code := parseArgs("project", args, parameters{limit: takesNoLimit}, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// The whole grammar is the bare command. A positional here would have
	// to name a Procedure, and there is no per-Procedure projection for it
	// to name (§9, ADR-0060).
	if len(parsed.positional) > 0 {
		fmt.Fprintf(stderr, "hyper project: takes no positional argument, got %s — projection is repo-wide and all-or-nothing\n", parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("project", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and long before the first
	// write: a Refusal here leaves the tree exactly as it stands (§9, §11,
	// ADR-0020).
	if code, _ := gateOnVersionPin("project", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper project: %s\n", err)
		return ExitUsage
	}

	if problems := verify.Repository(loaded); len(problems) > 0 {
		problem.Sort(problems)
		rows := checkRows(problems)
		page := func(w io.Writer, rows []render.Row) error { return render.WriteTable(w, checkColumns, rows) }
		if code := writeAnswer("project", stdout, stderr, parsed.json, rows, render.NewResultRow(false), page); code != 0 {
			return code
		}
		return ExitProblems
	}

	// Nothing is written until everything is computed: every wanted file's
	// bytes, and every standing file the namespace holds, both in hand
	// before the first byte lands. A refusal or a failure before this point
	// leaves the tree untouched (§10).
	wanted := verify.Projection(loaded, binaryVersion)
	standing, err := standingWorkflows(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper project: %s\n", err)
		return ExitProblems
	}
	unwanted := removed(wanted, standing)

	if path, err := writeProjection(repoRoot, wanted, unwanted); err != nil {
		// The file it died on, named, and the tree left as it stands.
		// git is the undo, the tree is under review, and a rollback path
		// is code that runs only when something has already gone wrong
		// and is therefore the least-tested thing in the command (§10).
		fmt.Fprintf(stderr, "hyper project: %s: %s\n", path, err)
		return ExitProblems
	}

	rows := projectionRows(loaded, wanted, unwanted)
	if code := writeAnswer("project", stdout, stderr, parsed.json, rows, render.NewResultRow(false), writeProjectionTable); code != 0 {
		return code
	}
	return ExitClean
}

// standingWorkflows is every file in the namespace `project` owns that the
// working tree already holds, in path order (§10).
//
// A repository with no `.github/workflows/` holds none, which is an answer
// rather than a fault: the directory is created where a file is written into it
// and is not a thing a repository has to have first.
//
// **A directory whose name is in the namespace is not a file in it.** Nothing
// here recurses and nothing here removes a directory: what `project` owns is
// files directly under `.github/workflows/`, and a tree somebody put there under
// a name that looks like one is theirs.
func standingWorkflows(repoRoot string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(workflow.Dir)))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var held []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := workflow.Dir + "/" + entry.Name()
		if _, inside := workflow.ProcedureOf(path); inside {
			held = append(held, path)
		}
	}
	return held, nil
}

// removed is the standing files no Procedure asks for any more: what `project`
// takes away in the same act that writes the rest.
//
// A Procedure that has dropped its Cadence and a `hyper-*.yml` naming no
// Procedure at all reach this the same way, and that is the point — the
// namespace is answered as a set rather than one file at a time, so there is no
// shape of leftover that has to be recognised for what it is (§10).
func removed(wanted []verify.ProjectedWorkflow, standing []string) []string {
	asked := make(map[string]bool, len(wanted))
	for _, file := range wanted {
		asked[file.Path] = true
	}

	var unwanted []string
	for _, path := range standing {
		if !asked[path] {
			unwanted = append(unwanted, path)
		}
	}
	return unwanted
}

// writeProjection is the one act: every wanted file written whole, then every
// unwanted one removed. It answers the path it died on and the reason, and the
// empty path where it wrote everything.
//
// **The order is write-then-remove and it matters at one point only**: nothing
// is both, `removed` having subtracted one set from the other, so what the order
// buys is that a tree interrupted half way carries a file too many rather than a
// Cadence with nothing behind it.
//
// The directory is created where a file goes into it and never otherwise: a
// repository that projects nothing gets no empty `.github/workflows/`, an empty
// directory being a thing git will not carry anyway.
func writeProjection(repoRoot string, wanted []verify.ProjectedWorkflow, unwanted []string) (string, error) {
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
	return "", nil
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
	Type      string   `json:"type"`
	Path      string   `json:"path"`
	Procedure string   `json:"procedure,omitempty"`
	Cadence   string   `json:"cadence,omitempty"`
	Phrase    string   `json:"phrase,omitempty"`
	Rate      *float64 `json:"rate,omitempty"`

	// rateText is the rate in the notation the page renders it in, and
	// facts are §10's two facts about how the executor will treat the
	// declaration. Both are off the wire and on the row for the reason the
	// review's own gloss carries them so: the page is written from the rows
	// like every other page, and a rounding done twice or a fact derived
	// twice is two answers to one question (§8, §10, ADR-0026).
	rateText string
	facts    []string
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
	written := make([]workflowRow, 0, len(wanted)+len(unwanted))
	for _, file := range wanted {
		written = append(written, writtenRow(file))
	}
	for _, path := range unwanted {
		written = append(written, removedRow(loaded, path))
	}
	slices.SortFunc(written, func(a, b workflowRow) int {
		return strings.Compare(a.orderedBy(), b.orderedBy())
	})

	rows := make([]render.Row, 0, len(written))
	for _, row := range written {
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
	if gloss, readable := cadence.Read(file.Cadence); readable {
		row.Cadence = gloss.Expression
		row.Phrase = gloss.Phrase
		row.Rate = &gloss.Rate
		row.rateText = gloss.RateText
		row.facts = cadence.Facts(file.Cadence)
	}
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
