package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// The path grammar: every path the Store branch holds, built here and read back
// here, so that nothing anywhere else joins two strings with a slash (§12).
//
// §12 closes the Store at six forms and no others:
//
//	STORE.md
//	records/<target>/<definition>/<name>/<run-id>-<nnnn>.json
//	journal/<yyyy>/<mm>/<dd>/<run-id>/run.json
//	journal/<yyyy>/<mm>/<dd>/<run-id>/steps/<nnnn>.json
//	journal/<yyyy>/<mm>/<dd>/<run-id>/outcome.json
//	journal/<yyyy>/<mm>/<dd>/<run-id>/closed-by/<closer-run-id>.json
//
// Every one of them carries the id of the Run that wrote it, and STORE.md —
// written once, by no Run — is the only exception (ADR-0076). That is what makes
// the push retry's re-application clean in every case rather than in every case
// but one: two Runs cannot mint one UUIDv7, so two Runs cannot write one path.
//
// A path is round-trippable in one direction only. Parsing answers the *shape*
// of a path — which form it is, which Run wrote it, which Step — and never which
// series it belongs to: an over-long identity segment is truncated, so the
// answer to *which series is this* is inside the file, where the Record version
// restates its target, definition and name unencoded and in full (§7).

// recordsPrefix is what every Record version's path begins with, and the whole
// of what an enumeration of the branch's series is listed over. It is spelled
// here with the others so that nothing outside this file joins a Store path
// together (§12).
const recordsPrefix = "records/"

// seriesPrefix is the listing one series' versions are read from: the series'
// own directory, and the separator that keeps the listing inside it.
func seriesPrefix(id Identity) string { return seriesDir(id) + "/" }

// IntroductionPath is where the branch introduces itself, and it is the one
// path in the whole Store that carries no Run id: every other path names the
// Run that wrote it, and this file is written by no Run (§12, ADR-0076).
//
// It is a constant rather than a constructor because it is written once, by an
// act that is not a Run, and takes nothing to build.
const IntroductionPath = "STORE.md"

// RunID is a Run's id: a UUIDv7, lowercase and hyphenated (§12).
//
// It holds its text unexported, on ParseRunID and MintRunID being its only two
// doors — the shape internal/store already uses for a Number, and for the same
// reason. A path is built from one of these without being handed an error to
// answer, so what makes a path well formed is checked once, where the id enters,
// rather than at every position that writes one down.
type RunID struct{ text string }

// String is the id as it is written: lowercase, hyphenated, and whole. An id a
// human retypes renders entire wherever it renders at all (ADR-0047), and this
// is the only text of one there is.
func (r RunID) String() string { return r.text }

// ParseRunID reads a Run id, and refuses everything a UUIDv7 is not: an
// uppercase spelling, an unhyphenated one, a UUIDv4, a UUID carrying no
// variant bits. The Store is written by two environments and read by both, so an
// id that reached a path unchecked would be a directory nothing could ever find
// again by looking for the id it was told.
func ParseRunID(text string) (RunID, error) {
	groups := strings.Split(text, "-")
	widths := [...]int{8, 4, 4, 4, 12}
	if len(groups) != len(widths) {
		return RunID{}, fmt.Errorf("%q is not a UUID: a UUID is five hyphenated groups", text)
	}
	for i, group := range groups {
		if len(group) != widths[i] {
			return RunID{}, fmt.Errorf("%q is not a UUID: group %d is %d digits, want %d", text, i+1, len(group), widths[i])
		}
		if !isLowerHex(group) {
			return RunID{}, fmt.Errorf("%q is not a UUID: %q is not lowercase hexadecimal", text, group)
		}
	}
	if groups[2][0] != '7' {
		return RunID{}, fmt.Errorf("%q is a UUID of version %c, not a UUIDv7", text, groups[2][0])
	}
	if !strings.ContainsRune("89ab", rune(groups[3][0])) {
		return RunID{}, fmt.Errorf("%q carries no UUID variant bits", text)
	}
	return RunID{text: text}, nil
}

