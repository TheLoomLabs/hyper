package cli

import (
	"fmt"
	"strconv"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// The Trigger a Run's entry carries: what caused the Run, which executor it
// happened on, and which occasion on that executor (§7, §12, issue #136).
//
// **It is filled by reading the environment and branching on nothing found in
// it.** Recording where a Run happened is a fact about the occasion; behaving
// differently on one would be the authority axis §5 does not have, and no rule
// anywhere in the specification reads either field back. That is why this file
// holds a reader and no decision: everything it produces lands in `run.json`
// and is read by a human or by §8's `runs` table.
//
// It lives here rather than in internal/run for the reason that package states
// about itself: the engine reaches no process fact of its own, so what the
// process says about the occasion is read at the surface and handed down.

// The environment GitHub Actions sets, and the whole of what hyper reads from
// it. They are the documented variables of that executor rather than a
// heuristic: `GITHUB_ACTIONS` is what says the Run is on a runner at all, and
// the rest are the occasion §7 says an entry carries so that a Run id and the
// job that emitted it are relatable.
const (
	envActions    = "GITHUB_ACTIONS"
	envActor      = "GITHUB_ACTOR"
	envEventName  = "GITHUB_EVENT_NAME"
	envRunID      = "GITHUB_RUN_ID"
	envRunAttempt = "GITHUB_RUN_ATTEMPT"
	envServerURL  = "GITHUB_SERVER_URL"
	envRepository = "GITHUB_REPOSITORY"
)

// scheduleEvent is the Actions event name a Cadence fires under, and the one
// value that makes a Run `cron`. Everything else on that executor is `manual` —
// a dispatched workflow run included, which is why the cause and the executor
// are two fields and not one (§12).
const scheduleEvent = "schedule"

// unnamedActor is what a Trigger's `actor` says where nothing could name one:
// no Actions actor, and a passwd database that answered nothing.
//
// It is a constant rather than an absence because `actor` is a member every
// Trigger carries (§7) — a Run that could not name one still has an entry to
// write, and failing to write it over a value nobody can supply would lose the
// whole account of a Run to a container with no passwd entry. It is not a claim
// about a person: it says hyper could not name one, which is what happened.
const unnamedActor = "unknown"

// readTrigger fills the Trigger from the environment and the machine.
//
// The two arms are §12's two executors and they are told apart by one variable.
// On Actions the actor, the cause and the occasion are all the executor's own
// facts, and `host` is written nowhere: a runner is a machine nobody will look
// for again. Locally the cause is always `manual` — hyper never sleeps, never
// daemonises and never watches a clock (§10), so a Cadence firing is something
// only an executor can do — and `host` is written beside the actor, which is
// what §8's header renders `igor@thinkpad` from.
func readTrigger(lookupenv func(string) (string, bool), user, hostname func() (string, error)) store.Trigger {
	if actions, _ := lookupenv(envActions); actions == "true" {
		event, _ := lookupenv(envEventName)
		cause := store.CauseManual
		if event == scheduleEvent {
			cause = store.CauseCron
		}
		actor, _ := lookupenv(envActor)
		executorRun, _ := lookupenv(envRunID)
		attempt, _ := lookupenv(envRunAttempt)
		if actor == "" {
			actor = unnamedActor
		}
		return store.Trigger{
			Cause:       cause,
			Executor:    store.ExecutorGitHubActions,
			Actor:       actor,
			ExecutorRun: executorRun,
			Attempt:     positiveInteger(attempt),
			JobURL:      jobURL(lookupenv),
		}
	}

	trigger := store.Trigger{Cause: store.CauseManual, Executor: store.ExecutorLocal, Actor: named(user)}
	if host, err := hostname(); err == nil {
		trigger.Host = host
	}
	return trigger
}

// named is what a read of the process answered, or the constant where it
// answered nothing.
func named(read func() (string, error)) string {
	value, err := read()
	if err != nil || value == "" {
		return unnamedActor
	}
	return value
}

// jobURL composes the URL of the job out of the four variables that name it.
// It is composed rather than read because Actions sets no variable carrying it,
// and it is written only where every part of it is there: a URL missing its
// repository is a link to somewhere else, which is worse than the absence the
// entry would otherwise carry (§7).
func jobURL(lookupenv func(string) (string, bool)) string {
	server, hasServer := lookupenv(envServerURL)
	repository, hasRepository := lookupenv(envRepository)
	run, hasRun := lookupenv(envRunID)
	attempt, hasAttempt := lookupenv(envRunAttempt)
	if !hasServer || !hasRepository || !hasRun || !hasAttempt {
		return ""
	}
	if server == "" || repository == "" || run == "" || attempt == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s/actions/runs/%s/attempts/%s", server, repository, run, attempt)
}

// positiveInteger reads a decimal counter the executor set, and answers 0 where
// it says something that is not one. 0 is the absence: the Store writes no
// `run_attempt` for it, which is the honest answer about a variable hyper could
// not read (§7).
func positiveInteger(text string) int {
	value, err := strconv.Atoi(text)
	if err != nil || value < 1 {
		return 0
	}
	return value
}
