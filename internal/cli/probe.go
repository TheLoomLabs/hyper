package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/render"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// localTarget is the one Target name the format reserves, and the one a Probe
// binds (§3, §9, ADR-0041). What the name reserves is that Target and nothing
// else about the file: a repository authors it in targets/ like any other and
// hyper ships none.
const localTarget = "local"

// RunProbe implements `hyper probe <provider> <operation>` — the smallest
// complete path through the tool that touches the world, and the first byte
// hyper ever sends anywhere: an artefact, a call, a response object, a
// projection, a page.
//
// It invokes a `read` Operation against `local` without a Definition, which is
// the whole of why it exists: §9's argument for a Probe is that the review
// model dies by volume, an agent authoring a Manifest against an unfamiliar API
// asking *what does this endpoint actually answer* twenty times. Every one of
// those questions costing a reviewed Definition is what ends with a repository
// full of Definitions nobody read.
//
// So it writes nothing. No Record, no Journal entry, no Trigger, no Provenance,
// no Disposition, no lock, no Store, no credential (ADR-0009). Its instant is
// its own start, recorded nowhere. It **exits 0 whatever came back** — a 503 as
// readily as a 200, and a host that answered nothing as readily as either —
// because a read never halts on what came back (ADR-0050) and a nonzero exit
// would be hyper deciding that a 503 is bad news. The exit code says whether
// the command did what it was asked; the rendering says what came back.
//
// Its reach is the one thing it does not escape, and there are two of those.
// The host it asks for is `local`'s to grant, and one outside that grant is
// `host-not-granted`, a Refusal at 77 (ADR-0042) — the reach comes from an
// artefact even where no artefact named the Operation. And a Probe may never
// invoke an `opaque` Operation, whatever any Target grants, which is a usage
// error rather than a Refusal because there is no edit that would make it work.
//
// It surfaces the raw response beside the projection, which no credentialled
// surface does (ADR-0017): a Probe binds `local`, which carries no credential
// slot, so the wire is visible by construction rather than by a flag.
func RunProbe(args []string, stdout, stderr io.Writer, process Process, wd, binaryVersion string) int {
	// --input is read off the argument list before the globals, because
	// parseArgs refuses a flag it does not know and this is the one command
	// that carries one. Everything past a `--` is left untouched, so a
	// positional spelled like a flag is still reachable.
	supplied, rest, fault := splitInputs(args)
	if fault != "" {
		fmt.Fprintf(stderr, "hyper probe: %s\n", fault)
		return ExitUsage
	}

	parsed, code := parseArgs("probe", rest, parameters{limit: takesNoLimit}, process.LookupEnv, stderr)
	if code != 0 {
		return code
	}
	if len(parsed.positional) != 2 {
		fmt.Fprintf(stderr, "hyper probe: %s\n", arityFault(parsed.positional, "Provider", "Operation"))
		return ExitUsage
	}
	providerName, operationName := parsed.positional[0], parsed.positional[1]

	repoRoot, code := resolveRepoRoot("probe", parsed.repoDir, process.LookupEnv, wd, stderr)
	if code != 0 {
		return code
	}
	if code := gateOnVersionPin("probe", repoRoot, binaryVersion, stderr); code != 0 {
		return code
	}

	loaded, err := repository.Load(repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "hyper probe: %s\n", err)
		return ExitUsage
	}

	// The two positionals resolve against the Provider set exactly as
	// `operation`'s do, and either matching nothing is the same usage error
	// in the same words (§9, ADR-0060).
	manifest, resolved := loaded.Manifests[providerName]
	if !resolved {
		fmt.Fprint(stderr, unresolvedProviderName("probe", providerName))
		return ExitUsage
	}
	operation, declared := loaded.Providers[providerName].Operations[operationName]
	if !declared {
		fmt.Fprint(stderr, unresolvedOperationName("probe", providerName, operationName))
		return ExitUsage
	}

	if fault := probeInvokes(providerName, operationName, operation); fault != "" {
		fmt.Fprintf(stderr, "hyper probe: %s\n", fault)
		return ExitUsage
	}

	inputs, faults := readInputs(operation, supplied)
	if len(faults) > 0 {
		for _, fault := range faults {
			fmt.Fprintf(stderr, "hyper probe: %s\n", fault)
		}
		return ExitUsage
	}

	// Its instant, read once: what tls.days_left counts from, recorded
	// nowhere (§9, ADR-0034).
	instant := process.Now()

	host, decline := probeHost(loaded, providerName, operationName, operation, inputs)
	if decline != nil {
		return decline.render(stderr, repoRoot)
	}

	// One lookup for the two readers below: where an Operation is declared
	// is internal/artefact's fact, and what its http: and record: blocks mean
	// belongs to the two packages that read them.
	declaration := artefact.OperationNode(manifest.Root, operationName)

	request, read := capability.ReadRequest(declaration)
	if !read {
		fmt.Fprintf(stderr, "hyper probe: %s %s declares no legible http: block; hyper check reports what is wrong with it\n",
			providerName, operationName)
		return ExitUsage
	}
	call, err := request.Build(host, inputs)
	if err != nil {
		fmt.Fprintf(stderr, "hyper probe: %s\n", err)
		return ExitUsage
	}

	detail := artefact.ReadOperationDetail(manifest.Root, operationName)
	ctx, cancel := capability.Deadline(context.Background(), detail.DeadlineSeconds)
	defer cancel()

	// A Probe binds `local`, which carries no `auth:` block and no
	// credential slot, so the credential it sends is the empty one and the
	// wire is visible by construction rather than by a flag (§3, §4,
	// ADR-0017, ADR-0024).
	response, err := call.Perform(ctx, process.Dial, instant, capability.Credential{})
	if err != nil {
		// A call that got no answer renders on stderr and nowhere else.
		// No member of the response object says what went wrong — that is
		// the catch-all bucket ADR-0017 closed, and what stands in its
		// place is the same absence a projection already reads (§12) — but
		// a reader is owed the reason all the same, and a Probe is the one
		// surface that may say it: it binds `local`, which carries no
		// credential slot, so nothing here can leak a secret into a
		// terminal or an Actions log (§9, ADR-0017).
		//
		// The deadline is named as itself rather than as the transport's
		// word for it, because it is the one of these an artefact declared
		// and therefore the one a reader can edit (§3).
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintf(stderr, "hyper probe: the Operation's deadline of %s was reached and no response arrived\n", detail.Deadline)
		} else {
			fmt.Fprintf(stderr, "hyper probe: no response arrived: %s\n", err)
		}
	}

	rows := []render.Row{probeResultRow{
		Type:       "probe_result",
		Provider:   providerName,
		Operation:  operationName,
		Projection: projection.Read(declaration).Project(response),
		Response:   response,
	}}

	// The terminal row is `result` and never `outcome`: a Probe has no
	// outcome triple to report (§8, §9, ADR-0009). It carries no truncation
	// marker either — one Probe is one answer, and there is no result set
	// for a limit to cut.
	if code := writeAnswer("probe", stdout, stderr, parsed.json, rows, render.NewResultRow(false), writeProbePage); code != 0 {
		return code
	}
	return ExitClean
}

