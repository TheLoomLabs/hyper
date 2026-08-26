package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/registry"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunInstall implements `hyper install <ref>` — the sixteenth of §9's sixteen,
// and the single point at which third-party data enters the repository (§9,
// §11, issue #187).
//
// **One act, two reads, one file.** It parses the ref against the grammar
// ADR-0087 fixes, fetches the Manifest at it, fetches `checksums.txt` in the
// ref's own directory, verifies the bytes it fetched against the line naming
// the ref's basename, and writes `providers/<basename>` with an `origin:` block
// appended recording the ref it resolved and the digest it verified. Nothing is
// written on any other path.
//
// **The ref grammar is decided with no network reached**, and an invocation
// outside it is §9's usage error: exit `2`, no `error_code`, no row stream, the
// rendering to stderr — the shape the other eight positionals already take, and
// decidable offline because that is the property ADR-0060 keeps `2` for. What a
// ref is, and why `hyper` names no registry, is ADR-0087's; what is here is
// where the grammar is applied.
//
// **It runs no static pass before writing, and that is load-bearing rather than
// an omission.** §4 states that an Extension the repository never installed
// lands at `check` as `artefact-absent` on the Definition's `provider:`, with no
// network reached — so the repository you install into is very often one that
// does not check, and the thing you are installing is the repair. A pre-write
// pass, which is `project`'s rule one command over, would make this command
// unrunnable exactly when it is wanted. It follows that `install` may write a
// file that immediately fails `check`, and that is the design: the four
// Extension codes stay `check`'s (§4, §11).
//
// **The path comes from the ref and not from the Manifest.** It is *digest
// only, never intent*: the bytes are not parsed before they are written, and a
// command reading a `provider:` key to choose a filename would be deciding on a
// parse it has no business making. `name-mismatch` already pins a Manifest's
// `provider:` to its file's basename, so a Manifest published under a filename
// that disagrees with its own name is reported by `check`, positioned, at the
// `provider:` scalar, the moment the file lands (§4, §11, ADR-0004).
//
// **It stands behind the pin gate like the other fifteen.** `project` is the
// one exemption and it is exempt for being the pin's only writer (§9, §11,
// ADR-0020).
//
// It takes the whole Process rather than environmentOnly's lookup, for
// `project`'s reason: it dials, and Dial is the member that says so. It reads
// no clock, mints no id and starts no child, and its signature says that too by
// what it never reaches for. There is no `--limit`, it naming no namespace to
// range over, and no `--dry-run`, `check` already reporting digest drift and the
// diff being the rehearsal; the three globals apply with no fourth (§9).
func RunInstall(args []string, to destination, process Process, wd, binaryVersion string) int {
	parsed, to, code := parseArgs("install", args, parameters{limit: takesNoLimit}, process.LookupEnv, to)
	if code != 0 {
		return code
	}
	// Exactly one positional, and the ref it holds read against the grammar
	// — both decided from the argument list alone and before any repository
	// is resolved, which is `provider`'s own arity rule and is what ADR-0087
	// means by *every clause is a parse*. The line the gate stands on is
	// what a command **resolves**, and here that is the ref against a
	// registry: the read is behind the gate, and the parse that decides
	// whether there is a read at all is in front of it (ADR-0060,
	// ADR-0087).
	if len(parsed.positional) != 1 {
		fmt.Fprintf(to.narrate(), "hyper install: %s\n", arityFault(parsed.positional, "ref"))
		return ExitUsage
	}
	ref, err := registry.ParseRef(parsed.positional[0])
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper install: %s\n", err)
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("install", parsed.repoDir, process.LookupEnv, wd, to.narrate())
	if code != 0 {
		return code
	}

	// The gate, before anything is dialled: a mismatched pin is 77 and no
	// request leaves the machine, which is the whole of what standing behind
	// it means for the one command that reaches the world on somebody else's
	// coordinate (§9, §11, ADR-0020).
	if code, _ := gateOnVersionPin("install", repoRoot, binaryVersion, to); code != 0 {
		return code
	}

	fetched, err := registry.Fetch(context.Background(), process.Dial, ref)
	if err != nil {
		return declineFetch(err, to)
	}

	path := repository.ProvidersDir + "/" + ref.Basename()
	if err := writeManifest(repoRoot, path, withOrigin(fetched.Bytes, ref.String(), fetched.Digest)); err != nil {
		// The file it died on, named, and the tree left as it stands —
		// `project`'s own rule, and true here for its reason: git is the
		// undo, the tree is under review, and a rollback path is code
		// that runs only when something has already gone wrong and is
		// therefore the least-tested thing in the command (§10).
		fmt.Fprintf(to.narrate(), "hyper install: %s: %s\n", path, reasonFor(err))
		return ExitProblems
	}

	rows := []render.Row{installedRow{Type: "manifest", Path: path, Digest: fetched.Digest}}
	// One row over two columns, and no empty form: a command that wrote no
	// file exited before there was a row to write (§9).
	page := func(w io.Writer, rows []render.Row) error { return render.WriteTable(w, installedColumns, rows) }
	if code := writeAnswer("install", to, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}
	return ExitClean
}

