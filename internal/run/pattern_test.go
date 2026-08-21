// This file tests issue #143: the three Patterns as `hyper` performs them —
// what makes one call happen after another, and what stops the sequence (§3,
// §6, §12, ADR-0018).
//
// It is a table rather than a corpus case, on the same exception
// `predicate_test.go` argues: `internal/run` is driven through `cli.Main`
// because its interface is a Run, and what stands here is not an arrangement
// but the **sequence** the corpus is unable to hold. A branch and a page record
// that twelve pages were fetched; they cannot record that the twelfth was asked
// for only after the eleventh had answered, or that a fourth attempt was never
// made.
//
// Nothing here reads a clock. That is the fence run_test.go holds over this
// package's source, and it is not worked around: the one claim that needs one —
// that a polling Pattern's interval falls **between** its calls and never before
// the first — is asserted end to end from internal/cli, where the elapsed time
// of a whole Run is a thing a test may read.
//
// What the corpus does hold is everything downstream of that: the Records every
// page projected, the account the Step file carries, and the halt an `until:`
// that cannot compare produces (testdata/run/README.md).
package run

import (
	"context"
	"testing"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// collection is a response carrying one page of a collection: the members it
// holds and, where the case wants a `cursor:` walk to continue, a next token.
func collection(members int, next string) capability.Object {
	held := make([]any, 0, members)
	for i := range members {
		held = append(held, map[string]any{"id": i})
	}
	body := map[string]any{"records": held}
	if next != "" {
		body["next"] = next
	}
	return capability.Object{
		{Name: capability.MemberHost, Value: "pages.hyper.dev"},
		{Name: capability.MemberStatus, Value: 200},
		{Name: capability.MemberBody, Value: body},
	}
}

// declaring reads one Operation's `patterns:` block out of its source, which is
// what readPatterns is handed — the node artefact.OperationNode finds.
func declaring(t *testing.T, source string) patterns {
	t.Helper()

	var root yaml.Node
	if err := yaml.Unmarshal([]byte(source), &root); err != nil {
		t.Fatal(err)
	}
	return readPatterns(root.Content[0])
}

// keeping is a read that answers a fixed number of members per page, walking
// the counts the case wrote down and repeating the last.
func keeping(counts ...int) (read, *int) {
	pages := 0
	return func(capability.Object) (int, error) {
		at := min(pages, len(counts)-1)
		pages++
		return counts[at], nil
	}, &pages
}

// TestReadPatterns_TheClosedThreeAndTheirParameters is §12's set read off a
// Manifest: both of pagination's forms, both of `into:`'s positions, and the
// two Patterns beside it.
func TestReadPatterns_TheClosedThreeAndTheirParameters(t *testing.T) {
	read := declaring(t, `
kind: read
patterns:
  pagination:
    cursor: {from: $.body.next, into: {query: cursor}}
  polling:
    interval: 5s
    until:
      - field: body.state
        equals: running
  retry: {attempts: 3}
`)
	if !read.pagination.declared || read.pagination.cursor != "$.body.next" {
		t.Errorf("pagination = %#v, want the cursor form reading $.body.next", read.pagination)
	}
	if read.pagination.into != (into{name: "cursor"}) {
		t.Errorf("into = %#v, want the query position named cursor", read.pagination.into)
	}
	if !read.polling.declared || read.polling.interval != 5*time.Second || len(read.polling.until) != 1 {
		t.Errorf("polling = %#v, want a 5s interval and one predicate", read.polling)
	}
	if read.retry != (retry{declared: true, attempts: 3}) {
		t.Errorf("retry = %#v, want three attempts", read.retry)
	}

	numbered := declaring(t, "kind: read\npatterns:\n  pagination:\n    page: {from: 1, into: {header: x-page}}\n")
	if numbered.pagination.cursor != "" || numbered.pagination.page != 1 {
		t.Errorf("pagination = %#v, want the page form starting at 1", numbered.pagination)
	}
	if numbered.pagination.into != (into{header: true, name: "x-page"}) {
		t.Errorf("into = %#v, want the header position named x-page", numbered.pagination.into)
	}

	none := declaring(t, "kind: read\n")
	if none.pagination.declared || none.polling.declared || none.retry.declared {
		t.Errorf("an Operation declaring no patterns: reads %#v, want none of the three", none)
	}
}

// TestPage_TheTokenIsWrittenIntoTheDeclaredPosition is what a pagination
// Pattern puts on the wire: the position `into:` names, replacing what the
// Manifest's own request wrote there rather than standing a second entry beside
// it.
func TestPage_TheTokenIsWrittenIntoTheDeclaredPosition(t *testing.T) {
	built := capability.Call{
		Host:    "pages.hyper.dev",
		Query:   []capability.Parameter{{Name: "limit", Value: "50"}, {Name: "cursor", Value: "start"}},
		Headers: []capability.Parameter{{Name: "accept", Value: "application/json"}},
	}

	walked := page{into: into{name: "cursor"}, value: "c2", written: true}.write(built)
	if got := walked.URL(); got != "https://pages.hyper.dev?limit=50&cursor=c2" {
		t.Errorf("the paged call reaches %s, want the cursor written over the one the request declared", got)
	}
	// The call is built once and paged many times, so a decoration that
	// wrote in place would have page three carrying page two's token.
	if built.Query[1].Value != "start" {
		t.Errorf("the built call's cursor moved to %q; a page decorates a copy", built.Query[1].Value)
	}

	headed := page{into: into{header: true, name: "x-page"}, value: "2", written: true}.write(built)
	if len(headed.Headers) != 2 || headed.Headers[1] != (capability.Parameter{Name: "x-page", Value: "2"}) {
		t.Errorf("the paged call's headers are %#v, want x-page appended", headed.Headers)
	}

	// The zero value is the first call of every Operation, and it decorates
	// nothing at all.
	if first := (page{}).write(built); first.URL() != built.URL() {
		t.Errorf("the first call reaches %s, want the request as it was built", first.URL())
	}
}

// TestPagination_PageNPlusOneIsNotAskedForBeforeNHasAnswered is the serialism,
// and it is constructional rather than declared: the token for the next call is
// read off the answer to this one, so there is nowhere for a second request to
// come from.
//
// It is asserted by the calls themselves — each records whether another was
// standing when it began — because *how many were in flight* is the whole claim
// and nothing a branch records can hold it.
func TestPagination_PageNPlusOneIsNotAskedForBeforeNHasAnswered(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  pagination:\n    cursor: {from: $.body.next, into: {query: cursor}}\n")

	inFlight, overlapped, asked := 0, false, []string{}
	call := func(_ context.Context, at page) (capability.Object, bool, error) {
		inFlight++
		overlapped = overlapped || inFlight > 1
		asked = append(asked, at.value)
		defer func() { inFlight-- }()
		switch len(asked) {
		case 1:
			return collection(2, "c2"), false, nil
		case 2:
			return collection(2, "c3"), false, nil
		default:
			return collection(2, ""), false, nil
		}
	}

	keep, pages := keeping(2, 2, 2)
	acted, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if overlapped {
		t.Error("two of a member's pages were in flight together")
	}
	if want := []string{"", "c2", "c3"}; len(asked) != 3 || asked[0] != want[0] || asked[1] != want[1] || asked[2] != want[2] {
		t.Errorf("the pages asked for were %v, want %v — each token read off the page before it", asked, want)
	}
	if *pages != 3 || acted.pages != 3 {
		t.Errorf("%d pages were read and %d accounted for, want 3 of each", *pages, acted.pages)
	}
}

// TestPagination_BothFormsStopWhenTheCollectionComesBackEmpty is §3's shared
// termination, and the `page:` form's only one: an integer `hyper` increments
// has no token to stop resolving.
func TestPagination_BothFormsStopWhenTheCollectionComesBackEmpty(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  pagination:\n    page: {from: 1, into: {query: page}}\n")

	var asked []string
	call := func(_ context.Context, at page) (capability.Object, bool, error) {
		asked = append(asked, at.value)
		return collection(1, ""), false, nil
	}

	keep, _ := keeping(1, 1, 0)
	acted, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if want := []string{"1", "2", "3"}; len(asked) != 3 || asked[2] != want[2] {
		t.Errorf("the pages asked for were %v, want %v — the declared starting value, incremented", asked, want)
	}
	if acted.pages != 3 {
		t.Errorf("pages = %d, want 3 — the empty one was fetched and is part of the account", acted.pages)
	}
}

// TestPagination_ACursorThatStopsResolvingEndsTheWalk is the `cursor:` form's
// second terminator: a page that handed no token back is the last one, and
// asking again with the token that reached this page would be repeating the
// call just made.
func TestPagination_ACursorThatStopsResolvingEndsTheWalk(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  pagination:\n    cursor: {from: $.body.next, into: {query: cursor}}\n")

	calls := 0
	call := func(context.Context, page) (capability.Object, bool, error) {
		calls++
		return collection(2, ""), false, nil
	}

	keep, _ := keeping(2)
	acted, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if calls != 1 {
		t.Errorf("%d calls were made, want one — the first page carried no next token", calls)
	}
	// One page fetched is the trivial single call, so the account says
	// nothing: a walk of one and no pagination at all are the same silence
	// on the page (§7).
	if acted != (account{}) {
		t.Errorf("a walk of one page accounts for %#v, want silence", acted)
	}
}

// TestPolling_AValueItCannotCompareHalts is ADR-0035 at the third root. Two of
// §12's three predicate roots contribute a Refusal and this one contributes a
// halt: it reads a response after the call went out, so there is none available.
//
// The halt names the field and what was found in it, and carries no
// `error_code` — which is a fact about the value this answers rather than about
// its wording: an error is narration's, and §12's codes are a check's.
func TestPolling_AValueItCannotCompareHalts(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  polling:\n    interval: 1s\n    until:\n      - field: body.state\n        equals: ready\n")

	calls := 0
	call := func(context.Context, page) (capability.Object, bool, error) {
		calls++
		return capability.Object{
			{Name: capability.MemberBody, Value: map[string]any{"state": 7}},
		}, false, nil
	}

	keep, _ := keeping(1)
	_, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err == nil {
		t.Fatal("an until: handed a number where its operator takes one did not halt")
	}
	if want := "the polling Pattern's until: field: body.state holds the number 7, and equals: takes a number"; err.Error() != want {
		t.Errorf("the halt reads %q, want %q", err, want)
	}
	if calls != 1 {
		t.Errorf("%d calls were made, want one — nothing is asked again once the answer cannot be read", calls)
	}
}

// TestRetry_TheAttemptsAreTheDeclaredCeiling is ADR-0018's Pattern: a failure
// that provably preceded the request is followed, and never more times than the
// Manifest declared.
func TestRetry_TheAttemptsAreTheDeclaredCeiling(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  retry: {attempts: 3}\n")

	calls := 0
	call := func(context.Context, page) (capability.Object, bool, error) {
		calls++
		// The object a call that got no answer carries: host and
		// nothing else, which is what an exhausted retry leaves for the
		// projection to read (§12, ADR-0050).
		return capability.Object{{Name: capability.MemberHost, Value: "gone.hyper.dev"}}, true, nil
	}

	keep, pages := keeping(1)
	acted, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("an exhausted retry answered a fault: %v — it leaves the response object for the projection to read", err)
	}
	if calls != 3 || acted.attempts != 3 {
		t.Errorf("%d calls were made and %d attempts accounted for, want three of each", calls, acted.attempts)
	}
	if *pages != 1 {
		t.Errorf("%d responses were projected, want one — a retry does not change the number of Records a Step affects", *pages)
	}
}

