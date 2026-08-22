package run

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// A Step's Expansion: the resolution of `over:` to the concrete Records the
// Step will act on, and the four checks that decide before its first call goes
// out (§5, §6, §12, issue #139).
//
// **It resolves before the Step runs**, which is what makes every decline here
// a Refusal rather than a halt: a guardrail declining after a member's turn
// would be declining after an effect. *Which three of the five* therefore has
// an answer, and a re-run attempts them in the same order.
//
// **The order is the one a reviewer can predict from what is in front of
// them**, which makes it two rules rather than one. Where the selector is a
// `values:` list the artefact states the order and that is the order — the list
// top-first, as authored. Where it is `assets:` or `observations:` there is no
// page to read an order off, so the Record `name` supplies one, sorted by
// Unicode code point: the name the Store holds, never the percent-encoded path
// segment §12 builds from it to reach a file (ADR-0044). The sort is total and
// needs no tie-break, one Expansion being one Target and one Definition.
//
// **The Bound is counted here**, which is the position §6 fixes for it and not
// a convenience: an Expansion has a count before the Step's first call goes out,
// and a guardrail declining after one would be a halt rather than a Refusal
// (§5, §7, ADR-0072, issue #149). What an effectful Expansion still adds is
// `skip-if-recorded`'s per-member test, which is a different moment and a later
// ticket's.
//
// **The Store shortens a `destroy`'s list and never lengthens it**, and that is
// the only thing resolved here that anything removes: a `values:` member whose
// head is a Tombstone is dropped before the count the Bound is read against is
// taken (§5, ADR-0033, literalMembers below).

// CodePredicateTypeMismatch is §6's half of the code §4 already fires where the
// fault is authored: an operator handed a **stored** value it cannot compare.
//
// The two halves are one check at two moments, which is why they are one code
// (ADR-0035). It is spelled here rather than imported from internal/artefact
// because that package's constant names an authored fault and this one names a
// value nobody wrote — the string is the contract and the two sites reach it
// independently, as §4's and §6's own texts do.
const CodePredicateTypeMismatch = "predicate-type-mismatch"

// CodeRecordIdentityCollision is the code both of the Expansion's identity
// comparands carry: two members of one Expansion resolving to one Record
// identity, and a resolved identity colliding with a series the Store already
// holds (§6, §7, §12, ADR-0070, ADR-0075).
const CodeRecordIdentityCollision = "record-identity-collision"

// CodeBoundExceeded is §6's half of the code §4 already fires where the count
// is authored: an Expansion resolving to more Records than the Step's declared
// Bound.
//
// It is one code because it is one check — what names a Refusal is the check
// that declined, never the moment it ran — and it is spelled here rather than
// imported from internal/artefact for CodePredicateTypeMismatch's reason above:
// §4's constant names a length a reader counts off the page and this one a count
// no file holds, and the string is the contract the two sites reach
// independently, as §4's and §6's own texts do (§4, §5, §6).
const CodeBoundExceeded = "bound-exceeded"

// selector is a Step's `over:` as it was authored: which of §12's three forms
// it is, and what that form carries.
//
// A Step carrying no `over:` has the zero value, whose Form is empty — it
// resolved no selector and holds none, which is a different thing from an
// Expansion that resolved to nothing (§7).
type selector struct {
	// Form is `values`, `assets` or `observations`, and "" where the Step
	// carries no selector at all.
	Form string
	// Values is the literal list in authored order, and List the predicate
	// list; each is empty on the form that does not carry it.
	Values []*yaml.Node
	List   []predicate
	// Declared is the selector as authored, in the canonical values the
	// Step file holds it in — held beside what it resolved to so that what
	// a Step reached is readable back from the entry long after the Run
	// without a checkout at the revision its Provenance names (§7).
	Declared store.Value
	// Line is where `over:` begins, which is what a Refusal citing the
	// selector rather than one of its predicates points at (§8).
	Line int
}

