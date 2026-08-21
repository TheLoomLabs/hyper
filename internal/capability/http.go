package capability

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// Scheme is the one scheme hyper requests, and there is no second one
// (ADR-0082). §12 fixes tls as present where the scheme was HTTPS, and since
// this is the only scheme, tls is present on every response that arrived and
// absent only where none did. Nothing in the authoring format chooses it: a
// hosts: grant enumerates hosts and carries no scheme, so there is no plain
// HTTP host to grant and no artefact that could name one (§3).
const Scheme = "https"

// Dial is how a connection to a host is made: the read cli.Process threads,
// and the one this package is handed rather than reaches for.
//
// It answers a connection that is already past its TLS handshake, which is
// what makes the scheme above a property of the process rather than of a
// switch: there is no plaintext path to configure because there is no
// plaintext dialer to supply. It is wired as http.Transport's DialTLSContext,
// so the certificate the peer presented is a real one off a real handshake and
// reaches the response object through http.Response.TLS.
type Dial func(ctx context.Context, network, address string) (net.Conn, error)

// holePattern matches one template hole, {name}, anywhere in a scalar's text —
// the one hole syntax in every artefact (§3, §12). It is spelled here as well
// as in internal/artefact because the two ask different questions of it: that
// package refuses a hole in a position §12 does not list, and this one fills
// the ones that survived.
var holePattern = regexp.MustCompile(`\{([^{}]+)\}`)

// reservedHeaders is the five headers hyper computes for itself — Host,
// Content-Length, Content-Type, Transfer-Encoding, Connection — reserved
// against every other writer and compared case-insensitively as an HTTP header
// name is (§3, §12).
//
// A Manifest naming one earned header-reserved at load, so nothing that
// reaches here should carry one. It is dropped here anyway, and for a reason
// rather than out of caution: a Probe re-runs no check (§9, ADR-0009), so this
// is the one path in the tool on which an authored header arrives without
// having been read against §4 first. What hyper computes wins, and the drop is
// what says so rather than the ordering of two Set calls.
var reservedHeaders = map[string]bool{
	"host": true, "content-length": true, "content-type": true,
	"transfer-encoding": true, "connection": true,
}

// Request is one Operation's http: block, read (§3): the method, the host
// template, the path, and the optional query, headers and body. It is the
// declaration and not the call — every string here may still carry template
// holes, and Build is what fills them.
//
// Query and Headers are ordered rather than mappings because a request is
// bytes: §3 fixes a body's keys in the order they were authored, and a query
// string is the same fact one position over. Nothing downstream sorts them, so
// what a Manifest wrote is what leaves.
type Request struct {
	Method  string
	Host    string
	Path    string
	Query   []Parameter
	Headers []Parameter
	// Body is the body: tree, and nil where the Operation declares none.
	Body *BodyNode
	// HostInput is the host-input: scalar naming the one input that carries
	// a whole host where the candidate set and the grant intersect to
	// several, and "" where the Operation declares none (§3, ADR-0029).
	HostInput string
}

// Parameter is one query parameter or one header: its name, and its value.
//
// The value is a template on a Request, where it may still carry holes, and it
// is what that template rendered to on a Call, where it may not. It is one
// field under one name rather than two types differing in a word, because a
// query parameter is a name and a value at both ends and the filling is Build's
// act rather than a change of subject.
type Parameter struct{ Name, Value string }

// BodyNode is one node of a body: tree — a mapping, a list, or a scalar (§3).
// It is a tree of its own rather than a yaml.Node because what a body position
// means is this package's: a literal carries its YAML 1.2 core type onto the
// wire and a hole carries its input's declared type, and neither is a fact the
// parse tree states (ADR-0078).
type BodyNode struct {
	// Kind is which of the three this node is.
	Kind BodyKind
	// Members is a mapping's entries in the order they were authored. A key
	// is always a literal string: a hole may not fill one (hole-illegal,
	// §3, §12).
	Members []BodyMember
	// Items is a list's members, in order.
	Items []BodyNode
	// Scalar is a scalar's text, and Tag the YAML tag its spelling resolved
	// to — which is what types a literal, YAML 1.2 core being what keeps the
	// Norway problem out (§3, ADR-0078).
	Scalar string
	Tag    string
}

