// Package capability is the two effects hyper performs on a Manifest's behalf,
// and it is the only package in the tool that touches a network or a child
// process (§5, issue #133). A Manifest declares the Capabilities it requires
// and a Target declaration grants them; what an Operation's request block says
// and what performing it does are one subject, and this is where both live.
//
// Both halves are here. The `http` one is a Manifest's http: block read, its
// template holes filled from an Operation's resolved inputs, the request
// performed, and the response object §12 closes at five members assembled back
// (issue #133). The `shell` one is an argv exec'd directly and the object §12
// closes at four assembled out of what the child did (issue #142) — shallower
// by a long way, and deliberately so: `hyper`'s own shell Provider is the only
// one that may declare that Capability (ADR-0039) and it knows nothing whatever
// about the command, so there is no request block to read and the words are the
// Step's.
//
// It is the milestone's deep module in the one sense that matters: everything
// above it is handed an object with named members and never a socket or a
// process. Nothing outside this package opens a connection, reads a status
// line, parses a body, looks at a certificate or waits on a child — and the
// projection above it (internal/projection) reads paths against the object
// rather than against the bytes that came back (§3, ADR-0040).
//
// Neither performer is reached for. Dial and Exec are threaded from
// cli.Process, which is what lets a case exercise a real handshake against a
// server standing in the test process and a real child against a script a
// fixture checked in, with the name resolution the only thing a fixture
// supplies (issues #134, #142).
package capability

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// The five members of the `http` response object, in §12's own order, which is
// the order they are assembled and rendered in. They are named here rather
// than spelled at each site because a projection path names them too — a
// Manifest writes $.tls.days_left — so a second spelling of one of them is a
// path that silently resolves to nothing.
const (
	MemberHost    = "host"
	MemberStatus  = "status"
	MemberHeaders = "headers"
	MemberBody    = "body"
	MemberTLS     = "tls"
)

// The four members of the tls member, in §12's order.
const (
	MemberNotAfter = "not_after"
	MemberDaysLeft = "days_left"
	MemberSubject  = "subject"
	MemberIssuer   = "issuer"
)

// Object is a response object: named members in the order §12 states them.
//
// It is ordered rather than a Go map for one reason, and it is the reason a
// response object exists at all: the object is the answer, and a surface
// renders it — the raw response beside the projection a Probe writes (§9,
// ADR-0017), and the `response` member of the probe_result row beside it. §12
// states five members in an order, and a rendering that sorted them or emitted
// them in whatever order a map iterated would be stating the tool's answer in
// an order nothing fixed.
//
// A member the object does not carry is absent from it entirely rather than
// held as null — the ordinary absence rule (§7) — which is what makes
// *resolved to nothing* and *resolved to null* two answers a projection tells
// apart (§12, internal/projection).
type Object []Member

// Member is one named member of an Object. Value is what the member holds: a
// string, an int, a nested Object, a mapping of header name to value, or —
// under body — whatever the response's JSON parsed to.
type Member struct {
	Name  string
	Value any
}

// Lookup answers what one member holds, and false where the object does not
// carry it. It is the first hop of every projection path, and the distinction
// it draws is the one §12 turns on: absent is not null.
func (o Object) Lookup(name string) (any, bool) {
	for _, m := range o {
		if m.Name == name {
			return m.Value, true
		}
	}
	return nil, false
}

// MarshalJSON writes the object compact and in its own member order: §8's wire
// encoding, which is the renderer's key order rather than the Store's code
// point order, and which nothing hashes or compares as bytes (§7, ADR-0079).
//
// HTML escaping is off, on internal/render's own rule: what came back off the
// wire renders as it came back, an & and a < included.
func (o Object) MarshalJSON() ([]byte, error) {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, m := range o {
		if i > 0 {
			out.WriteByte(',')
		}
		name, err := CompactJSON(m.Name)
		if err != nil {
			return nil, fmt.Errorf("response member name %q: %w", m.Name, err)
		}
		value, err := CompactJSON(m.Value)
		if err != nil {
			return nil, fmt.Errorf("response member %q: %w", m.Name, err)
		}
		out.Write(name)
		out.WriteByte(':')
		out.Write(value)
	}
	out.WriteByte('}')
	return out.Bytes(), nil
}

// CompactJSON is one value's JSON as hyper writes it anywhere a machine reads
// it: compact, and with HTML escaping **off** so that a value carrying an & or
// a < is one a consumer reads back as it was written (§8, internal/render).
//
// It is exported because it is the encoding of everything this package
// assembles, and the packages above it write the same values on the same wire:
// internal/projection renders a projected value with it, and a Probe's page and
// its row stream carry both. encoding/json's encoder is the only door to that
// switch, and it ends every value with a newline, which a member of an object
// may not carry.
//
// It is not the Store's encoding, which is indented and sorts a mapping's keys
// by code point because a git diff of it is read by a human (§7, ADR-0079).
// Nothing this writes is ever hashed or compared as bytes.
func CompactJSON(v any) ([]byte, error) {
	var out bytes.Buffer
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(out.Bytes(), []byte("\n")), nil
}
