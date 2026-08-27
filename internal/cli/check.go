// Package cli is the third seam #87's Implementation Decisions name: argument
// handling, path stats, the two renderings, ordering, filtering, exit codes,
// stream discipline. It owns no rule of its own — every problem it prints
// comes from the load (internal/repository, over internal/yamlsubset's own
// grammar), the pin gate (internal/pin), and an artefact's own schema
// (internal/artefact).
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/verify"
)

// RunCheck implements `hyper check [path...]`. wd is the working directory
// repository-root resolution walks up from, and that is the whole of what this
// command reads it for: a path positional is read against the repository root
// the walk arrives at, never against wd itself (ADR-0089). lookupenv reads
// HYPER_REPO_DIR and NO_COLOR; binaryVersion is what the pin gate compares
// against hyper.yaml's pin. All four are passed in rather than read from the
// process directly, so the whole command is exercisable without a subprocess.
func RunCheck(args []string, to destination, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, to, code := parseArgs("check", args, parameters{limit: takesNoLimit}, lookupenv, to)
	if code != 0 {
		return code
	}
	paths := parsed.positional

	repoRoot, code := resolveRepoRoot("check", parsed.repoDir, lookupenv, wd, to.narrate())
	if code != 0 {
		return code
	}

	// The gate, before check's own positionals and work: every command
	// compares itself against hyper.yaml's version: pin before reading a
	// second file, and Refuses on mismatch in either direction (§9, §11,
	// ADR-0020). check calls it rather than carrying it.
	if code, _ := gateOnVersionPin("check", repoRoot, binaryVersion, to); code != 0 {
		return code
	}

	// check reads its path arguments against the repository root and stats
	// them there, before loading a single artefact (§9, ADR-0089). One root,
	// and the repository is it: a path names a file this repository holds or
	// it names nothing, which is what makes the argument the same argument
	// on both surfaces and what keeps a filter from silently matching no
	// problem at all. A path naming no file exits 2 and reports no problems
	// at all, and so does one that resolves outside the repository — decided
	// before the stat, a path outside the repository naming nothing this
	// command could report on however well it names a file on the caller's
	// own disk.
	wanted := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, fault := repoRelative(repoRoot, p)
		if fault == "" {
			if _, err := os.Stat(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil {
				fault = p + ": no such file or directory"
			}
		}
		if fault != "" {
			fmt.Fprintf(to.narrate(), "hyper check: %s\n", fault)
			return ExitUsage
		}
		wanted = append(wanted, rel)
	}

	// The repository is loaded by one call, which walks the artefact
	// locations, reads and parses each file, and builds the four namespaces
	// (issue #109). check evaluates not one rule before that call returns,
	// and writes not one line of the load itself: four commands in this
	// milestone need the same read and every command after them will.
	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper check: %s\n", err)
		return ExitUsage
	}

	// The second pass, which is §4's and not the load's. It runs over an
	// already-parsed repository, which is what lets a Definition's
	// provider: and targets: resolve against the whole repository's names
	// rather than only the files walked before it (issue #93).
	//
	// It lives in internal/verify rather than here because this command is
	// not its only caller: a Run re-runs it in full at Run start, which is
	// how all thirty-two of §4's static codes reach a Run (§6, ADR-0061,
	// issue #137). What is left here is what is this command's — which
	// problems it reports, in what order, and what it says where there are
	// none.
	problems := verify.Repository(loaded)

	if len(wanted) > 0 {
		problems = filterByPaths(problems, wanted)
	}
	problem.Sort(problems)

	// checked is every artefact the load returned, which is every file hyper
	// check read plus the built-in shell Provider — what a clean run's
	// confirmation line names (issue #99). The built-in being a loaded
	// artefact like any other, the count no longer adds one for it (issue
	// #109).
	checked := len(loaded.Artefacts)

	// The two renderings are one list of rows written twice (ADR-0026): the
	// page and the --json stream come out of the same ordered rows, so they
	// cannot state different things. check takes no --limit and has no
	// truncation axis, so its terminal row's marker is always false (issue
	// #110).
	rows := checkRows(problems)
	page := func(w io.Writer, rows []render.Row) error { return writeCheckTable(w, rows, checked) }
	if code := writeAnswer("check", to, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}

	if len(problems) > 0 {
		return ExitProblems
	}
	return ExitClean
}

// absPath resolves p against base if p is not already absolute. Which base a
// path argument is read against is the argument's own — the caller's working
// directory for the two that name something *outside* the repository, --repo-dir
// and --secret-out, and the repository root for the one that names something
// inside it (ADR-0089).
func absPath(base, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(base, p)
}

// repositoryRoot is what a positional naming the repository itself comes back
// as — the way filepath.Rel spells a path that is the root — and what the
// filter below reads as *every problem there is*. It is named rather than
// spelt twice because two functions share the fact.
const repositoryRoot = "."

