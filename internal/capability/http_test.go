package capability_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"errors"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// instant is the clock every case here is driven at — the one §7's worked
// examples are written at, so a days_left in this file is the same number a
// golden case in internal/cli holds.
var instant = time.Date(2026, 4, 2, 9, 41, 14, 221_000_000, time.UTC)

// servedHost is the host every case dials. It is granted by nothing here —
// the grant is the caller's check and never this package's (§3, ADR-0042) —
// and it is a name rather than an address so that the handshake has a name to
// verify and the certificate has one to carry.
const servedHost = "status.hyper.dev"

// TestPerform_TheResponseObject is §12's five members, asserted against a real
// server: a real handshake, a real status line, real headers and a real parse.
// Nothing about the object is written down by the test — the fixture supplies
// what a server would, which is the only thing a fixture has any business
// supplying.
func TestPerform_TheResponseObject(t *testing.T) {
	for _, c := range []struct {
		name    string
		status  int
		headers map[string]string
		body    string
		want    string
	}{
		{
			name:    "a JSON body is parsed and carried",
			status:  200,
			headers: map[string]string{"Content-Type": "application/json"},
			body:    `{"result":{"id":"abc","count":3}}`,
			want:    `{"host":"status.hyper.dev","status":200,"headers":{"content-length":"33","content-type":"application/json","date":"Thu, 02 Apr 2026 09:41:14 GMT"},"body":{"result":{"count":3,"id":"abc"}},"tls":{"not_after":"2026-05-06T21:41:14Z","days_left":34,"subject":"CN=status.hyper.dev,O=hyper golden fixture","issuer":"CN=status.hyper.dev,O=hyper golden fixture"}}`,
		},
		{
			name:    "a body that is not JSON is absent and not an error",
			status:  503,
			headers: map[string]string{"Content-Type": "text/html"},
			body:    "<html>down</html>",
			want:    `{"host":"status.hyper.dev","status":503,"headers":{"content-length":"17","content-type":"text/html","date":"Thu, 02 Apr 2026 09:41:14 GMT"},"tls":{"not_after":"2026-05-06T21:41:14Z","days_left":34,"subject":"CN=status.hyper.dev,O=hyper golden fixture","issuer":"CN=status.hyper.dev,O=hyper golden fixture"}}`,
		},
		{
			name:    "a response carrying no body at all carries no body member",
			status:  503,
			headers: map[string]string{"Content-Type": "text/html"},
			body:    "",
			want:    `{"host":"status.hyper.dev","status":503,"headers":{"content-length":"0","content-type":"text/html","date":"Thu, 02 Apr 2026 09:41:14 GMT"},"tls":{"not_after":"2026-05-06T21:41:14Z","days_left":34,"subject":"CN=status.hyper.dev,O=hyper golden fixture","issuer":"CN=status.hyper.dev,O=hyper golden fixture"}}`,
		},
		{
			name:    "header names are lowercased and a repeated name is joined",
			status:  200,
			headers: map[string]string{"X-Trace": "one, two", "Retry-After": "30"},
			body:    "",
			want:    `{"host":"status.hyper.dev","status":200,"headers":{"content-length":"0","date":"Thu, 02 Apr 2026 09:41:14 GMT","retry-after":"30","x-trace":"one, two"},"tls":{"not_after":"2026-05-06T21:41:14Z","days_left":34,"subject":"CN=status.hyper.dev,O=hyper golden fixture","issuer":"CN=status.hyper.dev,O=hyper golden fixture"}}`,
		},
		{
			name:    "a redirect is the status it is and is never followed",
			status:  302,
			headers: map[string]string{"Location": "https://elsewhere.example.com/"},
			body:    "",
			want:    `{"host":"status.hyper.dev","status":302,"headers":{"content-length":"0","date":"Thu, 02 Apr 2026 09:41:14 GMT","location":"https://elsewhere.example.com/"},"tls":{"not_after":"2026-05-06T21:41:14Z","days_left":34,"subject":"CN=status.hyper.dev,O=hyper golden fixture","issuer":"CN=status.hyper.dev,O=hyper golden fixture"}}`,
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
				for name, value := range c.headers {
					w.Header().Set(name, value)
				}
				w.WriteHeader(c.status)
				io.WriteString(w, c.body)
			})

			object, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
				Perform(t.Context(), dial, instant)
			if err != nil {
				t.Fatalf("Perform: %v", err)
			}
			if got := encode(t, object); got != c.want {
				t.Errorf("response object:\n got:  %s\n want: %s", got, c.want)
			}
		})
	}
}

// TestPerform_NoResponseArrivedAtAll is §12's one case that reaches every
// member at once: the object is host and nothing else, and no member says what
// went wrong (ADR-0017, ADR-0050). The error beside it is narration's and never
// the object's.
func TestPerform_NoResponseArrivedAtAll(t *testing.T) {
	refused := capability.Dial(func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("connection refused")
	})

	object, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(t.Context(), refused, instant)
	if err == nil {
		t.Fatal("Perform answered no error against a host that refused the connection")
	}
	if got, want := encode(t, object), `{"host":"status.hyper.dev"}`; got != want {
		t.Errorf("response object = %s, want %s", got, want)
	}
}

