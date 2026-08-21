package cli

import (
	"errors"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Trigger a Run's entry carries, read off the environment and the machine
// (§7, §12, issue #136). The corpus drives the two executors end to end; what
// is here is the readings a golden cannot separate — the fallbacks, and the
// absences.

// environment is a lookup over a stated set: a variable the map does not name
// is absent, and one it names to the empty string is present and says so, which
// is the distinction os.LookupEnv draws and this reader depends on.
func environment(set map[string]string) func(string) (string, bool) {
	return func(name string) (string, bool) {
		value, present := set[name]
		return value, present
	}
}

func answering(value string) func() (string, error) {
	return func() (string, error) { return value, nil }
}

func failing() (string, error) { return "", errors.New("nothing answered") }

func TestReadTrigger_LocalNamesTheUserAndTheMachine(t *testing.T) {
	trigger := readTrigger(environment(nil), answering("igor"), answering("thinkpad"))

	want := store.Trigger{
		Cause:    store.CauseManual,
		Executor: store.ExecutorLocal,
		Actor:    "igor",
		Host:     "thinkpad",
	}
	if trigger != want {
		t.Errorf("the Trigger is %+v, want %+v", trigger, want)
	}
}

// hyper never sleeps, never daemonises and never watches a clock (§10), so a
// Cadence firing is something only an executor can do: a local Run is `manual`
// whatever the environment says.
func TestReadTrigger_ALocalRunIsAlwaysManual(t *testing.T) {
	trigger := readTrigger(environment(map[string]string{
		"GITHUB_EVENT_NAME": "schedule",
	}), answering("igor"), answering("thinkpad"))

	if trigger.Cause != store.CauseManual || trigger.Executor != store.ExecutorLocal {
		t.Errorf("the Trigger is %s on %s, want a manual Run on the local executor", trigger.Cause, trigger.Executor)
	}
}

// A machine that could not name itself writes no `host` at all — the ordinary
// absence rule, and better than a constant hyper invented for a machine it
// knows nothing about (§7).
func TestReadTrigger_AMachineThatCannotNameItselfWritesNoHost(t *testing.T) {
	if host := readTrigger(environment(nil), answering("igor"), failing).Host; host != "" {
		t.Errorf("host is %q, want it absent", host)
	}
}

// `actor` is a member every Trigger carries, so a Run that could not name one
// still has an entry to write: the constant says hyper could not name a user,
// which is what happened, rather than losing the whole account of a Run to a
// container with no passwd entry (§7).
func TestReadTrigger_AnUnnamedActorIsTheConstantAndNotAnAbsence(t *testing.T) {
	if actor := readTrigger(environment(nil), failing, answering("thinkpad")).Actor; actor != unnamedActor {
		t.Errorf("actor is %q, want the stated %q", actor, unnamedActor)
	}
	if actor := readTrigger(environment(map[string]string{"GITHUB_ACTIONS": "true"}), failing, answering("thinkpad")).Actor; actor != unnamedActor {
		t.Errorf("the Actions actor is %q, want the stated %q", actor, unnamedActor)
	}
}

// The occasion on Actions, and the URL composed out of the four variables that
// name it. A dispatched workflow run is `manual` on that executor, which is why
// the cause and the executor are two fields and not one (§12).
func TestReadTrigger_TheActionsOccasion(t *testing.T) {
	set := map[string]string{
		"GITHUB_ACTIONS":     "true",
		"GITHUB_ACTOR":       "TheLoomLabs",
		"GITHUB_EVENT_NAME":  "workflow_dispatch",
		"GITHUB_RUN_ID":      "17392044981",
		"GITHUB_RUN_ATTEMPT": "2",
		"GITHUB_SERVER_URL":  "https://github.com",
		"GITHUB_REPOSITORY":  "TheLoomLabs/hyper",
	}
	trigger := readTrigger(environment(set), answering("igor"), answering("thinkpad"))

	want := store.Trigger{
		Cause:       store.CauseManual,
		Executor:    store.ExecutorGitHubActions,
		Actor:       "TheLoomLabs",
		ExecutorRun: "17392044981",
		Attempt:     2,
		JobURL:      "https://github.com/TheLoomLabs/hyper/actions/runs/17392044981/attempts/2",
	}
	if trigger != want {
		t.Errorf("the Trigger is %+v, want %+v", trigger, want)
	}

	set["GITHUB_EVENT_NAME"] = "schedule"
	if cause := readTrigger(environment(set), answering("igor"), answering("thinkpad")).Cause; cause != store.CauseCron {
		t.Errorf("a schedule event reads as %s, want %s", cause, store.CauseCron)
	}
}

// A runner is a machine nobody will look for again, so `host` is written
// nowhere on that executor (§7).
func TestReadTrigger_ARunnerWritesNoHost(t *testing.T) {
	trigger := readTrigger(environment(map[string]string{"GITHUB_ACTIONS": "true", "GITHUB_ACTOR": "TheLoomLabs"}),
		answering("igor"), answering("thinkpad"))

	if trigger.Host != "" {
		t.Errorf("host is %q on the Actions executor, want it absent", trigger.Host)
	}
}

// A URL missing any of its four parts is a link to somewhere else, which is
// worse than the absence the entry would otherwise carry (§7).
func TestReadTrigger_AnIncompleteJobURLIsNotWritten(t *testing.T) {
	for name, missing := range map[string]string{
		"no server":     "GITHUB_SERVER_URL",
		"no repository": "GITHUB_REPOSITORY",
		"no run id":     "GITHUB_RUN_ID",
		"no attempt":    "GITHUB_RUN_ATTEMPT",
	} {
		t.Run(name, func(t *testing.T) {
			set := map[string]string{
				"GITHUB_ACTIONS":     "true",
				"GITHUB_RUN_ID":      "17392044981",
				"GITHUB_RUN_ATTEMPT": "2",
				"GITHUB_SERVER_URL":  "https://github.com",
				"GITHUB_REPOSITORY":  "TheLoomLabs/hyper",
			}
			delete(set, missing)
			if url := readTrigger(environment(set), answering("igor"), answering("thinkpad")).JobURL; url != "" {
				t.Errorf("job_url is %q with %s unset, want it absent", url, missing)
			}
		})
	}
}

// The run attempt is a decimal counter the executor set, and a variable saying
// something else writes no `run_attempt` at all rather than a number hyper
// invented from it.
func TestReadTrigger_AnAttemptThatIsNotACounterIsNotWritten(t *testing.T) {
	for _, value := range []string{"", "two", "2x", "-1", " 2"} {
		set := map[string]string{"GITHUB_ACTIONS": "true", "GITHUB_RUN_ATTEMPT": value}
		if attempt := readTrigger(environment(set), answering("igor"), answering("thinkpad")).Attempt; attempt != 0 {
			t.Errorf("GITHUB_RUN_ATTEMPT=%q reads as %d, want it absent", value, attempt)
		}
	}
}
