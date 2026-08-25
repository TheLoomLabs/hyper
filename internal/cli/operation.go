package cli

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunOperation implements `hyper operation <provider> <operation>` — §9's third
// discovery question, *how do I call it*, and the one place in this milestone
// where bytes matter rather than facts. It writes the Manifest lines declaring
// that Operation, verbatim, then the terminal row, and exits 0.
//
// Verbatim is the whole of it, and why is artefact.OperationSource's to state:
// what the range is, and what a re-encoding of it would silently break. What is
// this command's is that everything else in the milestone reads a parsed node
// and renders a value, and this finds a range in a file and copies it — so the
// range is that reader's, and the whole of what this adds to it is the two
// lookups and the two messages they write.
//
// It is `provider` in every other respect: the same globals, the same gate
// before the load, the same stream discipline, and the same two properties —
// it cannot exit 1, reporting facts rather than problems found (ADR-0064), and
// it reaches nothing: no network, no credential, no Store, no invocation. It
// takes no --limit either, for a reason of its own: it names one Operation, so
// there is no result set at all for a cap to cut (§9).
//
// The two positionals resolve in order and against two different namespaces,
// which is why a bad name has two messages rather than one. The Provider name
// resolves against the repository's Provider namespace; the Operation name
// against that Manifest's own Operation namespace, which does not exist until
// the first has resolved — so a bad Provider is reported and the Operation
// lookup is never attempted.
func RunOperation(args []string, stdout, stderr io.Writer, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, code := parseArgs("operation", args, parameters{limit: takesNoLimit}, lookupenv, stderr)
	if code != 0 {
		return code
	}
	// Exactly two positionals, decided from the argument list alone and
	// before any repository is resolved — the reading `provider` and
	// `completions` give their own arity, in the spelling all three share
	// (ADR-0060).
	if len(parsed.positional) != 2 {
		fmt.Fprintf(stderr, "hyper operation: %s\n", arityFault(parsed.positional, "Provider", "Operation"))
		return ExitUsage
	}
	providerName, operationName := parsed.positional[0], parsed.positional[1]

	repoRoot, code := resolveRepoRoot("operation", parsed.repoDir, lookupenv, wd, stderr)
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and before either positional
	// is resolved: a mismatched pin plus a name matching nothing is 77 and
	// not 2, because the gate fires first for fifteen of the sixteen (§9,
	// §11, ADR-0020, ADR-0060).
	if code, _ := gateOnVersionPin("operation", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper operation: %s\n", err)
		return ExitUsage
	}

	// The first lookup is `provider`'s own, and it is the same one: byte-exact
	// over UTF-8 and case-sensitive against the Manifests' own provider:
	// values, never settled by whether a filesystem open succeeded (§9,
	// ADR-0060). Its message is the same message, differing only in the
	// command it opens with.
	manifest, resolved := loaded.Manifests[providerName]
	if !resolved {
		fmt.Fprint(stderr, unresolvedProviderName("operation", providerName))
		return ExitUsage
	}

	// The second is into that Manifest's own Operation namespace, whose keys
	// are the operations: mapping's, and matching is the same rule again. The
	// built-in shell Provider needs no arm of its own here: the load carries
	// its compiled-in bytes on the same footing as a file's, so the one
	// Manifest a repository author cannot read any other way is read through
	// exactly this path (§12, ADR-0039).
	source, declared := artefact.OperationSource(manifest.Bytes, manifest.Root, operationName)
	if !declared {
		fmt.Fprint(stderr, unresolvedOperationName("operation", providerName, operationName))
		return ExitUsage
	}

	rows := []render.Row{newOperationDetailRow(source, artefact.ReadOperationDetail(manifest.Root, operationName))}

	// The terminal row is written with no marker: this command names one
	// Operation, so a stream it opened always carried everything there was
	// (§9).
	if code := writeAnswer("operation", stdout, stderr, parsed.json, rows, render.NewResultRow(false), writeOperationPage); code != 0 {
		return code
	}

	return ExitClean
}

