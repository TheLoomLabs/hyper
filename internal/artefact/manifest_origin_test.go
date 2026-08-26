package artefact

import (
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/schema"
)

// installed is what `hyper install` writes: the published bytes verbatim, then
// the block naming them — the digest taken over the file *without* that block,
// since a digest cannot cover itself (§11).
//
// recorded is passed rather than derived so that a case can put a digest beside
// bytes it does not cover, which is what an edited Manifest is. The digest is
// composed here rather than checked in as a constant: a constant goes stale
// silently, and an edit to `uptime` above would leave every case below passing
// for the wrong reason.
func installed(published, recorded string) string {
	return published + "origin:\n  ref: https://providers.example.com/acme/uptime.yaml\n  digest: " + recorded + "\n"
}

// TestCheckManifest_AnInstalledManifestUntouchedIsClean is the passing case,
// and it stands first because it is the one the failing case below is only
// meaningful against: the bytes `install` verified, recomputed offline by a
// second reader of the same file, agree.
func TestCheckManifest_AnInstalledManifestUntouchedIsClean(t *testing.T) {
	doc := installed(uptime, ManifestDigest([]byte(uptime)))
	mustNone(t, checkManifest(t, "providers/uptime.yaml", doc))
}

// TestCheckManifest_APublishedManifestWithNoTrailingNewlineIsClean is the
// second candidate, and it is exactness rather than laxity: `install` writes one
// newline of its own where the published bytes do not end in one, so the prefix
// recomputed here is a byte longer than what the publisher published. A
// publisher who omitted a trailing newline is not punished for it.
func TestCheckManifest_APublishedManifestWithNoTrailingNewlineIsClean(t *testing.T) {
	published := strings.TrimSuffix(uptime, "\n")
	doc := installed(published+"\n", ManifestDigest([]byte(published)))
	mustNone(t, checkManifest(t, "providers/uptime.yaml", doc))
}

// TestCheckManifest_AManifestWhoseBytesMovedIsAMismatch is the claim §11 rests
// the whole mechanism on, made checkable: the published half of the file is a
// byte different from what the block records, which is the direction every edit
// to an installed Manifest arrives from.
func TestCheckManifest_AManifestWhoseBytesMovedIsAMismatch(t *testing.T) {
	moved := strings.Replace(uptime, "deadline: 10s", "deadline: 90s", 1)
	doc := installed(moved, ManifestDigest([]byte(uptime)))

	got := checkManifest(t, "providers/uptime.yaml", doc)
	p := mustCode(t, got, CodeOriginDigestMismatch)
	if p.Field != "origin.digest" {
		t.Errorf("Field = %q, want origin.digest — the next act is an edit at the digest: scalar", p.Field)
	}
	// Both digests whole, neither abbreviated: this is the one row a reader
	// reaches for sha256sum over (§8, ADR-0047).
	for what, digest := range map[string]string{
		"recomputed": ManifestDigest([]byte(moved)),
		"recorded":   ManifestDigest([]byte(uptime)),
	} {
		if !strings.Contains(p.Message, digest) {
			t.Errorf("Message = %q, want it to name the %s digest %s whole", p.Message, what, digest)
		}
	}
}

// TestCheckManifest_TheCitationIsTheDigestScalar holds where the problem
// points: the `digest:` scalar's own line and column, so the caret §8 renders
// sits on the value a reader replaces.
func TestCheckManifest_TheCitationIsTheDigestScalar(t *testing.T) {
	doc := installed(uptime, "sha256:"+strings.Repeat("0", 64))
	got := checkManifest(t, "providers/uptime.yaml", doc)
	p := mustCode(t, got, CodeOriginDigestMismatch)

	lines := strings.Split(doc, "\n")
	if p.Line < 1 || p.Line > len(lines) {
		t.Fatalf("Line = %d, which is outside a file of %d lines", p.Line, len(lines))
	}
	if cited := lines[p.Line-1]; !strings.HasPrefix(strings.TrimSpace(cited), "digest:") {
		t.Fatalf("Line = %d, which is %q; want the digest: line", p.Line, cited)
	}
	if want := len("  digest: ") + 1; p.Column != want {
		t.Errorf("Column = %d, want %d — the scalar rather than its key", p.Column, want)
	}
}

// TestCheckManifest_AManifestWithNoOriginBlockIsClean is the first of the three
// files this check never reaches, and it is what dropping the block buys an
// author who deliberately modified an installed Manifest: a locally authored
// Provider, checked like any other and making no digest claim (§11, ADR-0073).
func TestCheckManifest_AManifestWithNoOriginBlockIsClean(t *testing.T) {
	mustNone(t, checkManifest(t, "providers/uptime.yaml", uptime))
}

// TestCheckBuiltinShellProvider_ReachesTheDigestCheckNever is the second: a
// built-in ships inside the binary, carries no block and makes no claim against
// a registry — and it is held to that by the pass it runs rather than by an
// exemption inside one (§11, ADR-0039).
func TestCheckBuiltinShellProvider_ReachesTheDigestCheckNever(t *testing.T) {
	mustNoCode(t, CheckBuiltinShellProvider(), CodeOriginDigestMismatch)
}

// TestCheckManifest_AMalformedOriginBlockIsSchemaMismatchAlone is the third:
// the schema requires both members where the block is present, so a block
// missing one has already earned its row and a second opinion here would put
// two on the page for one fault (§4, ADR-0064).
func TestCheckManifest_AMalformedOriginBlockIsSchemaMismatchAlone(t *testing.T) {
	doc := uptime + "origin:\n  digest: sha256:" + strings.Repeat("0", 64) + "\n"
	got := checkManifest(t, "providers/uptime.yaml", doc)
	mustCode(t, got, schema.CodeMismatch)
	mustNoCode(t, got, CodeOriginDigestMismatch)
}
