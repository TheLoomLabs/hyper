// Command hyper is the CLI surface over hyper's core (§9). This file is the
// whole of what the binary adds to the library behind it: it reads the process
// — the arguments, the ten reads cli.Process names, the facts Go's build
// stamped — and hands them to cli.Main, which decides which command runs.
//
// No command name appears here, and no dispatch does. This file has no golden
// coverage, which is the argument gate.go makes for the gate and cli.Main
// makes for the dispatch behind it (issues #102 and #107).
//
// Every one of the ten is handed over uncalled, so a read no command makes is a
// read that never happens. os.Getwd is the one that shows why: `version` and `completions` stand outside §9's tree of sixteen and
// resolve no repository, so a working directory that cannot be read must not
// stop them (§9, ADR-0020). The others follow the same rule for reasons
// process.go states one at a time, and nothing anywhere else in the tree
// reaches for any of them (§7, issue #125, issue #134, issue #142).
package main

import (
	"crypto/tls"
	"os"
	"os/signal"
	"os/user"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/version"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr, process(), version.Current()))
}

// process is the real ten: the environment read one variable at a time and
// read whole, the working directory, the user and the machine's name, the
// clock, the Run id mint, the dialer, the process launcher and the signals. It
// is the one place in the tree where the standard library's readings of the process are named, and
// it is a function rather than a literal inside main() so that the binary's own
// cases drive exactly the value main() does rather than a second assembly of it
// (issue #134).
func process() cli.Process {
	return cli.Process{
		LookupEnv: os.LookupEnv,
		// The whole environment, which is read for one thing: a `shell`
		// Operation's child inherits it, less every credential-slot
		// variable in the repository (§3, §11). LookupEnv cannot answer
		// it — a set of names nothing enumerates is not a set anything
		// can subtract from.
		Environ: os.Environ,
		Getwd:   os.Getwd,
		// Who is running hyper and on which machine, for the two values
		// in the tool that carry them: a Journal entry's Trigger writes
		// `actor` on both executors and `host` on `local` (§7, §12).
		//
		// The user comes from the passwd database rather than from
		// $USER, which is conventional rather than guaranteed and is
		// absent in a container often enough to matter. os/user falls
		// back to reading the environment itself where it can do no
		// better, so what is spelled here is the more reliable of the
		// two readings and not a second one.
		User:     currentUser,
		Hostname: os.Hostname,
		Now:      time.Now,
		// store.MintRunID over the instant it is handed. crypto/rand is
		// beneath it, which is why the mint is threaded and not called
		// where an id is wanted: the id is written into every Store path a
		// Run makes and onto the terminal line, and it renders whole
		// (ADR-0047).
		Mint: store.MintRunID,
		// The standard dialer, with no timeout of its own: a deadline
		// belongs to the Manifest that declared one and arrives on the
		// context, and a second one here would be a bound no artefact
		// agreed to (§3, ADR-0014).
		//
		// It is a TLS dialer because the scheme is https and there is no
		// second one (ADR-0082), so every connection hyper makes is a TLS
		// connection and there is no plaintext path to configure. The
		// configuration is the standard library's: the system roots, and
		// the server name taken from the address, which is the host the
		// grant was checked against (ADR-0029).
		Dial: new(tls.Dialer).DialContext,
		// The launcher, which internal/cli states in full: the child's
		// own process group, and a deadline that kills it whole. It is
		// named there rather than here because the corpus drives the same
		// value this line wires (§5, §6).
		Exec: cli.Child,
		// The signals, watched. This is the one place in the tree that
		// imports os/signal, and what it hands over is the whole of that
		// package's contract: a channel the signals arrive on, and the
		// stop that puts the default disposition back — which is what
		// makes the **second** interrupt kill the process after the
		// first has drained (§6, ADR-0015).
		Notify: watchSignals,
	}
}

// watchSignals is cli.Notify against the real process: signal.Notify onto a
// buffered channel, and signal.Stop as the release.
//
// The buffer is one, which is what os/signal requires of a caller that is not
// always at the receive: the delivery is non-blocking, so a signal arriving
// before the watch reads is held rather than dropped. One is enough because the
// watch reads exactly one — the second interrupt is the kernel's answer and not
// this channel's (internal/cli/signals.go).
func watchSignals(signals ...os.Signal) (<-chan os.Signal, func()) {
	arriving := make(chan os.Signal, 1)
	signal.Notify(arriving, signals...)
	return arriving, func() { signal.Stop(arriving) }
}

// currentUser is who is running hyper, as cli.Process.User: the passwd
// database's name for the account the process runs under.
func currentUser() (string, error) {
	who, err := user.Current()
	if err != nil {
		return "", err
	}
	return who.Username, nil
}
