package revision

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

// Object is one git object as this reader answers it: the blob id the name
// resolved to, and the bytes it holds.
//
// The two come back together because one read answers both and a caller wanting
// one always wants the other: a review names the blob on its header's range
// line and reads the bytes to mark the lines that moved, and a second call for
// the second half would be one subprocess spent re-asking a question already
// answered (§8).
type Object struct {
	Blob  string
	Bytes []byte
}

// Held answers the object an object name resolves to and whether this clone
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
// about per call, so the record is the whole answer, and the content is read
// by the size git states rather than by scanning for a separator: an artefact
// may hold any byte at all, a newline among them.
func Held(repoRoot, object string) (Object, bool, error) {
	if object == "" {
		return Object{}, false, nil
	}
	answer, err := repository(repoRoot).with([]byte(object+"\x00"), "cat-file", "--batch", "-z")
	if err != nil {
		return Object{}, false, err
	}

	record, err := readBatchRecord(answer, object)
	if err != nil {
		return Object{}, false, err
	}
	if record.missing() || record.kind() != "blob" {
		// An object git cannot produce, and one that is there and is not
		// a file: what the caller asked for is a file's bytes, and both
		// of those are an object it cannot read them from. Neither is a
		// failure — a Run recorded on a runner names an object a shallow
		// clone was never given (ADR-0071).
		return Object{}, false, nil
	}
	return Object{Blob: record.object(), Bytes: record.content}, true, nil
}