// MintRunID mints a Run id at the instant it is handed: a UUIDv7 over that
// instant's millisecond, and random bits under it (RFC 9562).
//
// The clock is threaded rather than read here, as every other clock in hyper is:
// the instant a Run started is the Run's, and the id and the Journal date
// partition are two renderings of it that must not be able to come apart.
//
// Two mints at one instant are two ids — 74 random bits stand beneath the
// millisecond — which is what makes a Run id mintable by the laptop and the
// runner alike with nothing between them to agree with (ADR-0006). It answers no
// error: crypto/rand does not fail without the process failing with it, and a
// caller minting an id has nothing to do with one.
func MintRunID(now time.Time) RunID {
	// Bytes 0-5 are the instant's millisecond, big-endian and so ordered by
	// a byte comparison; bytes 6-15 are random, less the version nibble and
	// the two variant bits written over them.
	var raw [16]byte
	binary.BigEndian.PutUint64(raw[:8], uint64(now.UTC().UnixMilli())<<16)
	rand.Read(raw[6:])
	raw[6] = 0x70 | raw[6]&0x0f
	raw[8] = 0x80 | raw[8]&0x3f

	text := hex.EncodeToString(raw[:])
	return RunID{text: strings.Join([]string{text[:8], text[8:12], text[12:16], text[16:20], text[20:]}, "-")}
}

// Identity is a Record's identity: its Target, its Definition and its name (§2).
// The three are one path segment each under the grammar, and they travel
// together because they are one fact — which series is this — rather than three
// arguments that happen to be adjacent.
//
// It is the same identity IdentityDigest is taken over, at a different grain:
// the digest is over the names of an Expansion's members, which are one Record
// identity each under a Step's own Definition and Target (ADR-0070), and this is
// one whole identity, which is what a path needs and a digest does not.
type Identity struct {
	Target     string
	Definition string
	Name       string
}

// RecordPath is where one version of a Record is written:
// records/<target>/<definition>/<name>/<run-id>-<nnnn>.json.
//
// The file name carries both the Run and the Step because <nnnn> names a Record
// version as well as a Step file (§12): two Steps of one Run writing one
// identity write two paths rather than one path twice, which is what keeps the
// series' versions a listing rather than an overwrite (ADR-0011).
//
// The identity it takes is the unencoded one. Encoding is this package's and
// happens here, at the one boundary where a Manifest-declared name — hostile
// input, and the reason the encoding exists at all — becomes a filename (§7).
func RecordPath(id Identity, run RunID, step int) string {
	return seriesDir(id) + "/" + run.text + "-" + stepNumber(step) + ".json"
}

// seriesDir is the directory one Record's versions sit in, and the whole of what
// a Head lookup lists. It is unexported and reached through seriesPrefix above,
// which is what the reader lists: nothing outside this file builds a Store path,
// and a directory without its separator is half of one.
//
// It is where an identity is required to be one: an empty component encodes to
// an empty segment, which is a path git cannot hold and the parser refuses, so
// the three are checked here rather than at the encoder — the encoder writes
// bytes, and which three components make an identity is this file's fact (§2).
func seriesDir(id Identity) string {
	for _, component := range []struct{ role, value string }{
		{"target", id.Target},
		{"definition", id.Definition},
		{"name", id.Name},
	} {
		if component.value == "" {
			impossible("an identity carries no %s: a Record is identified by its Target, its Definition and a name (§2)", component.role)
		}
	}
	return recordsPrefix + encodeSegment(id.Target) + "/" + encodeSegment(id.Definition) + "/" + encodeSegment(id.Name)
}

// stepNumber writes <nnnn>: the Step's position in the Run's written order, the
// first Step 0001, zero-padded to four digits and widening beyond four rather
// than wrapping (§12). Wrapping would put the ten-thousandth Step's file on top
// of the first's, which on an append-only branch is the one thing that cannot
// happen (ADR-0011).
//
// A position below one is not a position, and no rendering of one is a path.
func stepNumber(step int) string {
	if step < 1 {
		impossible("step %d is not a position: the first Step in a Run is 1", step)
	}
	return fmt.Sprintf("%04d", step)
}

