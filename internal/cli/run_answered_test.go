package cli_test

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
)

// **`answered` is effectful-only, and this holds it over the whole corpus** (§6,
// §7, ADR-0010, issue #148).
//
// Its presence on a Step file is the fact that something other than the
// ordinary answer decided that Step. A `read`'s status *is* the answer and
// belongs in the Record wherever its Manifest projected it; a Journal copy would
// add only a claim that `hyper` thought a `503` was untoward, which on a `read`
// it does not.
//
// It is asserted here rather than case by case because the claim is *nowhere*.
// A per-case golden says what that case wrote, and a hundred goldens that
// happen not to carry the key say nothing about the hundred-and-first — where
// this reads every Step file the corpus has ever landed and holds the rule over
// all of them at once. The corpus's `read` Steps include every shape a status
// can take: a host that answered nothing, a command that exited `1`, an
// exhausted retry, a `503` that was not retried.

// stepFileBlock matches one Step file in a branch golden: the header line the
// harness writes, and the path within it. Only Step files are read — a Record
// version and the two entry files carry no such key and never could.
var stepFileBlock = regexp.MustCompile(`(?m)^=== (journal/\d{4}/\d{2}/\d{2}/[^/]+/steps/\d+\.json) \(\d+ bytes\)\n`)

// TestAnswered_NoReadStepAnywhereWritesIt walks every branch golden the corpus
// holds and reads back each Step file it finds.
//
// It fails on a vacuous pass in both directions: a corpus with no `read` Step
// file would be asserting nothing, and one with no effectful Step carrying the
// key would be asserting a rule against a key nothing writes.
func TestAnswered_NoReadStepAnywhereWritesIt(t *testing.T) {
	reads, answered := 0, 0

	walkTestdata(t, "store.golden", func(dir string) {
		for path, step := range stepFilesIn(t, dir) {
			kind, _ := step["kind"].(string)
			_, carried := step["answered"]

			switch {
			case kind == "read" && carried:
				t.Errorf("%s carries answered on a read Step: a read's status is the answer and belongs in the Record (§7, ADR-0010)", path)
			case kind == "read":
				reads++
			case carried:
				answered++
			}
		}
	})

	if reads == 0 {
		t.Error("no branch golden in the corpus holds a read Step file; the rule was held over nothing")
	}
	if answered == 0 {
		t.Error("no branch golden in the corpus holds an effectful Step file carrying answered; the rule was held over a key nothing writes")
	}
}

// stepFilesIn is every Step file a case's branch goldens hold, decoded, keyed by
// where the reader would go to find it: the golden and the path within it, which
// is what a failure names. The path alone would not do — a case's two branch
// goldens hold the same paths, and what fails is one of them.
//
// It reads both `store.golden` and `remote.golden` where the case has them: a
// file that reached the remote is the same file, and a rule about what the Store
// holds holds wherever the Store is.
func stepFilesIn(t *testing.T, dir string) map[string]map[string]any {
	t.Helper()

	files := map[string]map[string]any{}
	for _, golden := range []string{"store.golden", "remote.golden"} {
		rendered := readFileAt(dir + "/" + golden)
		for _, found := range stepFileBlock.FindAllStringSubmatchIndex(rendered, -1) {
			path := rendered[found[2]:found[3]]
			// The block runs from the end of its header line to the
			// start of the next header, which is the harness's own
			// rendering: the bytes below a header are verbatim, and
			// the length on it is what makes them parsable.
			body := rendered[found[1]:]
			if next := strings.Index(body, "\n=== "); next >= 0 {
				body = body[:next+1]
			}

			var decoded map[string]any
			if err := json.Unmarshal([]byte(body), &decoded); err != nil {
				t.Fatalf("%s/%s: %s does not decode: %v", dir, golden, path, err)
			}
			files[dir+"/"+golden+" "+path] = decoded
		}
	}
	return files
}