// probeInvokes is the one Operation a Probe may invoke stated as what it
// refuses, and both refusals are usage errors naming no error_code: an
// error_code names a check that declined an artefact, and here nothing was
// reviewed to decline (§9, §12, ADR-0060).
//
// The opaque half is §9's own sentence. `shell`'s `read` is a read Kind and
// its class: is local, so `probe shell read` satisfies every other rule on that
// page — and what it would run is a command supplied at invocation, with no
// Definition, no Journal entry and no Record, which is to say with nothing
// reviewed anywhere and no evidence afterwards that it happened. It is a usage
// error rather than a Refusal because a Refusal's remediation points at an
// artefact to edit and there is no edit that would make this work.
//
// The Kind half is §9's opening sentence read as the rule it is: `probe
// <provider> <operation>` invokes a **read** Operation. A Probe writes no
// Record and takes no lock, so an effectful Operation reached through one would
// touch the world and leave nothing behind — the one thing an effectful path
// may not do (§3, §8). It declines for the same reason and in the same shape:
// no edit to any artefact makes a `destroy` a thing a Probe may run.
func probeInvokes(provider, operation string, info artefact.OperationInfo) string {
	if info.IsShell {
		return fmt.Sprintf("%s %s is an opaque Operation, and a Probe may never invoke one whatever any Target grants\n"+
			"  there is no edit that would make it work; run the command yourself",
			provider, operation)
	}
	if info.Kind != "read" {
		return fmt.Sprintf("%s %s declares kind: %s, and a Probe invokes a read Operation and nothing else\n"+
			"  an effectful Operation is reached through a Step of a reviewed Procedure — hyper run",
			provider, operation, info.Kind)
	}
	return ""
}