// readSelector reads a Step's `over:` as authored.
//
// It judges nothing: a mapping carrying two forms or none is one `check` has
// already refused, and this reader takes the first of §12's three it finds
// (ADR-0064).
func readSelector(over *yaml.Node) selector {
	if over == nil || over.Kind != yaml.MappingNode {
		return selector{}
	}

	read := selector{Declared: asStored(over), Line: over.Line}
	for i := 0; i+1 < len(over.Content); i += 2 {
		key, value := over.Content[i], over.Content[i+1]
		if key.Kind != yaml.ScalarNode || read.Form != "" {
			continue
		}
		switch key.Value {
		case "values":
			read.Form, read.Line = key.Value, key.Line
			if value.Kind == yaml.SequenceNode {
				read.Values = value.Content
			}
		case "assets", "observations":
			read.Form, read.Line = key.Value, key.Line
			read.List = readPredicates(value)
		}
	}
	return read
}

// asStored is an authored node as the canonical value a Step file holds it in.
//
// The scalar arm reads the node's **resolved tag** rather than its characters,
// which is the one place this package asks YAML what a value is: `bound: 5` is
// a number in the entry and `starts_with: preview-` is a string, and the
// difference is one a reader of the entry sees. Everywhere else `hyper` reads
// characters against a declared type (ADR-0081) — there is no declared type
// here, the selector being held as authored rather than as anything read.
func asStored(node *yaml.Node) store.Value {
	switch node.Kind {
	case yaml.MappingNode:
		mapping := store.Mapping{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			if key := node.Content[i]; key.Kind == yaml.ScalarNode {
				mapping[key.Value] = asStored(node.Content[i+1])
			}
		}
		return mapping
	case yaml.SequenceNode:
		array := make(store.Array, 0, len(node.Content))
		for _, element := range node.Content {
			array = append(array, asStored(element))
		}
		return array
	default:
		switch node.Tag {
		case "!!int", "!!float":
			if number, err := store.ParseNumber(node.Value); err == nil {
				return number
			}
		case "!!bool":
			return store.Bool(node.Value == "true")
		}
		return store.String(node.Value)
	}
}

// member is one member of an Expansion: one Record identity each (ADR-0070),
// and the value the Step's `{item:}` references resolve against.
type member struct {
	// Name is what `expanded_to` holds for this member — the Record `name`
	// where the selector ranges over series, and the literal itself where
	// it is a `values:` list.
	Name string
	// Item is what `{item: $}` names, which is the member itself, and Head
	// the head version's fields, which is what `{item: $.field}` reads. A
	// `values:` member has the first and not the second: it names something
	// `hyper` may never have recorded, which is the whole of what a literal
	// identifier is for (§5).
	Item store.Value
	Head store.Mapping
	// Inputs is the Step's `args:` resolved for this member, and Identity
	// the name its Record will be held under where the Operation's
	// `identity:` resolves before the call — "" where it reads from the
	// response and there is nowhere earlier than the answer to decide it
	// (§3, ADR-0072).
	Inputs map[string]schema.Scalar
	// Argv is the argv this member's call execs, and nil on every
	// Capability but `shell` — the one place a list-typed input reaches
	// anything at all (§3, §12, ADR-0051).
	Argv     []string
	Identity string
}

// expansion is one Step's Expansion, resolved: the selector as authored and the
// members it resolved to, in Expansion order.
//
// A Step carrying no selector holds one member and a zero selector, which is
// the set of one §6 states — vacuous to compare against itself, and compared
// against the Store like any other.
type expansion struct {
	Selector selector
	Members  []member
}

// names is what `expanded_to` holds: the members' names in **Expansion order**
// and not sorted, and the empty list where the Expansion resolved to nothing.
//
// It is nil where the Step carries no selector, which is what keeps the
// `selector` block off that Step's file entirely (§7).
func (e expansion) names() []string {
	if e.Selector.Form == "" {
		return nil
	}
	names := make([]string, 0, len(e.Members))
	for _, held := range e.Members {
		names = append(names, held.Name)
	}
	return names
}

// checks is what an Expansion's checks found, one bucket each, empty where the
// check declined nothing.
//
// **The order is the causal one** and it lives in declined below rather than in
// the order the buckets are filled: a predicate resolves the set, the set has
// the count a Bound is read against, the members' arguments fill the inputs an
// identity is projected from, and the identities are projected off the members
// the set holds. Where the last two are available at once the sibling collision
// is named first, being reproducible from the artefact alone and therefore
// pointing at an edit with no Store in hand (§6).
type checks struct {
	// Predicate is `predicate-type-mismatch`: an operator handed a stored
	// value it cannot compare (ADR-0035).
	Predicate []Refusal
	// Bound is `bound-exceeded`: the count the Expansion resolved to, read
	// against the maximum the Step's author declared (§5, §6).
	Bound []Refusal
	// Arguments is §6's half of `schema-mismatch`: an `args:` value
	// arriving from a reference that will not read as the type its input
	// declares, or resolving to nothing at all.
	Arguments []Refusal
	// Sibling and Stored are the two comparands of
	// `record-identity-collision`: the resolved identities against each
	// other, and against the series the Store already holds (§6, §7).
	Sibling, Stored []Refusal
}

