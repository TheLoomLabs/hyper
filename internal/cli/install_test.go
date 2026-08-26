package cli_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/cli"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// The things about `hyper install` that no case directory can state (§11, §12,
// issue #188), beside the two `project` has one command over.
//
// Everything else it does is a golden: what it fetched and what landed is
// testdata/install's `tree.golden`, what it reported is the two streams beside
// it, and every way the two reads can fail to produce verified bytes is a case
// of its own. What is here is what a corpus cannot reach — a write the
// filesystem refuses part-way through, and an **absence of egress** — because a
// case directory says what a repository and a registry hold and not what the
// disk will do with them, and a connection nobody attempted is not a byte any
// golden renders.
//
// The third test is the corpus read back rather than driven: `install`'s whole
// code set, held against the three §11 states it as.

// The ref every case here is written against, its basename, and the bytes the
// registry beneath it publishes — the corpus's own ref, so that a reader moving
// between the two is reading about one coordinate (testdata/install/README.md,
// ADR-0087).
const (
	installRef      = "https://providers.example.com/acme/dns.yaml"
	installHost     = "providers.example.com"
	installBasename = "dns.yaml"
	installManifest = "kind: provider\nprovider: dns\nschema-version: 1\nclass: cloudflare\n" +
		"capabilities: [http]\noperations:\n  list_zones:\n    kind: read\n    deadline: 30s\n" +
		"    http: {method: GET, host: \"{from-target}\", path: /zones}\n" +
		"    input: {type: object, properties: {}}\n" +
		"    record: {identity: $.id, fields: {id: $.id}}\n"
)

// installRepository writes the least `install` needs to stand in: a Repository
// declaration pinning the version this binary is, and nothing else.
//
// It is deliberately not a copy of a corpus fixture, on projectRepository's own
// footing: a case here is about a filesystem and about a dial that never
// happened, and the repository it stands on only has to clear the pin gate.
// `providers/` is absent, which is the state `install` creates the directory
// out of and the one every case below starts from.
func installRepository(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	declaration := "kind: repository-declaration\nversion: " + defaultVersion +
		"\ndigest: sha256:" + strings.Repeat("0", 64) + "\n"
	if err := os.WriteFile(filepath.Join(root, repository.DeclarationPath), []byte(declaration), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// standRegistry stands one TLS server publishing installManifest at every path
// but `checksums.txt`, and the line naming it there — which is the whole of
// what a registry is, and the same world the corpus's `serve/` entries stand
// (§11, ADR-0087).
//
// **It is the corpus's own world and not a second one.** The certificate is
// minted by golden_serve_test.go's own minter, over the ref's host and against
// the clock the stand-in process answers, and the dial verifies the name the
// caller asked for — so the handshake, the chain, the status line and the digest
// computed over bytes that crossed a socket are all real here exactly as they
// are there, and only the name resolution is a fixture. A server whose name the
// dial declined to check would be a second fixture, weaker than the one every
// case beside it stands in.
func standRegistry(t *testing.T) capability.Dial {
	t.Helper()

	certificate, pool := mintFixtureCertificate(t, fixedInstant, []string{installHost})
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/checksums.txt") {
			fmt.Fprintf(w, "%x  %s\n", sha256.Sum256([]byte(installManifest)), installBasename)
			return
		}
		fmt.Fprint(w, installManifest)
	}))
	server.TLS = &tls.Config{Certificates: []tls.Certificate{certificate}}
	server.StartTLS()
	t.Cleanup(server.Close)

	listening := server.Listener.Addr().String()
	return func(ctx context.Context, network, requested string) (net.Conn, error) {
		name, _, err := net.SplitHostPort(requested)
		if err != nil {
			return nil, err
		}
		dialer := tls.Dialer{Config: &tls.Config{
			RootCAs:    pool,
			ServerName: name,
			Time:       func() time.Time { return fixedInstant },
		}}
		return dialer.DialContext(ctx, network, listening)
	}
}

