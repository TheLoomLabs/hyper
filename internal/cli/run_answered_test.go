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

// **A `shell` Step's `answered` is never written empty, and the `command` in it
// is what keeps it from being** (§7, §12, issue #156).
//
// The key's presence is the fact that something other than the ordinary answer
// decided this Step, and under this Capability the case where that matters most
// is the one with the least in it: a child that could not be started at all has
// no exit code, no stdout and no stderr, so `command` is the whole of the block.
// Were it left to the identity set beside it the entry would encode to
// `{}`, which the encoding suppresses outright — and the fact would
// vanish exactly where it is least ordinary, on a `destroy` besides, which
// projects nothing and declares no identity anywhere in the entry.
//
// It is held over the corpus rather than case by case for the reason above: a
// golden says what one case wrote, and what this asserts is that no case can
// write the empty one.
func TestAnswered_NoShellStepAnywhereWritesItEmpty(t *testing.T) {
	commands, neverStarted := 0, 0

	walkTestdata(t, "store.golden", func(dir string) {
		for path, step := range stepFilesIn(t, dir) {
			entries, carried := step["answered"].([]any)
			if !carried {
				continue
			}
			if len(entries) == 0 {
				t.Errorf("%s carries an empty answered; a list with nothing in it is suppressed by the encoding and says nothing at all", path)
				continue
			}
			for _, held := range entries {
				answered, _ := held.(map[string]any)
				// A `member` alone does not save an entry: it
				// would say which member and nothing about what
				// it was told (§7, ADR-0126).
				if len(answered) == 0 || (len(answered) == 1 && answered["member"] != nil) {
					t.Errorf("%s carries an answer naming no host and no command; the fact something other than the ordinary answer reached this member would say nothing at all", path)
					continue
				}
				command, named := answered["command"].(string)
				if !named {
					continue
				}
				if command == "" {
					t.Errorf("%s names an empty command; the argv as run is what a shell answer is written from (§12)", path)
				}
				commands++
				if _, exited := answered["exit_code"]; !exited {
					neverStarted++
				}
			}
		}
	})

	if commands == 0 {
		t.Error("no branch golden in the corpus holds a shell answer; the rule was held over nothing")
	}
	// The shape the rule is bought for: `command` alone, with the three
	// members a child that never started leaves absent together.
	if neverStarted == 0 {
		t.Error("no branch golden in the corpus holds a shell answer for a child that never started; the case the key exists for is driven by nothing")
	}
}
