package run

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Secret sink: the one route by which a value a Manifest declared `secret:`
// leaves `hyper` at all (§9, ADR-0007, ADR-0148, issue #270).
//
// **It is a directory and not a file, and one file under it holds one value.**
// What a Run can produce is not one secret: `check` permits two secret-producing
// Steps in one Procedure, a Step that expands produces one value per member, and
// an Operation may declare more than one `secret:` field — so the addressable
// thing is the triple *Step, Record, field*, and the shape that expresses it
// with no key to design and no parser to run is the filesystem's own (ADR-0148).
//
// **It is written once and never read back.** No Run reads it, nothing here
// parses it, and it never reaches the Store — which is the whole point (§7,
// ADR-0007, ADR-0011). So the constraint is legibility rather than
// round-tripping: what a wrapper does with it is `cat` one path, and what makes
// two secrets unambiguous is that they are two paths.
//
// **The mode is the sink's own guarantee**: the directory `0700` and every file
// under it `0600`, which is what §9 has stated since it was written and what
// nothing wrote until this file (ADR-0146).

// secretSink is where this Run's secrets go: the directory the invocation named,
// already absolute, and "" where none was named.
//
// The path arrives resolved because this package reaches no process fact of its
// own — a relative path resolved here would be resolved against the process's
// working directory, which is the one read Request exists to keep out (run.go).
type secretSink struct{ root string }

// The two modes §9 states, and the only two nodes a sink has. They are named
// rather than spelled at each call because they are one promise made in four
// places, and because the `Chmod` beside each `Mkdir` has to be the same number
// as the `Mkdir` for the promise to be worth making (create, write).
const (
	sinkDirMode  = 0o700
	sinkFileMode = 0o600
)

// named says the invocation supplied a sink. It is what the §6 gate reads
// beside *does this Run reach a Step whose Operation declares secret output*,
// the two together being `secret-sink-absent` (gates.go).
func (s secretSink) named() bool { return s.root != "" }

// create makes the sink directory, `0700`, and answers what stopped it.
//
// **It must not already be there, and that is the rule rather than a
// precaution.** A directory this Run did not make may hold an earlier Run's
// files, and a file this Run did not write is one an operator reading the sink
// would take for this Run's — which is the stale read the empty-file shape was
// refused for (ADR-0146, ADR-0148). `os.Mkdir` is the test as well as the act,
// so there is no window between asking and answering.
//
// The parent must exist: `hyper` makes the sink and not the directory it sits
// in. A path whose parent is missing, or one whose parent this process may not
// write, is the same fault and is reported as the operating system worded it.
//
// **The mode is set rather than requested.** `os.Mkdir` and `os.WriteFile` take
// a mode the process umask then subtracts from, so a `hyper` running under
// `umask 0700` would create a sink its own operator cannot read — and §9
// promises `0700` and `0600` rather than *at most* them. The `Chmod` is what
// makes that a promise: the sink is not a file the operator's environment gets
// a say in, and a golden asserting the bits would otherwise be asserting the
// umask of whoever ran the suite.
//
// It is called at §6's gate — before Step 1 and before any effect on the world —
// so that a sink that cannot be made stops a Run that has not yet done
// anything, rather than a Run that has already mutated three Assets and has
// nowhere to put the fourth's credential (§6, gates.go).
//
// A Run that reaches no Step declaring secret output never calls it, so a sink
// named against a Procedure that produces none leaves no empty directory behind.
func (s secretSink) create() error {
	if !s.named() {
		return nil
	}
	if err := os.Mkdir(s.root, sinkDirMode); err != nil {
		return fmt.Errorf("the secret sink could not be created: %w", err)
	}
	if err := os.Chmod(s.root, sinkDirMode); err != nil {
		return fmt.Errorf("the secret sink could not be created: %w", err)
	}
	return nil
}

// write puts one Record's secrets into the sink: one file per declared-secret
// field, at `<nnnn>/<name>/<field>` under the root, each `0600`.
//
// **The three segments are the triple and nothing is folded out of it.**
// `<nnnn>` is the Step's position in the Run's written order — §12's own
// spelling, and the Step name that is unique by construction where an authored
// id is not, two invocations of one Procedure giving two Steps one id
// (sequence.go, store.StepNumber). `<name>` is the Record's name, which is the
// segment `hyper records --name` takes and the third of §12's own
// `records/<target>/<definition>/<name>/`; the Target and the Definition are
// not repeated because a Step binds one of each, so a Record is named by its
// name alone within one Step. `<field>` is the field the Manifest declared.
//
// The two hostile segments are encoded exactly as the Store encodes them, by
// the Store's own function: a name that reaches one grammar as `Ü ber` reaches
// the other the same way, so `hyper records` and `cat` name one Record
// (store.EncodeSegment, §12).
//
// **The file holds the value and nothing `hyper` added** — no trailing newline,
// no quoting, no wrapper. A newline `hyper` invented is a byte the endpoint
// never issued, and a wrapper that did not strip it would send a credential
// that is not the credential. `$(cat …)` strips one either way, so the shape
// that is right for a reader that strips is also the only one right for a
// reader that does not.
//
// A value that is not a string is written as the JSON it is, which is what a
// projected value reads as everywhere else on a surface (projection.Text, §9).
//
// **A secret in hand with no sink to put it in is a fault and never a silence.**
// The gate refuses such a Run before Step 1, so nothing can reach here that way;
// what would reach it is a caller that assembled a Request some other way, and
// the answer to that is the error rather than the discard this whole record
// exists to end (ADR-0146).
func (s secretSink) write(step int, name string, secrets map[string]string) error {
	if len(secrets) == 0 {
		return nil
	}
	if !s.named() {
		return fmt.Errorf("step %d projected a declared-secret field of %s and this Run holds no Secret sink to write it to", step, name)
	}

	// The two directories the triple's first two segments are, named
	// rather than walked: a `MkdirAll` would make them and leave their
	// modes to the umask, and the sink's whole guarantee is about the tree
	// and not only the leaves.
	step0 := filepath.Join(s.root, store.StepNumber(step))
	dir := filepath.Join(step0, store.EncodeSegment(name))
	for _, made := range []string{step0, dir} {
		if err := os.Mkdir(made, sinkDirMode); err != nil && !os.IsExist(err) {
			return sinkUnwritten(err)
		}
		// The mode is set rather than requested, and set on a directory
		// an earlier member of this Step already made too: `os.Mkdir`'s
		// argument is what the umask subtracts from, and §9 promises
		// `0700` rather than *at most* `0700`.
		if err := os.Chmod(made, sinkDirMode); err != nil {
			return sinkUnwritten(err)
		}
	}

	for field, value := range secrets {
		path := filepath.Join(dir, store.EncodeSegment(field))
		if err := os.WriteFile(path, []byte(value), sinkFileMode); err != nil {
			return sinkUnwritten(err)
		}
		if err := os.Chmod(path, sinkFileMode); err != nil {
			return sinkUnwritten(err)
		}
	}
	return nil
}

// sinkUnwritten is the one sentence every failure of a sink write wears. It is
// one function because the four calls above are one act — *this value did not
// reach the sink* — and four spellings of that would be four sentences a reader
// has to tell apart for no reason.
func sinkUnwritten(err error) error {
	return fmt.Errorf("the secret sink could not be written: %w", err)
}
