package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	api := &lookout{token: "fixture", monitors: seeded()}

	for _, call := range []struct{ method, path string }{
		{http.MethodGet, "/v1/monitors"},
		{http.MethodPost, "/v1/monitors"},
		{http.MethodGet, "/v1/monitors/mon_0d3e88"},
		{http.MethodDelete, "/v1/monitors/mon_0d3e88"},
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
	if held := len(api.monitors); held != len(seeded()) {
		t.Errorf("the monitors moved under a call nobody was allowed to make: %d stand, want %d", held, len(seeded()))
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
	api := &lookout{token: "fixture", monitors: seeded()}

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
	if len(whole.Data.Monitors) != len(seeded()) || whole.Data.Cursor != "" {
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
	api := &lookout{token: "fixture", monitors: seeded()}

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
	if held := len(api.monitors); held != len(seeded())+1 {
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
	api := &lookout{token: "fixture", monitors: seeded()}

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
	if held := page(t, api, "/v1/monitors?limit=100").Data.Monitors; len(held) != len(seeded())-1 {
		t.Errorf("%d monitors stand after a delete, want %d", len(held), len(seeded())-1)
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