// BodyKind is which of §3's three shapes a body node is.
type BodyKind int

// The three shapes, and nothing else: a body's top level is a mapping, and
// below it a value is a scalar, a mapping or a list (§3).
const (
	BodyScalar BodyKind = iota
	BodyMapping
	BodyList
)

// BodyMember is one entry of a body mapping: its authored key and its value.
type BodyMember struct {
	Name  string
	Value BodyNode
}

// Call is a request with every hole filled: what leaves, exactly. Host is the
// one host the candidate set and the grant intersected to, which is the value
// the Host header is derived from and the value the grant was checked against
// (§3, ADR-0029).
type Call struct {
	Host    string
	Method  string
	Path    string
	Query   []Parameter
	Headers []Parameter
	// Body is the serialised body, and nil where the Operation declares
	// none. Where it is non-nil hyper writes Content-Type and
	// Content-Length for it and nobody else may.
	Body []byte
}

// URL is the address the call reaches, and it is where the one scheme is
// written. It is a method rather than a field so that no caller can hold a
// Call whose scheme says one thing and whose host says another.
func (c Call) URL() string {
	address := url.URL{Scheme: Scheme, Host: c.Host, Path: c.Path}
	if query := c.rawQuery(); query != "" {
		address.RawQuery = query
	}
	return address.String()
}

// rawQuery writes the query string in the order the Manifest authored it,
// which url.Values.Encode does not: that sorts by name, and a Manifest's own
// order is a fact about the request the same way a body's key order is (§3).
func (c Call) rawQuery() string {
	var out strings.Builder
	for i, p := range c.Query {
		if i > 0 {
			out.WriteByte('&')
		}
		out.WriteString(url.QueryEscape(p.Name))
		out.WriteByte('=')
		out.WriteString(url.QueryEscape(p.Value))
	}
	return out.String()
}

// ReadRequest reads an Operation's http: block off the node that Operation is
// declared by, and false where it declares no legible one — a shell Operation,
// or a Manifest check has already refused. Which node that is is
// internal/artefact's to answer (artefact.OperationNode): this package knows
// what an http: block means and not where one lives.
//
// It judges nothing and drops what it cannot read, which is the rule every
// reader in internal/artefact follows: what is wrong with a Manifest is
// check's to report, and a reader that guessed would be a second opinion about
// an artefact nobody reviewed (ADR-0064).
func ReadRequest(operation *yaml.Node) (Request, bool) {
	block := mappingValue(operation, "http")
	if block == nil || block.Kind != yaml.MappingNode {
		return Request{}, false
	}

	request := Request{
		Method:    scalar(mappingValue(block, "method")),
		Host:      scalar(mappingValue(block, "host")),
		Path:      scalar(mappingValue(block, "path")),
		Query:     readParameters(mappingValue(block, "query")),
		Headers:   readParameters(mappingValue(block, "headers")),
		HostInput: scalar(mappingValue(block, "host-input")),
	}
	if body := mappingValue(block, "body"); body != nil {
		read := readBody(body)
		request.Body = &read
	}
	return request, true
}

// Build fills every hole from the Operation's resolved inputs and answers what
// leaves. host is the one host the intersection resolved to, supplied rather
// than read out of the template: the candidate set, the grant and their
// intersection are the caller's three steps and the grant is checked there
// (§3, ADR-0029, ADR-0042).
//
// An error names the one hole that could not be filled. Every hole in a
// request names an Operation input and every declared input is supplied
// (ADR-0081), so a hole with nothing behind it is a Manifest check has already
// refused — and filling it with the empty string would put a request on the
// wire that no artefact describes.
func (r Request) Build(host string, inputs map[string]schema.Scalar) (Call, error) {
	call := Call{Host: host, Method: r.Method}

	var err error
	if call.Path, err = Fill("path:", r.Path, inputs); err != nil {
		return Call{}, err
	}
	if call.Query, err = fillParameters("query:", r.Query, inputs); err != nil {
		return Call{}, err
	}
	if call.Headers, err = fillParameters("headers:", r.Headers, inputs); err != nil {
		return Call{}, err
	}
	if r.Body != nil {
		body, err := serialiseBody(*r.Body, inputs)
		if err != nil {
			return Call{}, err
		}
		call.Body = body
	}
	return call, nil
}

