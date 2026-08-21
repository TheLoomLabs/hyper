package cli_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// A golden case can reach the world, and this file is the whole of what that
// costs (issue #135).
//
// **The call is real; only the name resolution is a fixture.** A case that
// supplies a `serve/` directory has one in-process TLS server stood for it, its
// certificate minted against the case's own `now`, and the dialer cli.Main is
// handed maps every hostname the case serves to that listener. So a case
// exercises a real handshake, a real peer certificate, a real status line, real
// headers and a real JSON parse, and `tls.days_left` is a number read off an
// x509 chain rather than one a fixture wrote down. **Nothing about the response
// object is ever supplied by a test**; the fixture supplies what a *server*
// would, which is the only thing a fixture has any business supplying.
//
// **A host with no `serve/` entry gets its connection refused**, which is how
// *no response arrived at all* is driven with no second mechanism: a case
// naming a granted host it does not serve is a case about the silence, and it
// says so by the file it did not write.
//
// **A host that `hangs` accepts the connection and never answers**, which is
// the only way a case can reach the Operation's `deadline:` (issue #140). It is
// a different fact from the refused connection above and the two must not be
// confused: a refused connection is *no response arrived*, which a `read`
// records as the answer it is, and a deadline is `hyper` stopping — the one
// thing that fails a `read` Step short of its projection (§6, ADR-0050).
//
// **A host may answer more than once, and may refuse a fixed number of times
// first — as a refused connection, a name that did not resolve, or a handshake
// that failed** (issue #143). Both are the world changing between calls, which is what
// the Patterns exist to walk: a paginated read asks for the next page, a polled
// Operation asks again until its `until:` holds, and a retry follows a
// connection that was refused. Neither says anything about `hyper` — the case
// still supplies only what a *server* would.

// certificateLife is how long after a case's instant the fixture's certificate
// expires: thirty-four days and a half, so `tls.days_left` is **34** — floored
// off a real chain, and far enough from a day boundary that the seconds x509
// rounds a NotAfter to cannot move it.
const certificateLife = 34*24*time.Hour + 12*time.Hour

// servedResponse is one entry of a case's serve/ directory: what the world
// answers for one host. It is a status, headers and a body and nothing else —
// a case says what the world answers and nothing about what hyper does with it.
//
// Unknown fields are an error: a misspelt key would otherwise be a fixture that
// asserts less than its file says it does, which is the failure mode a golden
// corpus is least able to notice.
type servedResponse struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	// Body is the bytes the response carries, exactly. It is a string rather
	// than a JSON value because the corpus has to be able to serve something
	// that is not JSON at all: a body that is absent, or HTML, is what makes
	// `body` an absence a projection reads rather than an error (§12).
	Body string `json:"body"`
	// Hangs is the host that accepts the connection and answers nothing at
	// all, until the caller gives up on it. It is what a server that has
	// stopped answering does, and it is the whole of how a case reaches the
	// Operation's own `deadline:` — a number an artefact declared, which is
	// therefore a number a case can drive to (§3, §6, issue #140).
	//
	// It serves nothing else: a case using it asserts what `hyper` did with
	// the members beside this one, and this host contributes no response for
	// anything to read.
	Hangs bool `json:"hangs"`
	// Answers is what the host answers on **successive** requests, in
	// order, where one answer is not enough: the pages a paginated read
	// walks, and the states a polled Operation passes through before its
	// `until:` holds (issue #143). The last answer repeats once the list is
	// exhausted, so a case says what changes and stops there.
	//
	// It is deterministic because the thing it serves is: all three
	// Patterns are serial by construction, so a member is one request at a
	// time from the moment it is dispatched until its last page (§3, §6).
	// A case driving several members through one host would be depending on
	// something nothing fixes, and none does — a member that pages has a
	// host of its own.
	//
	// An answer carries a status, headers and a body and no more: `hangs`
	// and the echoes below are the host's and not one answer's.
	Answers []servedAnswer `json:"answers"`
	// RefuseFirstAs is **how** those connections fail, one entry per refusal
	// and in order: `refused` for a connection nothing accepted, `name` for
	// a host that did not resolve, and `handshake` for a TLS handshake that
	// did not complete. It is what lets a case drive ADR-0018's class
	// through a Run rather than only at the Capability — the three are one
	// class because each provably precedes the request, and a case that
	// could name only one of them would be asserting the class by its
	// smallest member.
	//
	// Absent, every refusal is `refused`, which is what every case that
	// names none means. A list shorter than RefuseFirst repeats its last
	// entry, on the rule Answers follows.
	RefuseFirstAs []string `json:"refuse_first_as"`
	// RefuseFirst is how many connections to this host are refused before
	// any is accepted — the refused connection above, arriving a fixed
	// number of times rather than forever.
	//
	// It is what a retry Pattern is driven against: ADR-0018's class is
	// *the request provably never left*, and a case that could only refuse
	// for ever could show a retry exhausting and never a retry succeeding.
	// It counts dials rather than requests, which is the same number here —
	// one call is one connection, hyper's client disabling keep-alives so
	// that the certificate a response reports is the one that call was
	// answered over.
	RefuseFirst int `json:"refuse_first"`
	// EchoRequestHeaders names request headers the fixture writes back as
	// the response body, a JSON object keyed by the lowered header name.
	//
	// It is the one thing here that is not what a server would supply, and
	// it exists for the one claim a golden cannot otherwise assert: **what
	// hyper put on the wire**. A credential is composed from a Manifest's
	// scheme parameters and a Target declaration's environment variable and
	// then leaves — it reaches no file, no row and no rendering (§7,
	// ADR-0007) — so the only place a corpus can observe it is at the far
	// end, which is here. A case using it serves a body of its own for
	// nothing else.
	EchoRequestHeaders []string `json:"echo_request_headers"`
}

