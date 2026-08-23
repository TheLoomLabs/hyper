package revision

import (
	"fmt"
	"strings"
)

// The code branch's objects as a reader that never fetches sees them (§8,
// ADR-0071, issue #164).
//
// A review's range opens at a revision the Store named, and what stands between
// that name and the bytes is whether this clone holds the object at all. It is
// one question asked of two forms — a blob id a Run recorded in Provenance, and
// the file one commit held at a path — and both are git's own object names, so
// one reader answers both and one absence comes back from either.
//
// **The absence is answered and never errored**, which is the rule ArtefactsAt
// already states one file over: a Run recorded on a runner names an object a
// shallow clone was never given, and *not held* is an ordinary fact about the
// clone rather than the world resisting. What a review does with it is render
// `not-in-clone` and mark nothing (§8).

// AtPath names one file inside a commit's tree, in git's own notation. It is
// the object name a range opens at where the artefact carries no revision of
// its own: the file's blob under the `repo_revision` the supplying Run recorded
// (§8, ADR-0067).
//
// It is spelled here rather than at the call site so that the one place that
// composes a commit and a path is the package that reads the object — a caller
// building `commit + ":" + path` is a caller that can build it wrong, and git's
// answer to a name that means something else is a blob id like any other.
func AtPath(commit, path string) string { return commit + ":" + path }

// Held answers the blob id an object name resolves to and whether this clone
// holds it.
//
// It answers false for an object the clone does not hold, for a name that
// resolves to nothing at all, and for one that resolves to something that is
// not a blob: what the caller asked for is a file's bytes, and a commit
// standing where a blob was named is an object it cannot read those from.
//
// **It never fetches.** Every git subprocess this package starts runs with lazy
// fetching off (environment, git.NoLazyFetch), so on a partial clone a promisor
// object that would need fetching answers *not held* like any other absent one
// rather than reaching a remote nobody asked it to (§8, ADR-0071).
//
// The name goes to git NUL-delimited and the answer is read as one record,
// which is what makes a path holding any byte at all — a space, a newline —
// answerable rather than a parse this reader gets wrong. One object is asked
// about per call, so the record is the whole answer.
func Held(repoRoot, object string) (string, bool, error) {
	if object == "" {
		return "", false, nil
	}
	answer, err := repository(repoRoot).with([]byte(object+"\x00"), "cat-file", "--batch-check", "-z")
	if err != nil {
		return "", false, err
	}

	record := strings.TrimSuffix(string(answer), "\n")
	if strings.HasSuffix(record, " missing") {
		// git's own word for an object it cannot produce, the name
		// echoed back ahead of it. It covers every way this call comes
		// back empty at once — an object the clone does not hold, a
		// commit it does not hold, a path that commit's tree does not
		// carry — and none of them is a failure.
		return "", false, nil
	}
	fields := strings.Fields(record)
	if len(fields) != 3 {
		return "", false, fmt.Errorf("git cat-file wrote %q for %s, which is neither <object> <type> <size> nor a missing object", record, object)
	}
	if fields[1] != "blob" {
		return "", false, nil
	}
	return fields[0], true, nil
}