// Deadline bounds a call by the Operation's own `deadline:` and by nothing
// else.
// There is no whole-invocation deadline and no flag: the bound is the Manifest
// author's, declared beside the request it bounds (§3, §6).
//
// seconds is what the Operation declared, and nil where hyper could not read
// one. A call bounded by nothing is what that answers, rather than a number
// substituted here: `deadline:` is mandatory and its absence is
// `schema-mismatch`, which is check's to report and never a performer's to
// paper over (§4, ADR-0064).
//
// It lives here rather than at either caller because the deadline is a fact
// about the request: reaching one kills the call, and on a `mutate` or
// `destroy` that is the ambiguity *attempted, outcome unknown* exists to carry
// (§6). A Probe and a Run's Step both bound their call this way, and two
// spellings of one deadline is where the day comes that they differ.
//
// It is named for the thing an artefact declared and never *Bound*: a Bound is
// the maximum number of Records an effectful Step may affect (§5), and a second
// use of that word for a length of time would put the tool's one blast-radius
// noun on a clock.
func Deadline(ctx context.Context, seconds *int) (context.Context, context.CancelFunc) {
	if seconds == nil {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, time.Duration(*seconds)*time.Second)
}

// Perform makes the call and assembles the response object §12 closes at five
// members. now is the instant the invocation fixed, and it is what
// tls.days_left counts from (ADR-0034).
//
// credential is the Auth scheme's header, already composed, and the zero value
// where the Provider names no scheme. It is a parameter rather than a member of
// Call because a Call is a value a caller may hold, compare and describe, and a
// credential is none of those: it arrives at the one call that puts bytes on
// the wire and reaches nothing else (§7, ADR-0007).
//
// The object is always usable: where no response arrived at all it is host and
// nothing else, which is the answer a read records rather than a failure it
// halts on (§6, §12, ADR-0050). The error beside it says what went wrong and
// is narration's alone — no member of the object says it, that being the
// catch-all bucket ADR-0017 closed, and a surface that wrote it into one would
// be minting a sixth member.
func (c Call) Perform(ctx context.Context, dial Dial, now time.Time, credential Credential) (Object, error) {
	object := Object{{Name: MemberHost, Value: c.Host}}

	request, err := c.request(ctx, credential)
	if err != nil {
		return object, err
	}

	response, err := client(dial).Do(request)
	if err != nil {
		return object, err
	}
	defer response.Body.Close()

	object = append(object,
		Member{Name: MemberStatus, Value: response.StatusCode},
		Member{Name: MemberHeaders, Value: lowercased(response.Header)},
	)
	if body, parsed := parseBody(response.Body); parsed {
		object = append(object, Member{Name: MemberBody, Value: body})
	}
	if certificates := peerCertificates(response); len(certificates) > 0 {
		object = append(object, Member{Name: MemberTLS, Value: tlsObject(certificates[0], now)})
	}
	return object, nil
}

// request is the *http.Request the call makes, with the five reserved headers
// written by hyper and by nobody else.
//
// Host is derived from the value the Target's grant was checked against, which
// is the whole reason the header is reserved at all: an author able to write
// it could reach one host while the grant was checked against another (§3,
// ADR-0029). Content-Type and Content-Length are written because hyper
// serialised the body, and are absent where it serialised none — a GET with no
// body carries neither.
//
// The Auth scheme's header is written here too, and it is the one header on
// this request whose value never came off an artefact: a Manifest supplies the
// position and a Target declaration names the variable, and the value goes from
// the environment onto the wire without passing through anything that renders
// (§3, §12, ADR-0007).
func (c Call) request(ctx context.Context, credential Credential) (*http.Request, error) {
	var body io.Reader
	if c.Body != nil {
		body = bytes.NewReader(c.Body)
	}
	request, err := http.NewRequestWithContext(ctx, c.Method, c.URL(), body)
	if err != nil {
		return nil, err
	}

	for _, header := range c.Headers {
		if reservedHeaders[strings.ToLower(header.Name)] {
			continue
		}
		request.Header.Set(header.Name, header.Value)
	}

	// The scheme's own header, written last and by nobody else. A Manifest
	// naming a position its scheme owns is `manifest-inconsistent` and a
	// scheme naming a header hyper computes is `header-reserved`, both of
	// them check's (§4) — so this cannot collide with an authored header on
	// a path that has been reviewed, and where it could it wins, which is
	// the same rule the five reserved names are dropped under above.
	if credential.declared() {
		request.Header.Set(credential.name, credential.value)
	}

	request.Host = c.Host
	if c.Body != nil {
		request.Header.Set("Content-Type", "application/json")
		request.ContentLength = int64(len(c.Body))
	}
	return request, nil
}

