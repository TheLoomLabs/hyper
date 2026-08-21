package capability

import (
	"context"
	"errors"
	"net"
)

// Whether a request provably never left, which is the one fact a retry Pattern
// is allowed to follow (§3, §6, ADR-0018, issue #143).
//
// **It is established by construction and never by reading an error.** A retry
// may follow only a failure that provably preceded the request, and *provably*
// is the whole weight of that sentence: a tool that decided by matching the
// text of a transport error would be guessing, and the guess would be wrong on
// the day a library reworded one. So the two performers mark the failures they
// know came before any byte left — the dialler answering nothing under `http`,
// and a child that never became a process under `shell` — and everything else
// is unmarked, whatever it says.
//
// **A timeout is never marked**, on either side of the mark's own test. A
// connect timeout is outside ADR-0018's class — the request may have left and
// the answer may be on its way — and the Operation's own `deadline:` is `hyper`
// stopping rather than the world refusing, which halts rather than retrying
// (§6).
//
// Nothing above this reads what went wrong, and this does not change that: the
// answer is a boolean about `hyper`'s own conduct, not a member of the response
// object and not a sixth thing a surface renders (ADR-0017).

// neverSent marks one error as a failure that provably preceded the request.
// It is unexported and has no constructor outside this package: the mark is a
// claim only the code that performed the call is in a position to make.
type neverSent struct{ err error }

func (n neverSent) Error() string { return n.err.Error() }
func (n neverSent) Unwrap() error { return n.err }

// NeverSent reports whether err is a failure that provably preceded the
// request — the closed class ADR-0018 retries: a refused connection, a name
// that did not resolve, a handshake that failed, and, the same fact one
// Capability over, a child that could not be started at all.
//
// It answers false for everything else, a status included: a status is an
// answer and never an error (ADR-0050), so nothing carrying one ever reaches
// here at all.
func NeverSent(err error) bool {
	var marked neverSent
	return errors.As(err, &marked)
}

// marking wraps a dialler so that every failure it answers is marked, which is
// what makes *the request never left* a fact about where the failure happened
// rather than about what it said.
//
// The dialler is the whole of the pre-send half of an `http` call: it is wired
// as the transport's DialTLSContext, so the TCP connect and the TLS handshake
// both happen inside it and the first byte of the request is written only after
// it has answered. A refused connection, a name that did not resolve and a
// failed handshake are therefore exactly the errors this sees, and they are
// exactly ADR-0018's class.
//
// Two failures pass through unmarked. One is the context being done — the
// Operation's `deadline:` reached while dialling, which is `hyper` stopping.
// The other is a timeout of the dialler's own, which is the connect timeout
// ADR-0018 declines to retry: it is *no answer within the time allowed* rather
// than *nothing left*, and the distinction is the whole reason the class is
// stated as three members rather than as *anything that went wrong early*.
func marking(dial Dial) Dial {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		connection, err := dial(ctx, network, address)
		if err == nil || ctx.Err() != nil {
			return connection, err
		}
		var expired net.Error
		if errors.As(err, &expired) && expired.Timeout() {
			return connection, err
		}
		return connection, neverSent{err}
	}
}