// declined is the Refusal the Expansion's checks produce, in §6's causal order,
// and nothing at all where none of them declined.
//
// It answers **one** check's finding rather than every check's: a Refusal is one
// phase's, and §7 fixes that the array holds more than one member only where the
// phase evaluates many checks together. The checks here are five moments rather
// than one, so the first that declined is the whole of the answer — and which
// that is is a fact about causality rather than about severity, which is what
// makes it stable across two Runs of one artefact.
func (c checks) declined() []Refusal {
	for _, found := range [][]Refusal{c.Predicate, c.Bound, c.Arguments, c.Sibling, c.Stored} {
		if len(found) > 0 {
			return found[:1]
		}
	}
	return nil
}

// expand resolves the Step's selector and runs the checks that decide before
// its first call goes out.
//
// The error it answers is what halted the Run — reaching the Store, which is
// nobody's guardrail — and the Refusals are what declined it. A Step reaching
// neither is a Step whose Expansion resolved.
func (r run) expand(bound binding, authored sequenced, position int) (expansion, []Refusal, error) {
	held := expansion{Selector: readSelector(authored.Over)}
	cited := r.citation(authored, position, held.Selector)

	// An `over:` carrying none of §12's three forms is a fault and never a
	// Step with no selector: `check` refuses the shape, and falling through
	// to make **one unselected call** would be this binary reading an
	// authored population as an absent one (§12, ADR-0064).
	if authored.Over != nil && held.Selector.Form == "" {
		return expansion{}, nil, fmt.Errorf("step %s carries an over: naming none of assets:, observations: or values: — hyper check reports it", named(authored))
	}

	var found checks
	switch held.Selector.Form {
	case "":
		// A Step carrying no `over:` resolves no selector and makes one
		// call, which is a set of one (§6).
		held.Members = []member{{}}
	case "values":
		members, err := r.literalMembers(bound, held.Selector, authored)
		if err != nil {
			return expansion{}, nil, err
		}
		held.Members = members
	default:
		members, declined, err := r.seriesMembers(held.Selector, authored, cited)
		if err != nil {
			return expansion{}, nil, err
		}
		held.Members, found.Predicate = members, declined
	}

	// The count the Bound is read against. It is filled here and **ordered
	// by declined above**, which is the file's rule for every bucket: a set
	// no predicate could resolve is not a set with a count, and the answer
	// that says so is reached before this bucket is.
	found.Bound = r.exceededBound(held, authored, cited)
	if declined := found.declined(); len(declined) > 0 {
		return held, declined, nil
	}

	// The arguments, and the identity each member projects where the
	// Operation's `identity:` resolves before the call. They are one walk
	// because the second is read off the first: an identity written as a
	// template hole is filled from the inputs the Expansion just resolved
	// (§3, §12).
	for index, resolving := range held.Members {
		read, declined, err := r.arguments(bound.operation, authored, resolving, cited)
		if err != nil {
			return expansion{}, nil, err
		}
		if declined != nil {
			found.Arguments = append(found.Arguments, *declined)
			continue
		}
		identity, err := identityBeforeTheCall(bound, held.Selector, resolving, read.Inputs, authored)
		if err != nil {
			return expansion{}, nil, err
		}
		held.Members[index].Inputs, held.Members[index].Argv = read.Inputs, read.Argv
		held.Members[index].Identity = identity
	}
	if declined := found.declined(); len(declined) > 0 {
		return held, declined, nil
	}

	sibling, stored, err := r.collisions(held, authored, cited)
	if err != nil {
		return expansion{}, nil, err
	}
	found.Sibling, found.Stored = sibling, stored
	return held, found.declined(), nil
}

