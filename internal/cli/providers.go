package cli

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunProviders implements `hyper providers` — the first of §9's three
// discovery questions, *which Provider*, and the one an agent asks before it
// can write a `provider:` at all. It writes one row per Provider hyper can
// load, built-in and Extension alike, and exits 0.
//
// It reports facts rather than problems, which fixes two things about it. It
// cannot exit 1: a Manifest that will not parse contributes no row and its
// faults are `check`'s to report (ADR-0064), so there is nothing here for a
// problem count to be. And it reaches nothing: no network, no credential, no
// Store, no invocation — the whole answer is the repository load, and the load
// reads the five artefact locations and nothing else.
//
// Its parameters are RunCheck's, for the reason RunCheck's are what they are:
// everything the command reads from the process arrives as an argument, so the
// whole of it is exercisable without a subprocess.
func RunProviders(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, code := parseArgs("providers", args, parameters{limit: defaultListLimit}, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// `providers` enumerates a namespace and resolves no name in one, so it
	// takes no positional at all: §9 gives a positional to nine of the
	// sixteen and this is not one of them, which makes `hyper providers
	// shell` the neighbouring command mistyped rather than a filter.
	if len(parsed.positional) > 0 {
		fmt.Fprintf(stderr, "hyper providers: takes no positional argument, got %s\n", parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("providers", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and before any row exists:
	// every command compares itself against hyper.yaml's version: pin and
	// Refuses on mismatch in either direction, with stdout left silent in
	// both modes (§9, §11, ADR-0020).
	if code := gateOnVersionPin("providers", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper providers: %s\n", err)
		return ExitUsage
	}

	rows := providerRows(loaded)
	kept, dropped := truncate(rows, parsed.limit)

	// The two renderings are one list of rows written twice (ADR-0026), and
	// the truncation is applied before either of them: the table and the
	// --json stream state the same facts because they are built from one row
	// set, cut in one place.
	page := func(w io.Writer, rows []render.Row) error { return render.WriteTable(w, providerColumns, rows) }
	if code := writeAnswer("providers", stdout, stderr, parsed.json, kept, render.NewResultRow(dropped > 0), page); code != 0 {
		return code
	}

	// The human rendering of a truncated result is a line on stderr, in both
	// modes, because it is narration rather than an answer (§9). A truncated
	// result must never look complete, and a table that simply stopped after
	// the last row it was allowed would.
	if dropped > 0 {
		fmt.Fprintf(stderr, "hyper providers: %s\n", truncationLine("Providers", len(kept), len(rows), parsed, ""))
	}

	return ExitClean
}

// providerRow is `providers`'s row, and its members are §9's own, in §9's
// order: {"type":"provider","name":…,"origin":…,"summary":…,
// "operation_count":…,"digest":…}. §9 writes that shape out once and milestone
// 11's MCP tool reuses this contract rather than minting a second one, so the
// declaration order here is the wire's and not a preference.
//
// Every member is always written: a Provider has a name, an origin, a derived
// summary, a count that is 0 where it declares no Operation, and a digest over
// whatever bytes it loaded from. There is nothing here the ordinary absence
// rule reaches.
//
// The digest is rendered whole in both forms. ADR-0047's abbreviation is for a
// fact to be *recognised* — a revision the eye matches against another — and a
// digest here is verified with sha256sum instead. The table is wide as a
// result, and that is accepted (§8).
type providerRow struct {
	Type           string `json:"type"`
	Name           string `json:"name"`
	Origin         string `json:"origin"`
	Summary        string `json:"summary"`
	OperationCount int    `json:"operation_count"`
	Digest         string `json:"digest"`
}

// Cells is the row's line on the page, in providerColumns' order. Every member
// the wire carries has a column: this row has no fact a consumer filters on
// that a reader would not also want.
func (r providerRow) Cells() []string {
	return []string{r.Name, r.Origin, r.Summary, strconv.Itoa(r.OperationCount), r.Digest}
}

// providerColumns is the page's header, and the columns are the row's own
// members in the row's own order: what a consumer filters on and what a reader
// reads down are one list here, this row carrying no fact that belongs to only
// one of them (§8, ADR-0026).
var providerColumns = []string{"NAME", "ORIGIN", "SUMMARY", "OPERATIONS", "DIGEST"}

// providerRows is the whole answer, ordered: one row per Provider in the
// repository's Provider namespace, by name ascending, byte-exact over UTF-8 —
// the same comparison §9 fixes for matching a name, and what makes two
// invocations against one repository produce identical bytes.
//
// It folds nothing itself. Which Manifests are in the namespace, and which file
// a name means where two declare one, is the load's single decision (§9,
// ADR-0064, issue #109) — so the Provider this command lists for a name and the
// Provider a Definition's provider: resolves to are the same file by
// construction rather than by two walks agreeing.
func providerRows(loaded repository.Loaded) []render.Row {
	rows := make([]render.Row, 0, len(loaded.Manifests))
	for _, name := range slices.Sorted(maps.Keys(loaded.Manifests)) {
		manifest := loaded.Manifests[name]
		info := loaded.Providers[name]
		rows = append(rows, providerRow{
			Type:           "provider",
			Name:           name,
			Origin:         manifest.Origin,
			Summary:        info.Summary(),
			OperationCount: len(info.Operations),
			Digest:         artefact.ManifestDigest(manifest.Bytes),
		})
	}
	return rows
}