// servedAnswer is one of a host's successive answers: a status, headers and a
// body, and nothing that is a fact about the host rather than about the answer.
type servedAnswer struct {
	Status  int               `json:"status"`
	Headers map[string]string `json:"headers"`
	Body    string            `json:"body"`
}

// Two placeholders a body may carry, which the fixture fills from the request
// it is answering: the raw query string, and one named request header.
//
// They are `echo_request_headers` written **inside** a body rather than instead
// of one, and they exist for the claim a golden cannot otherwise assert: **what
// hyper put on the wire**. A pagination Pattern writes its token or its number
// into a `query:` or a `header:` position an artefact named (§3), and a Record
// is projected out of a body — so a case that echoed the whole request would
// have nothing left to project, and one that echoed nothing could show only
// that the pages arrived in order.
//
// A body carrying neither is served exactly as the case wrote it, which is
// every case that landed before them.
const queryPlaceholder = "<<query>>"

var headerPlaceholder = regexp.MustCompile(`<<header:([^<>]+)>>`)

// fixtureServer is the world a case's serve/ directory stands: what each host
// answers, how many of its connections are still to be refused, and how many
// requests it has already answered.
//
// The two counters are the whole of what it holds beyond the case's own files,
// and both are per host: `refuse_first` counts the dials refused, and the
// answer index counts the requests answered, which is what walks a host through
// its `answers` list.
type fixtureServer struct {
	served   map[string]servedResponse
	mutex    sync.Mutex
	refused  map[string]int
	answered map[string]int
}

// servedHosts reads the case's serve/ directory: one entry per `<host>.json`,
// keyed by the host it answers for. A case with no such directory serves
// nothing and dials nothing.
func (c goldenCase) servedHosts(t *testing.T) map[string]servedResponse {
	t.Helper()

	entries, err := os.ReadDir(filepath.Join(c.dir, "serve"))
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatal(err)
		}
		return nil
	}

	served := map[string]servedResponse{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(c.dir, "serve", entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		var response servedResponse
		if err := decoder.Decode(&response); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		served[strings.TrimSuffix(entry.Name(), ".json")] = response
	}
	return served
}