// literalMembers is a `values:` list resolved: its members in authored order,
// top-first, each one the whole of what `{item: $}` names — less the ones the
// Store already holds a Tombstone for.
//
// **The Tombstone rule reads on a `values:` list too, and there it is a
// `destroy`'s**: a member whose head is a Tombstone is dropped and a member
// naming no series at all is reached. What that states is that `hyper` drops
// what it knows is gone and reaches what it has no record of, a member it never
// touched being in the second class — so a `values:` list left standing in a
// Procedure that runs on a Cadence is self-limiting rather than a Run that
// fails on a call the artefact never asked it to repeat (§5, ADR-0033).
//
// **A `mutate` reaches such a member instead**, and the same sentence is the
// reason: a create over a Tombstoned series is a call the artefact *is* asking
// for. Dropping it here would leave destroy-then-recreate reachable from a Step
// with no selector and from nowhere else (§5, ADR-0011). A `read` drops nothing
// either, expanding over a literal list the way it always did.
//
// **The Store shortens the list and never lengthens it**, which is what lets
// `check` count the **authored** length against the Bound offline: the count
// this Expansion is read against can only be the shorter one, so a list that
// passes offline cannot fail here for having grown (§4, §5, expand above).
//
// **What was dropped stays visible.** The selector is held as authored beside
// what it resolved to, which gives the entry an arithmetic no predicate can
// offer: a member present in `declared` and absent from `expanded_to` is one
// the Store already held a Tombstone for — *three authored, two expanded to,
// one already gone*, readable off the entry with no checkout (§7, ADR-0033).
//
// That arithmetic has **one** cause, and the skip below is not a second one: a
// `values:` member that is not a bare scalar is `schema-mismatch` at `check`,
// which a Run re-runs in full before Step 1, so no Run reaches here carrying
// one. It is skipped rather than read as a member all the same — a fault this
// file cannot reach is still one it must not silently reinterpret (§4,
// ADR-0064).
//
// The survivors keep the authored sequence. A `values:` list is the one
// selector form whose order a reader can predict from the page in front of
// them, and shortening it moves nobody up past anybody (§6, ADR-0044).
func (r run) literalMembers(bound binding, over selector, authored sequenced) ([]member, error) {
	members := make([]member, 0, len(over.Values))
	for _, value := range over.Values {
		if value.Kind != yaml.ScalarNode {
			continue
		}
		gone, err := r.dropped(bound, authored.identity(value.Value))
		if err != nil {
			return nil, err
		}
		if gone {
			continue
		}
		members = append(members, member{Name: value.Value, Item: store.String(value.Value)})
	}
	return members, nil
}

// dropped says this member does not survive the Expansion, which takes both of
// §5's halves and is why it answers the drop rather than the Store's fact: a
// member the Store holds a Tombstone for is dropped from a `destroy` and
// **reached** by a `mutate`, so *what the Store knows* is only half of what
// decides it.
//
// **The Kind is the first half and it is read first.** A `mutate` over a
// Tombstoned series is a call the artefact is asking for and a `read` drops
// nothing at all, so neither Kind reaches the branch here — which is the
// sentence literalMembers above argues, enacted where it costs a member its
// place (§5, ADR-0011).
//
// **The lookup is exact and is never folded.** A member authored `Foo` against
// a standing `foo` is not that series and survives this — the two names are not
// equal — and what reaches it instead is the Store comparand below, which
// Refuses `record-identity-collision` before the call rather than opening a
// colliding series with the call already gone out (§6, §7, ADR-0033).
//
// It reads the head of one series per member rather than enumerating the
// branch. A `values:` list is authored, so it is short by construction and its
// members are named up front, and the question asked of each is *does this one
// series read dead* — which is a listing of one directory (§7, store.Head).
func (r run) dropped(bound binding, id store.Identity) (bool, error) {
	if !bound.tombstones() {
		return false, nil
	}
	head, standing, err := r.request.Store.Head(id)
	if err != nil || !standing {
		return false, err
	}
	return head.Tombstone, nil
}

