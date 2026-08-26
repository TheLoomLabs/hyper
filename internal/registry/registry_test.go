package registry_test

import (
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/registry"
	"github.com/TheLoomLabs/hyper/internal/release"
)

// The ref every case here is written against, and the two URLs it resolves to:
// the Manifest where the caller pointed, and `checksums.txt` in that ref's own
// directory (ADR-0087).
const (
	ref           = "https://providers.example.com/acme/dns.yaml"
	checksumsURL  = "https://providers.example.com/acme/checksums.txt"
	basename      = "dns.yaml"
	publishedYAML = "kind: provider\nprovider: dns\nschema-version: 1\n"
)

// TestParseRef_TheGrammarAdmits is what a ref is: an absolute `https://` URL
// naming a Manifest, with a port that carries no meaning of its own and a path
// as deep as the publisher laid it out (ADR-0087).
func TestParseRef_TheGrammarAdmits(t *testing.T) {
	for _, admitted := range []struct{ typed, basename, checksums string }{
		{ref, basename, checksumsURL},
		{"https://providers.example.com/dns.yaml", "dns.yaml", "https://providers.example.com/checksums.txt"},
		{"https://providers.example.com:8443/acme/dns.yaml", "dns.yaml", "https://providers.example.com:8443/acme/checksums.txt"},
		{"https://providers.example.com/acme/v2/preview-dns.yaml", "preview-dns.yaml", "https://providers.example.com/acme/v2/checksums.txt"},
	} {
		t.Run(admitted.typed, func(t *testing.T) {
			parsed, err := registry.ParseRef(admitted.typed)
			if err != nil {
				t.Fatalf("ParseRef(%q) = %v, want the ref admitted", admitted.typed, err)
			}
			if parsed.String() != admitted.typed {
				t.Errorf("String() = %q, want %q — the recorded ref is what was typed", parsed.String(), admitted.typed)
			}
			if parsed.Basename() != admitted.basename {
				t.Errorf("Basename() = %q, want %q", parsed.Basename(), admitted.basename)
			}
			if parsed.ChecksumsURL() != admitted.checksums {
				t.Errorf("ChecksumsURL() = %q, want %q — the ref's last segment replaced", parsed.ChecksumsURL(), admitted.checksums)
			}
		})
	}
}

// TestParseRef_EveryClauseIsAParse is the whole grammar refusing, and the
// property the refusals share is that no clause reaches the network: a ref
// outside the grammar is decidable offline, which is what exit `2` is kept for
// (ADR-0060, ADR-0087).
func TestParseRef_EveryClauseIsAParse(t *testing.T) {
	for _, outside := range []string{
		"",
		"http://providers.example.com/acme/dns.yaml",
		"providers.example.com/acme/dns.yaml",
		"/etc/hyper/dns.yaml",
		"./dns.yaml",
		"https:///acme/dns.yaml",
		"https://user:token@providers.example.com/acme/dns.yaml",
		"https://providers.example.com:https/acme/dns.yaml",
		"https://providers.example.com/acme/dns.yml",
		"https://providers.example.com/acme/dns.yaml.sig",
		"https://providers.example.com/acme/",
		"https://providers.example.com",
		"https://providers.example.com/acme/..",
		"https://providers.example.com/acme/.",
		"https://providers.example.com/acme/%2e%2e%2fdns.yaml",
		"https://providers.example.com/acme/sub%2Fdns.yaml",
		"https://providers.example.com/acme/dns.yaml?token=t",
		"https://providers.example.com/acme/dns.yaml#origin",
		"https://providers.example.com/acme/dns dns.yaml",
	} {
		t.Run(outside, func(t *testing.T) {
			if parsed, err := registry.ParseRef(outside); err == nil {
				t.Errorf("ParseRef(%q) = %q, want it outside the grammar", outside, parsed.String())
			}
		})
	}
}