// probeInput is one `--input name=value` as it was typed, kept in the order it
// was typed so that a caller who named one input twice is told about the second
// rather than about whichever a mapping kept.
type probeInput struct{ name, value string }

// splitInputs takes the repeated `--input name=value` off the argument list and
// answers the pairs, what is left for parseArgs, and the one fault it can find
// — a flag with no value, or a value with no `=` in it.
//
// It runs before parseArgs rather than inside it because `--input` is one
// command's flag and the three globals are every command's: a parser that knew
// about both would be one every other command's signature had to admit. What it
// does share is the rule: `--` ends the flags, so everything past it is copied
// through untouched and a positional spelled like a flag is still reachable
// (§9, ADR-0014).
func splitInputs(args []string) (inputs []probeInput, rest []string, fault string) {
	for i := 0; i < len(args); i++ {
		argument := args[i]
		switch {
		case argument == "--":
			return inputs, append(rest, args[i:]...), ""
		case argument == "--input":
			i++
			if i >= len(args) {
				return nil, nil, "--input requires a value"
			}
			input, read := readInputPair(args[i])
			if !read {
				return nil, nil, fmt.Sprintf("--input %s: want name=value", args[i])
			}
			inputs = append(inputs, input)
		case strings.HasPrefix(argument, "--input="):
			input, read := readInputPair(strings.TrimPrefix(argument, "--input="))
			if !read {
				return nil, nil, fmt.Sprintf("%s: want name=value", argument)
			}
			inputs = append(inputs, input)
		default:
			rest = append(rest, argument)
		}
	}
	return inputs, rest, ""
}

// readInputPair splits one `name=value` at its first `=`, so a value carrying
// one of its own survives. A pair with no `=` at all names no value, and an
// empty name names no input.
func readInputPair(pair string) (probeInput, bool) {
	name, value, named := strings.Cut(pair, "=")
	if !named || name == "" {
		return probeInput{}, false
	}
	return probeInput{name: name, value: value}, true
}

// readInputs reads every supplied value against the Operation's declared input
// schema **at that position** rather than by what the value looks like
// (ADR-0081), and answers the faults it found rather than the first: a caller
// who mistyped two inputs is told about both rather than sent round the loop
// once per variable, which is the reading §6's credential gate already takes.
//
// Every fault here is a usage error carrying no error_code. It is the same
// fault §4 refuses as schema-mismatch and deliberately not that code, because
// an error_code names a check that declined an **artefact** and a value typed
// at a command line is not one (§9, ADR-0060).
func readInputs(operation artefact.OperationInfo, supplied []probeInput) (map[string]schema.Scalar, []string) {
	read := make(map[string]schema.Scalar, len(supplied))
	// named is every input the command line named, whether or not its value
	// read: an input that was supplied and would not read has one fault and
	// not two, the second being *nothing supplied it*, which is false.
	named := make(map[string]bool, len(supplied))
	var faults []string

	for _, input := range supplied {
		declared, isDeclared := operation.Inputs[input.name]
		if !isDeclared {
			faults = append(faults, fmt.Sprintf("--input %s: the Operation declares no input of that name\n  %s",
				input.name, declaredInputs(operation)))
			continue
		}
		if named[input.name] {
			faults = append(faults, fmt.Sprintf("--input %s: named twice; one input takes one value", input.name))
			continue
		}
		named[input.name] = true
		value, reads := schema.ReadScalar(schema.Type(declared.Type), input.value)
		if !reads {
			faults = append(faults, fmt.Sprintf("--input %s=%s: does not read as the %s the Operation declares it",
				input.name, input.value, declared.Type))
			continue
		}
		if len(declared.Enum) > 0 && !slices.Contains(declared.Enum, value.Text()) {
			faults = append(faults, fmt.Sprintf("--input %s=%s: the Operation admits %s and nothing else",
				input.name, input.value, strings.Join(declared.Enum, ", ")))
			continue
		}
		read[input.name] = value
	}

	// Every declared input is supplied: there is no null and no
	// key-omission syntax, so an input left out has no sink to render at
	// (§3, ADR-0081).
	for _, name := range sortedInputNames(operation) {
		if !named[name] {
			faults = append(faults, fmt.Sprintf("--input %s: the Operation declares it and nothing supplied it", name))
		}
	}
	return read, faults
}

// declaredInputs is the line that stands under an unknown input name: what the
// Operation does declare. It lists the namespace rather than guessing at a near
// miss, which is the rule every name in this tool resolves under (ADR-0047).
func declaredInputs(operation artefact.OperationInfo) string {
	names := sortedInputNames(operation)
	if len(names) == 0 {
		return "the Operation declares no inputs at all"
	}
	return "the Operation declares " + strings.Join(names, ", ")
}

