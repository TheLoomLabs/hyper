package cli

import (
	"fmt"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/presence"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
)

// RunTargets implements `hyper targets` — the one command in this milestone
// that asks the repository rather than a Provider, and the only surface
// anywhere that answers *is the credential in place*. It writes one row per
// Target declaration and exits 0.
//
// It is `providers` in every respect but its row: the same globals, the same
// --limit and truncation marker, the same gate before the load, the same
// stream discipline, and the same two properties — it cannot exit 1, a
// declaration that will not parse contributing no row and its faults being
// check's to report (ADR-0064), and it reaches no network, no Store and no
// invocation.
//
// What it does reach is the environment, which nothing else in this milestone
// does: presence is computed when the command runs, by asking what the
// environment did with the variable a credential slot names — it does not hold
// it, it holds it empty, or it fills it. The value behind a filled name is never
// read, and nothing on this surface has ever held a secret (§3, §9, ADR-0007,
// ADR-0145).
func RunTargets(args []string, to destination, lookupenv func(string) (string, bool), wd, binaryVersion string) int {
	parsed, to, code := parseArgs("targets", args, parameters{limit: defaultListLimit}, lookupenv, to)
	if code != 0 {
		return code
	}
	// `targets` enumerates a namespace and resolves no name in one, so it
	// takes no positional at all: §9 gives a positional to nine of the
	// sixteen and this is not one of them.
	if len(parsed.positional) > 0 {
		fmt.Fprintf(to.narrate(), "hyper targets: takes no positional argument, got %s\n", parsed.positional[0])
		return ExitUsage
	}

	repoRoot, code := resolveRepoRoot("targets", parsed.repoDir, lookupenv, wd, to.narrate())
	if code != 0 {
		return code
	}

	// The gate, before the repository is loaded and before any row exists
	// (§9, §11, ADR-0020).
	if code, _ := gateOnVersionPin("targets", repoRoot, binaryVersion, to); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(to.narrate(), "hyper targets: %s\n", err)
		return ExitUsage
	}

	rows := targetRows(loaded, lookupenv)
	kept, dropped := truncate(rows, parsed.limit)

	if code := writeAnswer("targets", to, kept, render.NewResultRow(dropped > 0), writeTargetTable); code != 0 {
		return code
	}

	if dropped > 0 {
		fmt.Fprintf(to.narrate(), "hyper targets: %s\n", truncationLine("Targets", len(kept), len(rows), parsed, nil))
	}

	return ExitClean
}

// targetRow is `targets`'s row, and its members are the issue's own, in its
// order: {"type":"target","name":…,"hosts":[…],"accepts_kinds":[…],
// "grants_capabilities":[…],"credentials":[…]}. Milestone 11's MCP tool reuses
// this contract rather than minting a second one, so the declaration order here
// is the wire's and not a preference.
//
// The host grant is hosts, an array, in the declaration's own order. §9 said
// *its endpoint* and its MCP sketch named the field endpoint, and §3's Target
// declaration has no such key: it has hosts:, and it never has a grant without
// an enumeration (ADR-0024). §12's opening rule — one fact reaching two wires
// reaches them under one name — decided it in favour of the artefact's own key,
// and a grant silently reduced to its first member is not a grant (ADR-0029).
// §9 now names the field hosts on both its surfaces, so the two agree; this
// comment keeps the disagreement it was resolved from.
//
// The four lists are omitempty, on the ordinary absence rule (§7): a
// declaration granting no http carries no hosts: at all, and a declaration
// named local carries no auth: block, so what stands there is nothing rather
// than an empty enumeration a reader would have to interpret.
type targetRow struct {
	Type               string           `json:"type"`
	Name               string           `json:"name"`
	Hosts              []string         `json:"hosts,omitempty"`
	AcceptsKinds       []string         `json:"accepts_kinds,omitempty"`
	GrantsCapabilities []string         `json:"grants_capabilities,omitempty"`
	Credentials        []credentialCell `json:"credentials,omitempty"`
}

// credentialCell is one credential slot on the wire: the slot, the environment
// variable it resolves from — the name, never the value — and what the
// environment did with that name.
//
// Presence is a pointer because a slot naming no variable has no presence to
// state, and a word here would be an answer to a question nothing asked: env:
// absent or malformed is credential-slot-malformed, which check reports, and no
// zero value here must stand in for it (§7, §8, ADR-0064). Where a variable is
// named, presence is always written — whichever of the three it is.
//
// It is a word out of §12's closed three rather than a boolean, and it replaced
// a boolean rather than growing a flag beside one: absent, empty and set are one
// question's three answers, and two booleans would have spelled them as four
// states of which one — present and not present but empty — is unreachable
// (§9, §12, ADR-0145). The rename is what makes the change loud: a consumer
// reading the old present finds no member rather than reading the string
// "absent" as true.
//
// The set is internal/presence's rather than a second spelling here, which is
// the whole point of it: the credential gate decides a Run under those three
// (§6), and a column that agreed with the gate by coincidence is a column a
// reader cannot use to predict a Run.
type credentialCell struct {
	Slot     string             `json:"slot"`
	Env      string             `json:"env,omitempty"`
	Presence *presence.Presence `json:"presence,omitempty"`
}

