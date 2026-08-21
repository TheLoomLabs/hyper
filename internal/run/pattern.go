package run

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// The three behaviours `hyper` performs around a call, which a Manifest
// parameterises and does not implement (§3, §6, §12, issue #143).
//
// **The set is closed at three** — pagination, polling to a terminal condition,
// and retry — and each is a key under an Operation's `patterns:` mapping. **A
// Pattern may never change an Operation's Kind or the number of Records it
// affects**, which is what keeps *how many times the world was touched* a fact
// about the artefact rather than about a release.
//
// **All three are serial, and by construction rather than by a rule imposed on
// them.** Pagination's two forms both terminate when the collection comes back
// empty, so there is no page after this one until this one has answered;
// polling waits an `interval:` between calls; and a retry follows a failure. No
// Pattern has any concurrency to govern, and an Operation's `concurrency:`
// limit reaches none of them (drain.go, ADR-0045). That the serialism is
// constructional is what makes it durable: a policy could be widened by a
// release and touch the world more times than the artefact says with nothing
// appearing in a diff, where a construction has nothing to widen.
//
// **They compose one inside the other**, innermost first: a retry follows one
// request's failure, a poll repeats that request until its `until:` holds, and
// a pagination walks the collection one such answer at a time. There is no
// fourth order to choose between — each learns whether there is another call to
// make only from the answer to the one before it, which is the same sentence
// that makes all three serial.
//
// **Nothing here judges an answer.** Every call is judged where every other
// call is (step.go, §6): there is no final call a rule could privilege without
// inventing one, and a Pattern may not change what an Operation does. What this
// file decides is only whether there is another call to make.
//
// The static half is `check`'s and is already landed: the closed three-member
// set, pagination's exactly-one form, `into:`'s exactly-one position, and
// `pagination` outside an `over:` as `manifest-inconsistent` (§4,
// internal/artefact/manifest.go). This reads a block that has been through it.

// patterns is an Operation's `patterns:` block read: which of §12's three it
// declares, and what each one was parameterised with.
//
// An Operation declaring none gets the zero value, under which perform makes
// exactly one call — which is what an Operation with no Patterns is, rather
// than a path around them.
type patterns struct {
	pagination pagination
	polling    polling
	retry      retry
}

// pagination is the `pagination:` Pattern: one of two forms and no others, and
// the request position the next call's token or number is written into.
//
// **A next-page URL read from a `Link` header or a response field is not a form
// and never will be.** Reach arriving from data is what ADR-0024 closed, and a
// URL a response hands back is exactly that: the population may come from data
// and the reach only from an artefact. An API paginated that way can be called
// and cannot be paged (§13).
type pagination struct {
	// declared says the Operation carries the Pattern at all. The two forms
	// are told apart below; this is what tells *no pagination* from either
	// of them, `page: {from: 0}` being a starting value an author may write.
	declared bool
	// cursor is the response path the next call's token is read from, and
	// "" on the `page:` form. Pagination terminates on it as well as on the
	// empty collection: a token that stopped resolving is a page there is
	// nothing to ask for.
	cursor string
	// page is the starting value of the `page:` form, written on the
	// **first** call and incremented from there — an integer `hyper`
	// increments rather than one a response hands back.
	page int
	// into is the position the token or the number is written into.
	into into
}

// into is `into:`, the single-key mapping naming the position a pagination
// Pattern writes into: `query:` or `header:`, and the name within it. It is a
// mapping rather than two flat keys so that exactly one of a closed two-member
// choice is writable (§3).
type into struct {
	header bool
	name   string
}

// polling is the `polling:` Pattern: the interval waited between calls, and the
// predicate list the calls stop on.
//
// It carries **no attempt count and no timeout of its own**: the Operation's
// `deadline:` bounds the whole call, polls included, and a second declared
// limit on the same clock can only ever disagree with the first (§3, §6).
type polling struct {
	declared bool
	interval time.Duration
	// until is §12's operator set rooted at its third scope — the response
	// object in hand, which is the same root a projection reads from. A
	// `field:` here is a path in the grammar written without the root
	// marker, a response having paths and no declared names (§12).
	until []predicate
}

