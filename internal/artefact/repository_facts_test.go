package artefact

import "testing"

// TestReadRepositoryFacts_ReadsTheDeclaredPolicyAndNothingElse is the one fact
// this reader carries, in the artefact's own spelling: `compact` names the
// policy it acted under and a reader goes and finds the line it came from, so
// what is answered is the text and never a normalisation of it (§3, §7, issue
// #131).
func TestReadRepositoryFacts_ReadsTheDeclaredPolicyAndNothingElse(t *testing.T) {
	facts := ReadRepositoryFacts(parse(t, `kind: repository-declaration
version: 1.4.0
digest: sha256:0000000000000000000000000000000000000000000000000000000000000000
retention: 90d
`))

	if facts.Retention != "90d" {
		t.Errorf("retention is %q, want the declaration's own 90d", facts.Retention)
	}
}

// TestReadRepositoryFacts_ADeclarationWithNoPolicyStatesNone is §3's own rule
// read off the value: omitted, nothing is ever removed. The empty string is
// what says so, and it is safe as the sentinel because no duration is empty.
func TestReadRepositoryFacts_ADeclarationWithNoPolicyStatesNone(t *testing.T) {
	for what, root := range map[string]string{
		"no retention: at all": `kind: repository-declaration
version: 1.4.0
`,
		"a retention: holding something this cannot read": `kind: repository-declaration
version: 1.4.0
retention: [90d]
`,
	} {
		t.Run(what, func(t *testing.T) {
			if got := ReadRepositoryFacts(parse(t, root)).Retention; got != "" {
				t.Errorf("retention is %q, want none stated", got)
			}
		})
	}
}

// TestReadRepositoryFacts_ANilRootStatesNothing is the load's own absence
// reaching this reader: a repository with no `hyper.yaml`, or one whose file
// would not parse, answers a nil root — and every key answers absent, which is
// the reading a lookup into a nil mapping already gives (issue #88, ADR-0064).
func TestReadRepositoryFacts_ANilRootStatesNothing(t *testing.T) {
	if got := ReadRepositoryFacts(nil).Retention; got != "" {
		t.Errorf("retention is %q, want none stated", got)
	}
}

// TestReadRepositoryFacts_ReadsTheDigestAsTheDeclarationSpellsIt is what
// `project` freezes into every generated workflow: the algorithm inline, as §3
// writes it, so that one split into the hex a `sha256sum -c -` line takes
// happens in the generator and nowhere else (§3, §11).
func TestReadRepositoryFacts_ReadsTheDigestAsTheDeclarationSpellsIt(t *testing.T) {
	facts := ReadRepositoryFacts(parse(t, `kind: repository-declaration
version: 1.4.0
digest: sha256:a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2
`))

	if want := "sha256:a3f1c07d2b9e4a6155c8e0d3f7b21ac49e5d8f0361b4c72ae9d05f83c1e6b7a2"; facts.Digest != want {
		t.Errorf("digest is %q, want the declaration's own %q", facts.Digest, want)
	}
}

// TestReadRepositoryFacts_ADeclarationWithNoDigestStatesNone is the reader
// declining to judge: the schema makes `digest:` required, so what is missing
// here is `check`'s to report and not this reader's to invent (ADR-0064).
func TestReadRepositoryFacts_ADeclarationWithNoDigestStatesNone(t *testing.T) {
	if got := ReadRepositoryFacts(parse(t, "kind: repository-declaration\nversion: 1.4.0\n")).Digest; got != "" {
		t.Errorf("digest is %q, want none stated", got)
	}
}
