package cli

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// RunCompact implements `hyper compact` — the second command to touch the
// record, and the first thing in the tool that removes anything (issue #131).
//
// What it removes is `internal/store`'s predicate and not this file's: for
// every Observation series the branch holds, a version is removable exactly
// where it is not the Head, not the series' first version, and older than
// `retention:`. What this file decides is everything around it — which policy
// is in force, which Refusals precede the work, and what the two surfaces say
// afterwards.
//
// The order is §9's and is the same everywhere: the repository root, the gate,
// then the command's own work. Past the gate it is the Store before the policy,
// because the two answers are not alike — a repository with no branch is a
// guardrail declining at 77 and a repository with no policy is a clean run that
// removed nothing, and a command that read the policy first would answer *there
// is nothing to remove* to a caller who has no Store at all.
//
// It is not a Run. It writes no Journal entry, terminates its stream with
// `result` rather than `outcome`, and can never exit 75: that code is a Run
// that lost the Store, and this command has no outcome triple to map onto (§9,
// §12).
func RunCompact(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string, now func() time.Time) int {
	// No --limit: `compact` reports what it just did rather than ranging
	// over a namespace, so there is no result set for a cap to cut (§9).
	// Every other flag a caller might reach for here — --dry-run,
	// --retention, --keep-versions — is an unknown flag by the same
	// omission: retention is read-time and lives in one reviewed artefact,
	// and a flag behind it would let one invocation remove more than the
	// repository ever agreed to (§7, ADR-0001, ADR-0014).
	parsed, code := parseArgs("compact", args, takesNoLimit, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// `compact` names no namespace and resolves no name in one, so it takes
	// no positional at all: §9 gives one to nine of the sixteen and this is
	// not one of them.
	if len(parsed.positional) > 0 {
		fmt.Fprintf(stderr, "hyper compact: takes no positional argument, got %s\n", parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("compact", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the branch is read and before any row exists: a
	// mismatched pin exits 77 without a single git subprocess running (§9,
	// §11, ADR-0020).
	if code := gateOnVersionPin("compact", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper compact: %s\n", err)
		return ExitUsage
	}
	policy, inForce := retentionPolicy(loaded)

	// The sync, then the handle. They are two calls because their failures
	// are two facts: a Store neither side holds is a guardrail declining,
	// and a remote that could not be reached is the world resisting — and
	// folding them together would tell a caller that a verbatim retry
	// Refuses identically, which is false the moment the network returns
	// (§7, §12, ADR-0061).
	instant := now()
	if err := store.Sync(repoRoot, instant); err != nil {
		return reportStoreFault(stderr, err)
	}
	held, err := store.Open(repoRoot, instant)
	if err != nil {
		return reportStoreFault(stderr, err)
	}

	// A repository that has not stated a policy has not agreed to lose
	// anything, so nothing is read and nothing is removed (§3). It names the
	// artefact and the key rather than printing a bare *nothing to do*: this
	// is the empty state most likely to be reached and the one most likely
	// to be read as a bug.
	//
	// Nothing is read is literal, and it decides one thing worth stating:
	// the schema ceiling is a **reader's** rule, so a repository with no
	// policy and a Store file above the ceiling exits 0 rather than
	// Refusing. `compact` tests the files it will read, and under no policy
	// it will read none (§7, ADR-0028, issue #131).
	if !inForce {
		if code := writeCompaction(stdout, stderr, parsed.json, store.Compaction{}, unboundedLine(policy)); code != 0 {
			return code
		}
		return ExitClean
	}

	done, err := held.Compact(policy)
	if err != nil {
		return reportStoreFault(stderr, err)
	}

	if code := writeCompaction(stdout, stderr, parsed.json, done, compactedLine(done, policy)); code != 0 {
		return code
	}
	return ExitClean
}

// retentionPolicy is the policy in force, and whether there is one to act
// under. The second answer is *in force* and not *stated*: a `retention:` the
// duration grammar does not admit is stated and is not a policy, and the two
// empty states are told apart further down by whether anything was declared at
// all.
//
// It is read from the Repository declaration and from nowhere else: no flag, no
// environment variable, and no default a binary carries — a repository that
// declared nothing keeps everything, forever (§7, ADR-0001, ADR-0014).
//
// A `retention:` that is not a duration is not a policy either, and it is
// stated as such rather than guessed at. The value is `schema-mismatch` and
// `check` is what reports it (ADR-0064); what this command does about it is
// remove nothing, which is the one reading that cannot lose a Record to an
// authoring mistake.
// The two accessors it goes through — repository.Loaded.Declaration and
// artefact.ReadRepositoryFacts — are readings of what the load already parsed
// and the schema already typed, and not additions to either's contract: issue
// #131 says the loader returns the parsed declaration and `internal/artefact`
// types the key as a duration, and both are still true. What they replace is a
// command comparing a path against the literal `hyper.yaml` and walking a YAML
// node for itself, which is the fourth site of one string and a second reader
// of one key.
func retentionPolicy(loaded repository.Loaded) (store.Retention, bool) {
	declared := artefact.ReadRepositoryFacts(loaded.Declaration()).Retention
	if declared == "" {
		return store.Retention{}, false
	}
	seconds, reads := schema.DurationSeconds(declared)
	if !reads {
		return store.Retention{Declared: declared}, false
	}
	return store.Retention{Declared: declared, Age: time.Duration(seconds) * time.Second}, true
}

// reportStoreFault renders whichever way the record stopped this command, and
// answers the exit code (§9, §12).
//
// The three are told apart by what it would take to clear them. A branch
// neither side holds Refuses `store-absent` at 77 naming `hyper store init` —
// the remedy is an act of somebody's, which is exactly what sorts 77 from 75
// (ADR-0061). A file written above this reader's ceiling Refuses
// `store-schema-unsupported` at the same code, the remedy there being a
// different binary (ADR-0028). Everything else is the world resisting at 1: a
// push exhausted, a remote unreachable, a git object that would not read. Never
// 75, which is a Run that lost the Store.
func reportStoreFault(stderr io.Writer, err error) int {
	var unsupported store.SchemaUnsupported
	switch {
	case errors.Is(err, store.ErrAbsent):
		return refuse(stderr, storeAbsentCode, "no "+store.BranchName+" branch in this repository — hyper store init")
	case errors.As(err, &unsupported):
		// The message carries the path the store package named, which is
		// the file the Refusal cites — §8 states that this code cites a
		// Store file, and it is the one Refusal whose subject is evidence
		// rather than an artefact.
		return refuse(stderr, store.SchemaUnsupportedCode, fmt.Sprintf("%s — install a hyper that reads it", err))
	}
	fmt.Fprintf(stderr, "hyper compact: %s\n", err)
	return ExitProblems
}

// storeAbsentCode is the error_code a command that needs the Store and does not
// find one Refuses under (§12). It is spelled here rather than in
// internal/store because that package holds no Run, renders no Refusal and
// knows no code — it reports the condition, and this maps it, exactly as
// SchemaUnsupported is reported there and rendered here.
const storeAbsentCode = "store-absent"

// writeCompaction writes the answer in whichever mode was asked for: one
// `version` row per removed version and the terminal `result`, or the page and
// the command's own line beneath it.
//
// Where nothing was removed, that line also goes to **stderr in --json mode**,
// which is `targets`' truncation line read once more: a fact that is not a row
// has no place on stdout, and a stream of no rows terminated by `result` says
// which nothing happened to nobody. It is written only there, and not beside a
// stream that carried rows, because the rows *are* the answer where there are
// any — the reason there are none is the one thing the wire cannot otherwise
// state, and §9's two empty states have to be told apart in both forms (§9,
// ADR-0026).
func writeCompaction(stdout, stderr io.Writer, asJSON bool, done store.Compaction, line string) int {
	rows := make([]render.Row, 0, len(done.Removed))
	for _, version := range done.Removed {
		rows = append(rows, versionRow{
			Type:       "version",
			Target:     version.Identity.Target,
			Definition: version.Identity.Definition,
			Name:       version.Identity.Name,
			RunID:      version.Run.String(),
			Step:       version.Step,
			WrittenAt:  store.InstantText(version.WrittenAt),
		})
	}
	page := func(w io.Writer, rows []render.Row) error { return writeVersionTable(w, rows, line) }
	// `truncated` is always false: `compact` takes no --limit and answers no
	// question with a result set to cut. The truncation axes §12 closes are
	// the record's two, and a report of what a command just did is on
	// neither.
	if code := writeAnswer("compact", stdout, stderr, asJSON, rows, render.NewResultRow(false), page); code != 0 {
		return code
	}
	if asJSON && len(rows) == 0 {
		fmt.Fprintf(stderr, "hyper compact: %s\n", line)
	}
	return 0
}

// versionRow is `compact`'s row, and one is written per removed version, in
// path order.
//
// It names the version by its Run and its Step — the two segments of its file
// name — and never by its ordinal, which this command is precisely the thing
// that moves: removing an interior version renumbers every version above it
// (ADR-0049). The value it is built from carries no ordinal at all, so there is
// none here to write by accident.
//
// The members are the issue's own, in its order:
// {"type":"version","target":…,"definition":…,"name":…,"run_id":…,"step":…,
// "written_at":…}. encoding/json marshals a struct's fields in declaration
// order, which is what fixes a row's key order on the wire, and the type
// declared first is what puts it first.
type versionRow struct {
	Type       string `json:"type"`
	Target     string `json:"target"`
	Definition string `json:"definition"`
	Name       string `json:"name"`
	RunID      string `json:"run_id"`
	Step       int    `json:"step"`
	WrittenAt  string `json:"written_at"`
}

// Cells is the row's line on the page, in versionColumns' order.
//
// `step` rides on the wire only, which is check's `column` read once more: a
// row carrying more than its page renders is not the two surfaces disagreeing
// but one of them having no column for a fact a consumer filters on. A Run id
// is unique to a Run and the Step disambiguates two versions of one series
// written by one Run — rare enough that a column for it would be blank-looking
// on every ordinary page.
//
// The Run renders whole. An id a human retypes renders entire wherever it
// renders at all, and there is no elided form of one anywhere (ADR-0047).
func (r versionRow) Cells() []string {
	return []string{r.Target, r.Definition, r.Name, r.WrittenAt, r.RunID}
}

// versionColumns is `compact`'s header: the series a removed version belonged
// to, when it was written, and what wrote it.
var versionColumns = []string{"TARGET", "DEFINITION", "RECORD", "WRITTEN", "RUN"}

// writeVersionTable is `compact`'s page: the removed versions, and the
// command's own line beneath them.
//
// The line is written in both cases and the header in only one, which is
// `internal/render`'s stated rule and `targets`' precedent: a command that
// removed nothing has no rows and therefore no table, and what stands in its
// place is the sentence that says which nothing happened (§9, issue #99).
func writeVersionTable(w io.Writer, rows []render.Row, line string) error {
	if len(rows) > 0 {
		if err := render.WriteTable(w, versionColumns, rows); err != nil {
			return err
		}
	}
	_, err := fmt.Fprintln(w, line)
	return err
}

// compactedLine is what the page says beneath the table: what was removed, the
// policy it was removed under, and how many series were left alone.
//
// The three are one sentence because they are the question an operator actually
// asks — *what did this take, by whose leave, and what is still there* — and the
// policy is named in the artefact's own spelling so that a reader can go and
// find the line it came from (§3, §8).
//
// A run that removed nothing says so in the same shape rather than in a
// sentence of its own, which is what tells it apart from the repository that
// declared no policy at all: the two empty states are reached by different
// roads and are read off different words, not off the exit code they share.
func compactedLine(done store.Compaction, policy store.Retention) string {
	removed := "nothing"
	if len(done.Removed) > 0 {
		removed = store.InteriorVersions(len(done.Removed))
	}
	// `series` is its own plural, so the count is written inline: there is
	// no `s` to hang off it and no singular form to choose between.
	return fmt.Sprintf("removed %s · retention %s · %d series untouched", removed, policy.Declared, done.Untouched)
}

// unboundedLine is what a repository that removes nothing by declaration says.
// It names the artefact and the key, which is the whole point of it: omitted,
// nothing is ever removed (§3), and that is the empty state most likely to be
// mistaken for a bug.
//
// A `retention:` that is not a duration is the same outcome by a different
// road, and it says so and sends the reader to `check` — the value is
// `schema-mismatch` and reporting it is that command's, not this one's
// (ADR-0064).
func unboundedLine(policy store.Retention) string {
	if policy.Declared != "" {
		return fmt.Sprintf("%s declares retention: %s, which is not a duration — run: hyper check", repository.DeclarationPath, policy.Declared)
	}
	return fmt.Sprintf("%s declares no retention: — nothing is ever removed", repository.DeclarationPath)
}