// seriesMembers is an `assets:` or `observations:` selector resolved: the Step's
// **own Definition and Target's** Record series, filtered by the predicate list
// and ordered by the Record `name`.
//
// **A predicate reads the head version of each series and no other.** *Any
// version* would have a selector reach a thing for what it used to be, and
// would make one artefact reach further every month the Store grows (§5).
//
// **A series whose head is a Tombstone stands for nothing** and is expanded
// over by neither form: what one Run destroyed the next does not reach again
// (§5, ADR-0027).
//
// **The predicate list is AND and does not short-circuit.** Every conjunct is
// evaluated against every candidate, so whether a Run Refuses does not depend
// on the order an author happened to write two conjuncts in — and a candidate an
// earlier conjunct excluded still has the rest evaluated against it, which is
// the silence ADR-0035 exists to remove.
func (r run) seriesMembers(over selector, authored sequenced, cited citation) ([]member, []Refusal, error) {
	records, err := r.request.Store.Records()
	if err != nil {
		return nil, nil, err
	}

	wanted := store.RecordObservation
	if over.Form == "assets" {
		wanted = store.RecordAsset
	}

	var members []member
	var declined []Refusal
	for _, series := range records {
		if series.Identity.Target != authored.Target || series.Identity.Definition != authored.Definition {
			continue
		}
		head, standing := series.Head()
		if !standing || head.RecordType != wanted || head.Tombstone {
			continue
		}
		version, err := r.request.Store.Read(head)
		if err != nil {
			return nil, nil, err
		}

		holds := true
		for _, conjunct := range over.List {
			held, mismatch := conjunct.holds(version.Fields, r.started)
			if mismatch != "" {
				declined = append(declined, r.refusal(CodePredicateTypeMismatch,
					fmt.Sprintf("on %s, %s", series.Identity.Name, mismatch),
					cited.at(conjunct.Line, fmt.Sprintf("over.%s[%d].%s", over.Form, conjunct.Index, conjunct.Operator))))
			}
			holds = holds && held
		}
		if holds {
			members = append(members, member{Name: series.Identity.Name, Head: version.Fields})
		}
	}
	if len(declined) > 0 {
		// A predicate that could not compare resolved no set, so what
		// the Step expanded to is nothing and its entry says so with
		// the empty list — rather than with the members that happened
		// to survive the conjunct that declined, which is an Expansion
		// nothing performed (§7, ADR-0061).
		return nil, declined, nil
	}

	// By the name the Store holds, sorted by Unicode code point — and never
	// by the percent-encoded path segment §12 builds from it, escaping
	// dragging every escaped character to the left of every unreserved one
	// (ADR-0044). Records answers in identity order, which is that order
	// over one (Target, Definition); the sort is written out anyway, one
	// Expansion's order being this function's to state rather than a
	// property of a listing it is reading.
	slices.SortFunc(members, func(a, b member) int { return cmp.Compare(a.Name, b.Name) })
	return members, declined, nil
}

// exceededBound is the Bound at the Expansion: what the selector resolved to,
// counted against the maximum the Step's author declared (§5, §6, §7).
//
// **It counts Records and does not weigh what happened to each one.** The
// failure it exists for is the runaway selector — the predicate that widened
// overnight, the list that grew — and it says nothing about how severe a single
// correctly-Bounded call is, which is a fact the gutter and `FLAGS` carry and
// no count could (§5).
//
// **It is the run-time half of the check `check` already performs offline**,
// where the count is authored and therefore readable off the file: an `over:`
// `values:` list longer than the Bound is refused before anything runs, the
// Store only ever shortening such a list (§4, §5). One check at two moments is
// one code, and the two differ only in what each cites — the authored length
// there, the Bound here, each being what the reader in front of it can edit.
//
// **A Step declaring no `bound:` counts nothing**, and a `read` Step is that
// Step by construction: a `bound:` written on one is `unknown-key` at `check`,
// which a Run re-runs in full before Step 1, so the node this reads stands on
// an effectful Step alone (§4, §6, ADR-0064).
func (r run) exceededBound(held expansion, authored sequenced, cited citation) []Refusal {
	declared, bounded := declaredBound(authored.Bound)
	observed := len(held.Members)
	if !bounded || observed <= declared {
		return nil
	}
	return []Refusal{r.compared(CodeBoundExceeded,
		fmt.Sprintf("expansion resolved %s on %s", counted(observed, held.Selector), authored.Target),
		cited.at(authored.Bound.Line, "bound"),
		store.Int(int64(declared)), store.Int(int64(observed)))}
}