// Cells is the row's line on the page, in targetColumns' order. Every member
// the wire carries has a column: this row has no fact a consumer filters on
// that a reader would not also want.
func (r targetRow) Cells() []string {
	return []string{
		r.Name,
		strings.Join(r.Hosts, ", "),
		strings.Join(r.AcceptsKinds, ", "),
		strings.Join(r.GrantsCapabilities, ", "),
		r.credentialsCell(),
	}
}

// credentialsCell is the human rendering of the credential column: each slot
// paired with its variable, and beside the pair whether that variable is
// present. The pair is written rather than a bare list of variable names
// because a declaration may carry slots for more than one scheme, and a list of
// names alone does not say which fills what (§9). A slot naming no variable
// renders as its name alone: there is nothing to pair it with, and nothing to
// ask the environment about — which is the same thing as having no presence to
// state, the two being written together or not at all.
func (r targetRow) credentialsCell() string {
	rendered := make([]string, 0, len(r.Credentials))
	for _, credential := range r.Credentials {
		if credential.Presence == nil {
			rendered = append(rendered, credential.Slot)
			continue
		}
		rendered = append(rendered, fmt.Sprintf("%s=%s (%s)", credential.Slot, credential.Env, string(*credential.Presence)))
	}
	return strings.Join(rendered, ", ")
}

// targetColumns is the page's header, and the columns are the row's own members
// in the row's own order. They are named for the declaration's own keys — a
// reader has the artefact open beside the table (§8, ADR-0026).
var targetColumns = []string{"NAME", "HOSTS", "KINDS", "CAPABILITIES", "CREDENTIALS"}

// writeTargetTable is targets's page: its five columns, and the line that
// stands where there are no rows. A repository that has declared no Target is
// not an error and not silence either — check's clean run states what it
// checked rather than printing nothing, and a header over no rows would state
// less than that (issue #99).
func writeTargetTable(w io.Writer, rows []render.Row) error {
	if len(rows) == 0 {
		_, err := fmt.Fprintln(w, "no Target declarations in targets/")
		return err
	}
	return render.WriteTable(w, targetColumns, rows)
}

// targetRows is the whole answer, ordered: one row per member of the Target
// namespace, by name ascending, byte-exact over UTF-8 — the same comparison §9
// fixes for matching a name, and what makes two invocations against one
// repository produce identical bytes.
//
// It folds nothing itself. Which declarations are in the namespace, and which
// file a name means where two declare one, is the load's single decision (§9,
// ADR-0064, issue #109) — so the Target this command lists for a name and the
// Target a Definition's targets: resolves to are the same declaration by
// construction rather than by two walks agreeing.
func targetRows(loaded repository.Loaded, lookupenv func(string) (string, bool)) []render.Row {
	rows := make([]render.Row, 0, len(loaded.TargetDeclarations))
	for _, name := range slices.Sorted(maps.Keys(loaded.TargetDeclarations)) {
		facts := artefact.ReadTargetFacts(loaded.TargetDeclarations[name])
		rows = append(rows, targetRow{
			Type:               "target",
			Name:               name,
			Hosts:              facts.Hosts,
			AcceptsKinds:       facts.Kinds,
			GrantsCapabilities: facts.Capabilities,
			Credentials:        credentialCells(facts.Credentials, lookupenv),
		})
	}
	return rows
}

// credentialCells pairs every slot the declaration carries with its variable
// and asks the environment what it did with that variable — which is the whole
// question, asked at the moment of the call. The value itself is never read: the
// one thing taken off it is whether it has any characters at all, and no
// credential's contents reach this row, this page or this stream (§9, ADR-0007,
// ADR-0145).
//
// Every slot is reported, which is deliberately wider than what a Run checks: a
// Run resolves the slots its bindings require, and this command has no
// Procedure in hand to narrow by — so an absent or empty slot here is not by
// itself a Run that will Refuse.
func credentialCells(slots []artefact.CredentialSlot, lookupenv func(string) (string, bool)) []credentialCell {
	cells := make([]credentialCell, 0, len(slots))
	for _, slot := range slots {
		cell := credentialCell{Slot: slot.Slot, Env: slot.Env}
		if slot.Env != "" {
			stated := presence.Of(lookupenv(slot.Env))
			cell.Presence = &stated
		}
		cells = append(cells, cell)
	}
	return cells
}
