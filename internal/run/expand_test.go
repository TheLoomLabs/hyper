package run

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// **Four checks decide at a Step's Expansion and they have an order, and it is
// the causal one** (§5, §6, §12, issue #139).
//
// A predicate resolves the set; the set has the count a Bound is read against;
// the members' arguments fill the inputs an identity is projected from; and the
// identities are projected off the members the set holds, against each other
// and against the Store. Where two are available at once the earlier in that
// sequence is the one named, and the sibling collision is named before the
// Store's — being reproducible from the artefact alone, and therefore pointing
// at an edit with no Store in hand.
//
// This is the exception the milestone's testing note names: `internal/run` is
// driven through `cli.Main` because its interface is a Run, and the order these
// checks decide in is a pure function of a resolved set. What the corpus holds
// beside it is each check declining on its own — and the two that can be
// available at once, in [the-sibling-collision-is-named-first].

// found is one Refusal in the shape only its code is read out of, since what
// this table is about is which check answered and not what it said.
func found(code string) []Refusal {
	return []Refusal{{RefusalMember: store.RefusalMember{ErrorCode: code}}}
}

// TestChecks_DeclineInTheCausalOrder walks the order cell by cell: which checks
// found something, and which one the Refusal names.
func TestChecks_DeclineInTheCausalOrder(t *testing.T) {
	predicate := found(CodePredicateTypeMismatch)
	bound := found(CodeBoundExceeded)
	arguments := found(schema.CodeMismatch)
	collision := found(CodeRecordIdentityCollision)

	for name, c := range map[string]struct {
		checks checks
		names  string
	}{
		"nothing declined": {
			checks{}, "",
		},
		"a predicate that cannot compare, alone": {
			checks{Predicate: predicate}, CodePredicateTypeMismatch,
		},
		"a predicate before the count the Bound is read against": {
			checks{Predicate: predicate, Bound: bound}, CodePredicateTypeMismatch,
		},
		"a predicate before the arguments the identity is projected from": {
			checks{Predicate: predicate, Arguments: arguments}, CodePredicateTypeMismatch,
		},
		"a predicate before either identity comparand": {
			checks{Predicate: predicate, Sibling: collision, Stored: collision}, CodePredicateTypeMismatch,
		},
		"the count before the identities projected off the set": {
			checks{Bound: bound, Sibling: collision, Stored: collision}, CodeBoundExceeded,
		},
		"an argument before the identity it fills": {
			checks{Arguments: arguments, Sibling: collision, Stored: collision}, schema.CodeMismatch,
		},
		"the sibling collision before the Store's": {
			checks{Sibling: collision, Stored: collision}, CodeRecordIdentityCollision,
		},
		"the Store's comparand alone": {
			checks{Stored: collision}, CodeRecordIdentityCollision,
		},
	} {
		t.Run(name, func(t *testing.T) {
			declined := c.checks.declined()
			if c.names == "" {
				if len(declined) != 0 {
					t.Fatalf("declined %v, want nothing", declined)
				}
				return
			}
			// **A Refusal holds exactly one member** at this
			// moment, which is what having an order is for (§7).
			if len(declined) != 1 {
				t.Fatalf("declined %d members, want exactly 1: %v", len(declined), declined)
			}
			if declined[0].ErrorCode != c.names {
				t.Errorf("declined %s, want %s", declined[0].ErrorCode, c.names)
			}
		})
	}
}

// TestChecks_TheSiblingIsNamedBeforeTheStoresOverTheSameSet holds the one pair
// the corpus can drive both halves of at once, over the values a resolution
// would build: two members that are one identity, one of which is also one the
// Store already holds. The sibling is named, and the Store's comparand — which
// is just as true — is not.
func TestChecks_TheSiblingIsNamedBeforeTheStoresOverTheSameSet(t *testing.T) {
	sibling := Refusal{RefusalMember: store.RefusalMember{
		ErrorCode: CodeRecordIdentityCollision,
		Message:   "alpha resolves to status.hyper.dev and beta resolves to Status.hyper.dev",
	}}
	stored := Refusal{RefusalMember: store.RefusalMember{
		ErrorCode: CodeRecordIdentityCollision,
		Message:   "alpha resolves to status.hyper.dev, and the Store already holds STATUS.HYPER.DEV",
	}}

	declined := checks{Sibling: []Refusal{sibling}, Stored: []Refusal{stored}}.declined()
	if len(declined) != 1 || declined[0].Message != sibling.Message {
		t.Errorf("declined %v, want the sibling alone — it is reproducible from the artefact with no Store in hand", declined)
	}
}