// TestRunInstall_AWriteThatFailsNamesTheFileItDiedOn is the verified bytes
// arriving and the filesystem declining them: exit `1`, the path on stderr, and
// the tree left as it stands.
//
// It is `project`'s own rule one command over and it holds here for that
// command's reasons — git is the undo, the tree is under review, and a rollback
// path is code that runs only when something has already gone wrong and is
// therefore the least-tested thing in the command (§10, §11).
//
// **It is exit `1` and never the `77`.** The bytes verified; what declined them
// was the disk. `origin-digest-mismatch` promises that a verbatim retry Refuses
// identically, and a path that is a directory this morning is a path somebody
// clears this afternoon — so this is the world resisting, which is where every
// other way `install` fails to finish already lives (§12, ADR-0060).
//
// The failure is arranged by standing a **directory** where the file goes,
// which is a write no permission bit has to be set for and one that behaves the
// same way for every account the suite might run as.
func TestRunInstall_AWriteThatFailsNamesTheFileItDiedOn(t *testing.T) {
	root := installRepository(t)
	wanted := repository.ProvidersDir + "/" + installBasename
	if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(wanted)), 0o755); err != nil {
		t.Fatal(err)
	}

	p := &process{wd: root}
	invocation := p.value()
	invocation.Dial = standRegistry(t)
	var stdout, stderr bytes.Buffer
	exit := cli.Main([]string{"install", "--repo-dir", root, installRef}, &stdout, &stderr, invocation, testFacts)

	if exit != cli.ExitProblems {
		t.Errorf("exit = %d, want %d — the world resisted a write", exit, cli.ExitProblems)
	}
	if !strings.Contains(stderr.String(), wanted) {
		t.Errorf("stderr = %q, want it to name %q, the file the write died on", stderr.String(), wanted)
	}
	// Once, and in the repository's own vocabulary — `project`'s rule, and
	// the same reason: os names the file absolutely in every *os.PathError
	// it hands back, and one fault reported as two files is worse than
	// either of them alone (§9).
	if strings.Contains(stderr.String(), root) {
		t.Errorf("stderr = %q, want the path named once and repo-relative, not again as %q", stderr.String(), root)
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout = %q, want it silent: there is no row for a file that did not land", stdout.String())
	}
	// And the tree is as it was: what stood in the way still stands, and
	// nothing was written beside it.
	held, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(wanted)))
	if err != nil {
		t.Fatalf("the directory the write died on is gone: %v", err)
	}
	if len(held) != 0 {
		t.Errorf("%s holds %d entries, want the tree left exactly as it stood", wanted, len(held))
	}
}

// TestRunInstall_AMalformedRefDialsNothing is ADR-0087's *every clause is a
// parse* asserted as an **absence of egress**: a ref outside the grammar is
// exit `2`, and the machine never reached for the network to decide it.
//
// It is the property exit `2` is kept for. `1` is where a ref the registry does
// not hold lives, precisely because *matches nothing* is an answer that had to
// be fetched; `2` stays decidable offline, and a command that dialled before
// reading its own argument would have collapsed the two (§11, ADR-0060).
//
// The corpus states half of it already — every one of `install`'s `usage-*`
// cases carries no `serve/` at all, and the harness fails a case that dials
// without one. What it cannot state is the count: the assertion there is
// carried by a directory that is not present, and a fixture that stopped
// refusing dials would take the claim with it silently. Here the dial is
// counted, which is the only way an absence is asserted rather than assumed.
//
// The repository beneath it is a real one that would clear the gate, so a ref
// refused here is refused for being outside the grammar and for no second
// reason.
func TestRunInstall_AMalformedRefDialsNothing(t *testing.T) {
	// The clauses spread across what they refuse: a scheme, a shape that is
	// no URL at all, a last segment that is not a filename, and a component
	// that would put a token into a tracked file.
	for _, typed := range []string{
		"http://providers.example.com/acme/dns.yaml",
		"providers.example.com/acme/dns.yaml",
		"https://providers.example.com/acme/..",
		"https://providers.example.com/acme/dns.yaml?token=t",
	} {
		t.Run(typed, func(t *testing.T) {
			root := installRepository(t)

			p := &process{wd: root}
			var stdout, stderr bytes.Buffer
			exit := cli.Main([]string{"install", "--repo-dir", root, typed}, &stdout, &stderr, p.value(), testFacts)

			if exit != cli.ExitUsage {
				t.Errorf("exit = %d, want %d — the grammar decided it", exit, cli.ExitUsage)
			}
			if p.dial != 0 {
				t.Errorf("a host was dialled %d times, want none: a ref outside the grammar is decided with no network reached", p.dial)
			}
			if stdout.Len() != 0 {
				t.Errorf("stdout = %q, want it silent: a usage error opens no row stream", stdout.String())
			}
			if !strings.Contains(stderr.String(), typed) {
				t.Errorf("stderr = %q, want it to quote %q, the ref as it was typed", stderr.String(), typed)
			}
			// And nothing was written, which here means the command never
			// reached a repository at all: `providers/` is still absent.
			if _, err := os.Stat(filepath.Join(root, repository.ProvidersDir)); !os.IsNotExist(err) {
				t.Errorf("stat %s = %v, want it never created", repository.ProvidersDir, err)
			}
		})
	}
}

// installCodeSet is what `hyper install` can exit with: the three §11 closes it
// at — `2` for an invocation the ref grammar rejects, `1` for a ref the registry
// does not hold **or** a fetch that did not complete, and `77` for
// `origin-digest-mismatch` — beside the `0` every command that did what it was
// asked exits with.
//
// **The one that is two states is `1`, and that is the whole design.** A ref
// names something in a registry's namespace, so *matches nothing* is an answer
// that had to be fetched: it can differ between two invocations of an identical
// command line, it is unavailable offline, and it arrives beside the answers
// that are unambiguously the world resisting. §11 spent a paragraph collapsing
// the distinction, and a fourth code here would be it reinvented (§11,
// ADR-0060).
//
// It is spelled out rather than derived from the corpus, on this suite's own
// footing everywhere else: a set read off the files it is held against would
// grow to fit whatever landed.
var installCodeSet = []int{cli.ExitClean, cli.ExitProblems, cli.ExitUsage, cli.ExitRefused}

