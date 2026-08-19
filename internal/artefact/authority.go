package artefact

import (
	"maps"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

// Authority is the supply behind §8's `AUTHORITY` table: the two namespaces the
// relation is read across, each keyed by the name its artefact declares for
// itself.
//
// It is one relation and not two tables. §5's authority rule is an
// intersection — a Definition claims Kinds and Targets, a Target declaration
// accepts Kinds — and an intersection privileges neither operand, so the
// artefact under review supplies one end of it and which end decides the filter
// and nothing else (ADR-0069). Three of the five artefacts supply an end and
// two are members of no pair at all.
//
// Both maps are the load's own folds, so the Definition a name means here and
// the Definition a Step's definition: resolves to are the same file rather than
// two walks agreeing (issue #109). A name absent from either is a name that
// resolved to nothing — which covers a file that is not there and one that is
// there and will not parse alike, the two differing in nothing this table can
// act on (ADR-0064).
type Authority struct {
	Definitions map[string]DefinitionFacts
	Targets     map[string]TargetFacts
}

// AuthorityTable is the relation on one artefact: whether it renders at all,
// whether its row set was discovered rather than authored, and the rows.
//
// The three travel together because a caller needs all three to render the
// block and no two of them are separable: a table that does not render has no
// rows and no discovery to have failed, and an empty table means one thing
// where an edit could produce a row and another where none could (§8).
type AuthorityTable struct {
	// Renders is false on the two artefacts that are members of no pair — a
	// Manifest, whose Operations declare the Kinds a Definition *may* claim
	// rather than claiming any, and a Repository declaration, which pairs
	// with nothing. On those the table is absent entire rather than empty.
	Renders bool
	// Discovered is true where the row set is discovered across
	// definitions/ rather than authored in the artefact under review, which
	// is the one filter a file that did not load can silently remove a row
	// from. It is why a discovery failure is counted beneath the table on
	// that artefact and nowhere else (ADR-0069).
	Discovered bool
	Rows       []AuthorityRow
}

// AuthorityRow is one pairing as the table states it: which Definition, which
// Target, what each claims and accepts, their intersection, and the `destroy`
// Operations the Definition names (§5, §8).
//
// The three Kind lists carry full names and never the page's initials. The
// initials are a notation this screen renders an intersection in and the names
// are the values, exactly as `envelope ✓` and `"envelope ok"` are one fact in
// two notations (§8, ADR-0026).
//
// DefinitionKinds carries `destroy` where the Definition's destroy: names any
// Operation, derived at that one position rather than read. §3 keeps `destroy`
// out of kinds: precisely so this column can derive it, granularity following
// severity (§8).
//
// A list is nil where its end of the pairing has no supply and non-nil
// otherwise, empty included — and that one encoding is the whole of what says
// so, a second boolean beside it being a fact stated twice. A Definition that
// claims no Kind at all claims none, where one that did not load claims nothing
// this table can read: the page renders the first as an em dash and the second
// as §8's `unresolved`, and a supply that resolved to nothing and one that
// resolved to something unreadable are the same absence here.
//
// Effective is the intersection, in the Definition's own claim order: §5's
// check reads a claim against a grant, so the claim is the operand the
// intersection is ordered by. It is nil where either end lacks a supply.
type AuthorityRow struct {
	Definition        string
	Target            string
	DefinitionKinds   []string
	TargetKinds       []string
	Effective         []string
	DestroyOperations []string
}

// Table is the relation on the artefact under review: the pairs its end of it
// supplies, each read across both namespaces, sorted.
//
// kind is §12's own kind: value, read off the load's path by the caller rather
// than off this file's key: a file whose directory and kind: disagree is a load
// error §12 already names, and the filter follows what the artefact is read as.
//
// The artefact under review supplies its own end from root, which stands in the
// namespace for the length of this rendering. The two agree in every repository
// a check passes on, and where two files declare one name they do not: what a
// review states is the file it was pointed at.
func (a Authority) Table(kind string, root *yaml.Node) AuthorityTable {
	switch kind {
	case KindDefinition:
		name := DeclaredName(root, KindDefinition)
		facts := ReadDefinitionFacts(root)
		supply := a.withDefinition(name, facts)
		return AuthorityTable{Renders: true, Rows: supply.rows(pairsWithTargets(name, facts.Targets))}
	case KindTargetDeclaration:
		name := TargetDeclarationName(root)
		supply := a.withTarget(name, ReadTargetFacts(root))
		return AuthorityTable{Renders: true, Discovered: true, Rows: supply.rows(supply.pairsClaiming(name))}
	case KindProcedure:
		return AuthorityTable{Renders: true, Rows: a.rows(boundPairs(root))}
	default:
		return AuthorityTable{}
	}
}

// withDefinition and withTarget are this supply with one name reading off the
// artefact under review's own bytes. They copy rather than write through: the
// namespaces are the load's, shared with every other reader of it, and what
// stands here is a reading for the length of one rendering.
func (a Authority) withDefinition(name string, facts DefinitionFacts) Authority {
	a.Definitions = withName(a.Definitions, name, facts)
	return a
}

func (a Authority) withTarget(name string, facts TargetFacts) Authority {
	a.Targets = withName(a.Targets, name, facts)
	return a
}

// withName is one namespace with one name answering to what was read here.
func withName[T any](namespace map[string]T, name string, facts T) map[string]T {
	copied := maps.Clone(namespace)
	if copied == nil {
		copied = map[string]T{}
	}
	copied[name] = facts
	return copied
}

// authorityPair is one row before either end is read: which Definition, which
// Target, both as some artefact wrote them.
type authorityPair struct{ definition, target string }

// pairsWithTargets is the left end's filter: one pair per Target a Definition
// claims, in the Definition's own order — which the sort below replaces and
// which the dedupe reads, a targets: list naming one Target twice claiming it
// once.
func pairsWithTargets(definition string, targets []string) []authorityPair {
	pairs := make([]authorityPair, 0, len(targets))
	for _, target := range targets {
		pairs = append(pairs, authorityPair{definition, target})
	}
	return pairs
}

// pairsClaiming is the right end's filter: one pair per Definition in the
// namespace whose targets: names this Target.
//
// The row set is discovered rather than authored, which is what makes this the
// rendering an unaided reading withholds and the one that earns the table most
// — the screen whose gutter marks what it grants and where nothing else says
// who took the grant (ADR-0069). It costs nothing at load: every artefact is
// already in memory, matched byte-exact on its own name: and never on whether
// an open succeeded (ADR-0064).
func (a Authority) pairsClaiming(target string) []authorityPair {
	if target == "" {
		// A declaration that names itself nothing is in no namespace, so
		// no targets: member can have resolved to it. Matching on "" would
		// pair it with every Definition whose own list is unreadable.
		return nil
	}
	var pairs []authorityPair
	for definition, facts := range a.Definitions {
		if slices.Contains(facts.Targets, target) {
			pairs = append(pairs, authorityPair{definition, target})
		}
	}
	return pairs
}

// boundPairs is the filter of the artefact that supplies neither end: one pair
// per Step that binds one, read off the Procedure's own steps:.
//
// It reads this Procedure's Steps and not the Steps of what it invokes. A
// nested invocation declares no definition: and binds no pairing of its own;
// what it reaches is the transitive envelope the gutter marks on that line
// (§3, §8).
//
// A Step naming only one of the two binds no pairing: the missing key is
// schema-mismatch and check's to report, and half a pairing has no row here to
// stand in (ADR-0064).
func boundPairs(root *yaml.Node) []authorityPair {
	stepsVal := topLevelFields(root, "steps")["steps"]
	if stepsVal == nil || stepsVal.Kind != yaml.SequenceNode {
		return nil
	}
	var pairs []authorityPair
	for _, entry := range stepsVal.Content {
		fields := topLevelFields(entry, "definition", "target")
		definition, named := resolveScalar(fields["definition"])
		if !named {
			continue
		}
		target, bound := resolveScalar(fields["target"])
		if !bound {
			continue
		}
		pairs = append(pairs, authorityPair{definition, target})
	}
	return pairs
}

// rows is the one renderer behind the three filters: each distinct pair read
// across both namespaces, sorted by (Target, Definition), each by Unicode code
// point.
//
// That is §7's identity-set ordering taken again rather than reinvented, so two
// renderings of one review are byte-identical, and it degenerates to Definition
// order where the Target column is constant. Step order is refused even where
// it exists: reading down the marker column *is* the step table, so a table
// beneath it in step order is a second copy of an ordering the reviewer already
// has, where a sorted one is a second index into the same rows (§8, ADR-0026).
func (a Authority) rows(pairs []authorityPair) []AuthorityRow {
	rows := make([]AuthorityRow, 0, len(pairs))
	seen := map[authorityPair]bool{}
	for _, pair := range pairs {
		if seen[pair] {
			continue
		}
		seen[pair] = true
		rows = append(rows, a.row(pair))
	}
	slices.SortFunc(rows, func(x, y AuthorityRow) int {
		if by := strings.Compare(x.Target, y.Target); by != 0 {
			return by
		}
		return strings.Compare(x.Definition, y.Definition)
	})
	return rows
}

// row is one pairing read across both ends. An end with no supply leaves its
// cells nil, which is what the page renders §8's one word for; the other end's
// are this artefact's own and render as they always do.
func (a Authority) row(pair authorityPair) AuthorityRow {
	row := AuthorityRow{Definition: pair.definition, Target: pair.target}
	if definition, claimed := a.Definitions[pair.definition]; claimed {
		row.DefinitionKinds = claimedKinds(definition)
		row.DestroyOperations = supplied(definition.Destroy)
	}
	if target, granted := a.Targets[pair.target]; granted {
		row.TargetKinds = supplied(target.Kinds)
	}
	if row.DefinitionKinds != nil && row.TargetKinds != nil {
		row.Effective = intersect(row.DefinitionKinds, row.TargetKinds)
	}
	return row
}

// supplied is a list read off a supply that is there: a copy of it, and the
// empty list where the key was absent. It is never nil, which is what separates
// a cell whose supply names nothing from one that has no supply at all — the
// distinction the whole table is built on (§7, §8).
func supplied(members []string) []string {
	copied := make([]string, 0, len(members))
	return append(copied, members...)
}

// claimedKinds is the Definition's claim as this column states it: its kinds:
// as written, with `destroy` appended where its destroy: names any Operation.
//
// The authored list is not reduced or re-sorted, which is ReadDefinitionFacts's
// own rule: what stands in this column is the claim the reviewer has open
// beside it. What is guarded is the derived member alone — §3 keeps `destroy`
// out of kinds:, so a Definition that writes it there has said something check
// names, and the column states the claim once either way.
func claimedKinds(facts DefinitionFacts) []string {
	claimed := make([]string, 0, len(facts.Kinds)+1)
	claimed = append(claimed, facts.Kinds...)
	if len(facts.Destroy) > 0 && !slices.Contains(claimed, "destroy") {
		claimed = append(claimed, "destroy")
	}
	return claimed
}

// intersect is what both ends name, in the claim's own order. It is what §5's
// two-key check admits, rendered: neither key alone reaches anything, and the
// column beneath is the Kinds an Operation bound here may actually carry.
//
// It is a set where the two columns beside it are enumerations, so a Kind
// claimed twice is admitted once: what the intersection states is what may be
// reached, and a member has no multiplicity there to state.
func intersect(claimed, accepted []string) []string {
	both := make([]string, 0, len(claimed))
	for _, kind := range claimed {
		if slices.Contains(accepted, kind) && !slices.Contains(both, kind) {
			both = append(both, kind)
		}
	}
	return both
}
