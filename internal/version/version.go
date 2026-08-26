// Package version names the version of the running binary — the fact the
// version pin gate compares against the Repository declaration's pin (§11,
// ADR-0020) — and, since issue #103, the whole of what `hyper version` prints:
// the facts a build stamps into a binary, and the page that states them.
//
// The page lives here rather than beside the command because its rules are
// properties of the facts and not of the surface: a fact the build did not
// stamp renders `unknown` rather than vanishing, and a commit from a modified
// tree is marked. Nothing in this package reads a file, an environment
// variable, or a network; the update check ADR-0019 declined has no half-way
// house here to grow in.
package version

import (
	"runtime"
	"runtime/debug"
	"strings"
)

// Version is the running binary's version — the fact the pin gate compares
// against the Repository declaration's `version:` for exact equality (§11,
// ADR-0020, internal/pin).
//
// **It is a placeholder, and no build replaces it.** The comment here once said
// it was a placeholder until `hyper project`'s release machinery landed; that
// machinery landed with issue #178, milestone 10 closed after it, and this
// constant never moved. Every binary this repository has produced reports
// `0.0.0-dev`, so the only value the pin gate has compared on a real build is
// this one and the mismatch arm is exercised by fixtures alone.
//
// **A `const` is also the one shape `-ldflags -X` cannot write.** The linker
// sets a string *variable*; a constant is inlined at compile time and the flag
// is ignored without complaint. So stamping is not a build-invocation change
// that could be made against this declaration as it stands — it starts by
// making this a `var`, which trades a fact the compiler proves for one the
// linker asserts. That, and the release publication `internal/release` already
// expects to fetch a `checksums.txt` from, are issue #191's.
//
// It is stated rather than left as a forward reference to a milestone that has
// been and gone: what a reader needs here is what is true now and where the
// work is filed, not a date that has passed.
const Version = "0.0.0-dev"

// unknown is what a fact the build did not stamp renders as. The five-line
// shape is fixed, so every fact has a value on the page even where the build
// supplied none — a page whose job is identifying bytes may not quietly
// identify bytes it cannot vouch for (issue #103).
const unknown = "unknown"

// Facts is everything `hyper version` states: the binary's own version, the
// revision and time the build stamped, whether that build came from a modified
// working tree, the toolchain, and the platform. Every member is a fact about
// the binary as built, which is why the whole value is passed to the command
// rather than read inside it — a page assembled from the running build changes
// with every commit, and a golden file cannot hold one.
//
// An empty string is a fact the build did not stamp, and renders unknown.
type Facts struct {
	// Version is what the binary claims to be — the same string the pin
	// gate compares and the Refusal quotes as *this binary*.
	Version string

	// Commit is the VCS revision the build was stamped with (`vcs.revision`).
	Commit string

	// Built is the VCS time of that revision (`vcs.time`), in RFC 3339.
	Built string

	// Modified is the build's `vcs.modified` flag: the tree carried edits the
	// commit above does not account for.
	Modified bool

	// Toolchain is the Go version that compiled the binary.
	Toolchain string

	// OS and Arch are GOOS and GOARCH, the platform the binary was built for.
	OS, Arch string
}

// Current reads the facts of the running binary out of Go's own build
// stamping. It is called once, at the entry point, and never from inside the
// command — which is what makes the command's output deterministic under test.
func Current() Facts {
	facts := Facts{
		Version:   Version,
		Toolchain: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return facts
	}
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			facts.Commit = setting.Value
		case "vcs.time":
			facts.Built = setting.Value
		case "vcs.modified":
			facts.Modified = setting.Value == "true"
		}
	}
	return facts
}

// Page renders the five lines `hyper version` writes to stdout, newline
// terminated. The first line is the version and nothing else, so a script that
// wants the bare version cuts one line rather than parsing a document — the
// accepted cost of `version` taking no `--json` (issue #103). The four below it
// carry fixed lowercase labels padded to one column.
func (f Facts) Page() string {
	var b strings.Builder
	b.WriteString("hyper " + orUnknown(f.Version) + "\n")
	writeLine(&b, "commit", f.commit())
	writeLine(&b, "built", orUnknown(f.Built))
	writeLine(&b, "go", orUnknown(f.Toolchain))
	writeLine(&b, "os/arch", f.platform())
	return b.String()
}

// labelWidth is the width every label is padded to: `os/arch`, the longest of
// the four, and one space after it. The set of labels is closed, so the width
// is a constant rather than a measurement.
const labelWidth = len("os/arch")

// writeLine writes one labelled fact. A label longer than labelWidth would
// break the column rather than the page — one space still separates it from its
// value — which is the right failure for a set of labels that is closed and
// changes only when someone edits Page directly above.
func writeLine(b *strings.Builder, label, value string) {
	b.WriteString(label)
	b.WriteString(strings.Repeat(" ", max(labelWidth-len(label), 0)+1))
	b.WriteString(value)
	b.WriteByte('\n')
}

// commit marks a revision built from a modified tree, so a hash on the page is
// never a claim about bytes edited after it. The suffix qualifies a hash and
// there is nothing to qualify without one: an unstamped revision renders
// unknown alone, `unknown-dirty` being a reading of the word as a commit.
func (f Facts) commit() string {
	if f.Commit == "" {
		return unknown
	}
	if f.Modified {
		return f.Commit + "-dirty"
	}
	return f.Commit
}

// platform is GOOS/GOARCH as one fact, because half a platform identifies
// nothing: `linux/unknown` reads as an architecture named unknown rather than
// as a build hyper cannot place.
func (f Facts) platform() string {
	if f.OS == "" || f.Arch == "" {
		return unknown
	}
	return f.OS + "/" + f.Arch
}

func orUnknown(fact string) string {
	if fact == "" {
		return unknown
	}
	return fact
}
