// Command hyper is the CLI surface over hyper's core (§9). This file is the
// whole of what the binary adds to the library behind it: it reads the process
// — the arguments, the six reads cli.Process names, the facts Go's build
// stamped — and hands them to cli.Main, which decides which command runs.
//
// No command name appears here, and no dispatch does. This file has no golden
// coverage, which is the argument gate.go makes for the gate and cli.Main
// makes for the dispatch behind it (issues #102 and #107).
//
// Every one of the six is handed over uncalled, so a read no command makes is a
// read that never happens. os.Getwd is the one that shows why: `version` and `completions` stand outside §9's tree of sixteen and
// resolve no repository, so a working directory that cannot be read must not
// stop them (§9, ADR-0020). The other five follow the same rule for reasons
// process.go states one at a time, and nothing anywhere else in the tree
// reaches for any of them (§7, issue #125, issue #134).
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/store"
	"github.com/TheLoomLabs/hyper/internal/version"
)

func main() {
	os.Exit(cli.Main(os.Args[1:], os.Stdout, os.Stderr, process(), version.Current()))
}

// process is the real six: the environment, the working directory, the clock,
// the Run id mint, the dialer and the process launcher. It is the one place in
// the tree where the standard library's readings of the process are named, and
// it is a function rather than a literal inside main() so that the binary's own
// cases drive exactly the value main() does rather than a second assembly of it
// (issue #134).
func process() cli.Process {
	return cli.Process{
		LookupEnv: os.LookupEnv,
		Getwd:     os.Getwd,
		Now:       time.Now,
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
		Exec: child,
	}
}

// child is cli.Process.Exec, which states what the two decisions here are for.
// This is how they are made.
//
// Setpgid is the process group, and it is set rather than the child being left
// in hyper's own: what a deadline kills is then the whole tree the argv
// started, a command that forks being the ordinary case rather than the exotic
// one. Cancel is replaced because exec.CommandContext's own cancellation
// signals the leader alone, which would leave the group Setpgid just made — so
// the signal goes to the negated pid, which is the group, and it is SIGKILL
// with nothing before it.
//
// argv is exec'd directly with no shell between the artefact and the process,
// and its head is read off it without a question asked: the authoring format
// admits no other shape and `check` refuses a command without a literal head
// long before a Run reaches a Capability (ADR-0051).
func child(ctx context.Context, argv []string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); !errors.Is(err, syscall.ESRCH) {
			return err
		}
		// The group went on its own between the deadline expiring and the
		// signal being sent, which is a race and not a fault. os/exec
		// surfaces any other error from Cancel out of Wait as *exec:
		// canceling Cmd*, and a command that merely finished in time would
		// then reach the Capability as a child that could not be started —
		// the one shape §12's response object reserves for an argv that
		// never ran at all. os.ErrProcessDone is what os/exec reads as
		// *there was nothing left to kill*.
		return os.ErrProcessDone
	}
	return cmd
}
