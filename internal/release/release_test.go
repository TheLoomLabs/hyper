package release_test

import (
	"context"
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
	"github.com/TheLoomLabs/hyper/internal/release"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

const version = "1.4.0"

// published is what a release's `checksums.txt` looks like: `sha256sum`'s own
// output, one line per file, the artefact this platform fetches among them.
var published = strings.Join([]string{
	"3a1f0b6c0d9e4a7b8c5d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b  hyper-" + version + "-aarch64-linux.tar.gz",
	"9f2c1b7a4e6d038c5b1f92a7de40cb83f5710e2d9a6c4b83fe012d75c9a4e6b1  " + workflow.ArtefactName(version),
	"",
}, "\n")

// TestDigest_FreezesTheLineNamingTheArtefact is the one network read the pin
// ever makes: the checksums file under the release tag, and the line in it
// naming the artefact the compiled-in template resolves to (§11).
func TestDigest_FreezesTheLineNamingTheArtefact(t *testing.T) {
	var asked string
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		asked = "https://" + r.Host + r.URL.Path
		fmt.Fprint(w, published)
	})

	got, err := release.Digest(context.Background(), dial, version)
	if err != nil {
		t.Fatalf("Digest() = %v, want the published checksum", err)
	}
	if want := "sha256:9f2c1b7a4e6d038c5b1f92a7de40cb83f5710e2d9a6c4b83fe012d75c9a4e6b1"; got != want {
		t.Errorf("Digest() = %q, want %q — the algorithm inline, as the declaration spells it", got, want)
	}
	if want := workflow.ChecksumsURL(version); asked != want {
		t.Errorf("it asked for %q, want %q", asked, want)
	}
}

// TestDigest_NoChecksumsFileUnderTheTagIsAbsent is three of the code's shapes
// arriving as one answer: a tag with no release under it, a release with no
// checksums file beside it and a release nobody may read unauthenticated are
// all `404` on a fetch that carries no credential, and the answer arrived,
// which is what makes it a check declining (§11, ADR-0007, ADR-0127).
func TestDigest_NoChecksumsFileUnderTheTagIsAbsent(t *testing.T) {
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Not Found", http.StatusNotFound)
	})

	_, err := release.Digest(context.Background(), dial, version)

	var absent *release.Absent
	if !errors.As(err, &absent) {
		t.Fatalf("Digest() = %v, want a %s", err, release.CodeArtefactAbsent)
	}
	for _, want := range []string{workflow.ChecksumsURL(version), "404"} {
		if !strings.Contains(absent.Error(), want) {
			t.Errorf("the fault is %q, want it to name %q", absent, want)
		}
	}
	if absent.Arrived() {
		t.Errorf("Arrived() = true, want false: nothing was read, and a private release answers this way too (#254)")
	}
}

// TestDigest_NoLineForTheArtefactIsAbsent is the shape that arrives as its own
// answer: the file is there and names no artefact for the platform `runs-on`
// fixes, which is a disagreement between the release and the template the
// binary holds and cannot be argued out of. It is also the only absence here
// that was observed rather than inferred, which is what Arrived reports and
// what parts the two remedies (§11, ADR-0127).
func TestDigest_NoLineForTheArtefactIsAbsent(t *testing.T) {
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "3a1f0b6c0d9e4a7b8c5d2e1f0a9b8c7d6e5f4a3b2c1d0e9f8a7b6c5d4e3f2a1b  hyper-"+version+"-aarch64-linux.tar.gz")
	})

	_, err := release.Digest(context.Background(), dial, version)

	var absent *release.Absent
	if !errors.As(err, &absent) {
		t.Fatalf("Digest() = %v, want a %s", err, release.CodeArtefactAbsent)
	}
	if !strings.Contains(absent.Error(), workflow.ArtefactName(version)) {
		t.Errorf("the fault is %q, want it to name the artefact it looked for", absent)
	}
	if !absent.Arrived() {
		t.Errorf("Arrived() = false, want true: the file was read, so this absence was observed rather than inferred (#254)")
	}
}

// TestDigest_AHostThatRefusedForItsOwnReasonsIsNotAbsent is the line exit `77`
// draws: a Refusal promises that a verbatim retry refuses identically, and a
// rate limit or a bad gateway from the release host promises the opposite. They
// arrive beside the answers that are unambiguously the world resisting, and
// `release-artefact-absent` would tell an author to publish a release that is
// already published (§11, §12, ADR-0060).
func TestDigest_AHostThatRefusedForItsOwnReasonsIsNotAbsent(t *testing.T) {
	for _, status := range []int{http.StatusTooManyRequests, http.StatusServiceUnavailable, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			dial := serve(t, func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, http.StatusText(status), status)
			})

			_, err := release.Digest(context.Background(), dial, version)

			if err == nil {
				t.Fatalf("Digest() = nil, want the %d reported", status)
			}
			var absent *release.Absent
			if errors.As(err, &absent) {
				t.Errorf("Digest() = %v, want it not to claim %s: a verbatim retry may answer differently", err, release.CodeArtefactAbsent)
			}
		})
	}
}

// TestDigest_AFetchThatDidNotCompleteIsNotAbsent is the line §11 draws through
// this one read: an answer that arrived is a check declining, and a host that
// never answered is the world resisting — two different exits, so they must be
// two different errors here (§11, ADR-0060).
func TestDigest_AFetchThatDidNotCompleteIsNotAbsent(t *testing.T) {
	refusing := capability.Dial(func(context.Context, string, string) (net.Conn, error) {
		return nil, errors.New("connect: connection refused")
	})

	_, err := release.Digest(context.Background(), refusing, version)

	if err == nil {
		t.Fatal("Digest() = nil, want the dial's own failure")
	}
	var absent *release.Absent
	if errors.As(err, &absent) {
		t.Errorf("Digest() = %v, want it not to claim %s: nothing answered", err, release.CodeArtefactAbsent)
	}
}

// serve stands one TLS server answering handler and answers the dialer that
// reaches it. Every name resolves to this listener, which is the whole of the
// isolation: the handshake, the certificate, the status line and the read are
// all real, and only the resolution is a fixture.
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