// TestFetch_ReadsTheManifestThenTheChecksumsBesideIt is the order ADR-0087
// fixes and the reason it is not arbitrary: the Manifest is the read that
// establishes the registry answers at all, and a ref naming nothing is the
// common case.
func TestFetch_ReadsTheManifestThenTheChecksumsBesideIt(t *testing.T) {
	var asked []string
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		asked = append(asked, "https://"+r.Host+r.URL.Path)
		answer(w, r, publishedYAML, textLine(publishedYAML, basename))
	})

	fetched, err := fetch(t, dial, ref)
	if err != nil {
		t.Fatalf("Fetch() = %v, want the verified bytes", err)
	}
	if string(fetched.Bytes) != publishedYAML {
		t.Errorf("Fetch() answered %q, want the published bytes verbatim", fetched.Bytes)
	}
	if want := digestOf(publishedYAML); fetched.Digest != want {
		t.Errorf("Fetch() verified against %q, want %q", fetched.Digest, want)
	}
	if want := []string{ref, checksumsURL}; !equal(asked, want) {
		t.Errorf("it asked for %q, want %q — the Manifest first", asked, want)
	}
}

// TestFetch_ReadsBothSpellingsOfASha256sumLine is the one grammar
// internal/release already holds, read here rather than parsed a second time:
// two spaces for a text read and ` *` for a binary one, because which mode a
// publisher used is not a fact about what they published (ADR-0087).
func TestFetch_ReadsBothSpellingsOfASha256sumLine(t *testing.T) {
	for name, line := range map[string]string{
		"text":   textLine(publishedYAML, basename),
		"binary": binaryLine(publishedYAML, basename),
	} {
		t.Run(name, func(t *testing.T) {
			dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
				answer(w, r, publishedYAML, line)
			})

			fetched, err := fetch(t, dial, ref)
			if err != nil {
				t.Fatalf("Fetch() = %v, want the published checksum read", err)
			}
			if want := digestOf(publishedYAML); fetched.Digest != want {
				t.Errorf("Fetch() verified against %q, want %q", fetched.Digest, want)
			}
		})
	}
}

// TestFetch_BytesThatDoNotMatchArePutBack is the one answer this package sorts
// out from every other: bytes that arrived and are not the bytes the publisher
// published. Both digests are carried whole, a digest being verified with
// `sha256sum` rather than recognised by eye (§11, ADR-0047).
func TestFetch_BytesThatDoNotMatchArePutBack(t *testing.T) {
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		answer(w, r, publishedYAML, textLine("something else entirely\n", basename))
	})

	_, err := fetch(t, dial, ref)

	var mismatch *registry.Mismatch
	if !errors.As(err, &mismatch) {
		t.Fatalf("Fetch() = %v, want a %s", err, registry.CodeOriginDigestMismatch)
	}
	for _, want := range []string{digestOf(publishedYAML), digestOf("something else entirely\n")} {
		if !strings.Contains(mismatch.Error(), want) {
			t.Errorf("the fault is %q, want it to name %q whole", mismatch, want)
		}
	}
}

// TestFetch_ARefTheRegistryDoesNotHoldIsNotAMismatch is §11's own collapse read
// from this side: *matches nothing* and *the fetch did not complete* are one
// answer, and the only thing this package sorts out of them is bytes that
// arrived and did not verify (§11, ADR-0060).
func TestFetch_ARefTheRegistryDoesNotHoldIsNotAMismatch(t *testing.T) {
	for name, world := range map[string]http.HandlerFunc{
		"the Manifest 404s": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Not Found", http.StatusNotFound)
		},
		"the checksums file 404s": func(w http.ResponseWriter, r *http.Request) {
			if strings.HasSuffix(r.URL.Path, "checksums.txt") {
				http.Error(w, "Not Found", http.StatusNotFound)
				return
			}
			fmt.Fprint(w, publishedYAML)
		},
		"the checksums file names every file but this one": func(w http.ResponseWriter, r *http.Request) {
			answer(w, r, publishedYAML, textLine(publishedYAML, "other.yaml"))
		},
		"the registry rate limits": func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := fetch(t, serve(t, world), ref)

			if err == nil {
				t.Fatal("Fetch() = nil, want the read reported")
			}
			var mismatch *registry.Mismatch
			if errors.As(err, &mismatch) {
				t.Errorf("Fetch() = %v, want it not to claim %s: no bytes were put beside a published digest", err, registry.CodeOriginDigestMismatch)
			}
		})
	}
}

