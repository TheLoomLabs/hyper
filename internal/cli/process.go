package cli

import (
	"os"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Process is everything `hyper` reads from the process it is running in, as one
// value.
//
// It is issue #100's property at the grain a milestone of eight reads needs:
// everything a command reads from the process is a parameter it is handed
// rather than a package it reaches for, which is what makes the whole dispatch
// exercisable without a subprocess. What travels beside this value rather than
// in it is the argv, which is what the dispatch decides on; the destination,
// which a command writes to rather than reads and which is assembled out of the
// two streams the process handed in (destination.go); and version.Facts, which
// the build stamped rather than the process holds.
//
// Six loose parameters threaded through repositoryCommand is where that
// property starts costing more than it buys, so the reads travel as one value:
// what a command may read is a type a reader opens rather than a parameter list
// they count, and the milestone that adds a seventh read changes no signature
// at all (issue #134). The seventh and the eighth landed with `run`, which is
// the claim made good: User and Hostname were added below and no signature in
// the tree moved. The ninth landed with the `shell` Capability and moved none
// either (issue #142), and the tenth landed with the signals a Run drains on
// (issue #145).
//
// It is one trade and worth naming. A command handed the whole value says *I
// may read the process* where its signature used to say *I read the clock and
// nothing else*, and for the two commands that take it — `store` and `compact`
// — the finer statement is gone. main.go's environmentOnly is what keeps it
// everywhere it can still be made: a command that reads only the environment
// takes only a lookup, and five of §9's sixteen still say so by their shape.
// `review` was the sixth until it opened a range and started reading the clock
// the age beside its gloss is measured against (issue #164), which is the trade
// working as stated — the signature moved because what the command reads did.
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

	// Environ is the whole environment, and it is read for exactly one
	// thing: a `shell` Operation's child inherits the invoking environment
	// with every credential-slot variable in the repository removed (§3,
	// §11, issue #142).
	//
	// It is a second read of one subject rather than a widening of the
	// first, and the two cannot be folded. LookupEnv answers *what does this
	// name hold*, which is the whole of what a credential slot and
	// HYPER_REPO_DIR ask; composing a child's environment is a subtraction,
	// and a set of names nothing enumerates is not a set anything can
	// subtract from. Nothing else in the tree reads it: the git subprocesses
	// internal/store runs compose their own deliberate inheritance, which is
	// the record's transport rather than anything an artefact named (§7,
	// ADR-0006).
	Environ func() []string

	// Getwd is where the invocation is standing. The dispatch calls it on the
	// repository commands' arm and hands the answer down, so no command
	// behind it has a reason to call it again — the third configuration layer
	// is the git root above the working directory, and it is resolved once,
	// where a command needs one. That the exempt arm calls it never is the
	// dispatch's own shape and is asserted there: a working directory that
	// cannot be read does not stop `hyper version` (§9, ADR-0020, issue
	// #103).
	Getwd func() (string, error)

	// User is who is running hyper, and Hostname is the machine they are
	// running it on. Both are read for one value each in the whole tool: a
	// Journal entry's Trigger carries `actor` on both executors and `host`
	// on `local`, which is what §8's header renders `igor@thinkpad` from
	// (§7, §12).
	//
	// They are two reads rather than one because they come from two places
	// — the passwd database and the kernel — and either can answer while
	// the other does not.
	//
	// They are threaded rather than read because everything in this value
	// is, and because a fact that lands in the record must be a fact a
	// fixture can supply: an entry whose `host` came from the machine the
	// suite ran on is a `store.golden` nobody can check in. Each answers an
	// error the way its standard-library reading does, and a Run that
	// cannot read the machine writes no `host` at all — the ordinary
	// absence rule, and better than a constant hyper invented for a machine
	// it knows nothing about.
	User     func() (string, error)
	Hostname func() (string, error)

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
	//
	// It names the type for Dial's own reason: one signature, spelled where
	// the Capability that starts a child through it is stated rather than
	// twice. Child below is the value the binary wires into it (issue #142).
	Exec capability.Exec

	// Notify is the process's signals, watched: it is handed the signals to
	// watch and answers the channel they arrive on and the function that
	// stops the watch (§6, ADR-0015, issue #145).
	//
	// It is the tenth read and it is threaded for the reason the other nine
	// are: a Run stopped by an interrupt writes a Journal entry and a
	// terminal line, and a case that could not deliver one could assert
	// neither. What it stands for is the whole of `os/signal` — nothing
	// behind the dispatch imports that package — so a case drives the same
	// drain the terminal does, with the delivery its own.
	//
	// **Stopping the watch is what makes a second interrupt kill the
	// process.** Go's signal package replaces the default disposition while
	// a channel is registered and restores it when the last one goes, so the
	// Run that has already drained on the first signal releases the handler
	// and the next one lands on the kernel's own answer — which is §6's open
	// entry, and the reason there is no second drain to write (§7).
	//
	// It may be nil, and a Run under a nil Notify is one nobody can
	// interrupt: nothing is installed, nothing is caught, and the Run
	// performs to its end. That is what every command but `run` is handed,
	// there being no Run under them to stop.
	Notify Notify
}

// Notify is how the signals are watched: the signals to watch for, and the
// channel and the release the watch is held by. It is a named type for Dial's
// and Exec's reason — one signature, spelled where the thing that uses it is
// stated rather than twice — and it is `os/signal`'s own shape, so the binary
// wires that package in one expression and nothing else in the tree imports it.
type Notify func(signals ...os.Signal) (arriving <-chan os.Signal, stop func())
