package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// The fixture's own fence (issue #227). What the `monitor-coverage` task is
// worth depends on this service answering the way `docs/lookout-api.md` says it
// does — that file is written into the repository the agent is handed, and it is
// the only description of this API anyone inside the seal can read, so a drift
// between the two is a sealed session graded against documentation that lies.
//
// The acceptance fence in `cmd/hyper` asserts that this service *starts*: a
// task's subtest runs the setup script, which waits for the report and fails
// under the task's own name if it never arrives. What it cannot assert is what
// the service answers, and three of those answers are load-bearing rather than
// incidental — the page size, the duplicate, and the type of `window` — because
// each is a trap the task's comment claims is reachable. Every route the shipped
// documentation describes is driven here, including the two the task never asks
// for, since the file is the contract and not the task's half of it.
//
// These cases drive the handler directly rather than over TLS: the certificate
// is main.go's business, and the transport is not what any of this is about.
//
// **There are three worlds now** (issues #255, #271). Everything below but the
// last four cases drives `monitor-coverage`'s, because that is the state these
// routes were written against and the one whose transcripts are compared across
// years. The last four, in the order they stand: the first world's bytes held
// against a golden so a later fixture cannot move them, the second world's own
// arrangement, the third world's, and the shipped documentation held to naming
// nothing the service does not answer.

// coverage is the monitors `monitor-coverage` starts with — the world every case
// below that does not say otherwise is driving, and the length the counts in them
// are against.
func coverage() []monitor { return fixtures()["coverage"].monitors }

// listing is the shape of an answer to the list route, spelled once because two
// helpers below read it.
type listing struct {
	Data struct {
		Monitors []monitor `json:"monitors"`
		Cursor   string    `json:"cursor"`
	} `json:"data"`
}

// TestLookout_ACallWithoutTheTokenIsRefusedWhateverItAsksFor holds the order the
// checks are in. The token is checked before the route is, so a call with no
// credential cannot learn what routes exist — and, more to the point here, an
// agent whose Manifest declares no Auth scheme meets one status rather than a
// different one per call (§3).
func TestLookout_ACallWithoutTheTokenIsRefusedWhateverItAsksFor(t *testing.T) {
	api := fixtures()["coverage"].serve("fixture")

	for _, call := range []struct{ method, path string }{
		{http.MethodGet, "/v1/monitors"},
		{http.MethodPost, "/v1/monitors"},
		{http.MethodGet, "/v1/monitors/mon_0d3e88"},
		{http.MethodDelete, "/v1/monitors/mon_0d3e88"},
		{http.MethodPost, "/v1/monitors/mon_0d3e88/credential"},
		{http.MethodGet, "/v1/nothing-here"},
	} {
		answer := ask(t, api, call.method, call.path, "", "")
		if answer.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a token answered %d, want 401", call.method, call.path, answer.Code)
		}
		if code := errorCode(t, answer.Body.String()); code != "unauthorized" {
			t.Errorf("%s %s without a token answered %q, want unauthorized", call.method, call.path, code)
		}
	}
	if held := len(api.monitors); held != len(coverage()) {
		t.Errorf("the monitors moved under a call nobody was allowed to make: %d stand, want %d", held, len(coverage()))
	}
}

