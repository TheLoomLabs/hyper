package workflow

import "strings"

// `hyper-*.yml` under `.github/workflows/` is the namespace `project` owns, and
// this file is the whole of what that means: where one Procedure's file sits,
// and which of the files already there are `project`'s to speak for (§10).
//
// It is here rather than at the one command that writes a file because the
// namespace is the generator's own. A workflow's path carries the Procedure's
// name verbatim, which is the same fact the file's `name:` carries, and two
// spellings of it would be two answers to *which file is this Procedure's* —
// the one question a whole-file, always-overwriting generation cannot afford to
// have two of.
//
// **A hand-written workflow whose name begins `hyper-` will be removed by the
// next `project`.** That is the cost of a namespace rather than a bug in one,
// and one `git mv` escapes it (§10).

const (
	// Dir is where every generated workflow sits, relative to the
	// repository root and spelled in slashes: it is a path inside a git
	// repository before it is a path on a disk, and the executor reads it
	// under that name on every platform.
	Dir = ".github/workflows"
	// prefix opens the name of every file in the namespace, and suffix
	// closes it. `.yml` and not `.yaml`: one spelling, so a Procedure has
	// one file and not two.
	prefix = "hyper-"
	suffix = ".yml"
)

// Path is where procedure's workflow sits, relative to the repository root —
// the Procedure's name verbatim between the two fixed halves, so run history in
// the executor's own UI is per-Procedure and nothing is ambiguous about which
// expression fired (§10).
func Path(procedure string) string {
	return Dir + "/" + prefix + procedure + suffix
}

// ProcedureOf is the Procedure name a path in the namespace carries, and false
// where the path is not in the namespace at all.
//
// It is Path read backwards and is exact about it: a file directly under Dir
// whose name opens with `hyper-` and closes with `.yml`, and nothing else. A
// `hyper-.yml` names the empty Procedure, which no Procedure is, and is in the
// namespace like any other — the file is `project`'s to remove, and what makes
// it so is where it sits rather than whether the name it carries resolves.
func ProcedureOf(path string) (string, bool) {
	name, inside := strings.CutPrefix(path, Dir+"/")
	if !inside || strings.Contains(name, "/") {
		return "", false
	}
	name, opens := strings.CutPrefix(name, prefix)
	if !opens {
		return "", false
	}
	name, closes := strings.CutSuffix(name, suffix)
	if !closes {
		return "", false
	}
	return name, true
}