// TestFetch_ABodyOverTheCapIsAFetchThatDidNotComplete is the cap on both reads,
// and the two values differ: a checksums file is a few hundred bytes by
// construction and takes internal/release's own, and a Manifest takes one
// generous enough that no honest Manifest reaches it. Nothing about a URL
// guarantees what is behind it (ADR-0087).
func TestFetch_ABodyOverTheCapIsAFetchThatDidNotComplete(t *testing.T) {
	for name, world := range map[string]http.HandlerFunc{
		"the Manifest": func(w http.ResponseWriter, r *http.Request) {
			answer(w, r, strings.Repeat("k", registry.MaxManifest+1), textLine(publishedYAML, basename))
		},
		"the checksums file": func(w http.ResponseWriter, r *http.Request) {
			answer(w, r, publishedYAML, strings.Repeat("k", release.MaxChecksums+1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := fetch(t, serve(t, world), ref)

			if err == nil {
				t.Fatal("Fetch() = nil, want the body reported as a read that did not complete")
			}
			var mismatch *registry.Mismatch
			if errors.As(err, &mismatch) {
				t.Errorf("Fetch() = %v, want it not to claim %s", err, registry.CodeOriginDigestMismatch)
			}
		})
	}
}

// fetch parses the ref and performs the two reads, which is what every case
// here drives: a ref outside the grammar never reaches this package's network
// half at all.
func fetch(t *testing.T, dial capability.Dial, typed string) (registry.Fetched, error) {
	t.Helper()

	parsed, err := registry.ParseRef(typed)
	if err != nil {
		t.Fatalf("ParseRef(%q) = %v, want the ref admitted", typed, err)
	}
	return registry.Fetch(context.Background(), dial, parsed)
}

// answer serves the Manifest at the ref and the checksums file beside it, which
// is what a static file host does and the whole of what a registry is (§11).
func answer(w http.ResponseWriter, r *http.Request, manifest, checksums string) {
	if strings.HasSuffix(r.URL.Path, "/checksums.txt") {
		fmt.Fprint(w, checksums)
		return
	}
	fmt.Fprint(w, manifest)
}

// textLine and binaryLine are `sha256sum`'s two spellings of one line: two
// spaces for a text read, and ` *` for a binary one.
func textLine(content, name string) string {
	return fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(content)), name)
}

func binaryLine(content, name string) string {
	return fmt.Sprintf("%x *%s\n", sha256.Sum256([]byte(content)), name)
}

// digestOf is a digest as the origin: block spells one — the algorithm inline,
// `sha256:` and the hex beside it (§3).
func digestOf(content string) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(content)))
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

// serve stands one TLS server answering handler and answers the dialer that
// reaches it — internal/release's own fixture, one command over. Every name
// resolves to this listener, which is the whole of the isolation: the
// handshake, the certificate, the status line and the read are all real, and
// only the resolution is a fixture.
func serve(t *testing.T, handler http.HandlerFunc) capability.Dial {
	t.Helper()

	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)

	pool := x509.NewCertPool()
	pool.AddCert(server.Certificate())
	address := server.Listener.Addr().String()

	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		dialer := tls.Dialer{Config: &tls.Config{RootCAs: pool, ServerName: "example.com"}}
		return dialer.DialContext(ctx, network, address)
	}
}
