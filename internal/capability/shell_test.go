package capability_test

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
)

// launcher is what these cases are handed in cli.Child's place: a child, with
// its name resolved against testdata/bin/ and against nothing else.
//
// The two launch decisions — the process group, and the SIGKILL a cancelled
// context sends it — are cli.Child's and are asserted beside it, where the
// binary's own value is. What these cases need is a process, and what they need
// of its name is that it is the fixture's: a case whose exit code came from the
// machine's own `false` would be asserting the machine.
//
// os/exec keeps the file and the word apart already — Path is what is executed
// and Args[0] is what the child is told it was invoked as — so this sets the
// first and leaves the second, which is what keeps `command` the argv the case
// wrote (§12).
func launcher(t *testing.T) capability.Exec {
	t.Helper()

	bin, err := filepath.Abs(filepath.Join("testdata", "bin"))
	if err != nil {
		t.Fatal(err)
	}
	return func(ctx context.Context, argv []string) *exec.Cmd {
		child := exec.CommandContext(ctx, argv[0], argv[1:]...)
		child.Path = filepath.Join(bin, argv[0])
		child.Err = nil
		return child
	}
}

// run performs one command with no deadline, in the directory and environment
// the case names, and answers the object and the error beside it.
func run(t *testing.T, argv []string, root string, environment capability.Environment) (capability.Object, error) {
	t.Helper()

	return capability.Command{Argv: argv}.Perform(t.Context(), launcher(t), root, environment)
}