// repoRelative reads one path positional against the repository root and gives
// it back in the spelling a problem's File is written in: slash-separated, and
// relative to that root. Where the argument names nothing this repository could
// hold it gives back the sentence saying so instead, and the caller writes that
// after its own name — the shape a usage error takes everywhere in this package
// (flags.go's arityFault, ADR-0060).
//
// A relative path is joined to the root and an absolute one is read as itself,
// and both are then held to the same bound: the repository. There is no second
// root and no arm — the caller's directory decides which repository is being
// checked (resolveRepoRoot) and never which file inside it is being reported on,
// which is what lets an agent that cannot see a working directory name a file at
// all (§9, ADR-0089).
//
// **The bound is decided lexically**, from the argument and the root and no disk
// read: it is a parse, in the sense ADR-0087 gives one, and it runs before the
// stat. The cost is stated in ADR-0089 — an absolute argument spelling a symlink
// prefix differently from the root is refused rather than resolved, which is a
// decline with a sentence and never a silent zero, and the repository-relative
// form is unaffected.
func repoRelative(repoRoot, p string) (rel, fault string) {
	if p == "" {
		return "", "the empty string names no path"
	}
	rel, err := filepath.Rel(repoRoot, absPath(repoRoot, p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", p + ": outside the repository"
	}
	return filepath.ToSlash(rel), ""
}

// filterByPaths keeps only the problems positioned in the paths named,
// having already loaded and checked every artefact in the repository (§9):
// every rule compares one artefact against another, so a subset of the
// repository is not checkable on its own — only reportable on its own.
//
// wanted is what repoRelative already made of the positionals, which is the
// spelling a problem's File carries. The two being one spelling is the whole of
// what this ticket was: a filter reading its argument against one root while the
// stat beside it read the same argument against another matched no problem at
// all, and reported clean over a repository full of them (ADR-0089, issue #205).
func filterByPaths(problems []problem.Problem, wanted []string) []problem.Problem {
	var kept []problem.Problem
	for _, prob := range problems {
		for _, want := range wanted {
			if want == repositoryRoot || prob.File == want || strings.HasPrefix(prob.File, want+"/") {
				kept = append(kept, prob)
				break
			}
		}
	}
	return kept
}

// problemRow is check's row on the row stream, and its field order is §9's
// example verbatim: {"type":"problem","file":...,"line":...,"column":...,
// "field":...,"error_code":...,"message":...}. encoding/json marshals a
// struct's fields in declaration order, which is what fixes a row's key order
// on the wire, and the type declared first is what puts it first; the corpus
// holds that rule against every stream any command writes.
//
// It lives here rather than in internal/render because a row type belongs to
// the command that writes it: the renderer owns the stream, and check owns
// what it puts on one.
type problemRow struct {
	Type      string `json:"type"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Field     string `json:"field"`
	ErrorCode string `json:"error_code"`
	Message   string `json:"message"`
}

// Cells is the row's line on check's page: the file, the line, the field, the
// error_code and the message. column rides on the wire only (§9) — a row
// carrying more than its page renders, which is not the two surfaces
// disagreeing but one of them having no column for a fact a consumer filters
// on.
//
// **A problem with no line renders an empty cell**, as one with no field
// already does. No file has a line 0, so printing the number would put a
// position on the page that cannot be gone to; the wire carries the zero, where
// it reads as the absence the `field` beside it reads as (§12). The one code
// this reaches is `projection-stale`, whose comparison is whole-file (§10).
func (r problemRow) Cells() []string {
	line := ""
	if r.Line != 0 {
		line = strconv.Itoa(r.Line)
	}
	return []string{r.File, line, r.Field, r.ErrorCode, r.Message}
}

// checkRows is the sorted problems as the ordered list both surfaces are
// written from. The order is the one problem.Sort fixed and neither rendering
// is free to change it: on the wire the rows arrive one line at a time and a
// consumer cannot re-sort what it has already printed (§9).
func checkRows(problems []problem.Problem) []render.Row {
	rows := make([]render.Row, 0, len(problems))
	for _, p := range problems {
		rows = append(rows, problemRow{
			Type:      "problem",
			File:      p.File,
			Line:      p.Line,
			Column:    p.Column,
			Field:     p.Field,
			ErrorCode: p.ErrorCode,
			Message:   p.Message,
		})
	}
	return rows
}

// checkColumns is check's header. The columns a page carries are the command's
// own — one renderer means one path from rows to bytes, not one table layout
// for every command.
var checkColumns = []string{"FILE", "LINE", "FIELD", "ERROR_CODE", "MESSAGE"}

// writeCheckTable is check's page: its five columns, and the line that stands
// where there are no rows. A clean run is not silent (issue #99): it names how
// many artefacts were checked and that nothing was found, checked being the
// load's own count — every repository file `hyper check` read plus the built-in
// shell Provider — rather than a header over no rows.
func writeCheckTable(w io.Writer, rows []render.Row, checked int) error {
	if len(rows) == 0 {
		plural := "s"
		if checked == 1 {
			plural = ""
		}
		_, err := fmt.Fprintf(w, "checked %d artefact%s: no problems found\n", checked, plural)
		return err
	}
	return render.WriteTable(w, checkColumns, rows)
}