// client is the one client hyper makes calls with. Two of its three settings
// are rules rather than tuning.
//
// CheckRedirect returns the 3xx rather than following it: a redirect target is
// reach arriving from data, and the grant was checked against the host the
// intersection resolved to and against no other (ADR-0029). On a read the
// status is simply what the Observation records (§6, ADR-0050).
//
// DialTLSContext is the threaded dialer, marked, which is what puts the whole
// handshake in one read and leaves this package with no TLS configuration of
// its own to get wrong — and what makes *the request provably never left* a
// fact about where a failure happened rather than about what it said (sent.go,
// ADR-0018). DisableKeepAlives is the third: one call is one
// connection, so the certificate the response reports is the certificate that
// call was answered over.
//
// There is no timeout here. A deadline belongs to the Manifest that declared
// one and arrives on the context; a second one written here would be a bound
// no artefact agreed to (§3).
func client(dial Dial) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialTLSContext:    marking(dial),
			DisableKeepAlives: true,
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// peerCertificates is the chain the peer presented, and it is empty only where
// the response arrived over something that was not TLS — which the one scheme
// above makes unreachable, and which is read for rather than asserted because
// a nil dereference is a worse answer than an absent member.
func peerCertificates(response *http.Response) []*x509.Certificate {
	if response.TLS == nil {
		return nil
	}
	return response.TLS.PeerCertificates
}

// tlsObject is §12's tls member: the expiry, the days remaining, and the two
// names off the peer's own certificate.
//
// days_left is a member because no artefact could compute one — there is no
// arithmetic in the format (ADR-0022) — and what it counts from is the instant
// the invocation fixed rather than the machine's clock (ADR-0034). It is
// floored rather than rounded: a certificate with fourteen hours left has zero
// whole days left, and a reader comparing greater_than 0 is asking whether
// there is a day in hand.
func tlsObject(certificate *x509.Certificate, now time.Time) Object {
	expiry := certificate.NotAfter.UTC()
	return Object{
		{Name: MemberNotAfter, Value: expiry.Format(time.RFC3339)},
		{Name: MemberDaysLeft, Value: int(math.Floor(expiry.Sub(now).Hours() / 24))},
		{Name: MemberSubject, Value: certificate.Subject.String()},
		{Name: MemberIssuer, Value: certificate.Issuer.String()},
	}
}

// lowercased is §12's headers member: a mapping of header name to value with
// the names lowered, because a header name is case-insensitive on the wire and
// a path is exact, so the lowering is what makes one path mean one thing.
//
// A name a response repeated is joined with ", ", which is the wire's own rule
// for combining field lines rather than a choice made here: §12 fixes one
// value per name, and dropping either of two would be answering a question
// with half of what came back.
func lowercased(header http.Header) map[string]string {
	lowered := make(map[string]string, len(header))
	for name, values := range header {
		lowered[strings.ToLower(name)] = strings.Join(values, ", ")
	}
	return lowered
}

