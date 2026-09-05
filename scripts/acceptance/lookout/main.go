// Command lookout is the fixture API the lookout acceptance tasks point a
// Provider Manifest at (issues #227, #255, #268 and #271). It mints a certificate, listens on a free loopback port, and
// serves a small JSON API behind a bearer token until it is killed. Nothing in
// `hyper` knows it exists: it is reached the way any vendor's API is, over TLS,
// through a Manifest an agent authored.
//
// **One service, three worlds.** `-fixture` names the initial state it starts in,
// and each of those states is a task's fiction: which monitors are already
// there, and which services will not answer the first look a create takes at
// them. The states and the argument for each are in api.go; nothing else about
// the service varies between them, so a Manifest written against one is a
// Manifest that works against the others.
//
// **Why any of this is here** is [ADR-0105](../../../docs/adr/0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md).
// A task that asks for a Manifest needs something for that Manifest to talk to,
// and the choice was between a public endpoint, a local one, and neither. A
// real vendor's API would put a live credential in a headless session running
// under `--permission-mode bypassPermissions`; an echo service is safe because
// nothing it accepts means anything; and either one buys a second variable —
// a rate limit, a schema change, an outage — into an experiment run a handful
// of times a year. What is not negotiable is that the trust be unreachable from
// where the agent writes: no artefact carries a root, a pin or a verification
// mode, so the sealed session trusts this certificate through `SSL_CERT_FILE`
// in the `hyper` process's environment and a Manifest that works here is one
// that would work against a vendor.
//
// **It is Go rather than a script**, and that is a rule rather than a taste.
// `scripts/acceptance/run.sh` declares the tools it needs — `bwrap git go
// python3` — and the fence asserts the same four, so a fifth would be an edit
// to the seam this task is required not to make. `python3` is declared and
// cannot mint a certificate; `go` is declared, already builds the binary under
// test, and buys one thing more: `go build ./...` compiles this file on every
// change, where a script in another language is text nothing reads until a
// sealed run.
//
// **The port comes from the kernel** because a fixed one is a port another
// process on this machine may already hold, and a task that failed on it would
// have failed for a reason that is not the task's. It costs no comparability
// between transcripts: `host:` is `"{from-target}"`, so the port lives in the
// Target declaration the setup script writes and appears in nothing the agent
// authors.
//
// # The shape of the API, and why it is awkward
//
// The fixture's shape is ours, which is the cost ADR-0105 records paying: an
// agent authoring a Manifest against an API we designed is graded against our
// own idea of one, and an API that fit §3 too neatly would flatter the
// transcript. So it is awkward exactly where real APIs are awkward, and each of
// the four is here to be survived rather than admired:
//
//   - **An envelope key.** Everything lands under `data`, so no path a
//     projection carries starts at the thing it is about.
//   - **A collection under a name.** The list is `data.monitors`, not `data`,
//     so `over:` names a member of a member.
//   - **An identity that is not the name.** A monitor is handled by an opaque
//     `ref` and describes a `service`, and which of the two a Record is named
//     by is the author's choice rather than the API's.
//   - **A create whose answer differs from an element of the list.** `POST`
//     answers a single object under `data.monitor` with a `state` of its own, so
//     a projection written off the list does not resolve against the create.
//
// A fifth is not awkwardness and is here for a different reason. **One route
// answers a value the service will not answer again** — a monitor's push
// credential, minted by a `POST` of its own — which is the class ADR-0007 names
// as not re-readable and the only thing this fixture offers that a Manifest
// would mark `secret:` (issue #271). Everything else here comes back as often as
// it is asked for.
//
// Three further facts of it are load-bearing rather than decorative. The list
// pages at two, so an agent that does not reach for a pagination Pattern sees
// half the monitors and concludes the other half are missing — and creating one
// that exists is refused `409`, which halts an effectful Step (§6). `window`
// is validated as a whole number of seconds, so a Manifest that sends the
// integer as a string — the composition rule ADR-0078 states, a stray character
// beside a hole — is refused `400` rather than quietly accepted. And a monitor
// is looked at as soon as it is added, so a service that does not answer that
// first look leaves a `201` in the caller's hands and nothing in the list —
// which is the only way a monitor `hyper` created is gone before `hyper`
// deleted it, and therefore the only way to a `404` on a `destroy` from inside
// the seal (issue #255). None of the three is announced as a trap anywhere:
// they are properties of an API, and the documentation the setup scripts ship
// describes them the way a vendor's would.
package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"log"
	"maps"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// What the fixture is, in numbers. The page size is small because it has to be
