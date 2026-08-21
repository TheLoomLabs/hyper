package store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// §12 closes the Store at six path forms and no others, and this file is the
// acceptance test for the grammar that builds and reads them. Every assertion
// here is a byte the specification wrote down: a path, a segment, an escape, a
// digest suffix (issue #128).

// A Run id. It is a UUIDv7, lowercase and hyphenated — time-ordered, and
// mintable by either environment alone, which a counter is not (§12, ADR-0006).

// theRunID is §7's own worked example, the one #124's demo prints. It is used
// rather than a minted one wherever a case asserts a path's bytes: a path built
// from a random id is a path no failure message can be read against.
const theRunID = "01984f1a-3c9f-7b04-9c2e-4f0b8d61a3e7"

func TestParseRunID_AcceptsALowercaseHyphenatedUUIDv7(t *testing.T) {
	run, err := store.ParseRunID(theRunID)
	if err != nil {
		t.Fatalf("ParseRunID(%q) = %v, want it accepted", theRunID, err)
	}
	if got := run.String(); got != theRunID {
		t.Errorf("run id = %q, want %q", got, theRunID)
	}
}

func TestParseRunID_RefusesWhatIsNotOne(t *testing.T) {
	for name, id := range map[string]string{
		"a UUIDv4":                "f47ac10b-58cc-4372-a567-0e02b2c3d479",
		"the same id uppercased":  "01984F1A-3C9F-7B04-9C2E-4F0B8D61A3E7",
		"an unhyphenated one":     "01984f1a3c9f7b049c2e4f0b8d61a3e7",
		"one carrying no variant": "01984f1a-3c9f-7b04-1c2e-4f0b8d61a3e7",
		"one digit short":         "01984f1a-3c9f-7b04-9c2e-4f0b8d61a3e",
		"a word":                  "the-last-run",
		"nothing at all":          "",
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := store.ParseRunID(id); err == nil {
				t.Errorf("ParseRunID(%q) = no error, want it refused", id)
			}
		})
	}
}

func TestMintRunID_MintsOneItAccepts(t *testing.T) {
	minted := store.MintRunID(theInstant).String()

	if _, err := store.ParseRunID(minted); err != nil {
		t.Errorf("ParseRunID(%q) = %v, want the minted id accepted", minted, err)
	}
}

func TestMintRunID_IsTimeOrderedAcrossSuccessiveMints(t *testing.T) {
	// A millisecond apart, which is the resolution a UUIDv7 orders on. The
	// ordering asserted is lexicographic over the text, because that is the
	// order a directory listing of the Journal gives.
	first := store.MintRunID(theInstant).String()
	second := store.MintRunID(theInstant.Add(time.Millisecond)).String()

	if first >= second {
		t.Errorf("minted %q at the earlier instant and %q at the later, want the earlier to sort first", first, second)
	}
}

func TestMintRunID_MintsTwoDifferentIdsAtOneInstant(t *testing.T) {
	// Two environments write one branch (ADR-0006), and the whole of what
	// keeps their paths disjoint is that two Runs cannot mint one id.
	if first, second := store.MintRunID(theInstant), store.MintRunID(theInstant); first == second {
		t.Errorf("two mints at one instant both gave %q, want two ids", first)
	}
}

// The Record version path, and the identity segments inside it. The name half
// of an identity is a Manifest-declared field of an upstream response, which
// makes it hostile input and makes this the boundary that holds (§7, §12).

// theRun is theRunID as the constructors take one. Parsing it here rather than
// minting an id is what lets every path below be written out in full.
func theRun(t *testing.T) store.RunID {
	t.Helper()
	run, err := store.ParseRunID(theRunID)
	if err != nil {
		t.Fatalf("ParseRunID(%q) = %v", theRunID, err)
	}
	return run
}

func TestRecordPath_IsTheFormSectionTwelveStates(t *testing.T) {
	got := store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: "status.hyper.dev"}, theRun(t), 1)

	want := "records/local/uptime/status.hyper.dev/" + theRunID + "-0001.json"
	if got != want {
		t.Errorf("record path = %q, want %q", got, want)
	}
}