// dialer is what the case hands cli.Main in the process's place. A case that
// serves nothing gets the stub that fails it: dialling is an axis a case opts
// into by writing a serve/ entry, and a case that reached the network without
// one is a case whose golden could not have been asserted.
func (c goldenCase) dialer(t *testing.T, instant time.Time) func(context.Context, string, string) (net.Conn, error) {
	t.Helper()

	served := c.servedHosts(t)
	if len(served) == 0 {
		return func(context.Context, string, string) (net.Conn, error) {
			t.Errorf("case %s dialled a host and supplies no serve/ entry for any", c.name)
			return nil, fmt.Errorf("the golden harness dials nothing for %s", c.name)
		}
	}

	closing := make(chan struct{})
	hosts := make([]string, 0, len(served))
	for host := range served {
		hosts = append(hosts, host)
	}
	slices.Sort(hosts)

	world := &fixtureServer{served: served, refused: map[string]int{}, answered: map[string]int{}}
	certificate, pool := mintFixtureCertificate(t, instant, hosts)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		world.answer(w, r, instant, closing)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	// The server's own log would otherwise write a handshake error to the
	// suite's stderr where a case drives a host it does not serve.
	server.Config.ErrorLog = discardLog()
	server.StartTLS()
	t.Cleanup(server.Close)
	// Registered after the close above and therefore run before it: a
	// handler that hangs is released first, and httptest's own wait for its
	// outstanding requests then has nothing to wait for.
	t.Cleanup(func() { close(closing) })

	listening := server.Listener.Addr().String()
	return func(ctx context.Context, network, requested string) (net.Conn, error) {
		name, _, err := net.SplitHostPort(requested)
		if err != nil {
			return nil, err
		}
		// A host with no serve/ entry gets its connection refused, which
		// is the whole of how *no response arrived at all* is driven —
		// and so does one whose entry says its first few are, which is
		// the same refusal arriving a fixed number of times.
		if _, serves := served[name]; !serves {
			return nil, refusal("refused", network, name)
		}
		if as, refuses := world.refusing(name); refuses {
			return nil, refusal(as, network, name)
		}
		// Time is the case's clock and not the machine's: the certificate
		// is minted against the instant the case is driven at, so the
		// chain it verifies against has to be read at that instant too.
		dialer := tls.Dialer{Config: &tls.Config{
			RootCAs:    pool,
			ServerName: name,
			Time:       func() time.Time { return instant },
		}}
		return dialer.DialContext(ctx, network, listening)
	}
}

// answer writes the response the case wrote down for the host that was asked
// for. Date is stamped from the case's clock rather than left to Go's, which
// would put the machine's wall clock into every asserted response object: a
// server answers with a Date and this one does too, and what the fixture fixes
// is which instant it names.
//
// closing is the suite taking the server down, and it releases a host that
// hangs where no caller ever gave up on it.
func (f *fixtureServer) answer(w http.ResponseWriter, r *http.Request, instant time.Time, closing <-chan struct{}) {
	name := r.Host
	if host, _, err := net.SplitHostPort(name); err == nil {
		name = host
	}
	response, serves := f.served[name]
	if !serves {
		// Unreachable: the dialer refuses a host with no entry before a
		// request is ever written. It answers rather than panicking so
		// that a harness fault reads as a case failing rather than as the
		// suite dying inside a goroutine.
		http.Error(w, "the golden harness serves no "+name, http.StatusInternalServerError)
		return
	}

	// The host that answers nothing: it holds the request open until the
	// caller gives up on it, which is the caller's own deadline arriving and
	// nothing this fixture decides.
	if response.Hangs {
		select {
		case <-r.Context().Done():
		case <-closing:
		}
		return
	}

	// Which of the host's successive answers this request gets. A host
	// answering one thing serves it to every request, which is every case
	// that landed before the list existed.
	//
	// An answer states its own status and body outright. Its headers are
	// written **over** the host's rather than instead of them, so a case
	// whose pages differ only in what they carry says the Content-Type once
	// — which is the difference between a fact about the host and a fact
	// about the answer, kept where the rest of this file keeps it.
	headers := response.Headers
	if next, walks := f.next(name, response); walks {
		response.Status, response.Body = next.Status, next.Body
		headers = merged(response.Headers, next.Headers)
	}

	w.Header().Set("Date", instant.UTC().Format(http.TimeFormat))
	for header, value := range headers {
		w.Header().Set(header, value)
	}
	w.WriteHeader(response.Status)
	io.WriteString(w, body(r, response))
}

// merged is the host's headers with one answer's written over them.
func merged(host, answer map[string]string) map[string]string {
	written := make(map[string]string, len(host)+len(answer))
	maps.Copy(written, host)
	maps.Copy(written, answer)
	return written
}

// refusing answers whether this dial is one of the ones the host's entry says
// are refused and how it fails, and counts it.
func (f *fixtureServer) refusing(host string) (string, bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	entry := f.served[host]
	if f.refused[host] >= entry.RefuseFirst {
		return "", false
	}
	as := "refused"
	if named := entry.RefuseFirstAs; len(named) > 0 {
		as = named[min(f.refused[host], len(named)-1)]
	}
	f.refused[host]++
	return as, true
}