// retry is the `retry:` Pattern, and it carries `attempts:` and nothing else.
//
// The failure class is fixed and **provably pre-send** (ADR-0018), which is
// internal/capability's to establish and never this file's to guess at
// (capability.NeverSent). **No status is ever retried**, and neither is a
// connect timeout.
//
// There is no backoff to declare. Backoff after a DNS failure is not a fact
// about the API a Provider author describes, it is `hyper`'s — like the account
// of the attempts the Disposition carries (§7).
type retry struct {
	declared bool
	attempts int
}

// account is `hyper`'s own account of the work, supplied by no Provider
// (ADR-0018): what one member's Patterns did to reach the answer that was
// recorded.
//
// It is what makes the same Disposition after five attempts a different fact on
// the page from the same Disposition after a single call (§7). A member whose
// Patterns did no more than the single call every Operation makes contributes
// nothing to it, which is what beyondTheTrivialCall is for.
type account struct{ attempts, pages, polls int }

// add sums one member's account into a Step's. A Step holds one account and an
// Expansion holds many members, so what the Step's file carries is the total
// work its Patterns did — five pages across two members being five pages
// fetched, which is what *what hyper did to reach that outcome* means (§7).
func (a *account) add(member account) {
	a.attempts += member.attempts
	a.pages += member.pages
	a.polls += member.polls
}

// **Two of the three count what came back, and the third counts what went
// out.** A page is a page of results and a poll iteration is an observation of
// state, so a call that answered nothing produced neither; an attempt is the
// one counter §7 ties to how many times `hyper` may have touched the world, so
// a call that went out and was cut off by the deadline counts (ADR-0018).
//
// Each is reduced where its own *single call* is defined — a retry's per
// request, a poll's per page, a page's per member — so that an Operation
// declaring two Patterns cannot have one of them inflate the other's count.
// written is the account as the Step's file carries it (§7). It is a method
// rather than three assignments at the writer because the two shapes are one
// fact under two encodings, and a caller unpacking it field by field is where
// the day comes that a fourth counter reaches one of them and not the other.
func (a account) written() store.Pattern {
	return store.Pattern{Attempts: a.attempts, Pages: a.pages, Polls: a.polls}
}

// beyondOne is a count that says something, and zero where it says only that
// one call was made — the whole of §7's *written where a Pattern did more than
// the trivial single call, and absent otherwise*, stated once and applied by
// each counter to its own definition of one call.
func beyondOne(count int) int {
	if count <= 1 {
		return 0
	}
	return count
}

// page is the position and value a pagination Pattern writes into one call, and
// the **zero value is the first call of every Operation** — the one an
// Operation declaring no pagination makes, and the first of a `cursor:` walk,
// which has no token until a response has handed one back.
type page struct {
	into    into
	value   string
	number  int
	written bool
}

// write decorates one call with this page's position, and answers the call
// unchanged where there is nothing to write.
//
// It replaces a parameter of the same name rather than adding a second, and
// appends where the request declares none: a Manifest may write the position's
// name in its own `query:` as a starting value, and two entries under one name
// would put a request on the wire that neither the artefact nor the Pattern
// describes. The name is matched byte-exact, as every name in the tool is (§7).
//
// The call is copied down to the slice it writes into. A Call is built once per
// member and paged many times, so a decoration that appended in place would
// have page two carrying page one's parameter beside its own.
func (at page) write(call capability.Call) capability.Call {
	if !at.written {
		return call
	}
	if at.into.header {
		call.Headers = replacing(call.Headers, at.into.name, at.value)
		return call
	}
	call.Query = replacing(call.Query, at.into.name, at.value)
	return call
}

// replacing is the parameter list with one name's value set, in place where the
// name was already there and appended where it was not — so the order a
// Manifest authored survives paging (§3).
func replacing(parameters []capability.Parameter, name, value string) []capability.Parameter {
	written := slices.Clone(parameters)
	for at, parameter := range written {
		if parameter.Name == name {
			written[at].Value = value
			return written
		}
	}
	return append(written, capability.Parameter{Name: name, Value: value})
}