// TestLookout_TheListPagesAtTwoAndLimitReachesTheRest is the one an author has
// to survive. Four monitors and a page of two means the first answer is half the
// list, and `data.cursor` and `?limit=` are the two routes out of it that the
// documentation describes — the third, a `pagination` Pattern, is §3's and needs
// nothing of this service. The cursor is asserted to be opaque rather than an
// offset spelled in the clear, since an offset an author can compute is one
// nobody has to carry.
func TestLookout_TheListPagesAtTwoAndLimitReachesTheRest(t *testing.T) {
	api := fixtures()["coverage"].serve("fixture")

	first := page(t, api, "/v1/monitors")
	if len(first.Data.Monitors) != pageSize {
		t.Fatalf("the first page held %d monitors, want %d", len(first.Data.Monitors), pageSize)
	}
	if first.Data.Cursor == "" {
		t.Fatal("the first page carried no cursor and there are two more monitors behind it")
	}
	if first.Data.Cursor == "2" {
		t.Error("the cursor is the offset in the clear; an offset an author can compute is one nobody carries")
	}

	second := page(t, api, "/v1/monitors?cursor="+first.Data.Cursor)
	if len(second.Data.Monitors) != 2 || second.Data.Cursor != "" {
		t.Errorf("the second page held %d monitors and cursor %q, want 2 and the end of the list",
			len(second.Data.Monitors), second.Data.Cursor)
	}
	if first.Data.Monitors[0].Service != "legacy-import" || second.Data.Monitors[1].Service != "billing" {
		t.Errorf("the list is not oldest-first: it opens on %q and ends on %q",
			first.Data.Monitors[0].Service, second.Data.Monitors[1].Service)
	}

	whole := page(t, api, "/v1/monitors?limit=100")
	if len(whole.Data.Monitors) != len(coverage()) || whole.Data.Cursor != "" {
		t.Errorf("limit=100 held %d monitors and cursor %q, want all of them and no cursor",
			len(whole.Data.Monitors), whole.Data.Cursor)
	}

	for _, limit := range []string{"0", "101", "many"} {
		answer := ask(t, api, http.MethodGet, "/v1/monitors?limit="+limit, "", "fixture")
		if code := errorCode(t, answer.Body.String()); answer.Code != http.StatusBadRequest || code != "invalid_limit" {
			t.Errorf("limit=%s answered %d %q, want 400 invalid_limit", limit, answer.Code, code)
		}
	}
	if code := errorCode(t, ask(t, api, http.MethodGet, "/v1/monitors?cursor=zzz", "", "fixture").Body.String()); code != "invalid_cursor" {
		t.Errorf("a cursor this service did not mint answered %q, want invalid_cursor", code)
	}
}

