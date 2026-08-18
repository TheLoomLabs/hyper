package cli

import (
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunProvider implements `hyper provider <name>` — §9's second discovery
// question, *which Operation*, and the first command in the tool that takes a
// name and resolves it. It writes the Manifest's own facts as a header row,
// emitted first, then one row per Operation the named Provider exposes, and
// exits 0.
//
// It is `providers` in every respect but its rows and its positional: the same
// globals, the same gate before the load, the same stream discipline, and the
// same two properties — it cannot exit 1, reporting facts rather than problems
// found (ADR-0064), and it reaches nothing: no network, no credential, no
// Store, no invocation. The Auth scheme it renders is the header the Manifest
// composes with the credential's position marked, which is a fact about the
// Manifest and needs no secret to state (§9, ADR-0007).
//
// What is new here is the positional, and the rule it establishes for every
// command after it: a name resolving to nothing is a usage error, exit 2,
// carrying no error_code. A Refusal is the artefacts declining an act and a
// usage error is there being no act to decline — nothing was reviewed, so
// nothing refused, and the remedy is a different name rather than an artefact
// edit (ADR-0036, ADR-0060).
//
// It takes no --limit: it names a Manifest rather than ranging over a
// namespace, so there is no result set for a cap to cut (§9).
func RunProvider(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, code := parseArgs("provider", args, takesNoLimit, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// Exactly one positional. Naming none and naming two are both faults in
	// the invocation as typed, decided from the argument list alone and
	// before any repository is resolved — the same reading `completions`
	// gives its own arity, and the same fault the shared spelling names
	// (ADR-0060).
	if len(parsed.positional) != 1 {
		fmt.Fprintf(stderr, "hyper provider: %s\n", arityFault(parsed.positional, "Provider"))
		return ExitUsage
	}
	name := parsed.positional[0]

	repoRoot, code := resolveRepoRoot("provider", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and before the positional is
	// resolved: a mismatched pin plus a name matching nothing is 77 and not
	// 2, because the gate fires first for all sixteen (§9, §11, ADR-0020,
	// ADR-0060).
	if code := gateOnVersionPin("provider", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper provider: %s\n", err)
		return ExitUsage
	}

	// The lookup is into the Provider namespace, whose keys are the
	// Manifests' own provider: values — so matching is byte-exact over UTF-8
	// and case-sensitive, and is never settled by whether a filesystem open
	// succeeded. A macOS filesystem is case-insensitive and a runner's is
	// not, so a fold that was the filesystem's would render on a laptop and
	// exit 2 in CI (§9, ADR-0060).
	manifest, resolved := loaded.Manifests[name]
	if !resolved {
		fmt.Fprint(stderr, unresolvedProviderName("provider", name))
		return ExitUsage
	}

	rows := manifestRows(manifest)

	// The terminal row is written with no marker: nothing here ranges over a
	// namespace and no --limit exists to cut a result short, so a stream
	// this command opens always carried everything it found (§9).
	if code := writeAnswer("provider", stdout, stderr, parsed.json, rows, render.NewResultRow(false), writeProviderPage); code != 0 {
		return code
	}

	return ExitClean
}

// unresolvedProviderName is what a Provider name matching nothing writes, and
// it goes to stderr with stdout left completely silent in both modes: no row
// stream opens, so there is no terminal row to be missing (§9, ADR-0060).
//
// It states three things and no fourth: the name that was typed, the namespace
// it was resolved against, and the command that enumerates that namespace. It
// lists no candidate and suggests no near miss — a suggestion is a partial name
// resolved on the caller's behalf, and a human who accepts one has run
// something they did not type (ADR-0047). Enumerating the namespace is
// `providers`'s job, which is why the remedy names that command rather than
// doing its work here.
//
// command is the command the caller typed, which is `provider` and is also
// `operation`: both resolve their first positional against the one Provider
// namespace, so both write this message and neither may write it in the other's
// name. One spelling rather than two is what keeps them from drifting into two
// accounts of one namespace.
func unresolvedProviderName(command, name string) string {
	return fmt.Sprintf("hyper %s: no Provider named %q in this repository's Provider namespace\n"+
		"  hyper providers lists every Provider in it\n", command, name)
}

// manifestRow is the header row, and its members are §9's own, in §9's order:
// {"type":"manifest","auth_scheme":…,"capabilities_required":[…],"digest":…,
// "schema_version":…,"origin_ref":…,"origin_digest":…}. §9 writes that shape
// out once and milestone 11's MCP tool reuses this contract rather than minting
// a second one, so the declaration order here is the wire's and not a
// preference.
//
// The row exists to state what a Manifest declares, and the origin: block is
// the one part of it no other surface renders — the fact check enforces as
// origin-digest-mismatch and Provenance carries as origin_digest. Its two
// members follow the ordinary absence rule: both written where the block is
// there, both absent where it is not, which is a built-in Provider and a
// locally authored Extension alike. Absent together they say the Manifest makes
// no digest claim, and that is the whole of what distinguishes an installed
// Extension from one an author wrote (§7, ADR-0073).
//
// The other three optional members are the same rule applied to a Manifest
// hyper could not read: an auth: block that names no scheme, a capabilities:
// that is not a list, a schema-version: that is not an integer. Each is a
// schema fault check reports, and a row that wrote "" or 0 or [] in its place
// would state a declaration that is not there (§7, ADR-0064) — an empty
// capabilities_required is the claim that this Provider requires no Capability,
// which is a different thing from a Manifest that failed to say.
//
// auth_scheme is the one to read carefully, because §12 closes what a rendering
// of a Provider's auth may say at two things: the header the scheme composes,
// or none. Both are claims about the Provider — none says no credential is
// sent — and neither can be made about a Manifest that declared a scheme hyper
// could not parse. So the third answer is not a third word but no member at
// all, which is the same rule `targets` renders its four lists under: the row
// states what the artefact stated, and what is wrong with the rest is check's
// to name.
//
// The digest is rendered whole, on providerRow's own reasoning: a digest is
// verified with sha256sum rather than recognised by eye (§8, ADR-0047).
type manifestRow struct {
	Type                 string   `json:"type"`
	AuthScheme           string   `json:"auth_scheme,omitempty"`
	CapabilitiesRequired []string `json:"capabilities_required,omitempty"`
	Digest               string   `json:"digest"`
	SchemaVersion        *int     `json:"schema_version,omitempty"`
	OriginRef            string   `json:"origin_ref,omitempty"`
	OriginDigest         string   `json:"origin_digest,omitempty"`
}

// Cells is empty: the header row states a Manifest's own facts, which are one
// value each rather than a line in a table of like rows, and the page renders
// them as the block writeManifestBlock writes. A row contributing no line is
// the shape the terminal row already has (ADR-0026).
func (r manifestRow) Cells() []string { return nil }

// operationRow is one Operation, and its members are §9's own, in §9's order:
// {"type":"operation","name":…,"kind":…,"opaque":…,"summary":…}.
//
// Kind is on every row at this level because it is what answers the two-key
// question before a single input schema has been read (§5), and it is read from
// the Manifest's kind: and never inferred from the Operation's name — an
// Operation called delete_thing that declares mutate is a mutate. Opaque is
// read from the request block rather than declared: opacity is a property of
// the Capability the Operation's request uses, so shell is opaque and http is
// not, no artefact declares it and every surface renders it (§12).
//
// Every member is always written: an Operation has a name, and opacity is a
// boolean whose false is an answer. Kind is written even where the Manifest
// declared none, an empty cell there being the fault check names rather than a
// fact this row may substitute for.
type operationRow struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Opaque  bool   `json:"opaque"`
	Summary string `json:"summary"`
}

// Cells is the row's line on the page, in operationColumns' order. Every member
// the wire carries has a column: this row has no fact a consumer filters on
// that a reader would not also want.
func (r operationRow) Cells() []string {
	return []string{r.Name, r.Kind, strconv.FormatBool(r.Opaque), r.Summary}
}

// operationColumns is the page's header for the Operation rows, and the columns
// are the row's own members in the row's own order (§8, ADR-0026).
var operationColumns = []string{"NAME", "KIND", "OPAQUE", "SUMMARY"}

// manifestRows is the whole answer: the header row, then one Operation row per
// Operation the Manifest declares.
//
// Operation rows are ordered by name, ascending, byte-exact over UTF-8 — the
// same comparison §9 fixes for matching a name. The Manifest's authored order
// is preserved exactly where it is the answer, which is `hyper operation`'s
// verbatim source; a listing is ranged over, and a normative order is what
// makes two renderings diffable (§9). The sort is here rather than in the
// reader for that reason: the reader states what the artefact states, and the
// order a listing is ranged over in is this surface's rule. It sorts a copy,
// the order ReadManifestFacts answers in being that reader's own contract and
// not this command's to reorder underneath it.
func manifestRows(manifest repository.LoadedManifest) []render.Row {
	facts := artefact.ReadManifestFacts(manifest.Root)

	rows := []render.Row{manifestRow{
		Type:                 "manifest",
		AuthScheme:           facts.AuthScheme,
		CapabilitiesRequired: facts.Capabilities,
		Digest:               artefact.ManifestDigest(manifest.Bytes),
		SchemaVersion:        facts.SchemaVersion,
		OriginRef:            facts.OriginRef,
		OriginDigest:         facts.OriginDigest,
	}}

	operations := slices.SortedFunc(slices.Values(facts.Operations),
		func(a, b artefact.OperationFacts) int { return strings.Compare(a.Name, b.Name) })
	for _, operation := range operations {
		rows = append(rows, operationRow{
			Type:    "operation",
			Name:    operation.Name,
			Kind:    operation.Kind,
			Opaque:  operation.Opaque,
			Summary: operation.Summary,
		})
	}
	return rows
}

// writeProviderPage is `provider`'s page: the Manifest's own facts as a block
// of labelled values, then the table of Operations beneath it. Two shapes
// rather than one because the two row types are two shapes — a Manifest's facts
// are one value each and the Operations are a list of like rows — and both are
// written from the one list of rows the --json stream is written from
// (ADR-0026).
//
// A Manifest declaring no Operation says so in words. A header over no rows
// would state less than a sentence does, which is the rule `check`'s clean run
// and `targets`'s empty repository already follow (issue #99).
func writeProviderPage(w io.Writer, rows []render.Row) error {
	// The header row is the first row and every row after it is an
	// Operation, which is what §9 fixes — the Manifest's own facts, emitted
	// first — and what manifestRows builds. Splitting the list on that fact
	// rather than testing each row's type is what keeps the page and the
	// stream one answer: the same ordering statement shapes both.
	operations := rows
	if len(operations) > 0 {
		if header, first := operations[0].(manifestRow); first {
			if err := writeManifestBlock(w, header); err != nil {
				return err
			}
			operations = operations[1:]
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if len(operations) == 0 {
		_, err := fmt.Fprintln(w, "no Operations in this Manifest")
		return err
	}
	return render.WriteTable(w, operationColumns, operations)
}

// writeManifestBlock writes the header row as labelled values, in the row's own
// member order — the wire's order, so a reader moving between the two surfaces
// reads the same facts in the same sequence.
//
// A member the row does not carry writes no line at all, which is the ordinary
// absence rule the wire already applies to it: the two origin members stand or
// fall together, and a page carrying "ORIGIN REF" against nothing would state a
// claim the Manifest never made (§7, ADR-0073).
func writeManifestBlock(w io.Writer, row manifestRow) error {
	values := []labelledValue{
		{"AUTH SCHEME", row.AuthScheme},
		{"CAPABILITIES", strings.Join(row.CapabilitiesRequired, ", ")},
		{"DIGEST", row.Digest},
	}
	if row.SchemaVersion != nil {
		values = append(values, labelledValue{"SCHEMA VERSION", strconv.Itoa(*row.SchemaVersion)})
	}
	values = append(values,
		labelledValue{"ORIGIN REF", row.OriginRef},
		labelledValue{"ORIGIN DIGEST", row.OriginDigest})

	return writeLabelledValues(w, values)
}