// TestRetry_AnAnswerIsNeverFollowed is the other half of ADR-0018 read from
// this side: a call that answered is not made again, whatever the answer was.
// Which failures are the class at all is internal/capability's
// (capability.NeverSent).
func TestRetry_AnAnswerIsNeverFollowed(t *testing.T) {
	declared := declaring(t, "kind: read\npatterns:\n  retry: {attempts: 3}\n")

	calls := 0
	call := func(context.Context, page) (capability.Object, bool, error) {
		calls++
		return capability.Object{
			{Name: capability.MemberHost, Value: "busy.hyper.dev"},
			{Name: capability.MemberStatus, Value: 503},
		}, false, nil
	}

	keep, _ := keeping(1)
	acted, err := declared.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if calls != 1 {
		t.Errorf("%d calls were made against a host that answered 503, want one — no status is ever retried", calls)
	}
	// One attempt and no retry declared are the same silence, so a Pattern
	// that was declared and never followed anything writes nothing (§7).
	if acted != (account{}) {
		t.Errorf("one attempt accounts for %#v, want silence", acted)
	}
}

// TestAccount_TheTrivialSingleCallIsSilent is §7's rule for the Disposition's
// third thing: it is written where a Pattern did **more than** the trivial
// single call and absent otherwise, so that *one attempt* and *no retry
// declared* stay the same silence on the page.
//
// It is asserted through perform rather than over a helper, because the rule's
// whole content is **where** each counter is reduced: a retry's per request, a
// poll's per page, a page's per member. An Operation that is paginated as well
// as retried and never retries anything is the case that tells them apart —
// three calls, three single attempts, and one Pattern that must not inflate the
// other's number.
func TestAccount_TheTrivialSingleCallIsSilent(t *testing.T) {
	both := declaring(t, "kind: read\npatterns:\n  pagination:\n    page: {from: 1, into: {query: page}}\n  retry: {attempts: 3}\n")

	calls := 0
	call := func(context.Context, page) (capability.Object, bool, error) {
		calls++
		return collection(1, ""), false, nil
	}
	keep, _ := keeping(1, 1, 0)
	acted, err := both.perform(t.Context(), time.Time{}, call, keep)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if calls != 3 {
		t.Fatalf("%d calls were made, want three — three pages, none of them retried", calls)
	}
	if acted != (account{pages: 3}) {
		t.Errorf("three unretried pages account for %#v, want three pages and **no** attempts — three single attempts are the same silence three times", acted)
	}

	// One Pattern doing nothing at all, which is the other end of the rule:
	// a single page under a single attempt says nothing on either counter.
	single, _ := keeping(0)
	quiet, err := both.perform(t.Context(), time.Time{}, call, single)
	if err != nil {
		t.Fatalf("perform err = %v", err)
	}
	if quiet != (account{}) {
		t.Errorf("one page under one attempt accounts for %#v, want silence", quiet)
	}
}

// TestAccount_AnExpansionsMembersSum is what a Step's file carries: the work
// its Patterns did across the whole Expansion, each member's own account
// already reduced (§7).
func TestAccount_AnExpansionsMembersSum(t *testing.T) {
	var step account
	for range 3 {
		step.add(account{pages: 3})
	}
	if step != (account{pages: 9}) {
		t.Errorf("three members walking three pages each account for %#v, want nine pages", step)
	}
	if written := step.written(); written.Pages != 9 || written.Attempts != 0 || written.Polls != 0 {
		t.Errorf("the Step file carries %#v, want nine pages and nothing else", written)
	}
}
