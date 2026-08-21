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
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

	hosts := make([]string, 0, len(served))
	for host := range served {
		hosts = append(hosts, host)
	}
	slices.Sort(hosts)

	certificate, pool := mintFixtureCertificate(t, instant, hosts)
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		answer(w, r, served, instant)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	// The server's own log would otherwise write a handshake error to the
	// suite's stderr where a case drives a host it does not serve.
	server.Config.ErrorLog = discardLog()
	server.StartTLS()
	t.Cleanup(server.Close)

	listening := server.Listener.Addr().String()
	return func(ctx context.Context, network, requested string) (net.Conn, error) {
		name, _, err := net.SplitHostPort(requested)
		if err != nil {
			return nil, err
		}
		// A host with no serve/ entry gets its connection refused, which
		// is the whole of how *no response arrived at all* is driven.
		if _, serves := served[name]; !serves {
			return nil, &net.OpError{
				Op: "dial", Net: network, Addr: fixtureAddress(name),
				Err: fmt.Errorf("connection refused"),
			}
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
func answer(w http.ResponseWriter, r *http.Request, served map[string]servedResponse, instant time.Time) {
	name := r.Host
	if host, _, err := net.SplitHostPort(name); err == nil {
		name = host
	}
	response, serves := served[name]
	if !serves {
		// Unreachable: the dialer refuses a host with no entry before a
		// request is ever written. It answers rather than panicking so
		// that a harness fault reads as a case failing rather than as the
		// suite dying inside a goroutine.
		http.Error(w, "the golden harness serves no "+name, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Date", instant.UTC().Format(http.TimeFormat))
	for header, value := range response.Headers {
		w.Header().Set(header, value)
	}
	w.WriteHeader(response.Status)
	io.WriteString(w, response.Body)
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
