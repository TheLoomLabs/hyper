package capability_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// ADR-0018's class, read off where the failure happened rather than off what it
// said (issue #143). A retry Pattern follows this answer and nothing else, so
// what these hold is the boundary: the three `http` failures that provably
// precede a request, the one `shell` failure that does, and the four things
// beside them that are answers or deadlines and are never retried.

// TestNeverSent_ARefusedConnection is the first member of the class: the
// dialler answered nothing, so no byte of the request was ever written.
func TestNeverSent_ARefusedConnection(t *testing.T) {
	refused := capability.Dial(func(_ context.Context, network, address string) (net.Conn, error) {
		return nil, &net.OpError{Op: "dial", Net: network, Err: errors.New("connection refused")}
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), refused, instant, capability.Credential{})
	if !capability.NeverSent(err) {
		t.Errorf("a refused connection answers NeverSent false: %v", err)
	}
}

// TestNeverSent_ANameThatDidNotResolve is the second, and it arrives through
// the same door: name resolution is the dialler's, so a name with nothing
// behind it fails before a request exists.
func TestNeverSent_ANameThatDidNotResolve(t *testing.T) {
	unresolvable := capability.Dial(func(context.Context, string, string) (net.Conn, error) {
		return nil, &net.DNSError{Err: "no such host", Name: servedHost, IsNotFound: true}
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), unresolvable, instant, capability.Credential{})
	if !capability.NeverSent(err) {
		t.Errorf("a name that did not resolve answers NeverSent false: %v", err)
	}
}

// TestNeverSent_AHandshakeThatFailed is the third. The dialler answers a
// connection already past its TLS handshake (§12, ADR-0082), so a handshake
// that failed is a dial that failed and lands in the class with the other two —
// which is what makes the class three members rather than a list of error
// types to keep up to date.
func TestNeverSent_AHandshakeThatFailed(t *testing.T) {
	unverified := capability.Dial(func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("tls: failed to verify certificate: x509: certificate signed by unknown authority")
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), unverified, instant, capability.Credential{})
	if !capability.NeverSent(err) {
		t.Errorf("a handshake that failed answers NeverSent false: %v", err)
	}
}

// TestNeverSent_AConnectTimeoutIsOutsideTheClass is ADR-0018 declining to retry
// one: a dial that ran out of time is *no answer within the time allowed*
// rather than *nothing left*, and the request may be on its way.
func TestNeverSent_AConnectTimeoutIsOutsideTheClass(t *testing.T) {
	expired := capability.Dial(func(_ context.Context, network, address string) (net.Conn, error) {
		return nil, &net.OpError{Op: "dial", Net: network, Err: connectTimeout{}}
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), expired, instant, capability.Credential{})
	if capability.NeverSent(err) {
		t.Errorf("a connect timeout answers NeverSent true: %v", err)
	}
}

// TestNeverSent_TheOperationsDeadlineIsNotTheClass is the other exclusion, and
// it is the one that matters most: the deadline is `hyper` stopping, which
// halts the Step rather than being retried into a second call the artefact
// never asked for (§6).
func TestNeverSent_TheOperationsDeadlineIsNotTheClass(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
	defer cancel()

	slow := capability.Dial(func(ctx context.Context, _, _ string) (net.Conn, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(ctx, slow, instant, capability.Credential{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Perform err = %v, want a deadline", err)
	}
	if capability.NeverSent(err) {
		t.Errorf("the Operation's deadline answers NeverSent true: %v", err)
	}
}

// TestNeverSent_AnAnswerIsNeverTheClass is ADR-0050 at this boundary: a status
// is an answer, so no status of any kind reaches NeverSent at all and a Call
// that got one answers false.
func TestNeverSent_AnAnswerIsNeverTheClass(t *testing.T) {
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), dial, instant, capability.Credential{})
	if err != nil {
		t.Fatalf("Perform err = %v, want none — a status is an answer", err)
	}
	if capability.NeverSent(err) {
		t.Error("a 503 answers NeverSent true")
	}
}

// TestNeverSent_AChildThatCouldNotBeStarted is the same fact one Capability
// over: a child that never became a process touched nothing, which is why it is
// the one `shell` failure a retry may follow (§6, ADR-0018, ADR-0062).
func TestNeverSent_AChildThatCouldNotBeStarted(t *testing.T) {
	command := capability.Command{Argv: []string{"/nonexistent/hyper-fixture-binary"}}
	_, err := command.Perform(t.Context(), direct, t.TempDir(), capability.Inherited(nil, nil))
	if !capability.NeverSent(err) {
		t.Errorf("a child that could not be started answers NeverSent false: %v", err)
	}
}

// TestNeverSent_ANonZeroExitIsNeverTheClass is ADR-0050 under `shell`: the code
// is the answer, so a command that ran and failed is not something to run
// again.
func TestNeverSent_ANonZeroExitIsNeverTheClass(t *testing.T) {
	command := capability.Command{Argv: []string{"/bin/sh", "-c", "exit 3"}}
	object, err := command.Perform(t.Context(), direct, t.TempDir(), capability.Inherited(nil, nil))
	if err != nil {
		t.Fatalf("Perform err = %v, want none — an exit code is an answer", err)
	}
	if code, held := object.Lookup(capability.MemberExitCode); !held || code != 3 {
		t.Fatalf("exit_code = %v (held %v), want 3", code, held)
	}
	if capability.NeverSent(err) {
		t.Error("a non-zero exit answers NeverSent true")
	}
}

// connectTimeout is a dialler that ran out of time, which is the one shape
// this boundary reads off the error rather than off the position: net.Error's
// own Timeout, which is what a connect timeout answers and what a refused
// connection does not.
type connectTimeout struct{}

func (connectTimeout) Error() string { return "i/o timeout" }
func (connectTimeout) Timeout() bool { return true }