func TestRecordPath_EncodesAllThreeIdentitySegments(t *testing.T) {
	// The target and the definition are authored and the name is projected,
	// and all three are one segment each: the encoding is the identity's,
	// never the hostile half of it alone (§7, §12).
	got := store.RecordPath(store.Identity{
		Target:     "prod/eu",
		Definition: "vm inventory",
		Name:       "Über-vm",
	}, theRun(t), 1)

	want := "records/prod%2Feu/vm%20inventory/%C3%9Cber-vm/" + theRunID + "-0001.json"
	if got != want {
		t.Errorf("record path = %q, want %q", got, want)
	}
}

func TestPaths_RefuseWhatNoPathCanBeBuiltFrom(t *testing.T) {
	// Everything a constructor builds is a path the parser reads back, so
	// what cannot be one is refused where it is handed over rather than
	// written into an append-only branch and found later (§12, ADR-0011).
	for name, build := range map[string]func(){
		"a Record version at no position": func() {
			store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: "a"}, theRun(t), 0)
		},
		"a Step file at no position": func() {
			store.JournalEntry{Run: theRun(t), Started: theInstant}.StepPath(0)
		},
		"an identity carrying no name": func() {
			store.RecordPath(store.Identity{Target: "local", Definition: "uptime"}, theRun(t), 1)
		},
		"an identity carrying no target": func() {
			store.RecordPath(store.Identity{Definition: "uptime", Name: "a"}, theRun(t), 1)
		},
		"a Run closing its own entry": func() {
			run := theRun(t)
			store.JournalEntry{Run: run, Started: theInstant}.ClosedByPath(run)
		},
	} {
		t.Run(name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Errorf("a path was built, want it refused")
				}
			}()
			build()
		})
	}
}

func TestRecordPath_EncodesEveryByteOutsideTheUnreservedSet(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"the unreserved set entire": {
			"AZaz09-_.",
			"AZaz09-_.",
		},
		"a slash, which would otherwise be a second segment": {
			"vms/web-01",
			"vms%2Fweb-01",
		},
		"a space and a colon": {
			"web 01:443",
			"web%2001%3A443",
		},
		"a tilde, which the truncation suffix owns": {
			"web~01",
			"web%7E01",
		},
		"a percent, which an escape is written with": {
			"100%",
			"100%25",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := segmentOf(t, tc.in); got != tc.want {
				t.Errorf("%q encodes as %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRecordPath_EncodesALeadingDotAndNotOneElsewhere(t *testing.T) {
	// A leading dot is encoded and a dot anywhere else is not (§12): the
	// segments `.` and `..` are the two a filesystem reads as a walk, and
	// `.git` is the directory a checkout of this branch would land beside.
	for name, tc := range map[string]struct{ in, want string }{
		"a hidden file's name":         {".hidden", "%2Ehidden"},
		"the directory itself":         {".", "%2E"},
		"the directory above":          {"..", "%2E."},
		"a dot anywhere but the front": {"status.hyper.dev", "status.hyper.dev"},
	} {
		t.Run(name, func(t *testing.T) {
			if got := segmentOf(t, tc.in); got != tc.want {
				t.Errorf("%q encodes as %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestRecordPath_PreservesCaseRatherThanFoldingIt(t *testing.T) {
	// The fold that decides whether two identities collide is hyper's and
	// belongs to a check (§6, §7), never to a filesystem: a git tree entry is
	// a byte string and case-sensitive everywhere.
	run := theRun(t)
	upper := store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: "Foo"}, run, 1)
	lower := store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: "foo"}, run, 1)

	if upper == lower {
		t.Errorf("Foo and foo built one path, %q, want two", upper)
	}
	if got := segmentOf(t, "Foo"); got != "Foo" {
		t.Errorf("Foo encodes as %q, want it unchanged", got)
	}
}

func TestRecordPath_WritesOnePercentEscapePerUTF8Byte(t *testing.T) {
	// Ü is two bytes in UTF-8, and each is one escape: the encoding is over
	// bytes, so a name is a path segment on a filesystem that has never
	// heard of a code point.
	if got := segmentOf(t, "Über-vm"); got != "%C3%9Cber-vm" {
		t.Errorf("Über-vm encodes as %q, want %q", got, "%C3%9Cber-vm")
	}
}

func TestRecordPath_NumbersAStepInTheRunsWrittenOrder(t *testing.T) {
	// <nnnn> is zero-padded to four digits and widens beyond four rather
	// than wrapping (§12), so a Run of ten thousand Steps writes ten
	// thousand paths rather than two Steps into one.
	for name, tc := range map[string]struct {
		step int
		want string
	}{
		"the first Step":                 {1, "0001"},
		"one inside the width":           {42, "0042"},
		"the last Step four digits hold": {9999, "9999"},
		"the one that widens it":         {10000, "10000"},
		"one far past it":                {123456, "123456"},
	} {
		t.Run(name, func(t *testing.T) {
			got := store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: "a"}, theRun(t), tc.step)
			want := "records/local/uptime/a/" + theRunID + "-" + tc.want + ".json"
			if got != want {
				t.Errorf("step %d gives %q, want %q", tc.step, got, want)
			}
		})
	}
}

// segmentOf is the encoded `name` segment of the Record path built from one, so
// that the encoding is asserted through the constructor a caller has rather than
// through an encoder reached behind it.
func segmentOf(t *testing.T, name string) string {
	t.Helper()
	return segmentIn(t, store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: name}, theRun(t), 1))
}