// TestLookout_ACreateAnswersOneObjectAndSaysNoTheDocumentedWays holds the
// create's shape and every way it says no. The shape is the awkwardness
// ADR-0105 asked for — one object under `data.monitor` where the list carries
// them under `data.monitors`, so a projection written off the list does not
// resolve against the create — and two of the five refusals are traps the task's
// comment claims are reachable: a duplicate is what an agent who read one page
// meets, and a `window` that is not a whole number of seconds is where
// ADR-0078's typing rule meets an API that checks.
func TestLookout_ACreateAnswersOneObjectAndSaysNoTheDocumentedWays(t *testing.T) {
	api := fixtures()["coverage"].serve("fixture")

	answer := ask(t, api, http.MethodPost, "/v1/monitors", `{"service":"checkout","window":60}`, "fixture")
	if answer.Code != http.StatusCreated {
		t.Fatalf("a good create answered %d, want 201: %s", answer.Code, answer.Body)
	}
	var created struct {
		Data struct {
			Monitor  monitor   `json:"monitor"`
			Monitors []monitor `json:"monitors"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Data.Monitors != nil {
		t.Error("the create answered a collection; the list's shape and the create's are two shapes on purpose")
	}
	if got := created.Data.Monitor; got.Service != "checkout" || got.Window != 60 || got.State != "pending" || got.Ref == "" {
		t.Errorf("the create answered %+v, want checkout on 60 seconds, pending, with a ref", got)
	}

	for _, refusal := range []struct {
		body, code string
		status     int
	}{
		{`{"service":"billing","window":60}`, "already_watched", http.StatusConflict},
		{`{"service":"search","window":"60"}`, "invalid_window", http.StatusBadRequest},
		{`{"service":"search"}`, "invalid_window", http.StatusBadRequest},
		{`{"service":"search","window":1}`, "window_out_of_range", http.StatusBadRequest},
		{`{"service":"search","window":86400}`, "window_out_of_range", http.StatusBadRequest},
		{`{"window":60}`, "invalid_service", http.StatusBadRequest},
		{`not json`, "invalid_body", http.StatusBadRequest},
	} {
		answer := ask(t, api, http.MethodPost, "/v1/monitors", refusal.body, "fixture")
		if code := errorCode(t, answer.Body.String()); answer.Code != refusal.status || code != refusal.code {
			t.Errorf("%s answered %d %q, want %d %s", refusal.body, answer.Code, code, refusal.status, refusal.code)
		}
	}
	if held := len(api.monitors); held != len(coverage())+1 {
		t.Errorf("%d monitors stand; a create that was said no to must leave the list where it was", held)
	}
}

// TestLookout_ARefReachesOneMonitorAndRetiresIt drives the two routes the task
// never asks for. They are in the documentation because a service that watches
// things is one that stops watching them, and a fixture whose documentation
// describes a route it does not have is one an agent can waste a session on.
// `DELETE` is also the one answer here with no body at all, which is the
// exception the envelope comment in api.go states.
func TestLookout_ARefReachesOneMonitorAndRetiresIt(t *testing.T) {
	api := fixtures()["coverage"].serve("fixture")

	answer := ask(t, api, http.MethodGet, "/v1/monitors/mon_0d3e88", "", "fixture")
	var one struct {
		Data struct {
			Monitor monitor `json:"monitor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &one); err != nil {
		t.Fatal(err)
	}
	if answer.Code != http.StatusOK || one.Data.Monitor.Service != "ingest" {
		t.Errorf("GET by ref answered %d and %q, want 200 and ingest", answer.Code, one.Data.Monitor.Service)
	}

	answer = ask(t, api, http.MethodDelete, "/v1/monitors/mon_0d3e88", "", "fixture")
	if answer.Code != http.StatusNoContent || answer.Body.Len() != 0 {
		t.Errorf("DELETE answered %d with %d bytes, want 204 and nothing", answer.Code, answer.Body.Len())
	}
	if held := page(t, api, "/v1/monitors?limit=100").Data.Monitors; len(held) != len(coverage())-1 {
		t.Errorf("%d monitors stand after a delete, want %d", len(held), len(coverage())-1)
	}

	for _, gone := range []string{"/v1/monitors/mon_0d3e88", "/v1/monitors/mon_nothing"} {
		for _, method := range []string{http.MethodGet, http.MethodDelete} {
			answer := ask(t, api, method, gone, "", "fixture")
			if code := errorCode(t, answer.Body.String()); answer.Code != http.StatusNotFound || code != "no_such_monitor" {
				t.Errorf("%s %s answered %d %q, want 404 no_such_monitor", method, gone, answer.Code, code)
			}
		}
	}
}

// TestLookout_ACredentialIsAnsweredOnceAndByNoOtherRoute is the fixture's half of
// what the `push-credential` task measures (issue #271). The task is the only one
// in the set that reaches a Step whose Operation declares `secret:` output, and it
// reaches it here: this is the one value the service issues that it will not issue
// again, which is the class ADR-0007 names as not re-readable.
//
// **The claim under assertion is the absence.** `docs/lookout-api.md` tells the
// agent the token is in this answer and in no other, and an author who marked the
// field `secret:` on the strength of that sentence has been told the truth only if
// no other route carries it. A service that leaked the value into the list would
// make the sink pointless and the documentation a lie, and nothing else here would
// notice: the value is random, so a golden cannot hold it and only this can.
//
// The two path faults are here for the same reason the create's five are. A `GET`
// at the credential path is a method the route declines and a second segment the
// service does not have is a route that is not there, and an author who met one
// wearing the other's answer would be debugging our routing.
func TestLookout_ACredentialIsAnsweredOnceAndByNoOtherRoute(t *testing.T) {
	api := fixtures()["coverage"].serve("fixture")

	first := issued(t, api, "mon_0d3e88")
	if first.Monitor != "mon_0d3e88" || first.Service != "ingest" {
		t.Errorf("the credential names monitor %q and service %q, want mon_0d3e88 and ingest", first.Monitor, first.Service)
	}
	if first.ID == "" || first.Token == "" || first.Issued == "" {
		t.Fatalf("the credential came back as %+v, want an id, a token and an issue time", first)
	}

	second := issued(t, api, "mon_0d3e88")
	if second.Token == first.Token || second.ID == first.ID {
		t.Errorf("a second mint answered id %q token %q, and the first answered %q and %q: asking again gets another one",
			second.ID, second.Token, first.ID, first.Token)
	}

	// The whole of the service's other surface, read back and searched for the
	// value it was told to carry nowhere.
	for _, path := range []string{"/v1/monitors?limit=100", "/v1/monitors/mon_0d3e88"} {
		body := ask(t, api, http.MethodGet, path, "", "fixture").Body.String()
		for _, token := range []string{first.Token, second.Token} {
			if strings.Contains(body, token) {
				t.Errorf("GET %s carries a minted token; the documentation says it is in the mint's answer and in no other", path)
			}
		}
	}
	if held := len(api.monitors); held != len(coverage()) {
		t.Errorf("%d monitors stand after two mints, want %d: issuing a credential is not creating a monitor", held, len(coverage()))
	}

	answer := ask(t, api, http.MethodPost, "/v1/monitors/mon_nothing/credential", "", "fixture")
	if code := errorCode(t, answer.Body.String()); answer.Code != http.StatusNotFound || code != "no_such_monitor" {
		t.Errorf("a mint against a ref we do not hold answered %d %q, want 404 no_such_monitor", answer.Code, code)
	}
	answer = ask(t, api, http.MethodGet, "/v1/monitors/mon_0d3e88/credential", "", "fixture")
	if answer.Code != http.StatusMethodNotAllowed {
		t.Errorf("GET at the credential path answered %d, want 405: the route is there and the method is not", answer.Code)
	}
	answer = ask(t, api, http.MethodPost, "/v1/monitors/mon_0d3e88/heartbeats", "", "fixture")
	if code := errorCode(t, answer.Body.String()); answer.Code != http.StatusNotFound || code != "no_such_route" {
		t.Errorf("a second segment the service does not have answered %d %q, want 404 no_such_route", answer.Code, code)
	}
}

// issued mints one credential and hands back what came out of it, failing the
// case rather than the assertion where the call did not answer `201`.
func issued(t *testing.T, api *lookout, ref string) credential {
	t.Helper()

	answer := ask(t, api, http.MethodPost, "/v1/monitors/"+ref+"/credential", "", "fixture")
	if answer.Code != http.StatusCreated {
		t.Fatalf("the mint for %s answered %d: %s", ref, answer.Code, answer.Body)
	}
	var minted struct {
		Data struct {
			Credential credential `json:"credential"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &minted); err != nil {
		t.Fatal(err)
	}
	return minted.Data.Credential
}

// ask puts one call through the handler and hands back what came out.
func ask(t *testing.T, api *lookout, method, path, body, token string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	answer := httptest.NewRecorder()
	api.ServeHTTP(answer, request)
	return answer
}

// page reads a listing, and fails the case rather than the assertion where the
// call did not answer `200` — a page that is not a page has nothing to compare.
func page(t *testing.T, api *lookout, path string) listing {
	t.Helper()

	answer := ask(t, api, http.MethodGet, path, "", "fixture")
	if answer.Code != http.StatusOK {
		t.Fatalf("GET %s answered %d: %s", path, answer.Code, answer.Body)
	}
	var read listing
	if err := json.Unmarshal(answer.Body.Bytes(), &read); err != nil {
		t.Fatal(err)
	}
	return read
}

// errorCode reads the stable half of an error answer, which is the half the
// documentation names and the half a reader of a transcript will quote.
func errorCode(t *testing.T, body string) string {
	t.Helper()

	var answer struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(body), &answer); err != nil {
		t.Fatalf("%v: %s", err, body)
	}
	return answer.Error.Code
}

// TestLookout_TheCoverageFixtureIsTheBytesThatTaskHasAlwaysSeen is a golden and
// nothing more. `monitor-coverage`'s transcripts are read against one another
// across years, and every one of them was produced against these four monitors in
// this order against a page size of two — so a seeded monitor renamed, reordered
// or added is a comparison that has quietly changed its subject. A second task
// arriving with its own world is exactly when that becomes possible, which is why
// this case arrives with it (issue #255).
func TestLookout_TheCoverageFixtureIsTheBytesThatTaskHasAlwaysSeen(t *testing.T) {
	want := []monitor{
		{Ref: "mon_4a91c7", Service: "legacy-import", Window: 300, Muted: true, State: "active", Created: "2025-11-02T08:41:19Z"},
		{Ref: "mon_0d3e88", Service: "ingest", Window: 60, State: "active", Created: "2026-01-14T10:02:47Z"},
		{Ref: "mon_b71f20", Service: "mailer", Window: 60, State: "active", Created: "2026-03-30T16:55:08Z"},
		{Ref: "mon_2c5a94", Service: "billing", Window: 60, State: "active", Created: "2026-06-11T09:20:33Z"},
	}
	if got := fixtures()["coverage"]; !reflect.DeepEqual(got.monitors, want) || got.unreachable != nil {
		t.Errorf("the coverage fixture is\n%+v\nand every monitor-coverage transcript was produced against\n%+v", got, want)
	}
	if len(want)%pageSize != 0 || len(want)/pageSize != 2 {
		t.Errorf("%d monitors at a page size of %d is no longer the two even pages that task's paging trap is", len(want), pageSize)
	}
}

// TestLookout_AServiceThatDoesNotAnswerItsFirstLookIsNotKept is the retirement
// fixture's own arrangement, and the whole of what it buys (issue #255). A
// `destroy` Step expanding over two Assets meets a `404` on one member and a
// `204` on the other only if one of the two monitors is gone without `hyper`
// having deleted it — §5 drops a Tombstoned member from a later Expansion, so
// re-running a retire Procedure cannot produce it, and nothing inside the seal
// can reach into the service and remove one by hand.
//
// The create's answer is asserted to be unchanged, because that is what makes the
// caller accountable for a monitor the world does not hold: a `503` here would be
// an ordinary failed call and would leave nothing behind to disagree with.
func TestLookout_AServiceThatDoesNotAnswerItsFirstLookIsNotKept(t *testing.T) {
	world := fixtures()["retirement"]
	api := world.serve("fixture")

	answer := ask(t, api, http.MethodPost, "/v1/monitors", `{"service":"pricing","window":60}`, "fixture")
	if answer.Code != http.StatusCreated {
		t.Fatalf("the create for an unreachable service answered %d, want 201: %s", answer.Code, answer.Body)
	}
	var created struct {
		Data struct {
			Monitor monitor `json:"monitor"`
		} `json:"data"`
	}
	if err := json.Unmarshal(answer.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	gone := created.Data.Monitor
	if gone.Ref == "" || gone.Service != "pricing" || gone.Window != 60 {
		t.Fatalf("the create answered %+v, want the monitor it minted for pricing", gone)
	}

	held := page(t, api, "/v1/monitors?limit=100").Data.Monitors
	if len(held) != len(world.monitors) {
		t.Errorf("%d monitors stand after a create that was not kept, want %d", len(held), len(world.monitors))
	}
	for _, standing := range held {
		if standing.Service == "pricing" {
			t.Errorf("pricing is in the list; a service that did not answer its first look is not one we kept")
		}
	}
	for _, method := range []string{http.MethodGet, http.MethodDelete} {
		answer := ask(t, api, method, "/v1/monitors/"+gone.Ref, "", "fixture")
		if code := errorCode(t, answer.Body.String()); answer.Code != http.StatusNotFound || code != "no_such_monitor" {
			t.Errorf("%s the dropped ref answered %d %q, want 404 no_such_monitor", method, answer.Code, code)
		}
	}

	// The other half of the same Step: a service that does answer is kept, and
	// its `DELETE` is the `204` the `404` above stands beside.
	if answer := ask(t, api, http.MethodPost, "/v1/monitors", `{"service":"warehouse","window":60}`, "fixture"); answer.Code != http.StatusCreated {
		t.Fatalf("the create for a reachable service answered %d, want 201: %s", answer.Code, answer.Body)
	}
	kept := page(t, api, "/v1/monitors?limit=100").Data.Monitors
	if len(kept) != len(world.monitors)+1 {
		t.Fatalf("%d monitors stand after a create that was kept, want %d", len(kept), len(world.monitors)+1)
	}
	if answer := ask(t, api, http.MethodDelete, "/v1/monitors/"+kept[len(kept)-1].Ref, "", "fixture"); answer.Code != http.StatusNoContent {
		t.Errorf("the delete for the kept monitor answered %d, want 204", answer.Code)
	}
}

// TestLookout_ThePushCredentialFixtureIsOnePageAndNothingInTheWay is the third
// world's arrangement, and every assertion in it is an absence (issue #271).
//
// That task's measurement is on the far side of a Run: a Refusal read, the same
// command again with a sink, and a value got out of the directory `hyper` made.
// A session that spent its turns on a paging trap would never reach any of it, so
// this world has none — two monitors against a page size of two is one page and no
// cursor, both name services the repository names, and nothing is switched off.
// The four awkwardnesses the API has by design are still there and are not this
// case's to assert; what is asserted here is that nothing *else* is.
//
// It is a fence rather than a description because the arrangement is a set of
// numbers that hold only while they agree with each other: a third monitor seeded
// here, or a page size raised to three in main.go, silently reintroduces or
// removes the trap this world exists without.
func TestLookout_ThePushCredentialFixtureIsOnePageAndNothingInTheWay(t *testing.T) {
	world := fixtures()["push-credential"]
	if len(world.monitors) != pageSize {
		t.Errorf("%d monitors against a page size of %d is not the one page this world is", len(world.monitors), pageSize)
	}
	if world.unreachable != nil {
		t.Errorf("this world switches off %v; nothing here is arranged to fail", world.unreachable)
	}

	api := world.serve("fixture")
	whole := page(t, api, "/v1/monitors")
	if len(whole.Data.Monitors) != len(world.monitors) || whole.Data.Cursor != "" {
		t.Fatalf("the first page held %d monitors and cursor %q, want all %d and no second page",
			len(whole.Data.Monitors), whole.Data.Cursor, len(world.monitors))
	}
	for _, held := range whole.Data.Monitors {
		if held.State != "active" {
			t.Errorf("%s is %q; both monitors here have settled and neither is a thing to wonder about", held.Service, held.State)
		}
		if minted := issued(t, api, held.Ref); minted.Service != held.Service || minted.Token == "" {
			t.Errorf("the mint for %s answered %+v, want a token for that service", held.Ref, minted)
		}
	}
}

// TestLookout_TheDocumentationQuotesTheAnswersTheServiceGives holds the file the
// sealed session is graded against to the service it describes. The document
// names a refusal one way throughout — a status and a code in one span, `409
// already_watched` — so that is what is read out of it and held against what
// `route` can actually answer.
//
// It is the one direction worth asserting. An author who cannot find a documented
// code is debugging our prose, where a code the documentation does not mention —
// `no_such_route`, `method_not_allowed` — is the ordinary reticence of a vendor's
// reference and no lie. The six named below are the ones an author meets on a
// documented route, and requiring them is what stops this case passing because a
// document said nothing at all.
//
// One document is shipped by both setup scripts rather than a heredoc in each
// (issue #255), which is what makes this one file to hold anything against.
func TestLookout_TheDocumentationQuotesTheAnswersTheServiceGives(t *testing.T) {
	statuses := map[string]int{
		"BadRequest": http.StatusBadRequest, "Unauthorized": http.StatusUnauthorized,
		"NotFound": http.StatusNotFound, "Conflict": http.StatusConflict,
		"MethodNotAllowed": http.StatusMethodNotAllowed,
	}
	source, err := os.ReadFile("api.go")
	if err != nil {
		t.Fatal(err)
	}
	answers := map[string]int{}
	for _, found := range regexp.MustCompile(`reject\(w, http\.Status(\w+), "([a-z_]+)"`).FindAllStringSubmatch(string(source), -1) {
		status, known := statuses[found[1]]
		if !known {
			t.Fatalf("api.go answers http.Status%s and this case does not know its number", found[1])
		}
		answers[found[2]] = status
	}
	if len(answers) == 0 {
		t.Fatal("no refusals were found in api.go; this case would pass for the wrong reason")
	}

	documentation, err := os.ReadFile("api.md")
	if err != nil {
		t.Fatal(err)
	}
	quoted := map[string]bool{}
	for _, found := range regexp.MustCompile("`([0-9]{3}) ([a-z_]+)`").FindAllStringSubmatch(string(documentation), -1) {
		status, answered := answers[found[2]]
		switch {
		case !answered:
			t.Errorf("api.md quotes %q and the service never answers it", found[2])
		case strconv.Itoa(status) != found[1]:
			t.Errorf("api.md quotes %s %s and the service answers %d", found[1], found[2], status)
		}
		quoted[found[2]] = true
	}
	for _, met := range []string{"already_watched", "no_such_monitor", "invalid_window", "window_out_of_range", "invalid_cursor", "invalid_limit"} {
		if !quoted[met] {
			t.Errorf("api.md does not quote %q, which an author meets on a documented route", met)
		}
	}
}