// declaredBound is the Bound the Step authored, and whether it authored one.
//
// It reads the node's characters as an integer, which is `check`'s own reading
// of that key (§4). A `bound:` that will not read as one is a repository a Run
// Refused before Step 1, so the false answer here means *this Step declares no
// Bound* and never *this Bound could not be read*: a fault this file cannot
// reach is still one it must not silently reinterpret (ADR-0064).
//
// A **negative** Bound reads as one and is not this file's to judge. §4 holds
// `bound:` to an integer and refuses one only where it may not stand at all, so
// `bound: -1` reaches here and Refuses every Expansion, which is the direction a
// Bound errs in: a guardrail nobody meant declines rather than admits.
func declaredBound(node *yaml.Node) (int, bool) {
	if node == nil || node.Kind != yaml.ScalarNode {
		return 0, false
	}
	declared, err := strconv.Atoi(node.Value)
	if err != nil {
		return 0, false
	}
	return declared, true
}

// counted is what the Refusal's message says the Expansion resolved: the count,
// and the word for what was counted beside it.
//
// The word is the **selector's own form** — `23 assets`, which is §7's own
// phrasing — so what a reader is told they have too many of is the thing the
// artefact in front of them named. A Step carrying no `over:` named nothing, so
// `record` stands there instead: it resolves no selector and holds a set of one
// (§6), and the count is read off that set like any other rather than written
// down here, a message that can only be right by construction being one nobody
// notices going wrong.
func counted(observed int, over selector) string {
	switch {
	case over.Form != "":
		return fmt.Sprintf("%d %s", observed, over.Form)
	case observed == 1:
		return "1 record"
	default:
		return fmt.Sprintf("%d records", observed)
	}
}

// identityBeforeTheCall is the name this member's Record will be held under
// where it is known without a response, and "" where it reads from one.
//
// Which of the two a Manifest declares is what decides whether an identity
// collision Refuses at Expansion or halts the Run: a template hole fills from
// the resolved inputs before the call, and a `$`-rooted path names a value that
// exists only once the call has gone out (§3, §6, ADR-0072, issue #144).
//
// **A `destroy` declares no identity, and its literal resolves one all the
// same.** It carries no `record:` at all, and the series its Tombstone goes
// into is the one the Expansion acted on — which where the selector is a
// `values:` list is a name an **author** wrote. That is the one Record name in
// the system with an author for its origin rather than a Manifest-declared
// field of a response, so it is the one that can be spelled wrong; it is known
// before the call like any other, and it is compared like one, which is what
// keeps `Foo` against a standing `foo` a Refusal rather than a colliding series
// opened with the call already gone out (§3, §7, ADR-0033, ADR-0037).
//
// **It stops at the literal, and that is a limit rather than an omission.** An
// `assets:` member's name came out of the Store, so it cannot be spelled wrong
// and it opens nothing — and the comparand it would meet answers the *other*
// spelling wherever the branch already holds both, a state the write cannot
// prevent and no edit can clear (store.Collision, ADR-0075). Refusing there
// would leave a series destroyable by neither selector form and correctable by
// nobody, which is the wall ADR-0033 declined to build rather than the
// guardrail §6 asked for.
//
// A `destroy` carrying no selector resolves nothing here either, its one member
// having no name — which is §5's *no series to write a Tombstone under*,
// standing where `check` leaves it (§5, ADR-0053).
func identityBeforeTheCall(bound binding, over selector, resolving member, inputs map[string]schema.Scalar, authored sequenced) (string, error) {
	if bound.tombstones() {
		if over.Form != "values" {
			return "", nil
		}
		return resolving.Name, nil
	}
	operation := bound.operation
	if operation.Identity == "" || strings.HasPrefix(operation.Identity, "$") {
		return "", nil
	}
	// A hole that will not fill **halts**. It is a Manifest naming an input
	// its Operation does not declare, which `check` refuses as
	// `hole-illegal` (§4) and no Run can reach — and answering "" for it
	// here would drop the member from **both** identity comparands, turning
	// a Refusal §6 states into #144's halt after the call had gone out. A
	// fault this milestone cannot reach is still a fault it must not
	// silently reinterpret (ADR-0064).
	filled, err := capability.Fill("identity:", operation.Identity, inputs)
	if err != nil {
		return "", fmt.Errorf("step %s binds %s, whose identity: %s does not fill from the inputs it was handed — hyper check reports it: %w",
			named(authored), authored.Operation, operation.Identity, err)
	}
	return filled, nil
}