// unresolvedOperationName is what an Operation name matching nothing writes,
// and it is unresolvedProviderName's counterpart in every respect but the
// namespace it names: stderr, stdout completely silent in both modes, no row
// stream and so no terminal row to be missing, no error_code — nothing was
// reviewed, so nothing was refused (§9, ADR-0060).
//
// The namespace it names is the Manifest's own rather than the repository's,
// which is why this is a second message and not the first one with a word
// changed: the two names were resolved against two different things, and the
// command that enumerates this one takes an argument. It lists no candidate and
// suggests no near miss, on ADR-0047's rule — enumerating the namespace is
// `hyper provider <name>`'s job, which is why the remedy names that command
// rather than doing its work here.
//
// command is a parameter for the reason unresolvedProviderName's already is:
// `probe` resolves its two positionals against the Provider set exactly as this
// command does (§9), so the two share the lookup and the message and differ
// only in the word a caller typed.
func unresolvedOperationName(command, provider, operation string) string {
	return fmt.Sprintf("hyper %s: no Operation named %q in Provider %q's own Operation namespace\n"+
		"  hyper provider %s lists every Operation in it\n", command, operation, provider, provider)
}

// operationDetailRow is the Operation's declaring lines and the facts derived
// beside them, and its members are §9's own:
// {"type":"operation_detail","source":…,"derived":{…}}. The two halves are one
// row rather than two, because §9 writes the shape out once and milestone 11's
// MCP tool reuses this contract rather than minting a second one.
//
// source is always written. An Operation this command resolved has declaring
// lines by construction — the lookup that found it read them off the same
// bytes — so there is no absence here for the ordinary rule to cover. derived
// is always written too, and for the same reason read the other way: the block
// is what `hyper` computed rather than what the Manifest stated, so there is
// always something it computed, down to the concurrency limit that is 1 for
// every Operation whatever the Manifest left out.
type operationDetailRow struct {
	Type    string           `json:"type"`
	Source  string           `json:"source"`
	Derived operationDerived `json:"derived"`
}

// operationDerived is the derived block, and its members are §9's own in §9's
// own order: the Capability the request is written under, the Bound, the
// Patterns resolved, the Record's cardinality and declared identity, the
// Repeatability in force, the deadline in seconds, and the effective
// concurrency limit.
//
// Three of the eight follow the ordinary absence rule and three do not, and
// which is which is the answer rather than the encoding. patterns_resolved is
// empty rather than absent: a caller asking which Patterns run around this call
// is answered *none of them*, which is a fact. concurrency_limit is always
// present, an effective limit existing for every Operation (ADR-0045). The
// Record pair is absent together on a destroy, which declares no record: at all
// (§3, ADR-0037), and capabilities, bound and repeatability are absent only
// where a Manifest hyper could not read left nothing to derive them from —
// which is check's to report and never this row's to substitute for (ADR-0064).
//
// deadline is the page's and never the wire's: §9 fixed the wire name and its
// unit with it, so what goes out is deadline_seconds, and what the table
// renders is the authored spelling, because that is what the source it stands
// beside says. It is unexported for exactly that reason — the two renderings
// are one row, and this member is the half of it only one of them has a line
// for.
type operationDerived struct {
	Capabilities      []string `json:"capabilities,omitempty"`
	Bound             string   `json:"bound,omitempty"`
	PatternsResolved  []string `json:"patterns_resolved"`
	RecordCardinality string   `json:"record_cardinality,omitempty"`
	RecordIdentity    string   `json:"record_identity,omitempty"`
	Repeatability     string   `json:"repeatability,omitempty"`
	DeadlineSeconds   *int     `json:"deadline_seconds,omitempty"`
	ConcurrencyLimit  int      `json:"concurrency_limit"`
	deadline          string
}

