package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"debug/buildinfo"
	"debug/macho"
	"encoding/binary"
	"encoding/hex"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/release"
	"github.com/TheLoomLabs/hyper/internal/workflow"
)

// released is the version these cases publish under. It is not this binary's:
// what a release process does is a function of the version it is handed, and a
// case that published the running version would agree with the templates for
// whichever reason the running version happens to agree with them.
const released = "1.4.0"

// The Mach-O spellings of what a signature is. `codesign -dv` renders the pair
// as `flags=0x20002(adhoc,linker-signed)`, which is the string ADR-0133 read
// off the Macs and the number this reads off the file.
const (
	lcCodeSignature      = 0x1d    // LC_CODE_SIGNATURE
	csAdhoc              = 0x0002  // CS_ADHOC — signed by nobody, with no identity
	csLinkerSigned       = 0x20000 // CS_LINKER_SIGNED — written by the linker rather than by codesign
	adhocLinkerSigned    = csAdhoc | csLinkerSigned
	superBlobMagic       = 0xfade0cc0 // CSMAGIC_EMBEDDED_SIGNATURE
	codeDirectoryMagic   = 0xfade0c02 // CSMAGIC_CODEDIRECTORY
	codeDirectoryFlagsAt = 12         // magic, length, version, then flags
)

// TestRelease_PublishesTheTwoFilesTheBinaryNames is the other half of the pin
// mechanism, exercised without a tag: the release script writes the artefact
// the compiled-in template names and a `checksums.txt` beside it, and the line
// in that file is one internal/release reads and agrees with (§11, ADR-0020,
// issue #191).
//
// It runs the script the tag runs rather than a transcription of it, which is
// the only way this case can fail when the release changes. The platform is
// named so that a laptop pays for one build rather than the set — what the
// case is about is the shape of the publication, and the set is the release's
// (docs/build/releasing.md).
func TestRelease_PublishesTheTwoFilesTheBinaryNames(t *testing.T) {
	published := publish(t, released, "x86_64-linux")

	name := workflow.ArtefactName(released)
	artefact := filepath.Join(published, name)
	if _, err := os.Stat(artefact); err != nil {
		t.Fatalf("the release published no %s: %v", name, err)
	}

	checksums, err := os.ReadFile(filepath.Join(published, "checksums.txt"))
	if err != nil {
		t.Fatalf("the release published no checksums.txt beside it: %v", err)
	}

	digest, named := release.DigestIn(string(checksums), name)
	if !named {
		t.Fatalf("checksums.txt is\n%s\nand names no %s; `project` would Refuse %s against this release", checksums, name, release.CodeArtefactAbsent)
	}
	if want := digestOf(t, artefact); digest != want {
		t.Errorf("checksums.txt records %s for %s, want %s — the digest `project` freezes is the artefact's own", digest, name, want)
	}
}

// TestRelease_TheArtefactHoldsTheBinaryTheInstallStepInvokes pins the layout
// the generated workflow's install step depends on: it untars into the
// checkout and invokes `./hyper`, so the archive holds that one file at its
// root and nothing else (§10, §11, internal/workflow).
func TestRelease_TheArtefactHoldsTheBinaryTheInstallStepInvokes(t *testing.T) {
	published := publish(t, released, "x86_64-linux")

	archive, err := os.Open(filepath.Join(published, workflow.ArtefactName(released)))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	unzipped, err := gzip.NewReader(archive)
	if err != nil {
		t.Fatal(err)
	}

	var held []string
	entries := tar.NewReader(unzipped)
	for {
		entry, err := entries.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		held = append(held, entry.Name)
		if entry.Name == "hyper" && entry.FileInfo().Mode()&0o111 == 0 {
			t.Errorf("the archive's hyper is mode %v, want it executable — the install step runs it", entry.FileInfo().Mode())
		}
	}

	if len(held) != 1 || held[0] != "hyper" {
		t.Errorf("the archive holds %v, want exactly [hyper] — `tar -xzf` runs in the checkout root", held)
	}
}

