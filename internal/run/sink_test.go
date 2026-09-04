package run

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The Secret sink's own three claims, each of which the golden corpus asserts
// the *outcome* of and none of which it can state as a rule (§9, ADR-0148,
// issue #270).
//
// A golden holds one Run's tree; these hold what any Run's tree is built from —
// the mode on the two kinds of node, the refusal to write into a directory this
// Run did not make, and the encoding that keeps a Record's name the same name
// under `cat` as under `hyper records`. Each is a pure function of the sink's
// own inputs, where a corpus case would need a fixture repository, a served
// host and a checked-in Store per claim to say the same three things.

// TestSink_CreatesTheDirectoryItselfAndRefusesOneAlreadyThere holds the rule the
// directory shape rests on: every file under a sink is **this Run's**.
//
// A path that is already there may hold an earlier Run's files, and a file this
// Run did not write is one an operator reading the sink takes for this Run's —
// the stale read the empty-file shape was refused for (ADR-0146, ADR-0148).
func TestSink_CreatesTheDirectoryItselfAndRefusesOneAlreadyThere(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "secrets")

	if err := (secretSink{root: root}).create(); err != nil {
		t.Fatalf("create: %v", err)
	}
	if mode := modeOf(t, root); mode != 0o700 {
		t.Errorf("the sink directory is %04o, want 0700", mode)
	}

	if err := (secretSink{root: root}).create(); err == nil {
		t.Error("a sink that is already there was created a second time; every file under a sink is this Run's")
	}

	// A path that is a *file* is the same fault under the same rule: what
	// the sink names has to be the sink, and this Run has to have made it.
	held := filepath.Join(base, "already-a-file")
	if err := os.WriteFile(held, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := (secretSink{root: held}).create(); err == nil {
		t.Error("a sink naming a file that is already there was accepted")
	}

	// A Run that named none makes nothing, which is what leaves a Run with
	// no secret-producing Step no directory behind (gates.go).
	if err := (secretSink{}).create(); err != nil {
		t.Errorf("a Run that named no sink: %v", err)
	}
}

// TestSink_WritesOneFilePerFieldAtTheTriple holds the path grammar and the
// bytes: `<nnnn>/<name>/<field>`, `0600`, and the value with nothing `hyper`
// added to it.
//
// The two hostile segments go through the Store's own encoder, so a Record whose
// name is not a filename is one name under `cat` and under `hyper records`
// (store.EncodeSegment, §12).
func TestSink_WritesOneFilePerFieldAtTheTriple(t *testing.T) {
	root := filepath.Join(t.TempDir(), "secrets")
	held := secretSink{root: root}
	if err := held.create(); err != nil {
		t.Fatal(err)
	}

	if err := held.write(1, "status.hyper.dev", map[string]string{"token": "sk-live-9f3a2b81c0"}); err != nil {
		t.Fatalf("write: %v", err)
	}
	// A second Step, an expanded member whose name is not a filename, and
	// two declared fields on one Record — the three ways one Run produces
	// more than one secret, all at once (ADR-0148).
	if err := held.write(12, "Über/vm", map[string]string{"token": "second", "cookie": "third"}); err != nil {
		t.Fatalf("write: %v", err)
	}

	want := []string{
		"0001/status.hyper.dev/token",
		"0012/%C3%9Cber%2Fvm/cookie",
		"0012/%C3%9Cber%2Fvm/token",
	}
	if got := filesUnder(t, root); !slices.Equal(got, want) {
		t.Errorf("the sink holds\n got:  %v\n want: %v", got, want)
	}

	for path, value := range map[string]string{
		"0001/status.hyper.dev/token": "sk-live-9f3a2b81c0",
		"0012/%C3%9Cber%2Fvm/token":   "second",
		"0012/%C3%9Cber%2Fvm/cookie":  "third",
	} {
		read, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		if string(read) != value {
			t.Errorf("%s holds %q, want %q — the file is the value and nothing hyper added", path, read, value)
		}
		if mode := modeOf(t, filepath.Join(root, filepath.FromSlash(path))); mode != 0o600 {
			t.Errorf("%s is %04o, want 0600", path, mode)
		}
	}
	// The member directory carries the root's own mode, a secret being no
	// less readable through the directory holding it.
	if mode := modeOf(t, filepath.Join(root, "0001")); mode != 0o700 {
		t.Errorf("the step directory is %04o, want 0700", mode)
	}
}

// TestSink_ASecretWithNowhereToGoIsAFault holds the one thing this whole record
// exists to end: a value produced and discarded in silence.
//
// The §6 gate refuses such a Run before Step 1, so nothing reaches here that
// way; what would is a caller that assembled a Request some other way, and the
// answer to that is an error and not a shrug (ADR-0146).
func TestSink_ASecretWithNowhereToGoIsAFault(t *testing.T) {
	if err := (secretSink{}).write(1, "status.hyper.dev", map[string]string{"token": "sk-live"}); err == nil {
		t.Error("a secret was written to no sink at all and nothing said so")
	}
	// A Step that declared no secret field writes nothing and is no fault:
	// the sink is silent about a Run that produced none.
	if err := (secretSink{}).write(1, "status.hyper.dev", nil); err != nil {
		t.Errorf("a Step that produced no secret: %v", err)
	}
}

// modeOf is one node's permission bits.
func modeOf(t *testing.T, path string) fs.FileMode {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

// filesUnder is every file under root, as slash-separated paths relative to it
// and in path order.
func filesUnder(t *testing.T, root string) []string {
	t.Helper()

	var found []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		found = append(found, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(found)
	return found
}