// segmentIn is the `name` segment of a Record version's path.
func segmentIn(t *testing.T, path string) string {
	t.Helper()
	segments := strings.Split(path, "/")
	if len(segments) != 5 {
		t.Fatalf("record path %q is %d segments, want 5", path, len(segments))
	}
	return segments[3]
}

// Truncation. An encoded segment longer than 200 bytes is cut at 200 on an
// escape boundary and suffixed with `~` and the first sixteen lowercase
// hexadecimal digits of the SHA-256 of the whole encoded segment (§12).
//
// The digests below were taken with `sha256sum` over the encoded segment
// written out in full, not computed the way the code computes them: an expected
// value a test derives the way the implementation derives it is a value that can
// never disagree with it.

func TestRecordPath_WritesASegmentOfExactlyTwoHundredBytesWhole(t *testing.T) {
	whole := strings.Repeat("a", 200)

	if got := segmentOf(t, whole); got != whole {
		t.Errorf("a 200-byte segment was written as %q, want it whole", got)
	}
}

func TestRecordPath_CutsAnOverLongSegmentOnAnEscapeBoundary(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"one byte over, the cut landing on no escape": {
			strings.Repeat("a", 201),
			strings.Repeat("a", 200) + "~a92efd82109373e5",
		},
		"an escape closing on the cut, taken whole": {
			// `%C3` occupies the 198th, 199th and 200th bytes, so
			// the cut falls where the rule names it and the escape
			// it ends is kept entire.
			strings.Repeat("a", 197) + "üü",
			strings.Repeat("a", 197) + "%C3~4dc838f990f63735",
		},
		"an escape spanning the cut, backed off two bytes": {
			// The 199th byte opens `%C3`, so a cut at 200 would
			// take `%C` and leave a path that decodes to nothing.
			strings.Repeat("a", 198) + "ü",
			strings.Repeat("a", 198) + "~4e3f7d9a34823376",
		},
		"an escape opening at the cut, backed off one byte": {
			strings.Repeat("a", 199) + "ü",
			strings.Repeat("a", 199) + "~4c96fd813464e6ac",
		},
	} {
		t.Run(name, func(t *testing.T) {
			path := store.RecordPath(store.Identity{Target: "local", Definition: "uptime", Name: tc.in}, theRun(t), 1)

			if got := segmentIn(t, path); got != tc.want {
				t.Errorf("segment = %q, want %q", got, tc.want)
			}
			// The parser refuses an escape cut through the middle,
			// so a cut path it accepts is a cut that cleared one.
			if _, err := store.ParsePath(path); err != nil {
				t.Errorf("ParsePath(%q) = %v, want a cut segment to be a path", path, err)
			}
		})
	}
}

func TestRecordPath_SeparatesTwoIdentitiesSharingTheirFirstTwoHundredBytes(t *testing.T) {
	// Truncation is lossy and the suffix is what keeps it from being
	// collision-prone as well: the digest is over the whole encoded segment,
	// never over the part that survived the cut.
	first := segmentOf(t, strings.Repeat("a", 200)+"b")
	second := segmentOf(t, strings.Repeat("a", 200)+"c")

	if first == second {
		t.Fatalf("two identities built one segment, %q, want two", first)
	}
	if want := strings.Repeat("a", 200) + "~830c8d218c20d177"; first != want {
		t.Errorf("segment = %q, want %q", first, want)
	}
	if want := strings.Repeat("a", 200) + "~bd99f8b68b44f2b7"; second != want {
		t.Errorf("segment = %q, want %q", second, want)
	}
}