// TestRelease_TheTagRunsTheScriptTheseCasesRun is what keeps the three above
// honest. They prove what the script publishes; this proves that a tag reaches
// the same script rather than a second copy of its commands, and that both
// files it wrote are uploaded to one release (§11, issue #191).
//
// It reads the steps and never the file, a workflow's own comments being the
// one place every string below could appear while the job did none of it. The
// reading is `suite_test.go`'s, which is where the two workflows' shared rules
// are held — one parser rather than two that drift (issue #243).
func TestRelease_TheTagRunsTheScriptTheseCasesRun(t *testing.T) {
	runs := runsOf(workflowOf(t, "release.yml"))
	if len(runs) == 0 {
		t.Fatal("the release workflow runs no step at all")
	}

	for _, want := range []string{"scripts/release.sh", "gh release create", "checksums.txt"} {
		if !slices.ContainsFunc(runs, func(run string) bool { return strings.Contains(run, want) }) {
			t.Errorf("no step of the release workflow runs %q; its steps are %q", want, runs)
		}
	}
}

// TestRelease_TheArtefactCarriesTheBinaryTheTagNames runs what the release
// published. The script spells the linker's symbol itself — a fourth reading of
// the one flag, beside the declaration, this package's own cases and
// docs/build/releasing.md — and a symbol the linker does not recognise is
// ignored without complaint, so a typo there publishes a release whose binary
// names a version nobody chose (§11, ADR-0020, issue #191).
//
// **What it would name is the checkout's, which is why this case runs where it
// does.** Since issue #263 a build the flag did not reach reports the module
// version Go recorded instead of `unknown`
// ([ADR-0138](../../docs/adr/0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)),
// and in a clean checkout standing exactly on the release tag that is the same
// string the flag would have written — so a dead flag is invisible there and
// costs nothing, both answers being the tag. Anywhere else, which is every
// checkout a contributor or this case runs in, it is a pseudo-version or a
// `+dirty` one and this comparison fails as it always did.
//
// It builds for the platform it is running on, which is the only one it can
// run, and skips where that is not a platform the release publishes — what the
// case is about is the stamp rather than the set.
func TestRelease_TheArtefactCarriesTheBinaryTheTagNames(t *testing.T) {
	platform := hostPlatform(t)
	published := publish(t, released, platform)

	stdout, _, exit := run(t, unpack(t, archiveOf(published, released, platform)), "version")

	if exit != 0 {
		t.Fatalf("hyper version exit = %d, want 0", exit)
	}
	if want := "hyper " + released + "\n"; !strings.HasPrefix(stdout, want) {
		t.Errorf("the released binary reports %q, want it to start %q — the release script stamped nothing", stdout, want)
	}
}

// TestRelease_EveryArchiveOfOneReleaseIsStampedFromACleanTree holds the fact
// the four archives of `v0.0.1-alpha` broke: `hyper version`'s second line
// appends `-dirty` where Go stamped `vcs.modified`, so that a hash on the page
// is never a claim about bytes edited after it — and three of the four
// published archives made that claim about a tree nobody had edited (issue
// #261, ADR-0133, ADR-0136).
//
// **It builds more than one platform, which is the whole of why it can see it.**
// The release wrote its own dirt: the first build's archive landed in the
// output directory, Go's stamping saw an untracked file in the checkout, and
// every build after the first was stamped from a tree the first had modified.
// One platform is always the first one, so every case above this one passes on
// a release that publishes three false pages.
//
// **And it publishes into the working tree, which is the other half.** The
// directory the release script is handed is where the dirt appears; handed a
// `t.TempDir()` outside the checkout there is none to see. `dist` inside the
// repository root is what `release.yml` hands it and what
// docs/build/releasing.md tells a person to hand it, so a directory of the
// case's own beside them is the honest place to reproduce from. An empty
// directory is not a modification — `git status` reports nothing for one — so
// the tree the builds are stamped against is the tree this case found.
//
// **A tree that already carries edits cannot show it**, because then every
// build is stamped `vcs.modified=true` honestly and the ordering makes no
// difference. That is a property of the checkout rather than of the machine, so
// it goes through `unavailable`: a laptop mid-change skips and says why, and a
// runner — where `actions/checkout` supplies exactly one tree and a release is
// cut from it — fails, this being the case that stands in front of publishing.
func TestRelease_EveryArchiveOfOneReleaseIsStampedFromACleanTree(t *testing.T) {
	if edits := worktreeEdits(t); edits != "" {
		unavailable(t, "this working tree carries edits, so every build in it is stamped vcs.modified=true whatever order the release builds in:\n%s", edits)
	}

	// The two the fault was reproduced on, and the cheapest pair: what the
	// case is about is that a second build exists, not which one it is.
	platforms := []string{"x86_64-linux", "aarch64-linux"}
	into := publishInto(t, directoryInTheWorktree(t), released, platforms...)

	for _, platform := range platforms {
		flag := modifiedFlagOf(t, unpack(t, archiveOf(into, released, platform)))
		if flag != "false" {
			t.Errorf("the %s archive of a release cut from a clean tree is stamped vcs.modified=%s, want false — `hyper version` appends `-dirty` on this, and would tell every operator running the release that it was built from an edited tree", platform, flag)
		}
	}
}

