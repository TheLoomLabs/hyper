package run_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/run"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Which lock a Run takes is decided from the Kinds `check` already computed,
// before any Step runs (§6, issue #138). These cases hold that decision over a
// repository loaded the way a Run loads one, because the whole claim is that
// the answer is readable off the artefacts and needs nothing from the world.

// TestLockMode_IsSharedWhereEveryStepIsARead is the monitoring cadence's case,
// and the reason there are two modes at all.
//
// It reads through a nested invocation as well: a Procedure invoking another
// runs as one Run, so what decides the lock is the Kinds of every Step the walk
// reaches and not the depth it reached them at (§6, issue #141).
func TestLockMode_IsSharedWhereEveryStepIsARead(t *testing.T) {
	loaded := loadFixture(t)

	for _, procedure := range []string{"watch-status", "watch-nested"} {
		if got := run.LockMode(loaded, procedure); got != store.Shared {
			t.Errorf("LockMode(%s) = %v, want Shared", procedure, got)
		}
	}
}

// TestLockMode_IsExclusiveWhereAnyStepIsEffectful is the other arm, over both
// effectful Kinds, over a Procedure whose effectful Step is not its first —
// *any* effectful Step, and never *the first one decides* — and over one whose
// only effectful Step is inside a Procedure it invokes.
func TestLockMode_IsExclusiveWhereAnyStepIsEffectful(t *testing.T) {
	loaded := loadFixture(t)

	for _, procedure := range []string{"publish-preview", "retire-preview", "read-then-publish", "watch-nested-effectful"} {
		if got := run.LockMode(loaded, procedure); got != store.Exclusive {
			t.Errorf("LockMode(%s) = %v, want Exclusive", procedure, got)
		}
	}
}

// TestLockMode_IsExclusiveWhereAStepsKindCannotBeRead is the conservative half.
// A Step whose binding does not resolve carries no Kind here; an invocation
// naming a Procedure that is not there, and one that is a cycle, leave Steps
// the walk never reached at all. A Run whose blast radius cannot be read is not
// a Run that may share the Store.
//
// None of them ever gets as far as its first Step: every one is `check`'s to
// refuse at Run start. The lock is taken before that, so what it does with them
// is a fact of its own.
func TestLockMode_IsExclusiveWhereAStepsKindCannotBeRead(t *testing.T) {
	loaded := loadFixture(t)

	for _, procedure := range []string{"watch-unresolvable", "watch-nested-absent", "watch-cyclic"} {
		if got := run.LockMode(loaded, procedure); got != store.Exclusive {
			t.Errorf("LockMode(%s) = %v, want Exclusive", procedure, got)
		}
	}
}

// TestLockMode_IsExclusiveForAProcedureThatIsNotThere is the answer to a name
// that resolves to nothing. It is unreachable from the CLI, which resolves the
// positional first, and it is the safe answer rather than the absent one:
// there is no Kind set to read, so there is nothing to share the Store on.
func TestLockMode_IsExclusiveForAProcedureThatIsNotThere(t *testing.T) {
	if got := run.LockMode(loadFixture(t), "no-such-procedure"); got != store.Exclusive {
		t.Errorf("LockMode(no-such-procedure) = %v, want Exclusive", got)
	}
}

// loadFixture writes the repository these cases read and loads it. It is
// written here rather than checked in because every fact it carries is one of
// these cases' own — one Provider with a Kind of each, and one Procedure per
// shape the decision has to answer for.
func loadFixture(t *testing.T) repository.Loaded {
	t.Helper()

	root := t.TempDir()
	for path, content := range map[string]string{
		"hyper.yaml": "" +
			"kind: repository-declaration\n" +
			"version: 1.4.0\n" +
			"retention: 90d\n",
		"targets/local.yaml": "" +
			"kind: target-declaration\n" +
			"target: local\n" +
			"class: local\n" +
			"kinds: [read, mutate, destroy]\n" +
			"capabilities: [http]\n" +
			"hosts: [status.hyper.dev]\n",
		"providers/uptime.yaml": "" +
			"kind: provider\n" +
			"provider: uptime\n" +
			"schema-version: 1\n" +
			"class: local\n" +
			"capabilities: [http]\n" +
			"operations:\n" +
			"  check_http:\n" +
			"    kind: read\n" +
			"    deadline: 10s\n" +
			"    http: {method: GET, host: \"{from-target}\", path: /}\n" +
			"    record: {identity: $.host, fields: {status: $.status}}\n" +
			"  raise_flag:\n" +
			"    kind: mutate\n" +
			"    repeatability: repeatable\n" +
			"    deadline: 10s\n" +
			"    http: {method: POST, host: \"{from-target}\", path: /flag}\n" +
			"    record: {identity: $.host, fields: {status: $.status}}\n" +
			"  lower_flag:\n" +
			"    kind: destroy\n" +
			"    repeatability: repeatable\n" +
			"    deadline: 10s\n" +
			"    http: {method: DELETE, host: \"{from-target}\", path: /flag}\n" +
			"    record: {identity: $.host, fields: {status: $.status}}\n",
		"definitions/uptime-check.yaml": "" +
			"kind: definition\n" +
			"definition: uptime-check\n" +
			"provider: uptime\n" +
			"kinds: [read]\n" +
			"targets: [local]\n",
		"definitions/uptime-flag.yaml": "" +
			"kind: definition\n" +
			"definition: uptime-flag\n" +
			"provider: uptime\n" +
			"kinds: [mutate, destroy]\n" +
			"targets: [local]\n",
		"procedures/watch-status.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-status\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: status, definition: uptime-check, operation: check_http, target: local}\n" +
			"  - {id: again, definition: uptime-check, operation: check_http, target: local}\n",
		"procedures/publish-preview.yaml": "" +
			"kind: procedure\n" +
			"procedure: publish-preview\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: raise, definition: uptime-flag, operation: raise_flag, target: local}\n",
		"procedures/retire-preview.yaml": "" +
			"kind: procedure\n" +
			"procedure: retire-preview\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: lower, definition: uptime-flag, operation: lower_flag, target: local, bound: 1}\n",
		"procedures/read-then-publish.yaml": "" +
			"kind: procedure\n" +
			"procedure: read-then-publish\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: status, definition: uptime-check, operation: check_http, target: local}\n" +
			"  - {id: raise, definition: uptime-flag, operation: raise_flag, target: local}\n",
		"procedures/watch-unresolvable.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-unresolvable\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: status, definition: no-such-definition, operation: check_http, target: local}\n",
		"procedures/watch-nested.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-nested\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: inner, procedure: watch-status}\n",
		"procedures/watch-nested-effectful.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-nested-effectful\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: status, definition: uptime-check, operation: check_http, target: local}\n" +
			"  - {id: inner, procedure: publish-preview}\n",
		"procedures/watch-nested-absent.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-nested-absent\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: inner, procedure: no-such-procedure}\n",
		"procedures/watch-cyclic.yaml": "" +
			"kind: procedure\n" +
			"procedure: watch-cyclic\n" +
			"targets: [local]\n" +
			"steps:\n" +
			"  - {id: status, definition: uptime-check, operation: check_http, target: local}\n" +
			"  - {id: again, procedure: watch-cyclic}\n",
	} {
		full := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := repository.Load(root)
	if err != nil {
		t.Fatalf("loading the fixture: %v", err)
	}
	return loaded
}
