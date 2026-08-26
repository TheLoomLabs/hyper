// Package release is the one thing the version pin ever reaches the network
// for: the published checksum of the release artefact a generated workflow
// installs (§11, ADR-0020, issue #178).
//
// **It is `hyper`'s own fetch and not an Operation.** No Capability is
// declared, no Target is bound, no credential resolves, no Journal entry is
// written and no Store is opened — `project` is not a Run, and what happens
// here is a tool reading a file its own compiled-in template names. What it
// shares with a Capability's call is the dialer alone, which is the process's
// one dialer rather than a grant: a case stands a server in the test process
// and asserts a real handshake rather than writing the answer down (§9,
// ADR-0009).
//
// **It happens attended, at review time, and lands in a diff.** Freezing the
// checksum is what converts a release tag — a mutable pointer, its asset
// replaceable after publication — into an immutable reviewed fact, and it is
// trust on first use, named rather than glossed (§11).
//
// **What it reads is the checksums file and never the artefact.** A few hundred
// bytes rather than a hundred megabytes, both being the same mutable source read
// in the same instant — so hashing bytes nothing on this machine will ever
// execute buys nothing over reading the checksum beside them (ADR-0046).
package release

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// CodeArtefactAbsent is the one error_code this package's answers reach (§12).
const CodeArtefactAbsent = "release-artefact-absent"

// Absent is `release-artefact-absent`'s three shapes as one error: no release
// under the tag, no checksums file beside it, and no line in that file for the
// artefact the compiled-in template names.
//
// The three are one code because all three are the same fact for a reader —
// there is no artefact to pin — and because the remedy for each is a released
// binary rather than an edit. Two of them are also one *answer*: a tag with no
// release and a release with no checksums file are both a request for something
// that is not there, and a second read to tell them apart would be `project`
// resolving twice to render a distinction nobody acts on (§11).
//
// What separates it from every other failure here is that the answer **arrived**,
// which is what makes it a check declining rather than the world resisting: a
// host that never responded is a fetch that did not complete, and that is
// `install`'s own rule one command over (§11, ADR-0060).
type Absent struct{ fault string }

func (a *Absent) Error() string { return a.fault }

// MaxChecksums is how much of a checksums file is read. It is a few hundred
// bytes by construction — one `sha256sum` line per published file — and the cap
// is here because nothing about a URL guarantees what is behind it: a body that
// is not the file it should be must fail as `release-artefact-absent`, not as a
// laptop reading a gigabyte into memory.
//
// It is exported for DigestIn's own reason: `install` reads a checksums file
// published beside a Manifest, which is the same kind of file this reads
// published beside a release artefact, and a second number for one fact is
// where the day comes that the two disagree (ADR-0087, internal/registry).
const MaxChecksums = 1 << 20

// Digest is the published checksum of the release artefact for version, as the
// Repository declaration spells it — the algorithm inline, `sha256:` and the hex
// beside it (§3).
//
// It answers an *Absent where the read arrived and named no artefact, and the
// transport's own error where the read did not complete — a host that never
// answered, and equally a body that stopped part way through one. Its caller
// exits on those two differently and on nothing else, so those are the two
// answers (§11).
//
// **It judges the checksum no further than the line it is on.** What the release
// published is what the declaration records, and the diff `project` writes is
// where a human reads it — a validity rule invented here would be a check no
// specification states, standing between an author and a fact they can see (§9,
// ADR-0064).
func Digest(ctx context.Context, dial capability.Dial, version string) (string, error) {
	url := workflow.ChecksumsURL(version)

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	response, err := Client(dial).Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if absent, answered := absentAt(response.StatusCode); absent {
		return "", &Absent{fmt.Sprintf("%s answered %d", url, response.StatusCode)}
	} else if !answered {
		return "", fmt.Errorf("%s answered %d", url, response.StatusCode)
	}

	published, err := io.ReadAll(io.LimitReader(response.Body, MaxChecksums))
	if err != nil {
		return "", err
	}

	name := workflow.ArtefactName(version)
	if digest, named := DigestIn(string(published), name); named {
		return digest, nil
	}
	return "", &Absent{fmt.Sprintf("%s names no %s", url, name)}
}

// absentAt sorts a status line into the two answers this package has: absent
// says the release does not hold what was asked for, and answered says the
// checksums file arrived.
//
// **`404` and `410` are the absence and every other refusal is not**, which is
// the line §12 draws through exit `77`: a Refusal promises that a verbatim retry
// refuses identically, and a `503` or a `429` from the release host promises the
// opposite — those can differ between two invocations of an identical command
// line, which is §11's own criterion for `1` and where `install` already puts
// them. Answering `release-artefact-absent` to a rate limit would tell an author
// to publish a release that is already published (§11, §12, ADR-0060).
func absentAt(status int) (absent, answered bool) {
	switch status {
	case http.StatusOK:
		return false, true
	case http.StatusNotFound, http.StatusGone:
		return true, false
	}
	return false, false
}

// DigestIn is the checksum one `sha256sum` line records for name, and false
// where no line in the file names it.
//
// The two spellings `sha256sum` writes are read alike — two spaces for a text
// read and ` *` for a binary one — because which mode a release process used is
// not a fact about the release, and a digest missed for a space would be
// `release-artefact-absent` reported against a file that names the artefact
// perfectly well.
//
// **It is exported because it is one grammar and not this package's own.** A
// checksum published beside a file, read by a tool about to trust the bytes, is
// the same fact whether the file is a release artefact or a Manifest a ref
// names — so `install` reads its `checksums.txt` through this rather than
// through a second parse of one format, which is where two readings of one line
// eventually drift apart (ADR-0087, internal/registry).
func DigestIn(published, name string) (string, bool) {
	for _, line := range strings.Split(published, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == name {
			return "sha256:" + fields[0], true
		}
	}
	return "", false
}

// Client is what a read of a published file is made with, and it is the `http`
// Capability's own client less everything a Capability needs: the threaded
// dialer wired as DialTLSContext, and keep-alives off because one read is one
// connection.
//
// **It is exported for DigestIn's own reason and no other.** `install` reads a
// Manifest and the checksums file beside it over exactly these terms — ADR-0087
// states them one by one and states them as this package's — so a second
// constructor spelling them again is where the day comes that one of them
// follows a redirect into plaintext and the other does not (internal/registry).
//
// **Redirects are followed**, which is where it parts company with a
// Capability's call. There a redirect is reach arriving from data and the grant
// was checked against one host (ADR-0029); here there is no grant and no
// authored host at all — the URL is compiled in, and the release host answers a
// download with a redirect to the store the bytes are actually in. A fetch that
// refused to follow one would resolve nothing anywhere.
//
// **Plaintext is refused wherever a redirect might reach for it.** The scheme is
// https and there is no second one (ADR-0082), so the transport is given no
// plaintext dialer and says why rather than quietly opening one.
//
// There is no timeout. No artefact declared one, and a bound written here would
// be one nobody agreed to; what a fetch that never completes costs is a command
// a human is sitting in front of, and the interrupt is theirs (§3, ADR-0014).
func Client(dial capability.Dial) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DialTLSContext:    dial,
			DisableKeepAlives: true,
			DialContext: func(context.Context, string, string) (net.Conn, error) {
				return nil, errors.New("hyper dials https and nothing else")
			},
		},
	}
}