// sortedInputNames is the Operation's declared inputs by name, sorted so that
// two runs of one faulty command line report the same faults in the same order.
func sortedInputNames(operation artefact.OperationInfo) []string {
	names := make([]string, 0, len(operation.Inputs))
	for name := range operation.Inputs {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// probeDecline is a Probe declining before it reaches the world, and the three
// shapes that decline takes. It is a value rather than three returns because
// the decision and its rendering are one fact: what a reader is owed is the
// artefact to author, and which of the three they get turns on whether there is
// a line to name.
//
// Two of them are the one Refusal §9 states — `host-not-granted`, exit 77,
// naming the Target declaration to author (ADR-0042) — and they differ only in
// whether that declaration exists to be pointed at.
//
// The third is not a Refusal at all, and the boundary is worth stating because
// the fault behind it *has* a code. A Manifest whose candidate set and the
// grant intersect to several hosts under an Operation declaring no
// `host-input:` is `manifest-inconsistent` where a Step binds it (§4), and §4
// finds it exactly there: the fault is decidable only at a binding, and a
// Manifest nothing binds passes `check` clean. A Probe is a binding no artefact
// wrote, so nothing was reviewed and nothing declined — hyper simply cannot say
// which host the request would reach. That is a usage error carrying no
// error_code, on ADR-0060's line: a code names a check that declined an
// artefact, and no check ran here.
type probeDecline struct {
	// positioned is the Refusal where there is a line to edit, and it
	// renders as the problem table check already renders.
	positioned *problem.Problem
	// code and message are the Refusal with no line at all — a repository
	// that declares no local Target, or one whose local writes no hosts:,
	// has no line to point at.
	code, message string
	// usage is a fault that is not a Refusal, rendered on stderr and exited
	// at 2 with no error_code and no row stream opened (§9, ADR-0060).
	usage string
}

// render writes the decline and answers the exit code its caller returns. The
// three shapes are one dispatch rather than three call sites, so which of them
// a decline takes is decided where the decline is built and stated nowhere
// else.
func (d probeDecline) render(stderr io.Writer, repoRoot string) int {
	switch {
	case d.usage != "":
		fmt.Fprintf(stderr, "hyper probe: %s\n", d.usage)
		return ExitUsage
	case d.positioned != nil:
		return refuseProblems(stderr, repoRoot, []problem.Problem{*d.positioned})
	default:
		return refuse(stderr, d.code, d.message)
	}
}

// probeHost is internal/artefact's answer rendered: it resolves the one host
// the call reaches and, where the call reaches none, which of the three
// declines that is.
//
// The deciding is not here. Where a Probe may reach is a fact about the
// artefacts — the candidate set, the grant, and their intersection (§3,
// ADR-0029, ADR-0042) — and artefact.ResolveHost is where it is read,
// beside the checks a Step's binding is held to. What is here is what a reader
// is owed for each answer, which is this surface's alone.
func probeHost(loaded repository.Loaded, provider, operationName string, operation artefact.OperationInfo, inputs map[string]schema.Scalar) (string, *probeDecline) {
	reach := artefact.ResolveHost(loaded.Providers[provider], operation,
		loaded.Targets[localTarget], operation.SuppliedHost(inputs))
	switch reach.Reach {
	case artefact.ReachGranted:
		return reach.Host, nil
	case artefact.ReachUndecidable:
		return "", &probeDecline{usage: fmt.Sprintf(
			"the candidate set and %s's hosts: grant intersect to %d hosts, and %s %s declares no host-input:\n"+
				"  which host the request would reach is undecidable — the Manifest names an input carrying one, or %s grants one",
			localTarget, reach.Granted, provider, operationName, localTarget)}
	case artefact.ReachIllegible:
		return "", &probeDecline{usage: fmt.Sprintf(
			"%s %s writes a host: hole that is neither from-target nor a declared enumeration\n"+
				"  hyper check reports what is wrong with the Manifest", provider, operationName)}
	default:
		return "", hostNotGranted(loaded, reach.Host)
	}
}

// hostNotGranted is the Refusal a host outside the grant earns, in whichever of
// its two shapes the repository leaves available: positioned on the `hosts:`
// line of the declaration that did not grant it, which is what makes the next
// act an edit rather than a search (§4, §8, ADR-0042), and unpositioned where
// there is no such line to name.
//
// There are two ways to have no line and they are one answer: a repository
// declaring no `local` at all, and a `local` that grants no `http` and so
// writes no `hosts:` — `hosts:` being present exactly where `capabilities:`
// grants `http` (§3). Both grant nothing, both decline under the same code, and
// neither has a coordinate, so both say what to author instead of pointing at a
// line that is not there.
func hostNotGranted(loaded repository.Loaded, host string) *probeDecline {
	localDeclaration, declared := loaded.TargetDeclaration(localTarget)
	if !declared {
		return &probeDecline{
			code: artefact.CodeHostNotGranted,
			message: fmt.Sprintf(
				"the repository declares no Target named %s, so it grants no host — author targets/%s.yaml granting %s",
				localTarget, localTarget, namedHost(host)),
		}
	}

	file := localDeclaration.Path
	line := artefact.TopLevelKeyLine(localDeclaration.Root, "hosts")
	if line == 0 {
		return &probeDecline{
			code: artefact.CodeHostNotGranted,
			message: fmt.Sprintf("%s writes no hosts:, so it grants no host — add %s to a hosts: list in %s",
				localTarget, namedHost(host), file),
		}
	}
	return &probeDecline{positioned: &problem.Problem{
		File:      file,
		Line:      line,
		Column:    1,
		Field:     "hosts",
		ErrorCode: artefact.CodeHostNotGranted,
		Message:   fmt.Sprintf("%s grants no %s", localTarget, granted(host)),
	}}
}

// namedHost and granted are the host a decline names, and what stands there
// where there is none to name: a `{from-target}` against a grant with nothing
// in it expands to nothing, so there is no host the call would have reached —
// the grant being the only thing that could have named one. They differ only in
// the sentence they sit in.
func namedHost(host string) string {
	if host == "" {
		return "the host this Operation would reach"
	}
	return fmt.Sprintf("%q", host)
}

func granted(host string) string {
	if host == "" {
		return "host at all"
	}
	return host
}

// probeResultRow is the whole of a Probe's answer as one row, and its members
// are §9's own: {"type":"probe_result","provider":…,"operation":…,
// "projection":…,"response":…}. §9 states the shape for the MCP tool and this
// surface reuses it rather than minting a second, on the rule that the two
// share one renderer.
//
// projection is what hyper derived, in the shape a Record would have held, and
// response is the raw response beside it — which no credentialled surface shows
// (ADR-0017). A Probe binds `local`, which carries no credential slot, so the
// wire is visible by construction rather than by a flag.
type probeResultRow struct {
	Type       string            `json:"type"`
	Provider   string            `json:"provider"`
	Operation  string            `json:"operation"`
	Projection projection.Fields `json:"projection"`
	Response   capability.Object `json:"response"`
}

// Cells is empty: the row is a page of two blocks rather than a line in a table
// of like rows, and writeProbePage is what writes it (ADR-0026).
func (r probeResultRow) Cells() []string { return nil }

// projectedFieldRow is one line of the FIELD/VALUE table, and it is the page's
// row and never the stream's: the wire carries the projection as one mapping
// inside probe_result, which is the shape a Record would have held, and the
// page carries the same mapping one line per member. Both are written from the
// one row above (ADR-0026).
type projectedFieldRow projection.Field

func (r projectedFieldRow) Cells() []string {
	return []string{r.Name, projection.Text(r.Value)}
}

// probeColumns is the projection table's header.
var probeColumns = []string{"FIELD", "VALUE"}

// writeProbePage is a Probe's page: the projection as a FIELD/VALUE table, then
// the response object beneath it under a RESPONSE heading — the two keys the
// probe_result row carries, one block each.
//
// A projection that resolved nothing writes no table at all, header included,
// which is the renderer's own rule: what stands in place of an empty table is
// the command's business, and here what stands there is the response, which is
// the whole reason the block beneath is on the page.
func writeProbePage(w io.Writer, rows []render.Row) error {
	if len(rows) == 0 {
		return nil
	}
	result, written := rows[0].(probeResultRow)
	if !written {
		return nil
	}

	fields := make([]render.Row, 0, len(result.Projection))
	for _, field := range result.Projection {
		fields = append(fields, projectedFieldRow(field))
	}
	if err := render.WriteTable(w, probeColumns, fields); err != nil {
		return err
	}
	if len(fields) > 0 {
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}

	response, err := result.Response.MarshalJSON()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "RESPONSE\n%s\n", response)
	return err
}