// **The Bound counts what the Expansion resolved and nothing else** (§5, §6,
// §7, issue #149).
//
// It is the run-time half of the check `check` already performs where the count
// is authored, and the two are one code. What the table below is about is the
// count alone: which numbers decline, which do not, and what the Refusal says
// it compared.

// bounded is one Step as this check reads it: an authored `bound:` at a line,
// and a selector that resolved to a count.
func bounded(declared string, form string, members int) (sequenced, expansion) {
	authored := sequenced{Step: artefact.Step{
		ID: "refresh", Definition: "preview-dns", Operation: "create_dns_record",
		Target: "cloudflare-prod", Line: 5,
	}}
	if declared != "" {
		authored.Bound = &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: declared, Line: 9}
	}
	held := expansion{Selector: selector{Form: form, Line: 10}}
	for i := range members {
		held.Members = append(held.Members, member{Name: fmt.Sprintf("preview-%d.example.com", 40+i)})
	}
	return authored, held
}

// TestExceededBound_CountsTheExpansionAgainstTheAuthoredMaximum walks the
// counts: the Bound that fits, the one that is met exactly, the one that is
// past, and the Step that declared none at all.
func TestExceededBound_CountsTheExpansionAgainstTheAuthoredMaximum(t *testing.T) {
	for name, c := range map[string]struct {
		declared string
		form     string
		members  int
		refuses  bool
	}{
		"a Step declaring no Bound counts nothing":   {"", "assets", 23, false},
		"a read Step, which carries no Bound at all": {"", "observations", 3, false},
		"an Expansion inside its Bound":              {"5", "assets", 3, false},
		"an Expansion that meets its Bound exactly":  {"3", "assets", 3, false},
		"an Expansion one past its Bound":            {"2", "assets", 3, true},
		"an Expansion far past its Bound":            {"2", "assets", 23, true},
		"an Expansion that resolved to nothing":      {"2", "assets", 0, false},
		"a set of one against a Bound of nought":     {"0", "", 1, true},
	} {
		t.Run(name, func(t *testing.T) {
			authored, held := bounded(c.declared, c.form, c.members)
			declined := run{}.exceededBound(held, authored, run{}.citation(authored, 3, held.Selector))
			if c.refuses != (len(declined) > 0) {
				t.Fatalf("declined %v, want refuses = %v", declined, c.refuses)
			}
			if !c.refuses {
				return
			}
			// **A Refusal holds exactly one member**: the count is
			// one check and it answered once (§7).
			if len(declined) != 1 {
				t.Fatalf("declined %d members, want exactly 1: %v", len(declined), declined)
			}
			if declined[0].ErrorCode != CodeBoundExceeded {
				t.Errorf("declined %s, want %s", declined[0].ErrorCode, CodeBoundExceeded)
			}
		})
	}
}

// TestExceededBound_CarriesTheTwoValuesItCompared holds §7's own shape: the
// authored Bound under `declared`, the Expansion's count under `observed`, and
// the citation pointing at the `bound:` line — which is the edit that widens it.
func TestExceededBound_CarriesTheTwoValuesItCompared(t *testing.T) {
	authored, held := bounded("5", "assets", 23)
	declined := run{}.exceededBound(held, authored, run{}.citation(authored, 3, held.Selector))
	if len(declined) != 1 {
		t.Fatalf("declined %d members, want exactly 1", len(declined))
	}

	refused := declined[0]
	if refused.Declared != store.Int(5) || refused.Observed != store.Int(23) {
		t.Errorf("compared declared %v against observed %v, want 5 against 23", refused.Declared, refused.Observed)
	}
	if refused.Line != 9 || refused.Field != "steps[0].bound" {
		t.Errorf("cites line %d field %q, want the bound: line and steps[0].bound", refused.Line, refused.Field)
	}
	if want := "expansion resolved 23 assets on cloudflare-prod"; refused.Message != want {
		t.Errorf("message = %q, want %q", refused.Message, want)
	}
	if refused.Step != 3 || refused.StepID != "refresh" {
		t.Errorf("cites step %d %q, want the Step's position in the Run and its authored id", refused.Step, refused.StepID)
	}
}

// TestExceededBound_NamesASetOfOneWhereThereIsNoSelectorToName holds the one
// count that has no authored key behind it: a Step carrying no `over:` resolves
// no selector and holds a set of one, which a Bound of nought is past (§6).
func TestExceededBound_NamesASetOfOneWhereThereIsNoSelectorToName(t *testing.T) {
	authored, held := bounded("0", "", 1)
	declined := run{}.exceededBound(held, authored, run{}.citation(authored, 1, held.Selector))
	if len(declined) != 1 {
		t.Fatalf("declined %d members, want exactly 1", len(declined))
	}
	if want := "expansion resolved 1 record on cloudflare-prod"; declined[0].Message != want {
		t.Errorf("message = %q, want %q", declined[0].Message, want)
	}
}