// reached by a list of four; the window bounds are a real service's ordinary
// sanity check, and they are what turns *checked every minute* into a value an
// author has to convert rather than copy. The validity is the certificate's, and
// it is a number chosen here rather than by a vendor's rotation, which is what
// makes `tls.days_left` a known quantity to anyone reading a transcript.
const (
	pageSize  = 2
	minWindow = 30
	maxWindow = 3600
	validity  = 30 * 24 * time.Hour
)

func main() {
	dir := flag.String("dir", "", "directory to write the certificate and the report into")
	name := flag.String("fixture", "", "the named initial state to serve")
	flag.Parse()
	if *dir == "" {
		log.Fatal("lookout: -dir is required")
	}
	// Named rather than defaulted, and fatal rather than empty (issue #255). A
	// default would be one task's world served to a task that forgot to say
	// which it wanted, and an unknown name that started an empty service would
	// be a transcript about a fixture nobody wrote. What a task passes is the
	// name; what the arrangement is *for* is the comment beside it in api.go.
	world, known := fixtures()[*name]
	if !known {
		log.Fatalf("lookout: -fixture is one of %s", strings.Join(slices.Sorted(maps.Keys(fixtures())), ", "))
	}
	// Absolute, because the report is read by a script and the path in it is
	// handed to a process with a working directory of its own.
	directory, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("lookout: %v", err)
	}

	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		log.Fatalf("lookout: %v", err)
	}
	api := world.serve(hex.EncodeToString(token))

	pair, root, err := mint()
	if err != nil {
		log.Fatalf("lookout: %v", err)
	}
	path := filepath.Join(directory, "lookout.pem")
	if err := os.WriteFile(path, root, 0o644); err != nil {
		log.Fatalf("lookout: %v", err)
	}

	// The kernel picks the port on the v4 address and the v6 listener is asked
	// for that same one, because `localhost` is what the Target grants and what
	// it resolves to is the machine's business: a host that answers on ::1 and
	// listens on 127.0.0.1 only is one whose first connection depends on a
	// resolver's ordering. A machine with no IPv6 loopback is not a failure —
	// there is nothing there to answer on.
	v4, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("lookout: %v", err)
	}
	port := v4.Addr().(*net.TCPAddr).Port
	server := &http.Server{
		Handler:   api,
		TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
	}
	if v6, err := net.Listen("tcp", net.JoinHostPort("::1", strconv.Itoa(port))); err == nil {
		go server.ServeTLS(v6, "", "")
	}

	// Written last and written atomically, so that a reader who finds this file
	// has a server behind it: the setup script waits for it rather than for a
	// sleep, and a rename is what makes *the file is there* and *the file is
	// complete* one fact (issue #227).
	report := fmt.Sprintf("port=%d\ncertificate=%s\ntoken=%s\n", port, path, api.token)
	if err := write(filepath.Join(directory, "lookout.report"), report); err != nil {
		log.Fatalf("lookout: %v", err)
	}
	log.Printf("lookout: listening on localhost:%d, certificate %s", port, path)
	log.Fatalf("lookout: %v", server.ServeTLS(v4, "", ""))
}

// write puts content at path through a temporary file and a rename, which is
// the whole of what makes the report's presence mean the report is readable.
func write(path, content string) error {
	temporary := path + ".partial"
	if err := os.WriteFile(temporary, []byte(content), 0o644); err != nil {
		return err
	}
	return os.Rename(temporary, path)
}

// mint makes the certificate the sealed session comes to trust. It is its own
// root, self-signed, valid for thirty days and carrying both spellings of the
// loopback in its SANs. A two-level chain buys a fixture nothing, and one
// certificate is one file for the setup script to name in `SSL_CERT_FILE`
// (ADR-0105). The validity is a number chosen here rather than one a vendor's
// rotation decided, which is what makes `tls.days_left` — the member ADR-0082
// spends its argument on — a known quantity to anyone reading a transcript.
func mint() (tls.Certificate, []byte, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	template := &x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: "lookout acceptance fixture"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(validity),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"localhost"},
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	root := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	parsed, err := x509.ParseCertificate(der)
	if err != nil {
		return tls.Certificate{}, nil, err
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key, Leaf: parsed}, root, nil
}
