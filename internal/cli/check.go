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

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunCheck implements `hyper check [path...]`. wd is the working directory
// repository-root resolution walks up from; getenv reads HYPER_REPO_DIR and
// NO_COLOR; binaryVersion is what the pin gate compares against hyper.yaml's
// pin. All four are passed in rather than read from the process directly, so
// the whole command is exercisable without a subprocess.
func RunCheck(args []string, stdout, stderr io.Writer, getenv func(string) string, wd, binaryVersion string) int {
	flags, paths, code := parseFlags(args, stderr)
	if code != 0 {
		return code
	}

	repoRoot, code := resolveRepoRoot(flags.repoDir, getenv, wd, stderr)
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

	// The second pass, which is check's own and not the load's. It runs
	// over an already-parsed repository, which is what lets a Definition's
	// provider: and targets: resolve against the whole repository's names
	// rather than only the files walked before it (issue #93). A failed
	// load does not stop it: reading or parsing one file stops every check
	// after it for that file — never for the repository (issue #88).
	var problems []problem.Problem
	for _, a := range loaded.Artefacts {
		problems = append(problems, a.Problems...)
		if !a.OK {
			continue
		}
		problems = append(problems, checkArtefact(a, loaded)...)
	}

	// The transitive walk — an invoked Procedure's own envelope against its
	// caller's, and the two Cadence rules that ride the same walk — needs
	// every procedures/ file at once, so it runs once here rather than per
	// file inside checkArtefact (issue #96).
	graph := artefact.BuildProcedureGraph(procedureRoots(loaded.Artefacts), loaded.Providers, loaded.Definitions)
	problems = append(problems, artefact.CheckProcedureGraph(graph)...)

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
	var renderErr error
	if flags.json {
		renderErr = render.WriteJSON(stdout, rows, render.NewResultRow(false))
	} else {
		renderErr = writeCheckTable(stdout, rows, checked)
	}
	if renderErr != nil {
		fmt.Fprintf(stderr, "hyper check: %s\n", renderErr)
		return ExitUsage
	}

	if len(problems) > 0 {
		return ExitProblems
	}
	return ExitClean
}

type checkFlags struct {
	json    bool
	noColor bool
	repoDir string
}

// parseFlags reads the three globals ADR-0014 fixes and nothing else: --json,
// --repo-dir (HYPER_REPO_DIR), --no-color (NO_COLOR). Anything else
// beginning "--" is a usage error. noColor is threaded through but has
// nothing to affect yet — the check table carries no colour of its own to
// suppress, which is why --no-color and NO_COLOR already produce identical
// bytes.
func parseFlags(args []string, stderr io.Writer) (checkFlags, []string, int) {
	var flags checkFlags
	var paths []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			paths = append(paths, args[i+1:]...)
			i = len(args)
		case a == "--json":
			flags.json = true
		case a == "--no-color":
			flags.noColor = true
		case a == "--repo-dir":
			i++
			if i >= len(args) {
				fmt.Fprintln(stderr, "hyper check: --repo-dir requires a value")
				return flags, nil, ExitUsage
			}
			flags.repoDir = args[i]
		case strings.HasPrefix(a, "--repo-dir="):
			flags.repoDir = strings.TrimPrefix(a, "--repo-dir=")
		case strings.HasPrefix(a, "--"):
			fmt.Fprintf(stderr, "hyper check: unknown flag %s\n", a)
			return flags, nil, ExitUsage
		default:
			paths = append(paths, a)
		}
	}
	return flags, paths, 0
}

// resolveRepoRoot applies flags → environment → defaults (ADR-0014):
// --repo-dir, then HYPER_REPO_DIR, then walking up from wd bounded by the
// git root.
func resolveRepoRoot(repoDirFlag string, getenv func(string) string, wd string, stderr io.Writer) (string, int) {
	repoDir := repoDirFlag
	if repoDir == "" {
		repoDir = getenv("HYPER_REPO_DIR")
	}
	if repoDir != "" {
		resolved := absPath(wd, repoDir)
		info, err := os.Stat(resolved)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(stderr, "hyper check: --repo-dir %s: not a directory\n", repoDir)
			return "", ExitUsage
		}
		return resolved, 0
	}

	root, ok := repository.FindGitRoot(wd)
	if !ok {
		fmt.Fprintln(stderr, "hyper check: not inside a git repository; pass --repo-dir or set HYPER_REPO_DIR")
		return "", ExitUsage
	}
	return root, 0
}

// procedureRoots is every loaded Procedure's root paired with its own file —
// what BuildProcedureGraph needs to cite a fault against the file that carries
// it (issue #96). A file that failed to parse contributes no root, on
// ADR-0064's own rule. The graph is check's and not the load's, so the shaping
// its input needs is here.
func procedureRoots(artefacts []repository.LoadedArtefact) []artefact.ProcedureRoot {
	var roots []artefact.ProcedureRoot
	for _, a := range artefacts {
		if a.OK && strings.HasPrefix(a.Path, "procedures/") {
			roots = append(roots, artefact.ProcedureRoot{File: a.Path, Root: a.Root})
		}
	}
	return roots
}

// checkArtefact runs one already-parsed artefact's own schema and the checks
// that read it against itself or against the repository: hyper.yaml, a file in
// targets/, a file in providers/, a file in definitions/ and, since issue #94,
// a file in procedures/ — the five artefacts this milestone's schema reaches —
// plus the built-in shell Provider, which the load carries like any other
// artefact and which is checked here with no exemption (§3, ADR-0081, issues
// #99, #109): a Provider is data, and data check may not read is an advisory
// analyzer wearing the tool's own badge.
//
// loaded is passed whole rather than as its four namespaces, which travel
// together everywhere and are what an artefact's authored names resolve
// against: providers and targets are the namespaces a Definition's provider:
// and targets: resolve against, definitions and procedures the namespaces a
// Step's definition: and a nested invocation's procedure: resolve against.
func checkArtefact(a repository.LoadedArtefact, loaded repository.Loaded) []problem.Problem {
	switch {
	case a.Path == artefact.BuiltinShellProviderPath:
		return artefact.CheckBuiltinShellProvider()
	case a.Path == "hyper.yaml":
		return artefact.CheckRepositoryDeclaration(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "targets/"):
		return artefact.CheckTargetDeclaration(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "providers/"):
		return artefact.CheckManifest(a.Path, a.Root)
	case strings.HasPrefix(a.Path, "definitions/"):
		return artefact.CheckDefinition(a.Path, a.Root, loaded.Providers, loaded.Targets)
	case strings.HasPrefix(a.Path, "procedures/"):
		return artefact.CheckProcedure(a.Path, a.Root, loaded.Providers, loaded.Definitions, loaded.Targets, loaded.Procedures)
	}
	return nil
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