// parseBody is §12's body member: the parsed JSON body, and absent where the
// response carried none or carried something else. Its absence is not an
// error — a site that is down answers with no body at all, and an uptime check
// is pointed at hosts that answer in HTML (§3, §12).
//
// Numbers are held as their literals rather than round-tripped through a
// float64, which is the rule the Store already writes them under: an integer
// past a float64's exact range is a Record identity on plenty of upstreams,
// and one that moved under a re-encode would mint a version on every Run (§7).
//
// *Carried something else* is decided by the parse and never by Content-Type.
// The header is what a server claims and the bytes are what it sent, and an API
// answering JSON under `text/plain` is one an author would otherwise have to
// route around with an artefact that cannot say so. The cost is the other
// direction and it is negligible: a response labelled HTML whose whole body is
// `123` parses as a number, and what a projection then reads off it is nothing,
// there being no member to name.
func parseBody(body io.Reader) (any, bool) {
	raw, err := io.ReadAll(body)
	if err != nil || len(bytes.TrimSpace(raw)) == 0 {
		return nil, false
	}

	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded any
	if err := decoder.Decode(&decoded); err != nil {
		return nil, false
	}
	if decoder.More() {
		// Two values where a body carries one is not JSON either, and a
		// reader that took the first would be describing half of what
		// came back as the whole of it.
		return nil, false
	}
	return decoded, true
}

// Fill renders one template against the resolved inputs: every hole replaced
// by the text form §12 fixes for the type its input declares, and the rest of
// the value left exactly as authored. position names the key in the error, so
// a Manifest with a hole nothing fills says which of its lines it was.
//
// Every position but body: is text on the wire, so there is nothing to type
// into here and a composition and a whole hole render identically — which is
// the difference §3 draws at the one sink that has types (ADR-0078).
//
// It is exported for the one hole outside a request that fills the same way: a
// `record:`'s `identity:` where it is a template rather than a response path
// resolves against the same inputs by the same rule (§3, §12). There is one
// hole syntax in every artefact, so there is one filler.
func Fill(position, template string, inputs map[string]schema.Scalar) (string, error) {
	var unfilled string
	filled := holePattern.ReplaceAllStringFunc(template, func(hole string) string {
		name := hole[1 : len(hole)-1]
		input, supplied := inputs[name]
		if !supplied {
			unfilled = name
			return hole
		}
		return input.Text()
	})
	if unfilled != "" {
		return "", fmt.Errorf("%s names an input %q that nothing supplied", position, unfilled)
	}
	return filled, nil
}

// fillParameters is fill over a query: or headers: mapping, whose values are
// text on the wire always — which is not a rule this format imposes but one
// the wire does (§3).
func fillParameters(position string, parameters []Parameter, inputs map[string]schema.Scalar) ([]Parameter, error) {
	if len(parameters) == 0 {
		return nil, nil
	}
	filled := make([]Parameter, 0, len(parameters))
	for _, p := range parameters {
		value, err := Fill(position+p.Name, p.Value, inputs)
		if err != nil {
			return nil, err
		}
		filled = append(filled, Parameter{Name: p.Name, Value: value})
	}
	return filled, nil
}

