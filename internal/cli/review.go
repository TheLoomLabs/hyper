package cli

import (
	"fmt"
	"io"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/cadence"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunReview implements `hyper review <artefact>` — §8's Definition review of
// the artefact named: every line of the working tree's file, verbatim, beneath
// a header saying what is being read (issue #118), annotated in place by the
// gutter to its left (issue #120).
//
// What the gutter marks on a Procedure is what a reviewer with the file open in
// a diff cannot see, because none of it is in the file: each Step's Kind,
// declared in a Manifest two directories away and never inferable from the
// Operation's name; the Target it binds; whether a `mutate` carries a Bound;
// whether the Operation's request is one `hyper` cannot describe; and the
// envelope check, which quantifies over every Step's `target:` at once. The
// other four artefacts' rosters are their own and land with them.
//
// It is `check` and `provider` in every other respect: the same globals, the
// same gate before the load, the same stream discipline, and the same reach —
// no credential resolves, no network is touched, no Store branch opens and no
// git object is read, so it answers on a fresh clone of a repository that has
// never run (§9).
//
// What is its own is the positional, which takes two forms and where neither is
// optional. A positional containing `/`, or ending in `.yaml`, is a
// repository-relative path resolved against the load's own paths; anything else
// is a name matched against the artefact's own `name:`. Both are mandatory and
// the reason is symmetric: the built-in Manifest has no file, so a path can
// never reach it, and `hyper.yaml` declares no name, so a path is the only
// thing that can (§9, ADR-0060).
//
// Its three exit codes are §9's and all three are reachable here. 0 where the
// review rendered, however much it had to say. 1 where the artefact under
// review is found and will not load, which writes `check`'s row for it. 2 where
// the positional named nothing — the usage error every command that takes a
// name answers with, no row stream opening and nothing reaching stdout. An
// artefact that loads and *names* one that is not there is none of those: it
// renders and exits 0, the fault being `check`'s to report (§9, ADR-0064).
//
// It takes no --limit: nothing on this screen is a result set, an artefact
// having neither an order nor a cap, so a review that dropped lines would be
// rendering something other than what is about to be approved (§8, §9).
func RunReview(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, code := parseArgs("review", args, takesNoLimit, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// Exactly one positional, decided from the argument list alone and
	// before any repository is resolved — the reading `provider` and
	// `operation` give their own arity, in the spelling all three share
	// (ADR-0060).
	if len(parsed.positional) != 1 {
		fmt.Fprintf(stderr, "hyper review: %s\n", arityFault(parsed.positional, "artefact"))
		return ExitUsage
	}
	named := parsed.positional[0]

	repoRoot, code := resolveRepoRoot("review", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and before the positional is
	// resolved: a mismatched pin plus a name matching nothing is 77 and not
	// 2, because the gate fires first for all sixteen (§9, §11, ADR-0020,
	// ADR-0060).
	if code := gateOnVersionPin("review", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper review: %s\n", err)
		return ExitUsage
	}

	found, code := resolveArtefact(named, loaded, stderr)
	if code != 0 {
		return code
	}
	reviewed := newReviewedArtefact(found, loaded)

	// Found and faulty is the one thing that is not a rendering: an artefact
	// the load could not read is reported as `check` reports it, one row per
	// problem, and exits 1 (§9). Found and *naming* something that is not
	// there is not this path — that renders and exits 0, the fault being
	// check's to report and this surface's to annotate (ADR-0064).
	if len(reviewed.problems) > 0 {
		rows := checkRows(reviewed.problems)
		page := func(w io.Writer, rows []render.Row) error { return render.WriteTable(w, checkColumns, rows) }
		if code := writeAnswer("review", stdout, stderr, parsed.json, rows, render.NewResultRow(false), page); code != 0 {
			return code
		}
		return ExitProblems
	}

	// One call to the absence pipeline, two notations out of it: the name
	// §12 closes the set at goes on the wire and the sentence §8 renders goes
	// on the page, so the two surfaces cannot name different absences.
	//
	// The page and the --json stream then come out of one row list
	// (ADR-0026): the header, and one `gutter` row per rendered line the
	// gutter has something to say about. The source is not on that list: a
	// review does not decompose into rows the way the change tables do, so
	// `review --json` emits the annotations and never the source, the
	// consumer already having the file (§8). So the page closes over the
	// lines it renders, exactly as `check`'s closes over the count its clean
	// line names, and reads the annotations off the rows like every other
	// page in this package.
	absent, sentence := rankedAbsence(reviewed)
	rows := append([]render.Row{newArtefactRow(reviewed, absent)}, gutterRows(reviewed.markers)...)
	rows = append(rows, authorityRows(reviewed.authority)...)
	rows = append(rows, flagRows(reviewed.flags)...)
	page := func(w io.Writer, rows []render.Row) error {
		return writeReviewPage(w, rows, reviewPage{
			sentence:  sentence,
			source:    reviewed.source,
			authority: reviewed.authority,
		})
	}
	if code := writeAnswer("review", stdout, stderr, parsed.json, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}

	return ExitClean
}

// reviewedArtefact is the artefact under review as this command reads it: what
// kind it is, where its bytes came from, the name it declares for itself, the
// lines it renders, and whatever the load found wrong with it.
//
// path is empty on the one artefact with no file in the repository, which is
// the fact two members of the header are read from — there is no path to state
// and no Run could have recorded a revision of what has none (ADR-0068). It is
// keyed on there being no file rather than on the artefact's kind, which is the
// decomposition ADR-0067 fixed for the range and ADR-0068 for the path.
type reviewedArtefact struct {
	kind     artefactKind
	path     string
	name     string
	cadence  string
	source   []string
	problems []problem.Problem

	// markers are the cells this artefact's marker column carries, in line
	// order and before the composition that aligns them. Five rosters reach
	// this one member: what an artefact marks is its kind's and what the
	// column does with a marker is every kind's (§8, issue #122).
	markers []reviewMarker

	// flags are §12's index into those marks, in the order both surfaces
	// carry them. They are read alongside the markers rather than after
	// them, which is what makes every row cite a line the gutter marked by
	// construction (§8, §12, issue #123).
	flags []reviewFlag

	// authority is the one relation §5 states, read from whichever end this
	// artefact supplies. Three of the five artefacts supply an end and two
	// are members of no pair at all (§8, ADR-0069).
	authority authorityBlock
}

// newReviewedArtefact is the resolved artefact read as the thing under review,
// and it is one constructor because the two forms of the positional resolve to
// the same value: which form found it is a fact about the argument and not
// about the artefact.
//
// It runs once, after the positional has settled, which is why the resolution
// above answers with a resolvedArtefact rather than with this: a Procedure's
// markers are read against the whole repository, and a constructor called once
// per candidate would build that walk for artefacts nobody is going to review.
//
// The Cadence is read on a Procedure and nowhere else. It is a Procedure's
// declaration (§10), so a key of that name on another artefact is not one — and
// reading it by kind rather than by key is what keeps the header from glossing
// something no author declared.
//
// The one artefact with no file in the repository carries no path here. The
// load gives it a pseudo-path so that it is a member on the same footing as the
// rest (ADR-0039); this surface has a path column of openable files, and what
// stands in it for an artefact that has none is nothing at all — the header's
// sentence beside it already says where the bytes are (ADR-0068).
func newReviewedArtefact(found resolvedArtefact, loaded repository.Loaded) reviewedArtefact {
	a, kind := found.artefact, found.kind
	file := a.Path
	if file == artefact.BuiltinShellProviderPath {
		file = ""
	}
	declared := ""
	if kind.wire == artefact.KindProcedure {
		declared = artefact.ProcedureCadence(a.Root)
	}
	marks := readMarks(found, loaded)
	return reviewedArtefact{
		kind:      kind,
		path:      file,
		name:      artefact.DeclaredName(a.Root, kind.nameKey),
		cadence:   declared,
		source:    artefact.SourceLines(a.Bytes),
		markers:   marks.markers,
		flags:     marks.flags,
		problems:  a.Problems,
		authority: readAuthority(found, loaded),
	}
}

// resolvedArtefact is what the positional resolved to: one artefact the load
// walked, and what kind the load's path says it is. It is the resolution's
// answer and not the review's subject — reading those bytes into what the page
// renders is newReviewedArtefact's, and it happens once, on the one artefact
// that survived the ambiguity below.
type resolvedArtefact struct {
	artefact repository.LoadedArtefact
	kind     artefactKind
}

// artefactKind is one of §12's five artefact `kind:` values with the three
// other spellings of it this surface needs: the word heading the marker column,
// the noun a message calls it by, and the key an artefact of that kind declares
// its own name under.
//
// The wire's value and the page's word are two spellings of one fact and not a
// disagreement: §12 closes the `kind:` values a repository author writes and
// the row carries one of those, while §8 fixes the five words the marker column
// is headed by — `MANIFEST` over a Provider's Manifest, `TARGET` over a Target
// declaration — because a header naming what the file is read as is what a
// reviewer reads down the column beneath it.
type artefactKind struct {
	wire, word, noun, nameKey string
}

// artefactKinds is §12's directory-to-`kind:` mapping, which is also the
// mapping from a loaded path to everything this surface says about what it is
// reading. `hyper.yaml` is the one artefact keyed by its filename rather than
// by a directory, and the one that declares no name — which is why a path is
// the only thing that reaches it.
var artefactKinds = []struct {
	// location is where an artefact of this kind is read from: a directory,
	// written with the trailing slash every path test in this repository
	// spells one with, or — for the one artefact keyed by its filename — the
	// file itself, at the repository root.
	location string
	kind     artefactKind
}{
	{"definitions/", artefactKind{"definition", "DEFINITION", "Definition", "definition"}},
	{"procedures/", artefactKind{"procedure", "PROCEDURE", "Procedure", "procedure"}},
	{"providers/", artefactKind{"provider", "MANIFEST", "Manifest", "provider"}},
	{"targets/", artefactKind{"target-declaration", "TARGET", "Target declaration", "target"}},
	{"hyper.yaml", artefactKind{"repository-declaration", "REPOSITORY", "Repository declaration", ""}},
}

// kindByWire is one of the five looked up by the `kind:` value it carries, and
// the zero kind for a value there is none for — unreachable from a resolved
// artefact, every one of them carrying one of the five.
func kindByWire(wire string) artefactKind {
	for _, mapped := range artefactKinds {
		if mapped.kind.wire == wire {
			return mapped.kind
		}
	}
	return artefactKind{}
}

// kindOf is what the load's path says the artefact is. It reads the location
// rather than the file's own `kind:` key because the two are one fact — a file
// whose directory and `kind:` disagree is a load error §12 already names — and
// because a review states what it is reading even where the key is the thing
// that is wrong with the file (ADR-0064).
func kindOf(loadedPath string) (artefactKind, bool) {
	if loadedPath == artefact.BuiltinShellProviderPath {
		// The one artefact with no file: a Provider's Manifest whose bytes
		// are compiled into the binary, and whose pseudo-path names no
		// directory to read a kind off (§11, ADR-0039).
		return kindByWire("provider"), true
	}
	for _, mapped := range artefactKinds {
		if inDirectory := strings.HasSuffix(mapped.location, "/"); inDirectory {
			if strings.HasPrefix(loadedPath, mapped.location) {
				return mapped.kind, true
			}
		} else if loadedPath == mapped.location {
			return mapped.kind, true
		}
	}
	return artefactKind{}, false
}

// locationOf is where an artefact of that kind is read from: the directory §12
// maps the kind: value to, or the file itself for the one artefact keyed by its
// filename. It reads the same table kindOf reads backwards, so a surface that
// counts what is in a location and the resolution that decides what a location
// means cannot name two different directories.
func locationOf(wire string) string {
	for _, mapped := range artefactKinds {
		if mapped.kind.wire == wire {
			return mapped.location
		}
	}
	return ""
}

// resolveArtefact is the positional resolved to the artefact under review, in
// whichever of its two forms it was written.
//
// The discriminator is a `/` or a `.yaml` suffix, and it decides which
// namespace the positional is resolved against rather than what is looked for
// first: a name is never tried as a path and a path is never tried as a name.
// A form that fell back to the other would make which artefact `hyper review
// deploy` renders depend on what else is in the repository (§9, ADR-0060).
func resolveArtefact(named string, loaded repository.Loaded, stderr io.Writer) (resolvedArtefact, int) {
	if isPathForm(named) {
		return resolveByPath(named, loaded, stderr)
	}
	return resolveByName(named, loaded, stderr)
}

// isPathForm says which form the positional was written in: a positional
// containing a `/`, or ending in `.yaml`, is a repository-relative path, and
// anything else is a name. The two spellings are what a caller has in hand —
// a path is what `check` cites and what an editor opens, and a name is what an
// artefact declares and what every other artefact refers to it by.
func isPathForm(named string) bool {
	return strings.Contains(named, "/") || strings.HasSuffix(named, ".yaml")
}

// resolveByPath matches the positional against the load's own paths. It is
// cleaned first, so `./procedures/x.yaml` is the path it names, and matched
// against nothing else: the built-in Manifest's pseudo-path is not a file and
// is not reachable this way, which is the other half of why both forms are
// mandatory (ADR-0039, ADR-0068).
func resolveByPath(named string, loaded repository.Loaded, stderr io.Writer) (resolvedArtefact, int) {
	wanted := path.Clean(named)
	for _, a := range loaded.Artefacts {
		if a.Path == artefact.BuiltinShellProviderPath {
			// The pseudo-path the load carries the built-in under is not
			// a file and is not a positional anybody can type: a path
			// resolves against the load's files, and this artefact has
			// none (ADR-0039, ADR-0068).
			continue
		}
		if a.Path != wanted {
			continue
		}
		kind, known := kindOf(a.Path)
		if !known {
			continue
		}
		return resolvedArtefact{artefact: a, kind: kind}, 0
	}
	fmt.Fprint(stderr, unresolvedArtefactPath(named))
	return resolvedArtefact{}, ExitUsage
}

// resolveByName matches the positional against the artefacts' own declared
// names: byte-exact over UTF-8, case-sensitive, with no normalisation, and
// never settled by whether a filesystem open succeeded (§9, ADR-0060). A
// macOS filesystem is case-insensitive and a runner's is not, so the fold is
// hyper's rather than the filesystem's or the same review renders on a laptop
// and exits 2 in CI.
//
// It matches across every artefact namespace at once, which is what makes a
// name matching in two of them answerable at all: the four namespaces are four
// namespaces, and a positional that is a Definition here and a Target
// declaration there names two artefacts. The command cannot pick one, so it
// says which kinds it matched and points at the form that names a file.
//
// A file that will not parse declares no name, so it is not in any namespace
// and this form does not reach it: `hyper review half-written` is the usage
// error above and not the load failure, because a name that resolves to nothing
// is there being no act to decline (ADR-0060, ADR-0064). The path form is what
// reaches a faulty file — a path resolves against the load's own paths and a
// file that failed to parse is one of them — and that is where §9's *found and
// faulty* exits 1 with `check`'s row. The two forms differing there is the
// namespaces differing, not two answers to one question.
func resolveByName(named string, loaded repository.Loaded, stderr io.Writer) (resolvedArtefact, int) {
	var matched []resolvedArtefact
	for _, a := range loaded.Artefacts {
		kind, known := kindOf(a.Path)
		if !known || kind.nameKey == "" {
			continue
		}
		if artefact.DeclaredName(a.Root, kind.nameKey) != named {
			continue
		}
		matched = append(matched, resolvedArtefact{artefact: a, kind: kind})
	}

	switch {
	case len(matched) == 0:
		fmt.Fprint(stderr, unresolvedArtefactName(named))
		return resolvedArtefact{}, ExitUsage
	case len(matched) > 1:
		fmt.Fprint(stderr, ambiguousArtefactName(named, matched))
		return resolvedArtefact{}, ExitUsage
	}
	return matched[0], 0
}

// unresolvedArtefactName is what a name matching nothing writes, and it is
// `provider`'s message under a wider namespace: stderr, stdout completely
// silent in both modes, no row stream and so no terminal row to be missing, no
// error_code — nothing was reviewed, so nothing was refused (§9, ADR-0060).
//
// It names what was typed, the namespaces it was resolved against, and how to
// enumerate them, and it suggests no near miss (ADR-0047): a suggestion is a
// partial name resolved on the caller's behalf, and a human who accepts one has
// reviewed something they did not type. Two of the four namespaces have a
// listing command and two have none, so the remedy names the two commands there
// are and the path form, which reaches every artefact that has a file.
func unresolvedArtefactName(named string) string {
	return fmt.Sprintf("hyper review: no artefact named %q in this repository's Definition, Procedure, Provider or Target namespaces\n"+
		"  hyper providers and hyper targets list two of the four\n"+
		"  a repository-relative path — one containing / or ending .yaml — names any artefact with a file\n",
		named)
}

// unresolvedArtefactPath is the same usage error read off the other form: a
// path resolving to no artefact the load walked. It names the five locations
// §12 fixes rather than a listing command, there being no command that
// enumerates artefact paths and none that would resolve this positional for the
// caller.
func unresolvedArtefactPath(named string) string {
	return fmt.Sprintf("hyper review: no artefact at %q in this repository\n"+
		"  an artefact's file is in definitions/, procedures/, providers/ or targets/, or is hyper.yaml itself\n",
		named)
}

// ambiguousArtefactName is what a bare name matching in more than one artefact
// namespace writes. It is a usage error like the two above and for the same
// reason: nothing was reviewed, and the remedy is a different positional rather
// than an artefact edit.
//
// It says which kinds it matched, because that is the whole of what is
// ambiguous, and points at the path form, which is the one positional that
// names a file and cannot be ambiguous at all.
func ambiguousArtefactName(named string, matched []resolvedArtefact) string {
	nouns := make([]string, 0, len(matched))
	for _, m := range matched {
		nouns = append(nouns, "a "+m.kind.noun)
	}
	return fmt.Sprintf("hyper review: %q names more than one artefact in this repository: %s\n"+
		"  name the file instead — a positional containing / or ending .yaml is a repository-relative path\n",
		named, strings.Join(nouns, " and "))
}

// artefactRow is the review's header on the wire, and it is one row rather than
// one per rendered line: a header cites no line, so a row per line would invent
// an anchor the surface does not have (§8). Its members are §8's own in §8's
// order — what is under review, where its bytes are, and the range it is read
// across.
//
// path is absent exactly where baseline_absent is `built-in`, and the two are
// one fact: an artefact with no file has no path, and no Run could have
// recorded a revision of what has none. The name already on the row is the
// discriminator and a second key would carry that fact twice (§8, ADR-0068).
//
// The gloss arrives as its parts and never as the composed string. A `gutter`
// row goes the other way and says why — a decomposition is a second rendering
// of one fact, and the second one can be wrong about the first — but a marker
// is one derived fact in one cell, where the gloss is several facts with
// several supplies sharing a line. Carrying both the parts and the string would
// be the failure that row names (§8, ADR-0063).
//
// rate is a pointer because zero is a rate: an expression the calendar has no
// instance of matches nothing, and a `rate` key merely missing where the gloss
// rendered would be the omission this screen forbids everywhere else (ADR-0026).
//
// `last_run` is not here, and its absence is one absence and not two: the range
// and *last ran* read the same Journal entry under the same filter, so the
// header states it once on the range's line and this line carries only what the
// artefact's own bytes supply. There is no Store in this milestone to hold one
// either way (§8, §10).
//
// baseline and baseline_absent are the two halves of one member: a review with
// a range carries the revision it opened at, and one with none carries which of
// §12's four names it is, a key merely missing collapsing four different facts
// into one reading. Only the second is declared here. No range opens in this
// milestone — there is no Store to ask — so `baseline` has nothing to carry,
// and a member declared against nothing is the stub the absence pipeline below
// refuses for the same reason: it arrives with the supply that fills it.
type artefactRow struct {
	Type           string   `json:"type"`
	Kind           string   `json:"kind"`
	Path           string   `json:"path,omitempty"`
	BaselineAbsent string   `json:"baseline_absent,omitempty"`
	Cadence        string   `json:"cadence,omitempty"`
	Phrase         string   `json:"phrase,omitempty"`
	Rate           *float64 `json:"rate,omitempty"`

	// rateText is the rate in the notation the page renders it in — the
	// `≈` where it was rounded, and the unit fixed at runs per month (§10).
	// It is off the wire because the wire carries the number, and it is on
	// the row because the page is written from the rows (ADR-0026): both
	// come out of one reading of the expression, so the digits the page
	// renders and the number the row carries are one rounding and cannot
	// disagree.
	rateText string
}

// newArtefactRow is the header as one row: the kind, the path where there is a
// file, the name the absence pipeline ranked, and — where the artefact declares
// a Cadence §10's grammar admits — the gloss's parts.
//
// An expression outside that grammar carries no gloss and is not refused:
// `cadence-malformed` is §12's static check and lands with the milestone that
// projects a Cadence into a workflow. Until it does, a review renders such an
// artefact like any other and says nothing about it — the gloss is a reading of
// the grammar, and what is not in the grammar has no reading (§10, ADR-0064).
func newArtefactRow(reviewed reviewedArtefact, absent string) artefactRow {
	row := artefactRow{
		Type:           "artefact",
		Kind:           reviewed.kind.wire,
		Path:           reviewed.path,
		BaselineAbsent: absent,
	}
	if gloss, readable := cadence.Read(reviewed.cadence); readable {
		row.Cadence = gloss.Expression
		row.Phrase = gloss.Phrase
		row.Rate = &gloss.Rate
		row.rateText = gloss.RateText
	}
	return row
}

// Cells is empty: the header is a block of one fact per line rather than a line
// in a table of like rows, and the page renders it as writeReviewPage writes it
// (ADR-0026).
func (r artefactRow) Cells() []string { return nil }

// absenceStage is one stage of §8's absent-baseline pipeline: the name §12
// closes the set at, and what the header's sentence says where this stage is
// the answer. A stage answers "" where it did not fire.
type absenceStage struct {
	name  string
	fires func(reviewedArtefact) string
}

// absencePipeline is §8's four absences ranked — and it is written as the
// pipeline it is rather than as a comparison between four facts, because that
// is how a reader checks a rendering against it: each stage is reachable only
// where the one before it did not fire (§8, §12).
//
// Two of the four are live here and two are not written at all. `built-in`
// fires on the one artefact with no file in the repository, and its absence is
// permanent — its bytes move only when the binary does. `no-store` fires on
// every other artefact and is the true answer, no repository having a Store
// before `store init` exists. `not-run` and `not-in-clone` are unreachable —
// there is no Journal to ask and no revision to fail to find — so they are
// absent from this list rather than stubbed in it, and the milestone that reads
// a Journal adds them between and after the two below rather than re-deriving
// the ranking.
var absencePipeline = []absenceStage{
	{"built-in", func(reviewed reviewedArtefact) string {
		if reviewed.path != "" {
			return ""
		}
		return reviewed.name + " ships in the binary"
	}},
	{"no-store", func(reviewedArtefact) string {
		return "no Store"
	}},
}

// rankedAbsence is the one absence the header states and the wire carries: the
// first stage to fire, and the sentence it says it in. One lead-in phrase
// serves all of them — a second form of words would make the eye sort the
// absences before reading them, which is the ranking `FLAGS` is the one surface
// permitted to do (§8, ADR-0026).
//
// It answers nothing where no stage fired, which is unreachable today: the last
// stage above is the true answer for every artefact that reaches it. It is
// written as an answer rather than as a fallthrough because the milestone that
// opens a range asks this only where none opened, and a pipeline that could not
// say *nothing is absent* would have to be re-derived there.
func rankedAbsence(reviewed reviewedArtefact) (name, sentence string) {
	for _, stage := range absencePipeline {
		if text := stage.fires(reviewed); text != "" {
			return stage.name, "no baseline — " + text
		}
	}
	return "", ""
}

// The review's geometry: the screen's own margins, and the one column width on
// it that is not read off what is being rendered.
const (
	// reviewIndent is what every line of the screen opens with, the gutter
	// included, exactly as §8 renders it.
	reviewIndent = "  "
	// reviewFieldGap is the least space this screen puts between two things
	// on one line: the marker column and the bar it stands left of, the bar
	// and the header's first field, and one header field and the next. It is
	// the two spaces every aligned page in this package separates a column
	// by, stated once here because the review draws its own alignment rather
	// than handing it to a tabwriter.
	reviewFieldGap = "  "
	// reviewMarkerPad is that gap as a width, which is what the marker
	// column is padded and the rule is drawn by. The column itself is as
	// wide as the widest marker in the rendering and the kind heading it.
	reviewMarkerPad = len(reviewFieldGap)
	// reviewCaptionGap is what stands between a block's name and the line
	// saying what the block is. It is wider than the gap between two
	// columns, and by one character rather than by an alignment: the caption
	// is a title and its second half a sentence, so nothing beneath it is
	// aligned against either, and §8 renders `AUTHORITY` and `FLAGS` alike
	// with it however long their names are.
	reviewCaptionGap = "   "
	// reviewSourceGap is what stands between the bar and the artefact's own
	// line. It is one character wide because the change column is not drawn
	// at all in this milestone — no range opens, so that column has no
	// content and no width, and the source sits two characters left of where
	// a ranged review would put it: the column and the space that would
	// separate it from the source (§8).
	reviewSourceGap = " "
	// reviewPathColumn is where the header's first line states the range, or
	// the absence standing in its place. A header is one row, so nothing on
	// the screen supplies a width to align it against; it is fixed so that
	// two reviews of two artefacts state the range in the same place, which
	// is what §8's own renderings do.
	reviewPathColumn = 40
)

// writeReviewPage is the review's page: the header, the rule beneath it, and
// then the artefact's own lines, each annotated in place by the gutter to its
// left (§8, ADR-0026). It is `review`'s own rather than a WriteTable because a
// review is stacked renderings and not a table of like rows, which is why
// writeAnswer takes a page function at all.
//
// The page's own supply is closed over rather than carried on a row: `review
// --json` emits the annotations and never the source, the consumer already
// having the file (§8).
//
// The gutter is two columns and here the second one is not drawn at all. There
// is no Store in this milestone and so no range ever opens, which means the
// change column has no content **and no width**, and the source sits two
// characters left of where a ranged review would put it — the absence being the
// state §8 names rather than a column pending its first mark. A blank column
// one character wide is the one thing this screen may not do.
func writeReviewPage(w io.Writer, rows []render.Row, page reviewPage) error {
	if len(rows) == 0 {
		return nil
	}
	header, written := rows[0].(artefactRow)
	if !written {
		return nil
	}

	facts := headerFacts(header, page.sentence)

	// The marker column is as wide as the widest marker in this rendering,
	// or the kind heading it where that is wider, and a marker is never
	// truncated: a Target name is an identity in a reviewed artefact, and
	// eliding a character of one is this screen stating something other than
	// what is about to be approved (§8). Nothing else on the screen is sized
	// to anything either: §9's truncation discipline governs a result set,
	// and an artefact has neither an order nor a limit.
	marked := markersByLine(rows)
	marker := utf8.RuneCountInString(header.markerHeading())
	for _, text := range marked {
		if width := utf8.RuneCountInString(text); width > marker {
			marker = width
		}
	}

	lines := make([]string, 0, len(facts)+len(page.source)+1)
	for i, fact := range facts {
		heading := ""
		if i == 0 {
			heading = header.markerHeading()
		}
		lines = append(lines, gutter(heading, marker)+reviewFieldGap+fact)
	}
	// The rule spans the header it stands under: the gutter's own width to
	// the left of the bar, and the widest header line to the right of it.
	lines = append(lines, reviewIndent+
		strings.Repeat("─", marker+reviewMarkerPad)+"┼"+
		strings.Repeat("─", widestOf(facts)+len(reviewFieldGap)))
	for n, line := range page.source {
		lines = append(lines, gutterLine(gutter(marked[n+1], marker)+reviewSourceGap, line))
	}

	for _, line := range lines {
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	if err := writeAuthorityBlock(w, header.Kind, rows, page.authority); err != nil {
		return err
	}
	return writeFlagsBlock(w, rows)
}

// reviewPage is what the screen renders that no row carries: the header's
// absence sentence, the artefact's own lines, and what the `AUTHORITY` block
// says about itself.
//
// The source is here rather than on a row because `review --json` emits the
// annotations and never the source, the consumer already having the file; the
// other two are here for ADR-0069's own reason, a rendering that emits no rows
// having nothing on the wire to carry its absence on (§8, ADR-0026).
type reviewPage struct {
	sentence  string
	source    []string
	authority authorityBlock
}

// markersByLine is the rendering's marker cells keyed by the line each stands
// beside — the page's own index into the rows it is written from, and the one
// place the page reads a `gutter` row. Every line number here is the working
// tree's, counted from one over every line of the file including blank ones,
// which is the numbering a flag's citation and a `gutter` row's `line` share
// (§8).
func markersByLine(rows []render.Row) map[int]string {
	marked := map[int]string{}
	for _, row := range rows {
		if g, drawn := row.(gutterRow); drawn {
			marked[g.Line] = g.markerText
		}
	}
	return marked
}

// markerHeading is the word standing at the top of the marker column: the
// artefact's kind, which is true on all five, where a header naming blast
// radius would describe a Procedure's marks and misdescribe a Repository
// declaration's retention policy (§8).
//
// It reads the row rather than the reviewed artefact because the page is
// written from the rows (ADR-0026), and §12's `kind:` value is what a row
// carries — the word is this surface's rendering of it.
func (r artefactRow) markerHeading() string { return kindByWire(r.Kind).word }

// headerFacts is the header's lines: one fact per line, each read from its own
// supply and citing no line (§8, ADR-0063).
//
// The first line carries the path and the range, and where no range opens it
// carries the absence in the range's own position. On an artefact with no file
// the path goes with it and the line collapses to its one field: the sentence
// beside it already says where the bytes are, so `path` is silenced rather than
// dropped, and the width it would have taken is not left as blank run-up —
// whitespace where a member was is omission wearing a rendering (ADR-0068).
//
// The second line is the Cadence gloss, and it takes a line of its own rather
// than being composed onto the first: composed, the screen's width would depend
// on how awkwardly an expression glosses, and §10's grammar admits expressions
// that gloss at very different lengths. It sits above the rule, because below
// it it would stand directly over `kind: procedure` and read as an annotation
// of that line, which is the one thing the header may not be (§8, ADR-0063).
//
// An artefact with no Cadence beneath it renders a one-line header: the gloss's
// absence takes the line rather than leaving one blank.
func headerFacts(header artefactRow, sentence string) []string {
	first := inColumn(header.Path, reviewPathColumn) + sentence
	if header.Path == "" {
		first = sentence
	}
	if header.Phrase == "" {
		return []string{first}
	}
	// How the parts are arranged is the surface's, and what they are is
	// not (§10). On a header they share one line, separated by `·`, the
	// expression already being on the `cadence:` line below.
	return []string{first, header.Phrase + " · " + header.rateText}
}

// gutter is one line's left-hand side: the indent, the marker cell padded to
// the column's width, the pad, and the bar the two halves of the screen meet
// at.
func gutter(marker string, width int) string {
	return reviewIndent + inColumn(marker, width+reviewMarkerPad) + "│"
}

// gutterLine is one rendered line: its gutter, and whatever stands to the right
// of it. A line the artefact left empty ends where the gutter does — the
// padding is this page's and a line does not end in it — while whatever the
// file wrote is handed through untouched, trailing spaces included, a review
// rendering the bytes that are about to be approved (§8, ADR-0057).
func gutterLine(prefix, content string) string {
	if content == "" {
		return strings.TrimRight(prefix, " ")
	}
	return prefix + content
}

// inColumn is s standing in a column of that width: padded out to it with
// spaces, or — where it is already at or past the width — ended with the least
// gap this screen puts between two things on one line, so that a long path
// pushes the range right rather than colliding with it.
//
// Width is counted in runes rather than bytes: the screen is aligned for an
// eye, and an em dash is one column wide however many bytes it takes.
func inColumn(s string, width int) string {
	if utf8.RuneCountInString(s) >= width {
		return s + reviewFieldGap
	}
	return padTo(s, width)
}

// padTo is s padded out to that width with spaces, and s itself where it is
// already at or past it. It is inColumn without the overflow gap, which is what
// a field *inside* a marker needs: the gap between a Kind and its Target is the
// composition's and is written once, where a column that supplied its own would
// double it on the widest token in the rendering.
func padTo(s string, width int) string {
	if run := utf8.RuneCountInString(s); run < width {
		return s + strings.Repeat(" ", width-run)
	}
	return s
}

// widestOf is the widest of the lines given, in runes.
func widestOf(lines []string) int {
	widest := 0
	for _, line := range lines {
		if width := utf8.RuneCountInString(line); width > widest {
			widest = width
		}
	}
	return widest
}
