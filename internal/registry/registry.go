// Package registry is `install`'s two reads: the Manifest a ref names, and the
// `checksums.txt` published beside it (§11, ADR-0087, issue #187).
//
// **What a ref is, and why `hyper` names no registry, is ADR-0087's** — an
// absolute `https://` URL naming the Manifest, the digest published in the
// ref's own directory, and the ref recorded being the one the caller typed. It
// is stated there rather than restated here: the load-bearing half of
// distribution belongs where a reader of `docs/adr/` finds it, and a package
// comment that argued it again would be a second copy to drift.
//
// **It is `hyper`'s own fetch and not an Operation**, which is
// internal/release's sentence one command over and true here for the same
// reasons: no Capability is declared, no Target is bound, no credential
// resolves, no Journal entry is written and no Store is opened. `install` is
// not a Run. What it shares with a Capability's call is the dialer alone —
// capability.Dial, threaded from cli.Process — which is what lets a case
// exercise a real handshake against a server standing in the test process (§9,
// ADR-0009).
//
// **It is internal/release's sibling and shares its reader.** The `sha256sum`
// line grammar and the client both come from there, one format having one
// parse and one set of transport terms having one place they are written
// down — threaded dialer as DialTLSContext, keep-alives off, redirects
// followed, plaintext refused, and no timeout, no artefact having declared one
// and the interrupt belonging to the human in front of the command (ADR-0014,
// ADR-0082).
//
// **What it answers is bytes and the digest they were verified against, or one
// error.** There is one sort in the whole package and it is not the status
// line: §11 puts *a ref the registry does not hold* and *a fetch that did not
// complete* on one exit code deliberately, so building a sort between them
// would be inventing a distinction the specification spent a paragraph
// collapsing. The one answer that is told apart is bytes that arrived and are
// not the bytes the publisher published (§11, ADR-0060).
package registry

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/release"
)

// CodeOriginDigestMismatch is the one error_code this package's answers reach
// (§12). §11 gives the same code to `check`, over the same fact recomputed
// offline against the tracked file — which is what will make the verification
// performed here repeatable by anyone reading the repository, long after the
// machine that performed it is gone. That half is not built (issue #189); this
// constant is named for the answer this package produces and not for it.
const CodeOriginDigestMismatch = "origin-digest-mismatch"

// MaxManifest is how much of a Manifest is read. A Manifest is a reviewed
// artefact a human reads in a diff, so the cap is generous enough that no
// honest one reaches it — and it is here for internal/release's own reason:
// nothing about a URL guarantees what is behind it, and a body that is not the
// file it should be must fail as a fetch that did not complete rather than as a
// laptop reading a gigabyte into memory (ADR-0087).
//
// It is this package's own number and the checksums file's is not: that file is
// a few hundred bytes by construction and takes internal/release's cap
// unchanged, the two reads being two different kinds of file rather than one
// read performed twice.
const MaxManifest = 1 << 22

// Ref is a ref the grammar admitted: the coordinate the caller typed, and the
// two things everything downstream derives from it.
//
// **The typed string is what is kept**, and it is what String answers. A ref
// records a location a later `install` is typed from, so a value normalised on
// the way in — a lowered host, a resolved dot segment, an escape rewritten —
// would put a coordinate the publisher never published into a tracked file
// (ADR-0087).
//
// It is a type rather than a string because the grammar is a parse and the two
// derivations are only sound behind it: ChecksumsURL cuts at the last `/`,
// which names the ref's own directory precisely because the last segment was
// established to hold no separator of its own.
type Ref struct {
	typed    string
	basename string
}

// String is the ref as the caller typed it, which is what the origin: block
// records and what a later `install` is typed from (ADR-0087).
func (r Ref) String() string { return r.typed }

// Basename is the ref's last path segment: the checksums line to read, and the
// name of the file `install` writes into providers/.
//
// **The path comes from the ref and never from the Manifest.** It is *digest
// only, never intent*, one aisle over: a command reading a `provider:` key to
// choose a filename would be deciding on a parse it has no business making, and
// `name-mismatch` already reports a Manifest published under a filename that
// disagrees with its own name (§4, §11).
func (r Ref) Basename() string { return r.basename }

// ChecksumsURL is `checksums.txt` in the ref's own directory — the ref with its
// last path segment replaced, which is the whole of the registry convention
// ADR-0087 states.
//
// It is derived from the **typed** ref rather than from wherever a redirect
// landed, for the reason the recorded ref is: the checksums file a publisher
// published sits beside the bytes they published, and the coordinate for it is
// the one that was typed.
func (r Ref) ChecksumsURL() string {
	return r.typed[:strings.LastIndex(r.typed, "/")+1] + "checksums.txt"
}