// first is the page the Operation's first call carries.
//
// The two forms differ here and nowhere else: a `page:` walk writes its
// declared starting value on the **first** call, there being an integer to
// write before anything has answered, and a `cursor:` walk writes nothing until
// a response has handed a token back.
func (p pagination) first() page {
	if !p.declared || p.cursor != "" {
		return page{}
	}
	return page{into: p.into, value: strconv.Itoa(p.page), number: p.page, written: true}
}

// next is the page after this one, and false where there is none to ask for.
//
// A `page:` walk always has one — it terminates on the empty collection alone,
// which the caller decides. A `cursor:` walk terminates here as well: a token
// whose path stopped resolving, or one that came back null, is a page there is
// nothing to ask for, and asking for it without one would be repeating the call
// just made.
func (p pagination) next(at page, response capability.Object) (page, bool) {
	if p.cursor == "" {
		number := at.number + 1
		return page{into: p.into, value: strconv.Itoa(number), number: number, written: true}, true
	}
	token, resolved := projection.Resolve(p.cursor, response)
	if !resolved || token == nil {
		return page{}, false
	}
	return page{into: p.into, value: projection.Text(token), written: true}, true
}

// holds answers whether the polling Pattern's `until:` list holds of the
// response in hand, and the mismatch where an operator was handed a value it
// cannot compare.
//
// **A predicate list does not short-circuit**: every conjunct is evaluated
// against the response whether or not an earlier one settled the answer, so
// whether a Run halts does not depend on the order two conjuncts were written
// in (ADR-0035).
//
// instant is the Run's start, used verbatim as it is at the other two roots:
// one instant covers every Step, every nested Procedure and all three roots, so
// nothing a Pattern does during a Run moves what a later Step reaches
// (ADR-0034).
func (p polling) holds(response capability.Object, instant time.Time) (bool, string) {
	held, found := true, ""
	for _, until := range p.until {
		matched, mismatch := until.holdsOfResponse(response, instant)
		if mismatch != "" && found == "" {
			found = mismatch
		}
		held = held && matched
	}
	if found != "" {
		return false, found
	}
	return held, ""
}

// request is one call as the Patterns make it: the page position written into
// it, the response object that came back, and the two things that can be said
// beside it.
//
// neverSent is whether the failure **provably preceded the request**, which is
// the only thing a retry follows and which internal/capability establishes by
// where the failure happened rather than by what it said (ADR-0018).
//
// halted is the fault that stops the Step — the Operation's own `deadline:`,
// which is `hyper` stopping rather than the world answering — worded by the
// Capability that reached it. Everything else is narration's and is dropped
// there: a `read` records a call that got no answer as the answer it is (§6,
// ADR-0050).
type request func(ctx context.Context, at page) (response capability.Object, neverSent bool, halted error)

// read is what one page's response becomes: the Records it projected, kept by
// whoever supplied it, and how many members it held.
//
// Pagination terminates on that count, which is why it is the answer rather
// than something this file resolves for itself: **both forms terminate when the
// collection `record.over` names comes back empty**, and the one reading of
// that collection is the projection's. Reading it twice would be two chances
// for *the collection was empty* and *nothing was projected* to disagree.
//
// The error is a projection that did not resolve, which halts the Run (§6).
type read func(response capability.Object) (members int, err error)