// encoded is one response object as a machine reads it, which is the encoding
// every surface that carries one writes: compact, in §12's member order, with
// HTML escaping off.
func encoded(t *testing.T, object capability.Object) string {
	t.Helper()

	out, err := object.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// TestCommandPerform_TheResponseObject is §12's four members, asserted against
// real children: a real exit status, real bytes on both streams. Nothing about
// the object is written down by the test beyond what the fixture scripts were
// told to print, which is the only thing a fixture has any business supplying.
func TestCommandPerform_TheResponseObject(t *testing.T) {
	for _, c := range []struct {
		name string
		argv []string
		want string
	}{
		{
			name: "what a command printed and what it exited with",
			argv: []string{"say", "out", "err", "0"},
			want: `{"command":"[\"say\",\"out\",\"err\",\"0\"]","exit_code":0,"stdout":"out","stderr":"err"}`,
		},
		{
			name: "a non-zero exit is recorded and is not an error",
			argv: []string{"say", "", "not found", "3"},
			want: `{"command":"[\"say\",\"\",\"not found\",\"3\"]","exit_code":3,"stdout":"","stderr":"not found"}`,
		},
		{
			name: "a command that printed nothing carries both streams empty",
			argv: []string{"say", "", "", "0"},
			want: `{"command":"[\"say\",\"\",\"\",\"0\"]","exit_code":0,"stdout":"","stderr":""}`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			object, err := run(t, c.argv, t.TempDir(), capability.Inherited(nil, nil))
			if err != nil {
				t.Fatalf("the command answered an error beside its object: %v", err)
			}
			if got := encoded(t, object); got != c.want {
				t.Errorf("the response object is\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

// TestCommandPerform_ACommandThatCouldNotBeStarted is the no-answer case one
// Capability over: no such binary, so the object is `command` and nothing else
// — no exit code, no stdout, no stderr — and a `read` records the attempt with
// its exit_code gone quiet (§12, ADR-0050).
func TestCommandPerform_ACommandThatCouldNotBeStarted(t *testing.T) {
	object, err := run(t, []string{"no-such-binary", "--now"}, t.TempDir(), capability.Inherited(nil, nil))
	if err == nil {
		t.Error("a command that never started answered no error beside its object, and a reader is owed the reason")
	}

	const want = `{"command":"[\"no-such-binary\",\"--now\"]"}`
	if got := encoded(t, object); got != want {
		t.Errorf("the response object is\n%s\nwant\n%s", got, want)
	}
	for _, absent := range []string{capability.MemberExitCode, capability.MemberStdout, capability.MemberStderr} {
		if _, carried := object.Lookup(absent); carried {
			t.Errorf("a command that never started carries %s, and the three are absent together", absent)
		}
	}
}

// TestCommandText_TwoArgvSpellingsAreTwoNames is why `command` is JSON rather
// than a joining rule: it must be injective. `[echo, "a b"]` and `[echo, a, b]`
// are two commands, and a joining rule silently makes them one series that
// record-identity-collision could never catch, the two names being genuinely
// equal (§12).
func TestCommandText_TwoArgvSpellingsAreTwoNames(t *testing.T) {
	one := capability.Command{Argv: []string{"echo", "a b"}}.Text()
	two := capability.Command{Argv: []string{"echo", "a", "b"}}.Text()

	if one == two {
		t.Fatalf("both argvs are named %s, and they are two commands", one)
	}
	if want := `["echo","a b"]`; one != want {
		t.Errorf("the argv is named %s, want %s", one, want)
	}
	if want := `["echo","a","b"]`; two != want {
		t.Errorf("the argv is named %s, want %s", two, want)
	}
	if strings.Contains(one, "\n") {
		t.Errorf("the argv is named across more than one line: %q", one)
	}
}

// TestCommandPerform_NothingStandsBetweenTheArtefactAndTheProcess is ADR-0051:
// the argv is exec'd directly, so a pipe, a glob and an && reach the child as
// literal words and are not writable as shell syntax (§13).
func TestCommandPerform_NothingStandsBetweenTheArtefactAndTheProcess(t *testing.T) {
	argv := []string{"argv", "a|b", "*.yaml", "&&", "$HOME", ">out"}

	object, err := run(t, argv, t.TempDir(), capability.Inherited([]string{"HOME=/home/nobody"}, nil))
	if err != nil {
		t.Fatalf("the command answered an error beside its object: %v", err)
	}

	stdout, carried := object.Lookup(capability.MemberStdout)
	if !carried {
		t.Fatal("the command carried no stdout")
	}
	want := "a|b\n*.yaml\n&&\n$HOME\n>out\n"
	if stdout != want {
		t.Errorf("the words that reached the process are\n%q\nwant\n%q", stdout, want)
	}
}

// TestCommandPerform_StdoutIsTextAndIsNeverParsed is ADR-0052 arriving in the
// projection: a command that answers in JSON is recorded as the string it
// printed, and $.stdout.result.id is not a path — the grammar §12 closes has
// three productions and none of them reaches inside a scalar.
func TestCommandPerform_StdoutIsTextAndIsNeverParsed(t *testing.T) {
	object, err := run(t, []string{"json"}, t.TempDir(), capability.Inherited(nil, nil))
	if err != nil {
		t.Fatalf("the command answered an error beside its object: %v", err)
	}

	stdout, carried := object.Lookup(capability.MemberStdout)
	if !carried {
		t.Fatal("the command carried no stdout")
	}
	if want := `{"result":{"id":"abc","count":3}}`; stdout != want {
		t.Errorf("stdout is %#v, want the string %q", stdout, want)
	}
	if _, resolved := projection.Resolve("$.stdout.result", object); resolved {
		t.Error("$.stdout.result resolved against a shell response, and stdout is text rather than a mapping to descend")
	}
	if _, resolved := projection.Resolve("$.stdout", object); !resolved {
		t.Error("$.stdout resolved to nothing, and a shell projection reaches it and nothing finer")
	}
}

// TestCommandPerform_TheDirectoryAndStdinAreFixed is the two things no artefact
// chooses: the child stands in the directory it is handed — the repository
// root, so a laptop and a runner agree without a line saying so — and its stdin
// is empty (§3).
func TestCommandPerform_TheDirectoryAndStdinAreFixed(t *testing.T) {
	root := t.TempDir()
	// The directory as the kernel reports it, which is not always the one
	// the temp directory was named: a path through a symlink resolves, and
	// what the child prints is the resolution.
	resolved, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}

	where, err := run(t, []string{"cwd"}, root, capability.Inherited(nil, nil))
	if err != nil {
		t.Fatalf("the command answered an error beside its object: %v", err)
	}
	stdout, _ := where.Lookup(capability.MemberStdout)
	if got, want := stdout, resolved+"\n"; got != want {
		t.Errorf("the child stood in %q, want the directory it was handed, %q", got, want)
	}

	handed, err := run(t, []string{"stdin"}, root, capability.Inherited(nil, nil))
	if err != nil {
		t.Fatalf("the command answered an error beside its object: %v", err)
	}
	stdout, _ = handed.Lookup(capability.MemberStdout)
	if got, want := stdout, "end\n"; got != want {
		t.Errorf("the child read %q off standard input, want nothing at all", got)
	}
}

// TestInherited_ACredentialSlotIsWithheld is §11's own sentence: the child
// inherits the invoking environment with every credential-slot variable in the
// repository removed, which is what stops a command reading one out of the
// environment `hyper` was invoked in.
func TestInherited_ACredentialSlotIsWithheld(t *testing.T) {
	invoking := []string{"WATCH_TOKEN=hunter2", "TZ=UTC", "PAID_KEY=k"}
	environment := capability.Inherited(invoking, []string{"WATCH_TOKEN", "PAID_KEY", "NEVER_SET"})

	for _, c := range []struct {
		variable, want string
	}{
		{"WATCH_TOKEN", "<unset>"},
		{"PAID_KEY", "<unset>"},
		{"TZ", "UTC"},
	} {
		t.Run(c.variable, func(t *testing.T) {
			object, err := run(t, []string{"variable", c.variable}, t.TempDir(), environment)
			if err != nil {
				t.Fatalf("the command answered an error beside its object: %v", err)
			}
			stdout, _ := object.Lookup(capability.MemberStdout)
			if got, want := stdout, c.want+"\n"; got != want {
				t.Errorf("the child read %s as %q, want %q", c.variable, got, want)
			}
		})
	}
}

// TestInherited_ComposesAnEnvironmentRatherThanInheritingOne is the one thing
// the type exists for: os/exec reads a nil Env as *inherit the parent's*, which
// would put every variable `hyper` was invoked with — credential slots included
// — into a process it cannot describe. An environment composed out of nothing
// is therefore *empty* and not *absent*, and the zero value is neither.
func TestInherited_ComposesAnEnvironmentRatherThanInheritingOne(t *testing.T) {
	if !capability.Inherited(nil, nil).Composed() {
		t.Error("an environment composed from nothing reads as uncomposed, which os/exec would answer with the parent's own")
	}
	if (capability.Environment{}).Composed() {
		t.Error("the zero Environment reads as composed, and nothing composed it")
	}

	object, err := run(t, []string{"variable", "TZ"}, t.TempDir(), capability.Environment{})
	if err == nil {
		t.Error("a child was started under an environment nothing composed")
	}
	if len(object) != 1 {
		t.Errorf("the response object is %s, want the command alone — no child ran", encoded(t, object))
	}
}

// TestInherited_ReadsAVariableByItsName is the one thing the composition has to
// get right beside the removal: a line that names no value is not a variable,
// and a value carrying an `=` is one value and not two.
func TestInherited_ReadsAVariableByItsName(t *testing.T) {
	environment := capability.Inherited([]string{"A=1", "malformed", "B=x=y", "C="}, []string{"A"})

	for _, c := range []struct{ variable, want string }{
		{"A", "<unset>"},
		{"B", "x=y"},
		{"C", ""},
		{"malformed", "<unset>"},
	} {
		t.Run(c.variable, func(t *testing.T) {
			object, err := run(t, []string{"variable", c.variable}, t.TempDir(), environment)
			if err != nil {
				t.Fatalf("the command answered an error beside its object: %v", err)
			}
			stdout, _ := object.Lookup(capability.MemberStdout)
			if got, want := stdout, c.want+"\n"; got != want {
				t.Errorf("the child read %s as %q, want %q", c.variable, got, want)
			}
		})
	}
}

// TestCommandPerform_ADeadlineIsTheOneErrorBesideTheObject: a command the
// context stopped is `hyper` stopping rather than the command answering, and it
// is the one error past a start that failed a caller reads (§6, ADR-0050).
// Nothing is recorded for a member that reached it. What the deadline does to
// the child's process group, and to the children the child started, is
// cli.Child's and is asserted beside it.
func TestCommandPerform_ADeadlineIsTheOneErrorBesideTheObject(t *testing.T) {
	// A child that outlives its deadline by a wide margin, so that what the
	// case reads is the deadline and never a race with a fast exit. It is
	// looked up rather than spelled as a path: what is under test is the
	// bound, and a case failing because /bin/sleep sat somewhere else would
	// be reporting the wrong thing.
	sleeping, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("this platform has no sleep to launch (%v); the deadline cannot be driven here", err)
	}

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	command := capability.Command{Argv: []string{sleeping, "30"}}
	object, err := command.Perform(ctx, direct, t.TempDir(), capability.Inherited(nil, nil))

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("a command that reached its deadline answered %v, want the deadline itself", err)
	}
	if _, carried := object.Lookup(capability.MemberCommand); !carried {
		t.Error("the object carries no command, and a call that answered nothing is still a call that was made")
	}
	if len(object) != 1 {
		t.Errorf("the response object is %s, want the command alone — nothing is recorded for a member that reached the deadline", encoded(t, object))
	}
}

// direct is the child an argv already naming a path needs: no resolution, and
// none of cli.Child's launch decisions, which are asserted where they are made.
func direct(ctx context.Context, argv []string) *exec.Cmd {
	return exec.CommandContext(ctx, argv[0], argv[1:]...)
}