// TestPerform_TheDeadlineBoundsTheCall is §3's deadline: reaching a call that
// would otherwise hang. What comes back is the object a call that got no answer
// carries, and the deadline is on the error, where narration reads it.
func TestPerform_TheDeadlineBoundsTheCall(t *testing.T) {
	// The release is deferred rather than a cleanup: the server's own Close
	// is a cleanup and waits for the handler in flight, cleanups run
	// last-registered-first, and a handler still blocked when Close runs is
	// a suite that hangs rather than a test that fails.
	release := make(chan struct{})
	defer close(release)
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		<-release
	})

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	object, err := (capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}).
		Perform(ctx, dial, instant)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Perform err = %v, want a deadline", err)
	}
	if got, want := encode(t, object), `{"host":"status.hyper.dev"}`; got != want {
		t.Errorf("response object = %s, want %s", got, want)
	}
}

// TestPerform_TheFiveReservedHeadersAreHypersOwn is §3's rule at the one
// surface that does not re-run check first: a Probe. Host is derived from the
// value the grant was checked against, Content-Type and Content-Length are
// written because hyper serialised a body, and an authored entry naming any of
// the five reaches the wire nowhere.
func TestPerform_TheFiveReservedHeadersAreHypersOwn(t *testing.T) {
	var seen *http.Request
	var body []byte
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		seen = r
	})

	call := capability.Call{
		Host:   servedHost,
		Method: http.MethodPost,
		Path:   "/v1/things",
		Headers: []capability.Parameter{
			{Name: "X-Trace", Value: "kept"},
			{Name: "content-type", Value: "application/x-www-form-urlencoded"},
			{Name: "Connection", Value: "close"},
			{Name: "Transfer-Encoding", Value: "chunked"},
			{Name: "Host", Value: "elsewhere.example.com"},
		},
		Body: []byte(`{"a":1}`),
	}
	if _, err := call.Perform(t.Context(), dial, instant); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if got, want := string(body), `{"a":1}`; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got, want := seen.Host, servedHost; got != want {
		t.Errorf("Host = %q, want %q — hyper derives it from the value the grant was checked against", got, want)
	}
	if got, want := seen.Header.Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q — hyper writes it because it serialised JSON", got, want)
	}
	if got, want := seen.ContentLength, int64(7); got != want {
		t.Errorf("Content-Length = %d, want %d", got, want)
	}
	if got, want := seen.Header.Get("X-Trace"), "kept"; got != want {
		t.Errorf("X-Trace = %q, want %q — a header hyper does not reserve is the Manifest's", got, want)
	}
	// Connection and Transfer-Encoding are not asserted from what arrived,
	// and cannot be: Go's own transport writes both, which is hyper writing
	// them, and neither is readable back as the value an author wrote. What
	// this case holds about them is that naming one is not an error and
	// changes nothing — the four assertions above are unmoved by the two
	// entries beside the two that are observable.
}