// TestInstallCorpus_ItsWholeCodeSetIsThree holds that set over `install`'s
// corpus in both directions: no case exits a code the command does not answer,
// and every member has a case that reaches it.
//
// Both halves are needed and neither is the other. A binary that grew a fifth
// answer — `75` for a registry that might be there next time, `130` for a fetch
// somebody interrupted — would land a case whose golden says so, and the first
// half is what notices. A corpus that lost the mismatch case would leave `77`
// asserted by nothing at all, and the second half is what notices that.
//
// It reads the corpus rather than driving it, through the walk the other
// assertions over checked-in exit codes already make: a case is found by the
// shape of its directory, and what its command is is the case's own to say.
//
// **What it can hold is every path a case reaches, which is not the same as
// every path.** The two tests above this one are the rest: a write the
// filesystem refused is the `1` no case directory can arrange, and a malformed
// ref is the `2` asserted beside a dial count. Between the three there is no way
// out of this command that nothing watches.
func TestInstallCorpus_ItsWholeCodeSetIsThree(t *testing.T) {
	reached := map[int]int{}
	forEachGoldenTriple(t, func(dir string, exit int) {
		if filepath.Dir(dir) != installCorpus {
			return
		}
		if !slices.Contains(installCodeSet, exit) {
			t.Errorf("case %s exits %d; `install` answers %v and nothing else", dir, exit, installCodeSet)
		}
		reached[exit]++
	})

	for _, code := range installCodeSet {
		if reached[code] == 0 {
			t.Errorf("no case under %s/ exits %d; that answer is asserted by nothing", installCorpus, code)
		}
	}
}

// TestRunInstall_TheRoundTripIsOneCase is the mechanism end to end, in one
// process and with no fixture standing in for either half: `install` fetches,
// verifies and writes; `check` recomputes the digest it recorded and finds
// nothing; one byte of the published half moves and `check` reports
// origin-digest-mismatch on it (§11, issues #187, #189).
//
// **It is one case rather than three because the claim is a round trip.** The
// two halves are asserted apart already — the write is testdata/install's
// `tree.golden` and the check is testdata/check/origin-digest-mismatch, each
// with a digest checked in beside it — and what neither of them can say is that
// the digest one wrote is the digest the other recomputes. A corpus can only
// hold a constant a human typed; a constant typed twice agrees with itself,
// which is not the same thing as two readers of one file agreeing about it.
//
// **The edit is one byte and it is inside the published half**, which is what
// makes the second `check` about drift rather than about a malformed file: the
// Manifest still parses, still names itself, still declares what it declared,
// and the only thing wrong with it is that it is not what was fetched.
func TestRunInstall_TheRoundTripIsOneCase(t *testing.T) {
	root := installRepository(t)
	installed := filepath.Join(root, filepath.FromSlash(repository.ProvidersDir+"/"+installBasename))

	p := &process{wd: root}
	invocation := p.value()
	invocation.Dial = standRegistry(t)
	var stdout, stderr bytes.Buffer
	if exit := cli.Main([]string{"install", "--repo-dir", root, installRef}, &stdout, &stderr, invocation, testFacts); exit != cli.ExitClean {
		t.Fatalf("install exit = %d, want %d: %s", exit, cli.ExitClean, stderr.String())
	}

	// Untouched, the file checks clean — and it is a `check` over the whole
	// repository rather than a call into the check, so what is asserted is
	// the pass a reader of this repository would run (§9).
	stdout.Reset()
	stderr.Reset()
	if exit := cli.Main([]string{"check", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts); exit != cli.ExitClean {
		t.Fatalf("check exit = %d, want %d — an installed Manifest nobody has touched: %s", exit, cli.ExitClean, stdout.String())
	}

	held, err := os.ReadFile(installed)
	if err != nil {
		t.Fatal(err)
	}
	moved := bytes.Replace(held, []byte("deadline: 30s"), []byte("deadline: 90s"), 1)
	if bytes.Equal(held, moved) {
		t.Fatal("the edit changed nothing; the published Manifest no longer holds the line this case moves")
	}
	if err := os.WriteFile(installed, moved, 0o644); err != nil {
		t.Fatal(err)
	}

	stdout.Reset()
	stderr.Reset()
	exit := cli.Main([]string{"check", "--repo-dir", root}, &stdout, &stderr, p.value(), testFacts)
	if exit != cli.ExitProblems {
		t.Fatalf("check exit = %d, want %d — the file moved after its block was written", exit, cli.ExitProblems)
	}
	if got := stdout.String(); !strings.Contains(got, artefact.CodeOriginDigestMismatch) {
		t.Errorf("check reported %q, want %s", got, artefact.CodeOriginDigestMismatch)
	}
	// The digest `install` verified, named whole by the check that declined
	// it: the two ends of the round trip spelling one value.
	if got := stdout.String(); !strings.Contains(got, artefact.ManifestDigest([]byte(installManifest))) {
		t.Errorf("check reported %q, want it to name the digest install verified, %s, whole",
			got, artefact.ManifestDigest([]byte(installManifest)))
	}
}