// TestRelease_TheMacOSArchivesCarrySignaturesNobodyIssued holds the fact
// docs/build/releasing.md states in full and docs/install.md states in one line
// where a person downloads: the two macOS archives are notarised by nobody, and
// they are not signed alike.
// Go's linker ad-hoc signs `darwin/arm64`, because Apple Silicon will not exec
// an unsigned binary at all, and leaves `darwin/amd64` alone because Intel will
// (§11, ADR-0133, ADR-0137, issue #262).
//
// **It is a claim the docs make and nothing else could catch.** The asymmetry
// is the toolchain's rather than this repository's — `release.sh` passes no
// signing flag and holds no identity — so a Go release that started signing
// `darwin/amd64`, or stopped signing `darwin/arm64`, would leave two documents
// describing a publication that had changed underneath them, and no case above
// this one reads a byte of either macOS archive. #247 read these flags with
// `codesign` on the machines themselves, which needed two Macs; a Mach-O load
// command is the same fact and needs none.
//
// What it does not assert is anything about notarisation, which is not in the
// bytes: notarisation is a ticket stapled to Apple's servers, and the whole of
// what a file can say about it is a stapled ticket this one does not carry.
// *Nobody notarised these* is a fact about the release process — there is no
// identity anywhere in it — and it is stated where a person reads it rather
// than asserted here against bytes that cannot answer.
func TestRelease_TheMacOSArchivesCarrySignaturesNobodyIssued(t *testing.T) {
	published := publish(t, released, "aarch64-darwin", "x86_64-darwin")

	for _, want := range []struct {
		platform string
		signed   bool
		because  string
	}{
		{"aarch64-darwin", true, "Apple Silicon will not exec an unsigned binary, so Go's linker ad-hoc signs one"},
		{"x86_64-darwin", false, "Intel execs an unsigned binary, so Go's linker leaves one alone"},
	} {
		unpacked := unpack(t, archiveOf(published, released, want.platform))
		flags, signed := codeDirectoryFlagsOf(t, unpacked)

		if signed != want.signed {
			t.Errorf("the %s archive's binary carries a code signature = %v, want %v — %s, and docs/build/releasing.md says so, docs/install.md carrying the half a person downloading meets",
				want.platform, signed, want.signed, want.because)
			continue
		}
		if !signed {
			continue
		}
		if flags != adhocLinkerSigned {
			t.Errorf("the %s archive's binary is signed with CodeDirectory flags %#x, want %#x (adhoc | linker-signed) — a signature from anything but the linker is one this release process cannot account for",
				want.platform, flags, adhocLinkerSigned)
		}
	}
}

// hostPlatform is this machine as the release names platforms.
func hostPlatform(t *testing.T) string {
	t.Helper()
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		return "x86_64-linux"
	case "linux/arm64":
		return "aarch64-linux"
	case "darwin/amd64":
		return "x86_64-darwin"
	case "darwin/arm64":
		return "aarch64-darwin"
	}
	// A skip and not `unavailable`: what this one says is that the release
	// publishes nothing for this architecture, which no preparation of the
	// machine could change. `SUITE_PREPARED` claims the tools are installed,
	// not that the runner is one of the four platforms (issue #243).
	t.Skipf("the release publishes nothing for %s/%s; a released binary cannot be run here", runtime.GOOS, runtime.GOARCH)
	return ""
}

