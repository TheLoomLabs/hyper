package cli

import (
	"context"
	"os/exec"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Process is everything `hyper` reads from the process it is running in, as one
// value.
//
// It is issue #100's property at the grain a milestone of six reads needs:
// everything a command reads from the process is a parameter it is handed
// rather than a package it reaches for, which is what makes the whole dispatch
// exercisable without a subprocess. What travels beside this value rather than
// in it is the argv, which is what the dispatch decides on; the two streams,
// which a command writes rather than reads; and version.Facts, which the build
// stamped rather than the process holds.
//
// Six loose parameters threaded through repositoryCommand is where that
// property starts costing more than it buys, so the reads travel as one value:
// what a command may read is a type a reader opens rather than a parameter list
// they count, and the milestone that adds a seventh read changes no signature
// at all (issue #134).
//
// It is one trade and worth naming. A command handed the whole value says *I
// may read the process* where its signature used to say *I read the clock and
// nothing else*, and for the two commands that take it — `store` and `compact`
// — the finer statement is gone. main.go's environmentOnly is what keeps it
// everywhere it can still be made: a command that reads only the environment
// takes only a lookup, and six of §9's sixteen still say so by their shape.
//
// Every member is a function rather than a resolved value, and for the reason
// the working directory already was one: a read a command never makes is a read
// that never happens. `version` and `completions` resolve no working directory
// and call no gate, and that exemption is a path not taken rather than a value
// quietly computed for nobody (§9, ADR-0020) — whatever a command does not read
// is still visible in what it is handed.
type Process struct {
	// LookupEnv reads one environment variable, and answers whether it was
	// set at all rather than only what it said. Both halves are load-bearing:
	// HYPER_REPO_DIR is the second of §9's three configuration layers, and a
	// credential slot is *present* or *absent* by whether its variable is
	// set — a variable set to the empty string is present and says so (§9,
	// §5, issue #112).
	LookupEnv func(name string) (string, bool)

	// Getwd is where the invocation is standing. The dispatch calls it on the
	// repository commands' arm and hands the answer down, so no command
	// behind it has a reason to call it again — the third configuration layer
	// is the git root above the working directory, and it is resolved once,
	// where a command needs one. That the exempt arm calls it never is the
	// dispatch's own shape and is asserted there: a working directory that
	// cannot be read does not stop `hyper version` (§9, ADR-0020, issue
	// #103).
	Getwd func() (string, error)

	// Now is the clock. Every commit `hyper` writes takes both its dates from
	// it, so a branch a fixture builds is reproducible and `git log` on the
	// Store is honest; retention is an age and is measured against it; and a
	// Run's start instant is read once and every date the Run writes is that
	// instant (§6, §7, issue #125, issue #131).
	Now func() time.Time

	// Mint mints a Run id at the instant it is handed.
	//
	// store.MintRunID reads crypto/rand, which is a read of the process
	// exactly as the clock is, and this is the member that is easiest to
	// miss: a Run id lands on the terminal line, in the `outcome` row, in
	// run.json and in every Store path a Run writes, so an id minted by
	// whichever function happened to want one makes every golden of a Run
	// unassertable. The fix is to thread the read rather than to normalise
	// the id out of the goldens — §8 states that a Run id renders **whole**
	// (ADR-0047), and a corpus asserting `<run-id>` could not check the one
	// rendering rule that surface has.
	Mint func(now time.Time) store.RunID

	// Dial is how a connection to a host is made, and it is what the `http`
	// Capability's client dials through. Threading it is what lets a golden
	// case exercise a real handshake, a real status line and a real parse
	// against a server standing in the test process, with the name
	// resolution the only thing a fixture supplies — the response object is
	// never written down by a test (§5, issue #133).
	//
	// It answers a connection that is already past its TLS handshake, and
	// that is a fact about hyper rather than about this signature: the
	// scheme is `https` and there is no second one (ADR-0082), so every
	// connection hyper makes is a TLS connection, there is no plaintext
	// dialer to supply, and the certificate the peer presented is a real one
	// off a real handshake — which is what tls.days_left is read from (§12).
	// internal/capability wires it as http.Transport's DialTLSContext and
	// holds no TLS configuration of its own, and names the type: one
	// signature, spelled where the Capability that dials through it is
	// stated rather than twice.
	Dial capability.Dial

	// Exec is how a child process is started: it answers the child that argv
	// names, ready to run, carrying the two launch decisions that belong to
	// the process rather than to a Capability — the child starts in its own
	// process group, and cancelling ctx kills that whole group with SIGKILL
	// and no grace period, which is what a Manifest's deadline is (§5, §6).
	//
	// argv is a list with a literal head and nothing stands between it and
	// the process: there is no shell here, so a pipe, a redirection, a glob
	// and an `&&` are not writable (ADR-0051). What the caller sets on the
	// answer is everything about the child that the Manifest decides — the
	// directory, the environment, the streams — and never a process
	// attribute, which is what keeps the process group decided in one place.
	//
	// It is a Capability's child and only that. The git subprocesses
	// internal/store runs are the record's transport rather than anything an
	// artefact named: their argv is compiled in, their environment is that
	// package's own deliberate inheritance, and a deadline that SIGKILLed a
	// push mid-write would be a bound nothing declared. They do not come
	// through here, and a Manifest cannot reach them (§7, ADR-0006).
	Exec func(ctx context.Context, argv []string) *exec.Cmd
}
