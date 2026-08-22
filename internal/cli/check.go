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
// repository-root resolution walks up from; lookupenv reads HYPER_REPO_DIR and
// NO_COLOR; binaryVersion is what the pin gate compares against hyper.yaml's
// pin. All four are passed in rather than read from the process directly, so
// the whole command is exercisable without a subprocess.
func RunCheck(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, code := parseArgs("check", args, takesNoLimit, lookupenv, stderr)
	if code != 0 {
		return code
	}
	paths := parsed.positional

	repoRoot, code := resolveRepoRoot("check", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before check's own positionals and work: every command
	// compares itself against hyper.yaml's version: pin before reading a
	// second file, and Refuses on mismatch in either direction (§9, §11,
	// ADR-0020). check calls it rather than carrying it.
	if code := gateOnVersionPin("check", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	// check stats its path arguments before loading a single artefact
	// (§9): a path naming no file exits 2 and reports no problems at all.
	for _, p := range paths {
		if _, err := os.Stat(absPath(wd, p)); err != nil {
			fmt.Fprintf(stderr, "hyper check: %s: no such file or directory\n", p)
			return ExitUsage
		}
	}

	// The repository is loaded by one call, which walks the artefact
	// locations, reads and parses each file, and builds the four namespaces
	// (issue #109). check evaluates not one rule before that call returns,
	// and writes not one line of the load itself: four commands in this
	// milestone need the same read and every command after them will.
	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper check: %s\n", err)
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

	if len(paths) > 0 {
		problems = filterByPaths(problems, paths, repoRoot, wd)
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
	if code := writeAnswer("check", stdout, stderr, parsed.json, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}

	if len(problems) > 0 {
		return ExitProblems
	}
	return ExitClean
}

// absPath resolves p against wd if p is not already absolute — the one rule
// every path argument this command takes (--repo-dir, a path positional) is
// read by.
func absPath(wd, p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(wd, p)
}

// filterByPaths keeps only the problems positioned in the paths named,
// having already loaded and checked every artefact in the repository (§9):
// every rule compares one artefact against another, so a subset of the
// repository is not checkable on its own — only reportable on its own.
func filterByPaths(problems []problem.Problem, paths []string, repoRoot, wd string) []problem.Problem {
	wanted := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(repoRoot, absPath(wd, p))
		if err != nil {
			rel = p
		}
		wanted = append(wanted, filepath.ToSlash(rel))
	}

	var kept []problem.Problem
	for _, prob := range problems {
		for _, want := range wanted {
			if prob.File == want || strings.HasPrefix(prob.File, want+"/") {
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
func (r problemRow) Cells() []string {
	return []string{r.File, strconv.Itoa(r.Line), r.Field, r.ErrorCode, r.Message}
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