// publish runs the release script into a directory of its own, outside the
// repository, and answers that directory. It is what a case that is about the
// shape of a publication wants: nothing the script writes can reach the tree it
// builds from.
func publish(t *testing.T, version string, platforms ...string) string {
	t.Helper()
	return publishInto(t, t.TempDir(), version, platforms...)
}

// publishInto runs the release script into a directory the caller names, which
// is the half of the invocation
// `TestRelease_EveryArchiveOfOneReleaseIsStampedFromACleanTree` cares about — a
// release publishes into the checkout it is cut from.
func publishInto(t *testing.T, into, version string, platforms ...string) string {
	t.Helper()
	needTools(t, "go", "bash", "tar", "sha256sum")

	command := exec.Command("bash", append([]string{"scripts/release.sh", version, into}, platforms...)...)
	command.Dir = root(t)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("scripts/release.sh %s %s %v: %v\n%s", version, into, platforms, err, out)
	}
	return into
}

// directoryInTheWorktree makes a directory of the case's own inside the
// repository, of the kind `dist` is: created empty, so that the tree a build is
// stamped against is the tree the case found, and removed with the case.
func directoryInTheWorktree(t *testing.T) string {
	t.Helper()

	into, err := os.MkdirTemp(root(t), "release-case-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(into) })
	return into
}

// worktreeEdits is what Go's build stamping reads before it writes
// `vcs.modified`: whatever this repository's working tree carries that the
// commit does not account for, empty where it carries nothing. A case reads it
// before it publishes, because what the release writes makes it non-empty.
//
// It answers the lines rather than a yes or no so that a case turned away by
// them can say which they were. On a prepared machine that answer is a failure
// (ADR-0123), and *this working tree carries edits* on a runner nobody edited
// is a sentence that needs the list beside it.
func worktreeEdits(t *testing.T) string {
	t.Helper()
	needTools(t, "git")

	command := exec.Command("git", "status", "--porcelain")
	command.Dir = root(t)
	out, err := command.Output()
	if err != nil {
		unavailable(t, "git status --porcelain failed in %s (%v); what a build here is stamped with cannot be read", root(t), err)
	}
	return strings.TrimSpace(string(out))
}

// archiveOf is the path the release script publishes one platform's archive at:
// the name the tag's URL carries, without the `v` (§11).
func archiveOf(published, version, platform string) string {
	return filepath.Join(published, "hyper-"+version+"-"+platform+".tar.gz")
}

// unpack extracts an archive's `hyper` into a directory of its own and answers
// the path to it — `tar -xzf`, which is what the install step in a generated
// workflow runs against the same bytes (§10).
func unpack(t *testing.T, archive string) string {
	t.Helper()
	needTools(t, "tar")

	into := t.TempDir()
	command := exec.Command("tar", "-xzf", archive, "-C", into)
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("tar -xzf %s: %v\n%s", archive, err, out)
	}
	return filepath.Join(into, "hyper")
}

// modifiedFlagOf reads `vcs.modified` out of a built binary — the same setting
// `version.Current` reads out of the running one, read here off a file so that
// an archive built for another platform can be asked.
//
// A binary with no such setting fails the case rather than skipping it. Go
// stamps `vcs.*` for a main package built inside a repository its tools can
// read, and the caller has already had `git status` answer in that repository,
// so an archive carrying no flag is a change in how Go stamps and not a
// property of the machine.
func modifiedFlagOf(t *testing.T, binary string) string {
	t.Helper()

	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		t.Fatalf("no build information in %s: %v", binary, err)
	}
	for _, setting := range info.Settings {
		if setting.Key == "vcs.modified" {
			return setting.Value
		}
	}
	t.Fatalf("%s carries no vcs.modified; this release was built in a checkout, and `hyper version` has no `-dirty` to withhold or append", binary)
	return ""
}