// The Journal's four forms. All of them sit under the entry's own directory —
// journal/<yyyy>/<mm>/<dd>/<run-id>/ — so *is this entry closed* is the listing
// of one directory, and a closing write by a later Run lands inside the entry it
// speaks about (§12, ADR-0076).

func TestJournalEntry_WritesTheFourFormsSectionTwelveStates(t *testing.T) {
	closerID := "01984f2b-1d7a-7c31-8d41-6b2f7ae05c19"
	closer, err := store.ParseRunID(closerID)
	if err != nil {
		t.Fatalf("ParseRunID(%q) = %v", closerID, err)
	}
	entry := store.JournalEntry{Run: theRun(t), Started: theInstant}

	const dir = "journal/2026/04/02/" + theRunID
	for name, tc := range map[string]struct{ got, want string }{
		"run.json":     {entry.RunPath(), dir + "/run.json"},
		"a step file":  {entry.StepPath(7), dir + "/steps/0007.json"},
		"outcome.json": {entry.OutcomePath(), dir + "/outcome.json"},
		"a closing write": {
			entry.ClosedByPath(closer),
			dir + "/closed-by/" + closerID + ".json",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("path = %q, want %q", tc.got, tc.want)
			}
		})
	}
}

func TestJournalEntry_PartitionsOnTheUTCDateOfTheRunsStart(t *testing.T) {
	// Two in the morning in Karachi is the previous day in UTC. The
	// partition is what a backward walk over directories reads, so a Run
	// filed under the operator's date rather than the record's would be
	// looked for on a day it is not on (§12).
	local := time.Date(2026, time.April, 2, 2, 0, 0, 0, time.FixedZone("+05:00", 5*60*60))
	entry := store.JournalEntry{Run: theRun(t), Started: local}

	want := "journal/2026/04/01/" + theRunID + "/run.json"
	if got := entry.RunPath(); got != want {
		t.Errorf("run.json path = %q, want %q", got, want)
	}
}

// Parsing. A path is read back for its shape — which form it is, which Run wrote
// it, which Step — and never for its identity, truncation having made that
// question one only the file itself answers (§7, §12).

func TestParsePath_RoundTripsEveryFormItWasBuiltFrom(t *testing.T) {
	run := theRun(t)
	closer, err := store.ParseRunID("01984f2b-1d7a-7c31-8d41-6b2f7ae05c19")
	if err != nil {
		t.Fatalf("ParseRunID = %v", err)
	}
	entry := store.JournalEntry{Run: run, Started: theInstant}
	// A name long enough to be truncated and one carrying an escape, so
	// that the round trip is asserted over the segments the grammar is
	// hardest on rather than over the two that read as ordinary words.
	record := store.RecordPath(store.Identity{
		Target:     "prod",
		Definition: "vm-inventory",
		Name:       strings.Repeat("ü", 300),
	}, run, 12)

	// The two axes every Journal form answers, and neither of which the two
	// forms that are not Journal entries carry.
	const (
		partition = "journal/2026/04/02"
		dir       = partition + "/" + theRunID
	)

	for name, tc := range map[string]struct {
		path string
		want store.Path
	}{
		"the introduction": {store.IntroductionPath, store.Path{Form: store.FormIntroduction}},
		"a Record version": {record, store.Path{Form: store.FormRecord, Run: run, Step: 12}},
		"run.json": {
			entry.RunPath(),
			store.Path{Form: store.FormRun, Run: run, Entry: run, Partition: partition, Dir: dir},
		},
		"a Step file": {
			entry.StepPath(9999),
			store.Path{Form: store.FormStep, Run: run, Entry: run, Step: 9999, Partition: partition, Dir: dir},
		},
		"outcome.json": {
			entry.OutcomePath(),
			store.Path{Form: store.FormOutcome, Run: run, Entry: run, Partition: partition, Dir: dir},
		},
		"a closing write": {
			// The Run speaking wrote it and the Run spoken about
			// owns the entry it sits in, which is the one form
			// where the two are not one Run (ADR-0076).
			entry.ClosedByPath(closer),
			store.Path{Form: store.FormClosedBy, Run: closer, Entry: run, Partition: partition, Dir: dir},
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := store.ParsePath(tc.path)
			if err != nil {
				t.Fatalf("ParsePath(%q) = %v", tc.path, err)
			}
			if got != tc.want {
				t.Errorf("ParsePath(%q) = %+v, want %+v", tc.path, got, tc.want)
			}
		})
	}
}