// impossible panics: a path constructor was handed something no path in §12's
// six forms can be built from — a Step at no position, an identity component
// that is not one, a Run closing its own entry.
//
// It panics rather than answering an error, on the reasoning canonical.go's own
// unreachable branch records: what a constructor would otherwise return is a
// path outside the grammar, and it is being written into an append-only branch
// by the next line. None of the three arrives from the world. A Step's position
// is hyper's own counter; the Run closing an entry is hyper's own id; and an
// identity component that is empty is stopped upstream of every write, by the
// identity check §7 states and §6 fires at Expansion — the caller this message
// is addressed to, since a path constructor is downstream of it and cannot
// decline on its behalf.
//
// It is deliberately not called a refusal. A Refusal is a Run outcome a
// guardrail reaches before any effect touches the world (§6), and this is
// hyper's own arithmetic being wrong, which is not a thing a Run outcome has a
// name for.
func impossible(format string, args ...any) {
	panic("store: " + fmt.Sprintf(format, args...))
}

// unreserved is the set an identity segment carries as itself: A–Z, a–z, 0–9,
// `-`, `_` and `.` (§12, RFC 3986). Every byte outside it is escaped, as is a
// leading `.`.
const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_."

// upperHex is the one place hyper writes a hexadecimal digit in uppercase. Every
// other digit it writes is lowercase (§7); a percent-escape is uppercase because
// RFC 3986 says so, and one rule with a stated exception is readable where two
// conventions and no rule is what an implementer guesses at.
const upperHex = "0123456789ABCDEF"

// isLowerHex answers whether every digit of s is a lowercase hexadecimal one,
// which is what a Run id's groups and a truncation suffix both are: every
// hexadecimal digit hyper writes is lowercase, and the percent-escape above is
// the one stated exception (§7).
func isLowerHex(s string) bool {
	return s != "" && strings.TrimLeft(s, lowerHex) == ""
}

// segmentLimit is the width an encoded segment is cut at, and digestWidth the
// number of digest digits the cut one carries after it (§12). The cut is at 200
// and backs off by at most two bytes to clear an escape, so a truncated segment
// is at most 217 bytes and at least 215.
const (
	segmentLimit = 200
	digestWidth  = 16
)

// encodeSegment writes one identity component as one path segment, cut to
// segmentLimit where the encoding runs past it.
//
// **Truncation makes the path lossy, and that is why the identity is restated
// inside the file**: a Record version carries its own target, definition and
// name unencoded and in full (§7), so *which series is this* is answered by
// reading the file and never by decoding a segment.
//
// The digest is over the **whole** encoded segment rather than over the part
// the cut discarded or the part it kept, which is what separates two identities
// agreeing in their first two hundred bytes. `~` is outside the unreserved set,
// so it never occurs in an encoding and the suffix is unambiguous.
func encodeSegment(component string) string {
	encoded := escape(component)
	if len(encoded) <= segmentLimit {
		return encoded
	}

	sum := sha256.Sum256([]byte(encoded))
	return encoded[:cutWidth(encoded)] + "~" + hex.EncodeToString(sum[:])[:digestWidth]
}

// cutWidth is how wide the surviving prefix is: the last position at or below
// segmentLimit that opens a byte rather than sitting inside an escape, which is
// both where the cut falls and how much of the segment it keeps.
//
// Cutting on an escape boundary is not an optimisation. A cut through the middle
// of a `%C3` leaves a path that decodes to nothing — and a reader that decoded
// it anyway would be reading a segment this package has already said is not an
// identity.
func cutWidth(encoded string) int {
	cut := 0
	for i := 0; i <= segmentLimit && i < len(encoded); {
		cut = i
		if encoded[i] == '%' {
			i += 3
			continue
		}
		i++
	}
	return cut
}

