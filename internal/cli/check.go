// Package cli is the third seam #87's Implementation Decisions name: argument
// handling, path stats, the two renderings, ordering, filtering, exit codes,
// stream discipline. It owns no rule of its own — every problem it prints
// comes from the loader (internal/yamlsubset), the pin gate (internal/pin),
// and an artefact's own schema (internal/artefact).
package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/pin"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/yamlsubset"
)

// Exit codes §12 defines. check uses four of the seven: 0 clean, 1 problems
// found, 2 usage error, 77 a guardrail declined.
const (
	ExitClean    = 0
	ExitProblems = 1
	ExitUsage    = 2
	ExitRefused  = 77
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

	// The pin gate: every command compares itself against hyper.yaml's
	// version: pin before reading a second file, and Refuses on mismatch in
	// either direction (§9, §11, ADR-0020).
	hyperYAML := filepath.Join(repoRoot, "hyper.yaml")
	data, err := os.ReadFile(hyperYAML)
	present := err == nil
	if err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(stderr, "hyper check: %s\n", err)
		return ExitUsage
	}
	if result := pin.Check(present, data, binaryVersion); result.Refused {
		fmt.Fprintf(stderr, "refused: %s\n  %s\n", result.Code, result.Message)
		return ExitRefused
	}

	// check stats its path arguments before loading a single artefact
	// (§9): a path naming no file exits 2 and reports no problems at all.
	for _, p := range paths {
		if _, err := os.Stat(absPath(wd, p)); err != nil {
			fmt.Fprintf(stderr, "hyper check: %s: no such file or directory\n", p)
			return ExitUsage
		}
	}

	files, err := repository.ArtefactFiles(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper check: %s\n", err)
		return ExitUsage
	}

	// check loads every artefact before evaluating a single rule, in two
	// passes: parsing every file first is what lets a Definition's
	// provider: and targets: resolve against the whole repository's names
	// rather than only the files loaded before it in artefactDirs' own
	// order (issue #93). A failed load does not stop the pass: reading or
	// parsing one file stops every check after it for that file — never
	// for the repository (issue #88). An artefact hyper cannot even read is
	// judged the same way as one that will not parse: exactly one problem,
	// on its own line, and every other artefact is untouched.
	loadedFiles := make([]loadedFile, 0, len(files))
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(repoRoot, rel))
		if err != nil {
			loadedFiles = append(loadedFiles, loadedFile{rel: rel, problems: []problem.Problem{{
				File:      rel,
				Line:      1,
				Column:    1,
				ErrorCode: yamlsubset.ErrorCode,
				Message:   err.Error(),
			}}})
			continue
		}
		root, probs, ok := yamlsubset.Parse(rel, data)
		if ok && root != nil {
			probs = append(probs, yamlsubset.Violations(root, rel)...)
		}
		loadedFiles = append(loadedFiles, loadedFile{rel: rel, root: root, problems: probs, ok: ok})
	}

	providers := artefact.BuildProviderIndex(rootsUnder(loadedFiles, "providers/"))
	targets := artefact.BuildTargetIndex(rootsUnder(loadedFiles, "targets/"))
	definitions := artefact.BuildDefinitionIndex(rootsUnder(loadedFiles, "definitions/"), targets)
	procedures := artefact.BuildProcedureIndex(rootsUnder(loadedFiles, "procedures/"))

	var problems []problem.Problem
	for _, lf := range loadedFiles {
		problems = append(problems, lf.problems...)
		if !lf.ok {
			continue
		}
		problems = append(problems, checkArtefact(lf.rel, lf.root, providers, targets, definitions, procedures)...)
	}

	if len(paths) > 0 {
		problems = filterByPaths(problems, paths, repoRoot, wd)
	}
	problem.Sort(problems)

	var renderErr error
	if flags.json {
		renderErr = render.WriteJSON(stdout, problems)
	} else {
		renderErr = render.WriteTable(stdout, problems)
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

// loadedFile is one artefact file's own parse, kept between check's two
// passes: root is what the second pass's schema and resolution checks read,
// and problems already holds whatever the read itself or yamlsubset's own
// grammar found. ok is false where the read failed or the file will not
// parse at all — the one case the second pass skips entirely, on the same
// "loading a file is the first check" rule yamlsubset.Parse's own ok return
// states (§4). A file that read and parsed but supplied no document at all
// (an empty file) is ok with root nil, which every check below reads the
// same way schema.Check already does: a required key nothing supplied.
type loadedFile struct {
	rel      string
	root     *yaml.Node
	problems []problem.Problem
	ok       bool
}

// rootsUnder returns the root of every loaded file whose path starts with
// prefix and that parsed at all — the roots BuildProviderIndex and
// BuildTargetIndex read the repository's provider and Target namespaces off
// of (issue #93). A file that failed to parse contributes no root and
// therefore no name to either namespace, on ADR-0064's own rule.
func rootsUnder(files []loadedFile, prefix string) []*yaml.Node {
	var roots []*yaml.Node
	for _, lf := range files {
		if lf.ok && strings.HasPrefix(lf.rel, prefix) {
			roots = append(roots, lf.root)
		}
	}
	return roots
}

// checkArtefact runs one already-parsed artefact's own schema and the
// checks that read it against itself or against the repository: hyper.yaml,
// a file in targets/, a file in providers/, a file in definitions/ and,
// since issue #94, a file in procedures/ — the five artefacts this
// milestone's schema reaches. providers and targets are the repository-wide
// namespaces a Definition's provider: and targets: resolve against;
// definitions and procedures are the namespaces a Step's definition: and a
// nested invocation's procedure: resolve against.
func checkArtefact(rel string, root *yaml.Node, providers artefact.ProviderIndex, targets artefact.TargetIndex, definitions artefact.DefinitionIndex, procedures artefact.ProcedureIndex) []problem.Problem {
	switch {
	case rel == "hyper.yaml":
		return artefact.CheckRepositoryDeclaration(rel, root)
	case strings.HasPrefix(rel, "targets/"):
		return artefact.CheckTargetDeclaration(rel, root)
	case strings.HasPrefix(rel, "providers/"):
		return artefact.CheckManifest(rel, root)
	case strings.HasPrefix(rel, "definitions/"):
		return artefact.CheckDefinition(rel, root, providers, targets)
	case strings.HasPrefix(rel, "procedures/"):
		return artefact.CheckProcedure(rel, root, providers, definitions, procedures)
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