func TestParsePath_RefusesAPathOutsideTheSixForms(t *testing.T) {
	const run = theRunID
	for name, path := range map[string]string{
		"nothing at all":                    "",
		"a root the grammar does not have":  "notes/2026/04/02/" + run + "/run.json",
		"the introduction with a tail":      "STORE.md/more",
		"a Record version of no name":       "records/local/uptime/" + run + "-0001.json",
		"a Record version of no step":       "records/local/uptime/a/" + run + ".json",
		"a Record version of no extension":  "records/local/uptime/a/" + run + "-0001",
		"a Record version under a UUIDv4":   "records/local/uptime/a/f47ac10b-58cc-4372-a567-0e02b2c3d479-0001.json",
		"a step numbered zero":              "journal/2026/04/02/" + run + "/steps/0000.json",
		"a step of three digits":            "journal/2026/04/02/" + run + "/steps/001.json",
		"a step padded past four":           "journal/2026/04/02/" + run + "/steps/01000.json",
		"a date that is not one":            "journal/2026/02/31/" + run + "/run.json",
		"a date of the wrong width":         "journal/2026/4/2/" + run + "/run.json",
		"a file the entry does not hold":    "journal/2026/04/02/" + run + "/started.json",
		"a closer that is not a Run id":     "journal/2026/04/02/" + run + "/closed-by/the-other-run.json",
		"a Run closing its own entry":       "journal/2026/04/02/" + run + "/closed-by/" + run + ".json",
		"a segment carrying a raw space":    "records/local/uptime/web 01/" + run + "-0001.json",
		"a segment escaped in lowercase":    "records/local/uptime/%c3%9cber/" + run + "-0001.json",
		"a segment escaped by half":         "records/local/uptime/%C3%9/" + run + "-0001.json",
		"a segment that walks upward":       "records/local/uptime/../" + run + "-0001.json",
		"a segment that is empty":           "records/local//a/" + run + "-0001.json",
		"a segment past the width uncut":    "records/local/uptime/" + strings.Repeat("a", 201) + "/" + run + "-0001.json",
		"a segment cut short of the width":  "records/local/uptime/" + strings.Repeat("a", 100) + "~a92efd82109373e5/" + run + "-0001.json",
		"a truncation suffix in uppercase":  "records/local/uptime/" + strings.Repeat("a", 200) + "~A92EFD82109373E5/" + run + "-0001.json",
		"a leading slash":                   "/STORE.md",
		"a trailing slash":                  "journal/2026/04/02/" + run + "/",
		"the entry's directory, not a file": "journal/2026/04/02/" + run,
	} {
		t.Run(name, func(t *testing.T) {
			if got, err := store.ParsePath(path); err == nil {
				t.Errorf("ParsePath(%q) = %+v, want it refused", path, got)
			}
		})
	}
}

func TestRecordPath_OrdersNothing(t *testing.T) {
	// The encoding names a file and orders nothing (§12). Escaping drags
	// every escaped byte to the left of every unreserved one, so Über-vm
	// sorts after zone-a by name and before it by path — and an Expansion is
	// ordered by the Record name itself (§6, ADR-0044). It is asserted here
	// rather than warned about, because a listing of one of these
	// directories is exactly what invites the mistake.
	const uber, zone = "Über-vm", "zone-a"
	if !(uber > zone) {
		t.Fatalf("%q sorts before %q by name, which is the premise this case rests on", uber, zone)
	}

	byPath := []string{
		store.RecordPath(store.Identity{Target: "local", Definition: "vms", Name: uber}, theRun(t), 1),
		store.RecordPath(store.Identity{Target: "local", Definition: "vms", Name: zone}, theRun(t), 1),
	}
	if byPath[0] > byPath[1] {
		t.Errorf("%q sorts after %q by path, want the two orderings to disagree", byPath[0], byPath[1])
	}
}