// refusal is the error one dial answers with, in the shape the failure it names
// really has: a refused connection is the transport's, a name that did not
// resolve is the resolver's, and a handshake that failed is the TLS stack's.
//
// The shapes matter rather than the words. `hyper` establishes ADR-0018's class
// by **where** a failure happened and not by reading it, so what a case is
// really driving is that all three arrive through the dialler — but a fixture
// answering one error type three times would be checking that with less than it
// could.
func refusal(as, network, host string) error {
	switch as {
	case "name":
		return &net.DNSError{Err: "no such host", Name: host, IsNotFound: true}
	case "handshake":
		return &tls.CertificateVerificationError{
			Err: fmt.Errorf("x509: certificate signed by unknown authority"),
		}
	default:
		return &net.OpError{
			Op: "dial", Net: network, Addr: fixtureAddress(host),
			Err: fmt.Errorf("connection refused"),
		}
	}
}

// next is the answer this request gets off the host's answers list, and false
// where the host answers one thing. The last answer repeats once the list is
// exhausted, so a case writes what changes and stops there.
func (f *fixtureServer) next(host string, response servedResponse) (servedAnswer, bool) {
	if len(response.Answers) == 0 {
		return servedAnswer{}, false
	}

	f.mutex.Lock()
	defer f.mutex.Unlock()

	at := min(f.answered[host], len(response.Answers)-1)
	f.answered[host]++
	return response.Answers[at], true
}

// body is what the response carries: the bytes the case wrote down, or — where
// the case asked for it — what arrived on the request, as a JSON object keyed
// by the lowered header name.
//
// The names are lowered and sorted so that the body is one byte sequence
// whatever order Go's header map iterates in, which is what makes the Record a
// case asserts a checked-in constant. A header the request did not carry is
// written as the empty string rather than left out: *the header was not sent*
// is precisely what such a case is asserting, and an absent key would leave the
// projection with nothing to record and the golden with nothing to show.
func body(r *http.Request, response servedResponse) string {
	if len(response.EchoRequestHeaders) == 0 {
		return filled(r, response.Body)
	}
	echoed := map[string]string{}
	for _, name := range response.EchoRequestHeaders {
		echoed[strings.ToLower(name)] = r.Header.Get(name)
	}
	encoded, err := json.Marshal(echoed)
	if err != nil {
		return response.Body
	}
	return string(encoded)
}

// filled writes back what the server saw, where the case's body asked for it:
// the raw query string in place of `<<query>>`, and one named request header in
// place of `<<header:name>>`. A body carrying neither is answered exactly as it
// was written.
//
// A header the request did not carry fills as the empty string, which is what
// `echo_request_headers` already does one position over and for the same
// reason: *the header was not sent* is precisely what such a case asserts.
func filled(r *http.Request, written string) string {
	written = strings.ReplaceAll(written, queryPlaceholder, r.URL.RawQuery)
	return headerPlaceholder.ReplaceAllStringFunc(written, func(placeholder string) string {
		return r.Header.Get(headerPlaceholder.FindStringSubmatch(placeholder)[1])
	})
}

// mintFixtureCertificate mints one self-signed certificate covering every host
// the case serves, valid from a day before the case's instant to
// certificateLife after it.
//
// It is minted rather than checked in for the reason the git fixture is built
// rather than committed: a certificate has an expiry, and one checked in would
// make the corpus fail on a date nobody chose. Minting it against the case's
// own clock is what makes `tls.days_left` a checked-in constant instead.
func mintFixtureCertificate(t *testing.T, instant time.Time, hosts []string) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: hosts[0], Organization: []string{fixtureIdentityName}},
		NotBefore:             instant.Add(-24 * time.Hour),
		NotAfter:              instant.Add(certificateLife),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              hosts,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}

	pool := x509.NewCertPool()
	pool.AddCert(parsed)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: parsed}, pool
}

// fixtureAddress is what a refused connection names in its error. It is the
// host rather than an address, because that is what the invocation asked for
// and no address was ever resolved.
type fixtureAddress string

func (a fixtureAddress) Network() string { return "tcp" }
func (a fixtureAddress) String() string  { return string(a) }

// discardLog is a logger that writes nowhere, for the server's own error log:
// a handshake the fixture meant to refuse is a case's subject and not a line in
// the suite's output.
func discardLog() *log.Logger { return log.New(io.Discard, "", 0) }