// collisions is the Expansion's two identity comparands, both run **once** over
// the identities it resolved rather than at each member's turn — which is what
// keeps them Refusals rather than declines after an effect (§6).
//
// **The sibling comparand** is the resolved identities against each other:
// every member of an Expansion is one Record identity (ADR-0070), so two
// members that are one identity under §7's fold are several calls writing
// several versions of one series, and the entry would then say *three expanded
// to, one concluded about* — the phrase reserved for a call that may have
// reached the world (§3).
//
// **The Store comparand** is the same identities against the series the branch
// already holds, under the same fold and carrying the same code. One that is
// byte-equal to a standing series is the ordinary further version and nothing
// at all. It reaches a Step carrying no `over:` as well — vacuous against
// itself, and not against the Store.
//
// **It reaches a `destroy`'s `values:` literal**, which is the one Record name
// in the system an author wrote rather than a Manifest projected, and therefore
// the one that can be spelled wrong. `Foo` against a standing `foo` survives
// the head lookup that shortens the list — the two names are not equal — and
// would otherwise open a colliding series with the call already gone out (§7,
// ADR-0033). It reaches no other of that Kind's members, for the reason
// identityBeforeTheCall above states.
//
// **A member that is its own identity is named twice and the sentence is left
// alone.** On a `destroy` the member and the name it resolved are one string,
// so the message reads `Foo resolves to Foo`; a second phrasing for that case
// would be a second sentence to keep true of every form, where what a reader
// needs from this one is both spellings verbatim and the series they collide
// with (§7).
//
// Both are silent where the Operation reads its `identity:` from the response:
// there is nowhere earlier than the answer to decide it, and what happens then
// is §6's halt rather than a Refusal (ADR-0072).
func (r run) collisions(held expansion, authored sequenced, cited citation) (sibling, stored []Refusal, err error) {
	resolved := projectedIdentities(held, authored)
	if len(resolved) == 0 {
		return nil, nil, nil
	}

	// The sibling comparand, in Expansion order: the first member to hold a
	// folded identity keeps it, and every later member that is one with it
	// is the collision — so what a Refusal names is the member an edit
	// would remove rather than the one that was already there.
	first := map[store.Identity]resolvedIdentity{}
	for _, member := range resolved {
		folded := store.Folded(member.id)
		earlier, taken := first[folded]
		if !taken {
			first[folded] = member
			continue
		}
		sibling = append(sibling, r.refusal(CodeRecordIdentityCollision,
			fmt.Sprintf("%s resolves to %s and %s resolves to %s, which are one Record identity — every member of an Expansion is one identity",
				expansionMember(earlier.member, cited), earlier.id.Name, expansionMember(member.member, cited), member.id.Name),
			cited.selector()))
	}

	collided, err := r.request.Store.Collisions(identitiesOf(resolved))
	if err != nil {
		return nil, nil, err
	}
	for _, member := range resolved {
		standing, found := collided[member.id]
		if !found {
			continue
		}
		stored = append(stored, r.refusal(CodeRecordIdentityCollision,
			fmt.Sprintf("%s resolves to %s, and the Store already holds %s under %s/%s — the two are one Record identity",
				expansionMember(member.member, cited), member.id.Name, standing.Name, standing.Target, standing.Definition),
			cited.selector()))
	}
	return sibling, stored, nil
}

// resolvedIdentity is one Record identity an Expansion resolved before its first call,
// and the member that resolved it. The two travel together because both
// comparands report both: the identity is what collides and the member is what
// an edit would reach.
type resolvedIdentity struct {
	member string
	id     store.Identity
}

// projectedIdentities is the identities the Expansion resolved, in Expansion
// order, and nothing at all for the members whose `identity:` reads from a
// response that has not come back.
func projectedIdentities(held expansion, authored sequenced) []resolvedIdentity {
	resolved := make([]resolvedIdentity, 0, len(held.Members))
	for _, member := range held.Members {
		if member.Identity == "" {
			continue
		}
		resolved = append(resolved, resolvedIdentity{
			member: member.Name,
			id:     store.Identity{Target: authored.Target, Definition: authored.Definition, Name: member.Identity},
		})
	}
	return resolved
}

// identitiesOf is the identities alone, which is what the Store's comparand is
// asked over.
func identitiesOf(resolved []resolvedIdentity) []store.Identity {
	ids := make([]store.Identity, len(resolved))
	for i, member := range resolved {
		ids[i] = member.id
	}
	return ids
}