// serialiseBody writes the body: tree as compact JSON — no insignificant
// whitespace, keys in the order they were authored, and each value carrying the
// type ADR-0078 fixes for it (§3).
func serialiseBody(node BodyNode, inputs map[string]schema.Scalar) ([]byte, error) {
	var out bytes.Buffer
	if err := writeBody(&out, node, inputs); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// writeBody is serialiseBody's recursion, and the whole of what compact means
// here: no insignificant whitespace anywhere, a mapping's keys written in the
// order they were authored rather than sorted, and every scalar written by the
// rule below. It is the one encoder in the tool that is not the Store's, and it
// differs from it in exactly those two respects — the Store sorts keys and
// indents, because a git diff of a Record version is read by a human and a
// request body is read by an API (§3, §7, ADR-0079).
func writeBody(out *bytes.Buffer, node BodyNode, inputs map[string]schema.Scalar) error {
	switch node.Kind {
	case BodyMapping:
		out.WriteByte('{')
		for i, member := range node.Members {
			if i > 0 {
				out.WriteByte(',')
			}
			name, err := CompactJSON(member.Name)
			if err != nil {
				return err
			}
			out.Write(name)
			out.WriteByte(':')
			if err := writeBody(out, member.Value, inputs); err != nil {
				return err
			}
		}
		out.WriteByte('}')
		return nil
	case BodyList:
		out.WriteByte('[')
		for i, item := range node.Items {
			if i > 0 {
				out.WriteByte(',')
			}
			if err := writeBody(out, item, inputs); err != nil {
				return err
			}
		}
		out.WriteByte(']')
		return nil
	default:
		token, err := bodyScalar(node, inputs)
		if err != nil {
			return err
		}
		out.WriteString(token)
		return nil
	}
}

// bodyScalar is ADR-0078 in one function: the three things a scalar in a body:
// position can be, and the type each carries onto the wire.
//
//   - A hole that is the **whole** of the value carries the declared type of
//     the input it resolves to — {type: integer} reaches the wire as a JSON
//     number.
//   - A hole beside other characters is a composition, which has no meaning
//     but a string, and the input is rendered into it in its text form.
//   - A literal carries its YAML 1.2 core type: false is a JSON boolean,
//     2592000 a JSON number, "2592000" a JSON string.
//
// The boundary is on the line the reviewer reads and so is its cost: a stray
// space changes the type on the wire and nothing else changes with it.
func bodyScalar(node BodyNode, inputs map[string]schema.Scalar) (string, error) {
	if hole := holePattern.FindStringSubmatchIndex(node.Scalar); hole != nil && hole[0] == 0 && hole[1] == len(node.Scalar) {
		name := node.Scalar[hole[2]:hole[3]]
		input, supplied := inputs[name]
		if !supplied {
			return "", fmt.Errorf("body: names an input %q that nothing supplied", name)
		}
		return input.JSON(), nil
	}
	if strings.Contains(node.Scalar, "{") {
		composed, err := Fill("body:", node.Scalar, inputs)
		if err != nil {
			return "", err
		}
		return jsonString(composed), nil
	}

	switch node.Tag {
	case "!!null":
		// There is no null in the scalar vocabulary (§12), and the loader
		// refuses one at every position of every artefact. A Probe re-runs
		// no check (ADR-0009), so this is the one path that can meet one:
		// it says so rather than inventing a byte, a body carrying `null`
		// and a body carrying `"null"` being two different requests.
		return "", fmt.Errorf("body: carries a null, and there is no null in the scalar vocabulary")
	case "!!int", "!!float":
		text, ok := store.NumberText(node.Scalar)
		if !ok {
			return "", fmt.Errorf("body: carries %q, which YAML read as a number and JSON will not", node.Scalar)
		}
		return text, nil
	case "!!bool":
		// YAML 1.2 core spells a boolean true or false, with its case
		// variants and nothing else, so the JSON token is the same word
		// lowered (§3).
		return strings.ToLower(node.Scalar), nil
	default:
		return jsonString(node.Scalar), nil
	}
}

// jsonString is one text as a JSON string, which is what every body position
// that is not a number, a boolean or a typed hole writes: a literal spelled
// as a string, and a composition, which has no meaning but a string (§3,
// ADR-0078). It reads the text at a string position rather than quoting it
// here, so the escaping is the one rule internal/schema already holds.
func jsonString(text string) string {
	read, _ := schema.ReadScalar(schema.String, text)
	return read.JSON()
}

// readBody reads a body: tree off the parse tree. A node that is neither a
// mapping nor a sequence is a scalar, which is the reading a document check
// has already refused still gets: nothing here judges, and a body that is not
// a body writes bytes nobody will read (ADR-0064).
func readBody(node *yaml.Node) BodyNode {
	switch node.Kind {
	case yaml.MappingNode:
		read := BodyNode{Kind: BodyMapping}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			read.Members = append(read.Members, BodyMember{Name: key.Value, Value: readBody(value)})
		}
		return read
	case yaml.SequenceNode:
		read := BodyNode{Kind: BodyList}
		for _, item := range node.Content {
			read.Items = append(read.Items, readBody(item))
		}
		return read
	default:
		return BodyNode{Kind: BodyScalar, Scalar: node.Value, Tag: node.Tag}
	}
}

// readParameters reads a query: or headers: mapping into the order it was
// authored in.
func readParameters(node *yaml.Node) []Parameter {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var read []Parameter
	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]
		if key.Kind != yaml.ScalarNode || value.Kind != yaml.ScalarNode {
			continue
		}
		read = append(read, Parameter{Name: key.Value, Value: value.Value})
	}
	return read
}

// mappingValue is the value one key of a mapping holds, and nil where the node
// is not a mapping or does not carry that key.
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if k := node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// scalar is a node's text, and "" where the node is absent or is not a plain
// scalar.
func scalar(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}