// TestBuild_TheRequestTheManifestDescribes is §3's request read and filled:
// the method, the path and query with their holes filled, the headers likewise,
// and the body serialised compact with the type ADR-0078 fixes for each
// position — a whole hole carrying its input's declared type, a composition a
// string, and a literal its YAML 1.2 core type.
func TestBuild_TheRequestTheManifestDescribes(t *testing.T) {
	request, read := capability.ReadRequest(operation(t, `
kind: mutate
deadline: 30s
http:
  method: POST
  host: "{from-target}"
  path: /client/v4/zones/{zone_id}/keys
  query:
    preview: "{description}"
    always: "1"
  headers:
    X-Trace: "trace-{description}"
  body:
    description: "{description}"
    expirySeconds: "{expiry_seconds}"
    label: "ci-{description}"
    ratio: "{ratio}"
    enabled: "{enabled}"
    capabilities:
      devices:
        create:
          reusable: false
          retries: 0755
          tags: ["tag:ci", 2592000, true]
`))
	if !read {
		t.Fatal("ReadRequest read no http: block")
	}

	call, err := request.Build(servedHost, map[string]schema.Scalar{
		"zone_id":        read1(t, schema.String, "z1"),
		"description":    read1(t, schema.String, "ci-runner"),
		"expiry_seconds": read1(t, schema.Integer, "2592000"),
		"ratio":          read1(t, schema.Number, "1.50"),
		"enabled":        read1(t, schema.Boolean, "false"),
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if got, want := call.URL(), "https://status.hyper.dev/client/v4/zones/z1/keys?preview=ci-runner&always=1"; got != want {
		t.Errorf("URL = %q, want %q", got, want)
	}
	if got, want := call.Headers[0], (capability.Parameter{Name: "X-Trace", Value: "trace-ci-runner"}); got != want {
		t.Errorf("header = %+v, want %+v", got, want)
	}
	want := `{"description":"ci-runner","expirySeconds":2592000,"label":"ci-ci-runner","ratio":1.5,"enabled":false,` +
		`"capabilities":{"devices":{"create":{"reusable":false,"retries":755,"tags":["tag:ci",2592000,true]}}}}`
	if got := string(call.Body); got != want {
		t.Errorf("body:\n got:  %s\n want: %s", got, want)
	}
}

// TestBuild_AHoleNothingSupplies is the one fault Build reports. Every
// declared input is supplied (ADR-0081), so a hole with nothing behind it is a
// Manifest check has already refused — and filling it with the empty string
// would put a request on the wire that no artefact describes.
func TestBuild_AHoleNothingSupplies(t *testing.T) {
	request, _ := capability.ReadRequest(operation(t, `
http:
  method: GET
  host: "{from-target}"
  path: /status/{id}
`))

	if _, err := request.Build(servedHost, nil); err == nil {
		t.Fatal("Build filled a hole nothing supplied")
	}
}

// TestBuild_ABodyCarryingANull is the one scalar a body: position cannot hold.
// There is no null in the vocabulary (§12), the loader refuses one at every
// position of every artefact, and a Probe re-runs no check (ADR-0009) — so this
// is the one path that can meet one, and it says so rather than inventing a
// byte: `null` and `"null"` are two different requests.
func TestBuild_ABodyCarryingANull(t *testing.T) {
	request, _ := capability.ReadRequest(operation(t, `
http:
  method: POST
  host: "{from-target}"
  path: /v1/things
  body:
    label: null
`))

	if _, err := request.Build(servedHost, nil); err == nil {
		t.Fatal("Build serialised a body carrying a null")
	}
}

// TestReadRequest_AShellOperationHasNoHTTPBlock is the reader's own absence
// rule: it judges nothing and drops what it cannot read (ADR-0064).
func TestReadRequest_AShellOperationHasNoHTTPBlock(t *testing.T) {
	if _, read := capability.ReadRequest(operation(t, "shell: {}\n")); read {
		t.Error("ReadRequest read an http: block off a shell Operation")
	}
	if _, read := capability.ReadRequest(nil); read {
		t.Error("ReadRequest read an http: block off an Operation that is not declared")
	}
}

// read1 reads one input the way a Probe's --input does, failing the test where
// the fixture itself is wrong about a type.
func read1(t *testing.T, declared schema.Type, value string) schema.Scalar {
	t.Helper()
	read, ok := schema.ReadScalar(declared, value)
	if !ok {
		t.Fatalf("the fixture's %s %q does not read as one", declared, value)
	}
	return read
}

// operation parses one Operation's declaration and answers its node, which is
// what ReadRequest is handed — the node artefact.OperationNode finds in a
// Manifest.
func operation(t *testing.T, source string) *yaml.Node {
	t.Helper()
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(source), &root); err != nil {
		t.Fatal(err)
	}
	return root.Content[0]
}

// encode is the object as the wire carries it, which is the rendering both the
// probe_result row and the page beneath it write.
func encode(t *testing.T, object capability.Object) string {
	t.Helper()
	encoded, err := json.Marshal(object)
	if err != nil {
		t.Fatal(err)
	}
	return string(encoded)
}

// serve stands one TLS server answering handler, and answers the dialer that
// reaches it: every name resolves to this listener, which is the whole of the
// isolation. The handshake, the certificate, the status line and the parse are
// all real.
func serve(t *testing.T, handler http.HandlerFunc) capability.Dial {
	t.Helper()

	certificate, pool := mintCertificate(t, servedHost)
	// Date is stamped from the case's clock rather than left to Go's, which
	// would put the machine's wall clock into every asserted response
	// object. A server answers with a Date and this one does too; what the
	// fixture fixes is which instant it names.
	stamped := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Date", instant.UTC().Format(http.TimeFormat))
		handler(w, r)
	}
	server := httptest.NewUnstartedServer(http.HandlerFunc(stamped))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.StartTLS()
	t.Cleanup(server.Close)

	address := server.Listener.Addr().String()
	return func(ctx context.Context, network, requested string) (net.Conn, error) {
		name, _, err := net.SplitHostPort(requested)
		if err != nil {
			return nil, err
		}
		// Time is the case's clock and not the machine's: the certificate
		// is minted against the instant the case is driven at, so the
		// chain it verifies against has to be read at that instant too.
		dialer := tls.Dialer{Config: &tls.Config{
			RootCAs:    pool,
			ServerName: name,
			Time:       func() time.Time { return instant },
		}}
		return dialer.DialContext(ctx, network, address)
	}
}

// mintCertificate mints a self-signed certificate for hosts, expiring
// 34 days and 12 hours after the instant above — so days_left is 34, floored
// off a real x509 chain rather than written down by a test.
func mintCertificate(t *testing.T, hosts ...string) (tls.Certificate, *x509.CertPool) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: hosts[0], Organization: []string{"hyper golden fixture"}},
		NotBefore:             instant.Add(-24 * time.Hour),
		NotAfter:              instant.Add(34*24*time.Hour + 12*time.Hour),
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