// perform makes one member's calls, from the first to the last page, and
// answers `hyper`'s own account of what that took.
//
// **The Operation's `deadline:` bounds the whole of it**, polls and pages
// included: the context arrives already bounded and every call below is made
// under it, so there is no second limit anywhere to disagree with the first
// (§3, §6).
//
// The account it answers is the member's own and is answered **whether or not
// the member halted**: a Step's file says what `hyper` did to reach its
// outcome, and a member that polled four times and then reached the deadline
// did four polls (§7).
func (p patterns) perform(ctx context.Context, instant time.Time, send request, keep read) (acted account, halted error) {
	// The pages are counted as they answer and reduced once, however this
	// returns: a walk that fetched one page did what every Operation does,
	// and §7 requires that to be the same silence as no pagination at all.
	fetched := 0
	defer func() { acted.pages = beyondOne(fetched) }()

	at := p.pagination.first()
	for {
		response, err := p.polled(ctx, &acted, instant, at, send)
		if err != nil {
			return acted, err
		}
		if p.pagination.declared {
			fetched++
		}
		members, err := keep(response)
		if err != nil {
			return acted, err
		}
		// A Pattern's whole authority is *is there another call to
		// make*, and here there is not: an Operation declaring no
		// pagination makes one call, and a page whose collection came
		// back empty is the walk's end under both forms.
		if !p.pagination.declared || members == 0 {
			return acted, nil
		}
		next, more := p.pagination.next(at, response)
		if !more {
			return acted, nil
		}
		at = next
	}
}

// polled makes one page's call, repeating it until the polling Pattern's
// `until:` holds.
//
// **An `until:` handed a value it cannot compare halts the Run** rather than
// Refusing. It reads a response after the call went out, so there is no Refusal
// available: the Run halts, carries **no `error_code`**, and names the field and
// what was found in it. That is why two of §12's three predicate roots
// contribute a Refusal and this one contributes a halt (§6, ADR-0035,
// ADR-0072).
//
// The interval is waited **between** calls and never before the first: an
// Operation that was ready when it was asked is answered at once, and a poll
// that slept first would put its interval on every call an artefact makes.
//
// The wait ends at the Operation's deadline as well as at the interval, and
// what it does then is go round again rather than answer: the next call is made
// under a context that is already done, and the Capability that makes it words
// the deadline it reached — which is what keeps the deadline worded in one
// place per Capability rather than in three (step.go, §6).
func (p patterns) polled(ctx context.Context, acted *account, instant time.Time, at page, send request) (capability.Object, error) {
	// One page's polls, reduced here and summed into the member's account:
	// a page that was ready when it was asked was answered by a single
	// call, whatever a page beside it took.
	answered := 0
	defer func() { acted.polls += beyondOne(answered) }()

	for {
		response, err := p.retried(ctx, acted, at, send)
		if err != nil {
			return nil, err
		}
		if !p.polling.declared {
			return response, nil
		}
		answered++

		held, mismatch := p.polling.holds(response, instant)
		if mismatch != "" {
			return nil, fmt.Errorf("the polling Pattern's until: %s", mismatch)
		}
		if held {
			return response, nil
		}
		waiting(ctx, p.polling.interval)
	}
}

// retried makes one call, and makes it again where the failure provably
// preceded the request and the Pattern has attempts left.
//
// **An exhausted retry leaves the response object for the projection to read.**
// It is not a fault: on a `read` that is an Observation whose `status` has gone
// quiet, recorded rather than halted on (§6, §12, ADR-0050). The Disposition
// that says the world was provably untouched belongs to the effectful Kinds and
// lands with them (ADR-0062).
//
// An Operation declaring no retry makes exactly one call and counts nothing:
// *one attempt* and *no retry declared* are the same silence on the page, and
// §7 requires them to stay one (§7).
//
// Where one does declare it, the count is every call **this request** took, so
// an Operation that is paginated as well as retried and asked one page for
// twice reads `attempts: 2, pages: 3` — two attempts at one page, rather than
// one number standing for both Patterns (§7, ADR-0018).
func (p patterns) retried(ctx context.Context, acted *account, at page, send request) (capability.Object, error) {
	// One request's attempts, reduced here and summed into the member's
	// account. Reducing per **request** is what keeps a second Pattern from
	// inflating this one: an Operation that is paginated as well as retried
	// and never retries anything makes three calls and has made three
	// single attempts, which is the silence §7 requires — where reducing
	// per member would write `attempts: 3` and read as an exhausted retry.
	made := 0
	defer func() { acted.attempts += beyondOne(made) }()

	attempts := 1
	if p.retry.declared {
		attempts = max(p.retry.attempts, 1)
	}
	for attempt := 1; ; attempt++ {
		response, neverSent, halted := send(ctx, at)
		if p.retry.declared {
			made++
		}
		if halted != nil {
			return nil, halted
		}
		if !neverSent || attempt >= attempts {
			return response, nil
		}
	}
}