// digestOf is the artefact's checksum computed here, in the spelling the
// Repository declaration uses — an independent reading of the same bytes
// `sha256sum` read, so the case compares two answers rather than one.
func digestOf(t *testing.T, path string) string {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	sum := sha256.New()
	if _, err := io.Copy(sum, file); err != nil {
		t.Fatal(err)
	}
	return "sha256:" + hex.EncodeToString(sum.Sum(nil))
}

// needTools skips where the machine is missing something a case needs on
// PATH — building or unpacking a release here, sealing the acceptance harness
// in the case next door. A case that cannot run says so rather than passing on
// an assertion it never reached, and on a machine that claims it was prepared
// it says so by failing, which is `unavailable`'s rule (suite_test.go, issue
// #243).
func needTools(t *testing.T, tools ...string) {
	t.Helper()
	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			unavailable(t, "%s is not on PATH; a case that needs it cannot run here", tool)
		}
	}
}

// codeDirectoryFlagsOf answers the CodeDirectory flags of a Mach-O binary and
// whether it carries a signature at all — `codesign -dv`'s two answers, read
// off a file by a machine that is not a Mac.
//
// It reads the embedded signature the way the format is laid out rather than
// the way a tool reports it: LC_CODE_SIGNATURE names an offset, a SuperBlob
// there indexes blobs by type, and the CodeDirectory's fourth word is the
// flags. A binary with no such load command is not signed, which is an answer
// and not a failure — it is the answer `x86_64-darwin` gives.
func codeDirectoryFlagsOf(t *testing.T, path string) (uint32, bool) {
	t.Helper()

	image, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	file, err := macho.NewFile(bytes.NewReader(image))
	if err != nil {
		t.Fatalf("%s is not a Mach-O binary: %v — the release publishes this platform as one", path, err)
	}

	for _, load := range file.Loads {
		raw, ok := load.(macho.LoadBytes)
		if !ok || len(raw) < 16 {
			continue
		}
		if file.ByteOrder.Uint32(raw[0:4]) != lcCodeSignature {
			continue
		}

		// The load command's byte order is the binary's; everything inside
		// the signature is big-endian whatever the architecture, which is
		// why the two are read with different hands.
		at := file.ByteOrder.Uint32(raw[8:12])
		if magic := wordAt(t, path, image, at); magic != superBlobMagic {
			t.Fatalf("%s has an LC_CODE_SIGNATURE whose blob magic is %#x, want %#x — this is not the embedded signature the format describes", path, magic, superBlobMagic)
		}

		// The SuperBlob indexes blobs by type, and the CodeDirectory is the
		// one carrying the flags. It is found by its magic rather than by
		// its position: what comes first in the blob is not fixed.
		count := wordAt(t, path, image, at+8)
		for i := uint32(0); i < count; i++ {
			blob := at + wordAt(t, path, image, at+12+i*8+4)
			if wordAt(t, path, image, blob) != codeDirectoryMagic {
				continue
			}
			return wordAt(t, path, image, blob+codeDirectoryFlagsAt), true
		}
		t.Fatalf("%s carries an embedded signature with no CodeDirectory in it; there is no flags word to read", path)
	}
	return 0, false
}

// wordAt reads the four bytes at an offset inside an embedded code signature,
// where every word is network order on both architectures the release
// publishes.
//
// **It bounds-checks because the offsets come out of the bytes they index.** A
// SuperBlob says where its own blobs are, so a truncated or malformed archive
// walks off the end — and a case that reports `index out of range` has told
// nobody which archive was short. Failing here says which file and how far past
// it went, which is the manner `modifiedFlagOf` next door already sets.
func wordAt(t *testing.T, path string, image []byte, at uint32) uint32 {
	t.Helper()
	if uint64(at)+4 > uint64(len(image)) {
		t.Fatalf("%s's code signature names an offset %d, and the file is %d bytes; the signature runs past the end of the archive's own binary", path, at, len(image))
	}
	return binary.BigEndian.Uint32(image[at:])
}

// root is the repository root, which is where the release script and the
// workflow that runs it both live.
func root(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return path
}
