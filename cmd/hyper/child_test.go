package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"
	"testing"
)

// The two decisions child makes, asserted rather than assumed (issue #134).
//
// They are here and not in internal/cli because they are the *production*
// implementation of one of the six: cli.Process says what Exec must do and the
// golden harness proves nothing calls it yet, so the only place the real launch
// can be held to its criterion is beside the file that writes it. m5.9 is the
// milestone that will exec through this; the process group it depends on is
// decided here and re-decided nowhere, which is exactly why it is pinned before
// its first caller exists.

// TestChild_StartsInItsOwnProcessGroup is the criterion the shell Capability
// stands on: the child leads a process group of its own rather than joining
// hyper's, so a deadline can reach everything the argv started and not only the
// process hyper waited on (§5, §6).
func TestChild_StartsInItsOwnProcessGroup(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cmd := child(ctx, []string{sleepBinary(t), "30"})
	if err := cmd.Start(); err != nil {
		t.Fatalf("the child would not start: %v", err)
	}
	// Cancelling is what stops it, which is the other decision doing its
	// work: the wait answers the cancellation rather than an exit status.
	defer func() {
		cancel()
		_ = cmd.Wait()
	}()

	group, err := syscall.Getpgid(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("the child's process group could not be read: %v", err)
	}
	if group != cmd.Process.Pid {
		t.Errorf("the child's process group is %d, want its own pid %d", group, cmd.Process.Pid)
	}
	if group == syscall.Getpgrp() {
		t.Errorf("the child is in hyper's own process group %d, want a group of its own", group)
	}
}

// TestChild_ACancellationThatFindsNothingToKillIsNotAFault is the race the
// negated-pid kill opens and the ESRCH arm closes: a child that finished
// between the deadline expiring and the signal being sent.
//
// os/exec surfaces any error from Cancel out of Wait as *exec: canceling Cmd*,
// so an unmapped ESRCH would reach the Capability as a child that could not be
// started — the one shape §12's response object reserves for an argv that never
// ran at all. The child here is run to completion and reaped first, which is
// that race with the timing taken out of it.
func TestChild_ACancellationThatFindsNothingToKillIsNotAFault(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	cmd := child(ctx, []string{sleepBinary(t), "0"})
	if err := cmd.Run(); err != nil {
		t.Fatalf("the child would not run: %v", err)
	}

	if err := cmd.Cancel(); !errors.Is(err, os.ErrProcessDone) {
		t.Errorf("cancelling a finished child answered %v, want os.ErrProcessDone", err)
	}
}

// sleepBinary is a child that does nothing for as long as it is told to, which
// is the whole of what these two cases need one for. It is resolved rather than
// spelled as a path: what is under test is the launch, and a case that failed
// because /bin/sleep sat somewhere else would be reporting the wrong thing.
func sleepBinary(t *testing.T) string {
	t.Helper()

	path, err := exec.LookPath("sleep")
	if err != nil {
		t.Skipf("this platform has no sleep to launch (%v); the probe cannot be built here", err)
	}
	return path
}