// escape writes every byte outside the unreserved set as `%` and two uppercase
// hexadecimal digits over its UTF-8 bytes, and a leading `.` likewise.
//
// It walks bytes rather than runes, which is not an optimisation: the escape is
// per UTF-8 byte, so a two-byte Ü is two escapes, and a byte sequence that is
// not UTF-8 at all still becomes a segment rather than a replacement character
// nothing can be traced back to.
//
// Case is preserved rather than folded. The fold that decides whether two
// identities collide is hyper's and belongs to a check (§6, §7): a filesystem's
// fold is one platform's, and the branch is written by two.
func escape(component string) string {
	var b strings.Builder
	for i := range len(component) {
		if c := component[i]; strings.IndexByte(unreserved, c) >= 0 && !(i == 0 && c == '.') {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(upperHex[component[i]>>4])
		b.WriteByte(upperHex[component[i]&0xf])
	}
	return b.String()
}

// JournalEntry is where one Run's entry sits: the Run's id, and the instant it
// started, which the date partition is the UTC date of.
//
// The two travel together because the entry's directory is built from both, and
// a path form that took them apart would let a caller file a Run under a date
// that is not its own. It carries no account of how the Run ended: an entry's
// account is a classification over the files present under this directory, and
// classifying is milestone 4.6's (§7).
type JournalEntry struct {
	Run     RunID
	Started time.Time
}

// RunPath is journal/<yyyy>/<mm>/<dd>/<run-id>/run.json, written at Run start.
func (e JournalEntry) RunPath() string { return e.dir() + "/run.json" }

// StepPath is journal/<yyyy>/<mm>/<dd>/<run-id>/steps/<nnnn>.json, one per Step
// reaching a Disposition. The number is the Step's position in the Run's written
// order — a nested Procedure's Steps counted in that order, the invocation
// itself being no Step and writing no file (§7, §12).
func (e JournalEntry) StepPath(step int) string {
	return e.dir() + "/steps/" + stepNumber(step) + ".json"
}

// OutcomePath is journal/<yyyy>/<mm>/<dd>/<run-id>/outcome.json, written when
// the Run ends — by the Run itself, and by nobody else. A later Run that infers
// this one died writes ClosedByPath instead, which is what leaves no path in the
// Store two Runs can reach (ADR-0076).
func (e JournalEntry) OutcomePath() string { return e.dir() + "/outcome.json" }

// ClosedByPath is journal/<yyyy>/<mm>/<dd>/<run-id>/closed-by/<closer-run-id>.json,
// written by a Run closing an entry it does not own.
//
// It names both Runs: the entry it speaks about, in the directory, and the Run
// speaking, in the file name. That is what makes an entry holding two accounts
// two files rather than one contested one — the reaper's inference and the
// owner's observation both stand, and hyper picks no side (§7, ADR-0076).
func (e JournalEntry) ClosedByPath(closer RunID) string {
	if closer == e.Run {
		impossible("run %s is closing its own entry: the Run that owns an entry writes outcome.json (§12, ADR-0076)", e.Run)
	}
	return e.dir() + "/closed-by/" + closer.text + ".json"
}

// dir is the entry's own directory, and the whole of what *is this entry closed*
// lists. The date is the UTC date of the Run's start — the operator's own
// midnight being nothing the record knows about — which is what makes finding a
// Step's previous Run a backward walk over directories rather than a scan of
// everything (§12).
func (e JournalEntry) dir() string {
	return "journal/" + e.Started.UTC().Format("2006/01/02") + "/" + e.Run.text
}

// Form is which of §12's six path forms a path is. The zero value is no form:
// a Path is answered by ParsePath or it is not answered at all.
type Form int

const (
	// FormIntroduction is STORE.md, the one path in the Store carrying no
	// Run id, because no Run writes it.
	FormIntroduction Form = iota + 1
	// FormRecord is one version of a Record.
	FormRecord
	// FormRun is an entry's run.json, written at Run start.
	FormRun
	// FormStep is one Step of a Run reaching a Disposition.
	FormStep
	// FormOutcome is an entry's outcome.json, written by its own Run.
	FormOutcome
	// FormClosedBy is a closing write by a Run that does not own the entry.
	FormClosedBy
)

// Path is what a Store path says about itself: which form it is, which Run
// wrote it, whose entry it sits in, and which Step it names where the form has
// one.
//
// What it does not carry is an identity. A Record version's three segments are
// percent-encoded and an over-long one is truncated, so decoding them recovers a
// name only where none was cut — and a reader that decoded them anyway would
// have two answers to *which series is this*, one of them wrong on exactly the
// identities the encoding exists to survive. The answer is inside the file (§7).
type Path struct {
	// Form is which of the six forms this is.
	Form Form
	// Run is the Run that wrote the path, on every form but the
	// introduction, which no Run wrote — the invariant the whole grammar is
	// arranged around (ADR-0076). On a closing write it is the Run speaking,
	// never the entry's own.
	Run RunID
	// Entry is the Run whose Journal entry the path sits in: the same Run on
	// the three a Run writes into its own entry, the dead Run on a closing
	// write, and the zero id on the two forms that are not Journal entries.
	//
	// It is written even where it repeats Run, rather than left to a caller
	// to derive from the form. *Who wrote this* and *whose entry is this*
	// are two questions, they part on exactly one form, and a field a caller
	// has to know when to read is a field that is read wrong on the form it
	// was written for.
	Entry RunID
	// Step is the Step's position on the two forms carrying one — a Record
	// version and a Step file — and zero on the rest, a Step's positions
	// beginning at one.
	Step int
}

// ParsePath reads a Store path back to its shape, and refuses everything outside
// §12's six forms rather than guessing at it. The Store holds these paths and no
// others, so a path that is not one of them is not a path this package can
// answer about — and answering anyway is how a file nothing wrote acquires a
// meaning nothing gave it.
func ParsePath(path string) (Path, error) {
	parsed, err := parsePath(path)
	if err != nil {
		return Path{}, fmt.Errorf("%q is not a Store path: %w", path, err)
	}
	return parsed, nil
}

// parsePath is ParsePath without the path in the message, which the one caller
// wraps on rather than every arm of the grammar carrying it.
func parsePath(path string) (Path, error) {
	if path == IntroductionPath {
		return Path{Form: FormIntroduction}, nil
	}

	segments := strings.Split(path, "/")
	switch segments[0] {
	case "records":
		return parseRecordPath(segments)
	case "journal":
		return parseJournalPath(segments)
	}
	return Path{}, fmt.Errorf("the Store has %s, records/ and journal/, and nothing else", IntroductionPath)
}

// parseRecordPath reads records/<target>/<definition>/<name>/<run-id>-<nnnn>.json.
func parseRecordPath(segments []string) (Path, error) {
	if len(segments) != 5 {
		return Path{}, fmt.Errorf("a Record version is a name under three identity segments, and this is %d segments", len(segments))
	}
	for _, segment := range segments[1:4] {
		if err := checkIdentitySegment(segment); err != nil {
			return Path{}, err
		}
	}

	name, found := strings.CutSuffix(segments[4], ".json")
	if !found {
		return Path{}, fmt.Errorf("a Record version is a .json file")
	}
	// The last hyphen and not the first: a UUIDv7 carries four of its own, so
	// cutting at the first would name a Run that stopped after eight digits.
	hyphen := strings.LastIndex(name, "-")
	if hyphen < 0 {
		return Path{}, fmt.Errorf("a Record version is named <run-id>-<nnnn>")
	}

	run, err := ParseRunID(name[:hyphen])
	if err != nil {
		return Path{}, err
	}
	step, err := parseStepNumber(name[hyphen+1:])
	if err != nil {
		return Path{}, err
	}
	return Path{Form: FormRecord, Run: run, Step: step}, nil
}

// parseJournalPath reads journal/<yyyy>/<mm>/<dd>/<run-id>/ and the four files
// that sit under it.
func parseJournalPath(segments []string) (Path, error) {
	if len(segments) < 6 {
		return Path{}, fmt.Errorf("a Journal path is a file under an entry's own directory")
	}
	partition := strings.Join(segments[1:4], "/")
	if day, err := time.Parse("2006/01/02", partition); err != nil || day.Format("2006/01/02") != partition {
		return Path{}, fmt.Errorf("%q is not a <yyyy>/<mm>/<dd> date partition", partition)
	}
	run, err := ParseRunID(segments[4])
	if err != nil {
		return Path{}, err
	}

	switch tail := segments[5:]; {
	case len(tail) == 1 && tail[0] == "run.json":
		return Path{Form: FormRun, Run: run, Entry: run}, nil

	case len(tail) == 1 && tail[0] == "outcome.json":
		return Path{Form: FormOutcome, Run: run, Entry: run}, nil

	case len(tail) == 2:
		// The two directories under an entry hold files and nothing
		// else, so the extension is cut once for both rather than
		// twice with two ways of saying the same thing.
		name, found := strings.CutSuffix(tail[1], ".json")
		if !found {
			return Path{}, fmt.Errorf("every file an entry holds is a .json file")
		}
		switch tail[0] {
		case "steps":
			step, err := parseStepNumber(name)
			if err != nil {
				return Path{}, err
			}
			return Path{Form: FormStep, Run: run, Entry: run, Step: step}, nil

		case "closed-by":
			closer, err := ParseRunID(name)
			if err != nil {
				return Path{}, err
			}
			if closer == run {
				return Path{}, fmt.Errorf("run %s closes its own entry: a closing write is a Run closing an entry it does not own (ADR-0076)", run)
			}
			return Path{Form: FormClosedBy, Run: closer, Entry: run}, nil
		}
	}
	return Path{}, fmt.Errorf("an entry holds run.json, steps/, outcome.json and closed-by/, and nothing else")
}

// parseStepNumber reads <nnnn> back to the position it renders. It is stated as
// the rendering rather than as a grammar of its own — a number whose rendering
// is not the text it was read from is a number written by something other than
// this package, and there is one spelling of every position (§12).
func parseStepNumber(number string) (int, error) {
	step, err := strconv.Atoi(number)
	if err != nil || step < 1 || stepNumber(step) != number {
		return 0, fmt.Errorf("%q is not a Step's position: the first Step is 0001 and the widths beyond four are unpadded", number)
	}
	return step, nil
}

// checkIdentitySegment answers whether a segment is one this package could have
// written: escapes whole and uppercase, no byte outside the unreserved set left
// bare, no leading `.`, and a width the truncation rule accounts for.
//
// It checks the shape and decodes nothing. What the segment means is not a
// question a path answers (§7), and the check is here so that a path from
// somewhere else is refused at the door rather than at the read.
func checkIdentitySegment(segment string) error {
	body, digest, truncated := strings.Cut(segment, "~")
	switch {
	case body == "":
		return fmt.Errorf("an identity segment is not empty")
	case truncated && len(digest) != digestWidth:
		return fmt.Errorf("a truncated segment carries %d digest digits, want %d", len(digest), digestWidth)
	case truncated && !isLowerHex(digest):
		return fmt.Errorf("%q is not lowercase hexadecimal", digest)
	case truncated && (len(body) > segmentLimit || len(body) < segmentLimit-2):
		return fmt.Errorf("a segment cut at %d is %d bytes wide, and the cut clears an escape by at most two", segmentLimit, len(body))
	case !truncated && len(body) > segmentLimit:
		return fmt.Errorf("a segment of %d bytes is past the %d the grammar cuts at", len(body), segmentLimit)
	case body[0] == '.':
		return fmt.Errorf("a leading . is escaped")
	}

	for i := 0; i < len(body); i++ {
		if body[i] != '%' {
			if strings.IndexByte(unreserved, body[i]) < 0 {
				return fmt.Errorf("%q is written as itself and is outside the unreserved set", body[i:i+1])
			}
			continue
		}
		if i+2 >= len(body) || strings.IndexByte(upperHex, body[i+1]) < 0 || strings.IndexByte(upperHex, body[i+2]) < 0 {
			return fmt.Errorf("%q is not a %% and two uppercase hexadecimal digits", body[i:])
		}
		i += 2
	}
	return nil
}