// TestCodeBoundExceeded_IsOneCodeAtBothSites holds the string this package
// spells against the one internal/artefact spells for the offline half.
//
// The two constants are deliberately separate — each names the check its own
// section states, and neither imports the other's word for it — so what makes
// them one code is that they are one string. Nothing else compares them, and a
// drift would otherwise show up only as a golden file that moved.
func TestCodeBoundExceeded_IsOneCodeAtBothSites(t *testing.T) {
	if CodeBoundExceeded != artefact.CodeBoundExceeded {
		t.Errorf("§6 spells %q and §4 spells %q; what names a Refusal is the check that declined, and it is one check",
			CodeBoundExceeded, artefact.CodeBoundExceeded)
	}
}

// **A `destroy` declares no identity, and its literal resolves one all the
// same** (§3, §7, ADR-0033, ADR-0037, issue #151).
//
// Which name a member's Record will be held under is what decides whether the
// two identity comparands can run before the first call, and a `destroy`
// carries no `record:` block for one to be read out of. What a `values:` member
// resolves instead is the literal an author wrote — a name known before the
// call like any other, and therefore compared like one, with nothing touched.
//
// **And it stops there**, which is the row below that answers nothing: an
// `assets:` member's name came out of the Store, so it opens no series and
// cannot be spelled wrong, and the comparand it would meet answers the other
// spelling wherever the branch already holds both — a wall no edit could clear
// (ADR-0033, ADR-0075).
//
// It is a unit test for this file's own reason: the answer is a pure function
// of an Operation's Kind, its `identity:` and the selector the member came out
// of, where the corpus would need one repository per Manifest to say the same
// five things.

// identityCase is one Operation's `identity:` met by one member of an
// Expansion: what the Manifest declared, which of §12's forms resolved the
// member, what it resolved to, and the name the member's Record is held under
// before the call — "" where there is none until the answer comes back.
type identityCase struct {
	kind, identity, form, member, holds string
}

func TestIdentityBeforeTheCall_ResolvesWhatIsKnownWithoutAResponse(t *testing.T) {
	// The inputs the Expansion resolved, which is what a template hole
	// fills from. A `destroy` reads none of them: it fills no hole, having
	// no `identity:` to write one in.
	filler, read := schema.ReadScalar(schema.String, "preview-42.example.com")
	if !read {
		t.Fatal("the fixture's own input does not read as a string")
	}
	inputs := map[string]schema.Scalar{"name": filler}

	for named, c := range map[string]identityCase{
		// The literal **is** the Record name, with no branch on whether
		// a series is already there (ADR-0033).
		"a destroy over a values: literal": {
			"destroy", "", "values", "/srv/app/releases/r41", "/srv/app/releases/r41",
		},
		// And over a series it resolves nothing: that name is the
		// Store's own and opens nothing, so what a comparand could find
		// there is a collision the branch already holds.
		"a destroy over an assets: selector": {
			"destroy", "", "assets", "preview-42.example.com", "",
		},
		// A destroy carrying no selector has no member to name, which
		// is §5's *no series to write a Tombstone under*.
		"a destroy carrying no selector": {
			"destroy", "", "", "", "",
		},
		// The ordinary pre-call identity: a hole filled from the inputs
		// the Expansion just resolved.
		"a mutate whose identity: is a template hole": {
			"mutate", "{name}", "values", "preview-42.example.com", "preview-42.example.com",
		},
		// A `$`-rooted path names a value that exists only once the call
		// has gone out, so nothing resolves here and what reaches a
		// collision is §6's halt (ADR-0072).
		"a mutate whose identity: reads from the response": {
			"mutate", "$.body.result.name", "assets", "preview-42.example.com", "",
		},
	} {
		t.Run(named, func(t *testing.T) {
			bound := binding{operation: artefact.OperationInfo{Kind: c.kind, Identity: c.identity}}
			resolving := member{Name: c.member, Item: store.String(c.member)}

			held, err := identityBeforeTheCall(bound, selector{Form: c.form}, resolving, inputs, sequenced{})
			if err != nil {
				t.Fatalf("identityBeforeTheCall: %v", err)
			}
			if held != c.holds {
				t.Errorf("resolved %q, want %q", held, c.holds)
			}
		})
	}
}