// ParseRef reads a ref against ADR-0087's grammar and answers what it is, or
// the clause it fell outside.
//
// **Every clause is a parse and nothing here reaches the network**, which is
// the property the rule exists to keep: its caller exits `2`, and ADR-0060
// keeps that code decidable without a round trip. Where the grammar ends is
// where the network begins, and everything the network answers is its caller's
// other code.
//
// The error is the whole message its caller renders, less the command name: the
// ref as typed, and the one clause it broke. One clause rather than every
// clause it breaks — a caller who typed `http://` is told the scheme is `https`
// and does not need to be told about the query string they also wrote.
func ParseRef(typed string) (Ref, error) {
	// A space or a control character is refused before anything is parsed,
	// which is the first half of *an absolute URL*: no URL carries one
	// unescaped. It is also what keeps the recorded ref a YAML scalar
	// `hyper` can write plainly into a tracked file — the block is read back
	// by the same loader every artefact goes through (§3, §4).
	if strings.IndexFunc(typed, func(r rune) bool { return r <= ' ' || r == 0x7f }) >= 0 {
		return Ref{}, refusal(typed, "a URL carries no space and no control character")
	}
	// A query and a fragment are refused on the string as typed rather than
	// on what the parse made of them: `?` opens a query and `#` opens a
	// fragment wherever either stands, so this is the clause exactly and it
	// catches the empty spellings of both (ADR-0087).
	if strings.ContainsAny(typed, "?#") {
		return Ref{}, refusal(typed, "a ref carries no query and no fragment — a ref is recorded, rendered and typed again, and either would let two callers name one set of bytes two ways")
	}

	parsed, err := url.Parse(typed)
	if err != nil {
		return Ref{}, refusal(typed, "it does not parse as a URL")
	}
	// A port is admitted and carries no meaning of its own, a registry being
	// wherever it is served; one that is not a decimal number is outside the
	// grammar rather than something to resolve, and that clause is the parse
	// above rather than a case below — net/url refuses a port that is not
	// digits, and a second reading of it here would be one that could
	// disagree (ADR-0087).
	switch {
	case parsed.Scheme != "https":
		return Ref{}, refusal(typed, "the scheme is https and there is no second one")
	case parsed.User != nil:
		return Ref{}, refusal(typed, "a ref carries no userinfo — it is written into a tracked file and read in a diff, and a credential in it is the one place hyper would write a secret down")
	case parsed.Host == "":
		return Ref{}, refusal(typed, "a ref names a host")
	}

	// The last segment is read off the **escaped** path, so that an escape
	// is judged as the character it decodes to rather than after a decode
	// has already hidden it in a path split: `%2F` is a separator, and a
	// segment carrying one is a traversal in the one command that writes a
	// path derived from a string a caller typed (ADR-0087).
	segments := strings.Split(parsed.EscapedPath(), "/")
	basename, err := url.PathUnescape(segments[len(segments)-1])
	if err != nil {
		return Ref{}, refusal(typed, "its last path segment carries an escape that does not decode")
	}
	//
	// **The filename clauses are read before the `.yaml` one**, though
	// ADR-0087 states them the other way round. Neither `.` nor `..` ends in
	// `.yaml`, so a suffix test placed first would answer every traversal
	// with *this path does not end in .yaml* — which is true and is not what
	// happened, and would leave the clause that exists to refuse a traversal
	// with nothing it is the reported reason for.
	switch {
	case basename == "." || basename == "..":
		return Ref{}, refusal(typed, "a ref's last path segment is a providers/ filename, and . and .. are not filenames")
	case strings.ContainsAny(basename, `/\`) || strings.IndexFunc(basename, func(r rune) bool { return r < ' ' || r == 0x7f }) >= 0:
		return Ref{}, refusal(typed, "a ref's last path segment is a providers/ filename, carrying no path separator and no escape that decodes to one")
	case !strings.HasSuffix(basename, ".yaml"):
		return Ref{}, refusal(typed, "a ref's path ends in .yaml — the loader reads providers/*.yaml and nothing else, so bytes landing anywhere else would never be read")
	}
	return Ref{typed: typed, basename: basename}, nil
}

// refusal is the one shape every clause above answers in: what was typed, and
// the clause it fell outside. One spelling rather than one per clause, so a
// caller reads the same sentence whichever way they got it wrong.
func refusal(typed, clause string) error {
	return fmt.Errorf("%q is not a ref: %s", typed, clause)
}

// Fetched is what a verified read answers: the published bytes exactly as they
// arrived, and the digest they were verified against.
//
// The bytes are the **published** ones — the file without the origin: block
// naming them, a digest being unable to cover itself — so what a caller writes
// is these bytes verbatim and then the block (§11, ADR-0087).
type Fetched struct {
	Bytes  []byte
	Digest string
}

// Mismatch is bytes that arrived and are not the bytes the publisher published:
// `origin-digest-mismatch`, and the one answer this package tells apart from
// every other.
//
// It is a check declining rather than the world resisting, which is what puts
// it at a different exit code from everything else here: the read completed,
// the digest was published, and a verbatim retry declines identically — the
// remedy is the publisher's rather than another attempt (§11, §12, ADR-0060).
//
// **Both digests are carried whole.** A digest is verified with `sha256sum`
// rather than recognised by eye, and an abbreviation is a value a reader has to
// go somewhere else to complete (ADR-0047).
type Mismatch struct {
	Ref       string
	Published string
	Fetched   string
}

func (m *Mismatch) Error() string {
	return fmt.Sprintf("%s published %s and answered bytes that are %s", m.Ref, m.Published, m.Fetched)
}

// Fetch performs the two reads and answers the verified bytes.
//
// **The Manifest is read first and the order is not arbitrary.** It is the read
// that establishes the registry answers at all, and a ref that names nothing is
// the common case — reading the checksums file first would spend a request
// proving a file exists beside a file that does not (ADR-0087).
//
// It answers a *Mismatch where the bytes arrived and did not verify, and an
// ordinary error everywhere else: a status line that is not `200`, a checksums
// file naming every published file but this one, a host that never answered, a
// body that stopped part way through and a body over the cap. Its caller exits
// on those two differently and on nothing else (§11).
//
// **Nothing is judged beyond the digest.** The bytes are not parsed, the
// Manifest's own name is not read, and no static pass runs here or in the
// caller: it is *digest only, never intent*, and what a Manifest declares is
// `check`'s to report the moment the file lands (§11, ADR-0004).
func Fetch(ctx context.Context, dial capability.Dial, ref Ref) (Fetched, error) {
	manifest, err := read(ctx, dial, ref.String(), MaxManifest)
	if err != nil {
		return Fetched{}, err
	}
	checksums, err := read(ctx, dial, ref.ChecksumsURL(), release.MaxChecksums)
	if err != nil {
		return Fetched{}, err
	}

	published, named := release.DigestIn(string(checksums), ref.Basename())
	if !named {
		return Fetched{}, fmt.Errorf("%s names no %s", ref.ChecksumsURL(), ref.Basename())
	}
	if fetched := digestOf(manifest); fetched != published {
		return Fetched{}, &Mismatch{Ref: ref.String(), Published: published, Fetched: fetched}
	}
	return Fetched{Bytes: manifest, Digest: published}, nil
}

// digestOf is a digest as the origin: block spells one — the algorithm inline,
// `sha256:` and the hex beside it, over the exact bytes that arrived (§3, §7).
func digestOf(data []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(data))
}

// read is one GET, bounded.
//
// **Every status but `200` is one answer**, which is §11's collapse honoured
// rather than re-litigated: `404`, `410`, `429`, `500` and a body that stopped
// part way are all *the fetch did not answer with the file*, and they exit
// together. The message names which URL was asked and what it said, because
// *the Manifest 404'd* and *the checksums file 404'd* are different acts for
// whoever has to fix it (§11, ADR-0060).
//
// **The limit is read one byte past itself**, and that is what this read has
// that internal/release's does not: a body **at** the limit is a body that
// fits, and a body over it is a failure rather than a truncation. There a
// truncated checksums file names no artefact and comes back as the absence it
// already had a code for; here truncated Manifest bytes would be verified
// against a digest they could never match, which reports a publisher's file as
// tampered with because it was large. That difference is the whole reason the
// two reads are two functions rather than one: what they share is the client
// and the line grammar, and those are imported rather than copied.
func read(ctx context.Context, dial capability.Dial, from string, limit int) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, from, nil)
	if err != nil {
		return nil, err
	}
	response, err := release.Client(dial).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s answered %d", from, response.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(response.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(body) > limit {
		return nil, fmt.Errorf("%s answered more than %d bytes", from, limit)
	}
	return body, nil
}
