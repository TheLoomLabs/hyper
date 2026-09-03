package version

import "testing"

// The rows below are the builds issue #263 measured on linux/amd64 with
// go1.26.0, against the published `v0.0.1-alpha`, plus one that was not
// measured and is marked as such: a build information carrying no module
// version at all, which is a guard rather than a state anything here produces.
// What each build stamps is Go's answer and not this package's; what this
// package decides is which of the two stampers is read, and that is the whole
// of what these cases pin.

// TestStampedVersion_TheLinkerWins is the guarantee the fallback may not touch.
// `-X` is what `scripts/release.sh` writes and what
// `docs/build/releasing.md` states for a human, so every binary a release
// publishes answers from the flag — even though the module version sitting
// beside it in the same build information would answer too (§11, ADR-0020,
// issues #191 and #263).
func TestStampedVersion_TheLinkerWins(t *testing.T) {
	if got, want := stampedVersion("1.4.0", "v0.0.1-alpha"), "1.4.0"; got != want {
		t.Errorf("stampedVersion(%q, %q) = %q, want %q — the flag decides wherever it wrote", "1.4.0", "v0.0.1-alpha", got, want)
	}
}

// TestStampedVersion_TheModuleAnswersWhereTheLinkerDidNot is the change issue
// #263 buys: a build nobody passed a flag to used to have no answer at all,
// and the toolchain had already derived one from the repository the source sat
// in. Every row is a fact about the bytes rather than about the machine, which
// is the line ADR-0020 draws.
//
// The `v` belongs to the tag and no filename under it carries one (§11), so it
// is stripped here for the same reason `releasing.md` strips it.
func TestStampedVersion_TheModuleAnswersWhereTheLinkerDidNot(t *testing.T) {
	for _, tc := range []struct {
		name   string
		module string
		want   string
	}{
		{"the source is the tag", "v0.0.1-alpha", "0.0.1-alpha"},
		{"the tree carried edits", "v0.0.1-alpha+dirty", "0.0.1-alpha+dirty"},
		{"the commit is not a release", "v0.0.1-alpha.0.20260902184134-c9cf477bd361", "0.0.1-alpha.0.20260902184134-c9cf477bd361"},
		{"a go test binary", "(devel)", unknown},
		{"no module version at all — not measured, a guard", "", unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := stampedVersion(unknown, tc.module); got != tc.want {
				t.Errorf("stampedVersion(%q, %q) = %q, want %q", unknown, tc.module, got, tc.want)
			}
		})
	}
}