// declineFetch is what a fetch that produced no verified bytes exits with. It
// is called on a failure and never on a success, so the sort below is the whole
// of it.
//
// **There are two answers and the status line is not what sorts them.** §11
// puts *a ref the registry does not hold* and *a fetch that did not complete*
// on one code deliberately — a ref names something in a registry's namespace,
// and *matches nothing* is an answer that had to be fetched, so it can differ
// between two invocations of an identical command line and it arrives beside
// the answers that are unambiguously the world resisting. That is exit `1`, and
// building a sort inside it would be inventing a distinction the specification
// spent a paragraph collapsing (§11, ADR-0060).
//
// **`origin-digest-mismatch` is the one `77`**, and it is a check declining
// bytes that did arrive: the read completed, the digest was published, and a
// verbatim retry declines identically — the remedy is the publisher's rather
// than another attempt. It is rendered in the two-line form with no caret, the
// fault having no artefact coordinate: nothing was written, so there is no file
// and no line to point at (§8, §12).
func declineFetch(err error, to destination) int {
	var mismatch *registry.Mismatch
	if errors.As(err, &mismatch) {
		return refuse(to, artefact.CodeOriginDigestMismatch, mismatch.Error())
	}
	fmt.Fprintf(to.narrate(), "hyper install: %s\n", err)
	return ExitProblems
}

// withOrigin is the file `install` writes: the verified bytes verbatim, then
// the block naming them.
//
// **The block is appended and it is the last thing in the file.** The digest
// covers the *published bytes* — the file without the block naming them, a
// digest being unable to cover itself — and this is what fixes that byte range
// for the check that recomputes it: the file's bytes up to the start of the last
// line beginning `origin:` at column 0 (§11, ADR-0087).
//
// **One newline of `hyper`'s own is written only where the published bytes do
// not already end in one.** A publisher who omitted a trailing newline gets a
// well-formed file rather than an `origin:` welded onto their last line, and a
// publisher who did not gets their bytes back untouched — which is what keeps
// the recomputation exact rather than normalised.
//
// The ref is written as a plain scalar, which the grammar is what makes safe: a
// ref carries no space and no control character, so there is no spelling of one
// that needs quoting (ADR-0087).
//
// The other end of this seam is artefact.checkOriginDigest, which takes the
// same range back out of the tracked file and recomputes the digest over it —
// including the two candidates the newline above makes necessary. What is
// written here and what is read there are one rule, and the round trip is
// driven in one process rather than asserted as a constant typed twice
// (§11, issue #189, install_test.go).
func withOrigin(published []byte, ref, digest string) []byte {
	written := make([]byte, 0, len(published)+len(ref)+len(digest)+32)
	written = append(written, published...)
	if len(published) > 0 && published[len(published)-1] != '\n' {
		written = append(written, '\n')
	}
	return fmt.Appendf(written, "origin:\n  ref: %s\n  digest: %s\n", ref, digest)
}

// writeManifest writes the one file, creating the directory it goes in where
// the repository holds none — a repository installing its first Extension is
// the common case, and an empty directory is a thing git will not carry anyway
// (§12).
//
// The directory is the path's own rather than the constant a second time: the
// caller already composed `providers/<basename>`, and a second reading of where
// that goes is where the day comes that the two disagree.
//
// It is whole-file and always overwriting, never merging: re-installing is how
// an Extension is updated, and the diff is the review (§11).
func writeManifest(repoRoot, path string, data []byte) error {
	written := filepath.Join(repoRoot, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(written), 0o755); err != nil {
		return err
	}
	return os.WriteFile(written, data, 0o644)
}

// installedRow is `install`'s row, and the only one it writes: §9 closes it at
// the file it wrote and the digest it verified, and states two members.
//
// **The ref is not on it.** It was typed by the caller, it is in the file, and
// `provider <name>` reports it beside the Manifest's other declared facts — a
// row that carried it back would be stating the argument (§9, §11).
//
// It carries no `outcome` key and terminates in `result`, `install` not being a
// Run (§8, §9).
//
// The digest is rendered whole on both surfaces, on providerRow's own
// reasoning: ADR-0047's abbreviation is for a fact to be *recognised*, and a
// digest here is verified with `sha256sum` instead (§8, §9).
type installedRow struct {
	Type   string `json:"type"`
	Path   string `json:"path"`
	Digest string `json:"digest"`
}

// Cells is the row's line: where the file landed, and what its bytes were
// verified against.
func (r installedRow) Cells() []string { return []string{r.Path, r.Digest} }

// installedColumns is `install`'s header. The digest stands last because it is
// the widest thing on the page and nothing has to be aligned past it (§8,
// render.WriteAligned).
var installedColumns = []string{"PATH", "DIGEST"}