// waiting is the interval a polling Pattern waits between calls, ended early by
// the Operation's deadline.
//
// It is real time and not a clock this package reads, which is the same
// treatment the deadline beside it already has: `capability.Deadline` is a
// context timeout on the machine's own clock, and `Request.Now` is threaded
// because the instants a Run **records** must be a fixture's rather than a
// machine's. Nothing waited here is ever recorded.
func waiting(ctx context.Context, interval time.Duration) {
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// readPatterns reads an Operation's `patterns:` block off the node that
// Operation is declared by. Which node that is is internal/artefact's to answer
// (artefact.OperationNode).
//
// It judges nothing and drops what it cannot read, which is every reader's rule
// in this tool: a block naming none of the three, a `pagination:` carrying both
// forms or neither, an `interval:` that is not a duration and an `attempts:`
// that is not an integer are all `check`'s to report, and a reader that guessed
// would be a second opinion on an artefact nobody reviewed (§4, ADR-0064).
func readPatterns(operation *yaml.Node) patterns {
	block := patternValue(operation, "patterns")
	if block == nil {
		return patterns{}
	}
	return patterns{
		pagination: readPagination(patternValue(block, "pagination")),
		polling:    readPolling(patternValue(block, "polling")),
		retry:      readRetry(patternValue(block, "retry")),
	}
}

// readPagination reads the two forms and the position they write into. A block
// carrying both is read as the `cursor:` one and is `schema-mismatch` from
// check either way: what matters here is that one reading is fixed rather than
// left to a mapping's key order.
func readPagination(block *yaml.Node) pagination {
	if block == nil {
		return pagination{}
	}
	if cursor := patternValue(block, "cursor"); cursor != nil {
		return pagination{
			declared: true,
			cursor:   patternScalar(patternValue(cursor, "from")),
			into:     readInto(patternValue(cursor, "into")),
		}
	}
	if numbered := patternValue(block, "page"); numbered != nil {
		from, _ := strconv.Atoi(patternScalar(patternValue(numbered, "from")))
		return pagination{declared: true, page: from, into: readInto(patternValue(numbered, "into"))}
	}
	return pagination{}
}

// readInto reads the single-key mapping naming the position: `query:` or
// `header:`, and the name within it.
func readInto(block *yaml.Node) into {
	if header := patternValue(block, "header"); header != nil {
		return into{header: true, name: patternScalar(header)}
	}
	return into{name: patternScalar(patternValue(block, "query"))}
}

// readPolling reads the interval and the predicate list, which is the same
// reader every other predicate list in the tool goes through: one operator set,
// three roots, and the root decided by the position rather than by the entry
// (§12, predicate.go).
func readPolling(block *yaml.Node) polling {
	if block == nil {
		return polling{}
	}
	seconds, _ := schema.DurationSeconds(patternScalar(patternValue(block, "interval")))
	return polling{
		declared: true,
		interval: time.Duration(seconds) * time.Second,
		until:    readPredicates(patternValue(block, "until")),
	}
}

// readRetry reads `attempts:`, which is the whole of the Pattern.
func readRetry(block *yaml.Node) retry {
	if block == nil {
		return retry{}
	}
	attempts, _ := strconv.Atoi(patternScalar(patternValue(block, "attempts")))
	return retry{declared: true, attempts: attempts}
}

// patternValue is the value one key of a mapping holds, and nil where the node
// is not a mapping or does not carry that key.
func patternValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if k := node.Content[i]; k.Kind == yaml.ScalarNode && k.Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// patternScalar is a node's text, and "" where the node is absent or is not a
// plain scalar.
func patternScalar(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}