// newOperationDetailRow is the whole answer as one row: the Manifest's own
// bytes, and the reader's derived facts under the wire's own member names. The
// mapping is written out here rather than by hanging json tags off the reader's
// own type, on manifestRow's rule — the wire shape is this surface's and a
// domain value carries no serialisation it has no other use for.
func newOperationDetailRow(source string, detail artefact.OperationDetail) operationDetailRow {
	return operationDetailRow{
		Type:   "operation_detail",
		Source: source,
		Derived: operationDerived{
			Capabilities:      detail.Capabilities,
			Bound:             detail.Bound,
			PatternsResolved:  detail.PatternsResolved,
			RecordCardinality: detail.RecordCardinality,
			RecordIdentity:    detail.RecordIdentity,
			Repeatability:     detail.Repeatability,
			DeadlineSeconds:   detail.DeadlineSeconds,
			ConcurrencyLimit:  detail.ConcurrencyLimit,
			deadline:          detail.Deadline,
		},
	}
}

// Cells is empty: this row is a block of an artefact's own lines rather than a
// line in a table of like rows, and the page renders it as writeOperationPage
// writes it. A row contributing no line is the shape the terminal row already
// has (ADR-0026).
func (r operationDetailRow) Cells() []string { return nil }

// writeOperationPage is `operation`'s page: the source, then the derived facts
// beneath it as labelled values. The source carries no header, no frame and no
// label — a frame around bytes a caller is meant to copy into a Definition is a
// frame they would have to strip — and the page and the --json stream state the
// same facts because they are written from the one row (§9, ADR-0026).
//
// A source whose last line carries no newline — an author whose editor wrote
// none, the range running to the end of the file — is ended with one here. That
// is the page's own business rather than the range's: a terminal writes lines,
// and the bytes the wire carries stay exactly what the file held.
//
// The row it writes is the first one, because the command builds exactly one:
// reading the list's shape off the statement that built it is what keeps the
// page and the stream one answer, which is the reading writeProviderPage gives
// its own header row rather than testing every row's type (ADR-0026).
func writeOperationPage(w io.Writer, rows []render.Row) error {
	if len(rows) == 0 {
		return nil
	}
	detail, written := rows[0].(operationDetailRow)
	if !written {
		return nil
	}

	source := detail.Source
	if !strings.HasSuffix(source, "\n") {
		source += "\n"
	}
	if _, err := io.WriteString(w, source); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	return writeDerivedBlock(w, detail.Derived)
}

// writeDerivedBlock writes the derived facts as labelled values, in the row's
// own member order — the wire's order, so a reader moving between the two
// surfaces reads the same facts in the same sequence.
//
// A member the row does not carry writes no line at all, which is the rule
// writeLabelledValues holds for every block of them: a destroy's Record pair is
// absent together, and a line reading "RECORD CARDINALITY" against nothing
// would state a projection the Operation does not have. An Operation declaring
// no Pattern is that rule read from its other end — the wire says *none of
// them* with an empty list and the page says it by having no line — and the
// concurrency limit always has one, there being an effective limit for every
// Operation (ADR-0045).
//
// DEADLINE is the authored spelling and not the wire's seconds, because the
// source it stands beneath says 30s and a page restating it as 30 would be a
// second spelling of one fact on one screen.
func writeDerivedBlock(w io.Writer, derived operationDerived) error {
	return writeLabelledValues(w, []labelledValue{
		{"CAPABILITIES", strings.Join(derived.Capabilities, ", ")},
		{"BOUND", derived.Bound},
		{"PATTERNS RESOLVED", strings.Join(derived.PatternsResolved, ", ")},
		{"RECORD CARDINALITY", derived.RecordCardinality},
		{"RECORD IDENTITY", derived.RecordIdentity},
		{"REPEATABILITY", derived.Repeatability},
		{"DEADLINE", derived.deadline},
		{"CONCURRENCY LIMIT", strconv.Itoa(derived.ConcurrencyLimit)},
	})
}
