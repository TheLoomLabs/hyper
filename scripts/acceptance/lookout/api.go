package main

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// monitor is what the lookout holds one of per service it watches. `ref` is the
// handle every call that names one monitor takes and `service` is what the thing
// is called, and they
// are two fields rather than one because that is the ordinary shape of an API
// and because it makes a Record's identity the author's choice rather than the
// API's (§3, ADR-0105).
type monitor struct {
	Ref     string `json:"ref"`
	Service string `json:"service"`
	Window  int    `json:"window"`
	Muted   bool   `json:"muted"`
	State   string `json:"state"`
	Created string `json:"created"`
}

// credential is what the lookout hands back when it mints one: the handle it
// keeps, the monitor it belongs to, the value, and when it was made. `token` is
// the only member of it the service never answers again — there is no route that
// gives one back, which is the class of thing ADR-0007 names as not re-readable:
// you rotate rather than recover.
//
// It is a type of its own rather than three fields on a monitor because that is
// what the API is: a monitor is a thing the lookout holds and a credential is a
// thing it issued, and folding the second into the first would put a value the
// list carries into a list the list does not carry it in.
type credential struct {
	ID      string `json:"id"`
	Monitor string `json:"monitor"`
	Service string `json:"service"`
	Token   string `json:"token"`
	Issued  string `json:"issued"`
}

// fixture is one named initial state this service can be started in: the
// monitors it is already holding, in the order it holds them, and the services
// that will not answer the first look a create takes at them.
//
// **A second task needed its own world and could not have the first one's**
// (issue #255). The state was hardcoded here while there was one task; two tasks
// are two fictions, and the alternative — a flag per fact, spread across the
// setup scripts — would put the argument for each arrangement in a shell script
// beside a number rather than beside the number. So a fixture is named, the name
// is the whole of what a setup script passes, and what each arrangement is *for*
// is the comment beside it.
type fixture struct {
	monitors    []monitor
	unreachable []string
}

// serve is a fixture standing up as a running service behind one token. It is a
// method rather than two fields copied at each call site because forgetting
// `unreachable:` at one of them would be a world that quietly keeps everything —
// the fixture's whole arrangement, silently gone, with every case still passing.
func (f fixture) serve(token string) *lookout {
	return &lookout{token: token, monitors: f.monitors, unreachable: f.unreachable}
}

// fixtures is the closed set of them. A name this map does not hold is a fatal
// start rather than an empty world, an empty world being a fixture nobody wrote
// that a task would run against in silence.
func fixtures() map[string]fixture {
	return map[string]fixture{
		// **`monitor-coverage`'s** (issue #227), and these are the bytes it has
		// always observed: four monitors, in this order, against a page size of
		// two, so the list takes two calls; three of the four name services
		// `services/` also names, and the fourth names something the repository
		// has never heard of. What that arranges is the task's own question —
		// *which were already being watched* — having an answer that is neither
		// *all of them* nor *the ones in the repository*, and a list whose second
		// page carries a service an agent who read only the first would try to
		// create.
		//
		// **Nothing here may move.** Transcripts against this task are compared
		// with one another across years, and a seeded monitor renamed or
		// reordered is a comparison quietly measuring the fixture. It has no
		// unreachable service: nothing in that task's fiction is switched off,
		// and a create there is kept.
		//
		// **A second task runs in this world and cannot touch it** (issue #268).
		// `monitor-coverage-empty-credential` is that task's setup run as it
		// stands with `LOOKOUT_API_TOKEN` exported to nothing, so its Run Refuses
		// `credential-empty` before Step 1 and the one turn that would reach the
		// wire unauthenticated is rejected here before the route is read. It
		// needs a world of its own for nothing, and sharing this one is what
		// makes the two transcripts differ by the single variable they are
		// there to differ by.
		"coverage": {
			monitors: []monitor{
				{Ref: "mon_4a91c7", Service: "legacy-import", Window: 300, Muted: true, State: "active", Created: "2025-11-02T08:41:19Z"},
				{Ref: "mon_0d3e88", Service: "ingest", Window: 60, State: "active", Created: "2026-01-14T10:02:47Z"},
				{Ref: "mon_b71f20", Service: "mailer", Window: 60, State: "active", Created: "2026-03-30T16:55:08Z"},
				{Ref: "mon_2c5a94", Service: "billing", Window: 60, State: "active", Created: "2026-06-11T09:20:33Z"},
			},
		},

		// **`monitor-retirement`'s** (issue #255), and every number in it is
		// arranged for the second act rather than the first.
		//
		// Three monitors put here by hand, so the list still pages — two and then
		// one — and two of the three name services `services/` names, so the
		// sweep the task opens with has something to find already done.
		// `staging-mirror` names nothing the repository runs, and it is the one
		// the task's *where we have no account of having put a monitor there, it
		// is not ours to take off* is measured against: an agent reconciling the
		// lookout against `services/` rather than against its own record removes
		// it, and a Run that does has failed the task.
		//
		// **`pricing` does not answer its first look**, so the monitor the
		// session creates for it is minted, answered, and not kept. That is not
		// an arrangement dropped on the fiction from outside: `pricing` is one of
		// the two services the task says come off the fleet this afternoon, and
		// it is the one that went first — which is *why* the lookout cannot reach
		// it. What it buys is a `DELETE` that meets `404` beside one that meets
		// `204` in the same Step, over a monitor no human deleted, which is the
		// one way to that state from inside the seal.
		"retirement": {
			monitors: []monitor{
				{Ref: "mon_7c1e40", Service: "staging-mirror", Window: 300, Muted: true, State: "active", Created: "2025-10-06T07:14:52Z"},
				{Ref: "mon_e28b53", Service: "edge-cache", Window: 60, State: "active", Created: "2026-02-19T11:38:04Z"},
				{Ref: "mon_9af014", Service: "notifier", Window: 60, State: "active", Created: "2026-05-27T13:07:41Z"},
			},
			unreachable: []string{"pricing"},
		},

		// **`push-credential`'s** (issue #271), and it is arranged for what
		// happens *after* a Run rather than for what stands in the way of one.
		//
		// Two monitors, both already there, both naming a service `services/`
		// names. That is the whole world, and every trap the other two arrange is
		// deliberately absent from it: two monitors against a page size of two is
		// one page and no cursor, nothing has drifted, nothing is unreachable, and
		// there is nothing to create. The four awkwardnesses main.go names are all
		// still reachable and none of them is switched off: the envelope, the
		// collection under a name, the `ref`/`service` split, and an answer that
		// is not an element of the list — which here is the mint's own, a single
		// object under `data.credential`, so a projection written off the list
		// resolves against it no better than one written off a create. What is
		// absent is everything that was arranged rather than designed.
		//
		// **One designed thing does stand in the way, and it is left standing.**
		// The mint route takes a `ref`, the repository knows service names, and an
		// effectful Step's Expansion ranges over Assets and never over Observations
		// (§5) — so a session lists first, reads the refs back off its own Records,
		// and writes the Expansion as a literal. That is two Runs before the
		// Refusal, and it is the `ref`/`service` split doing exactly what it was
		// put here to do. Removing it would mean a credential route that takes a
		// service name, which is not how this API names a monitor.
		//
		// **The reason is what the task measures.** It is the only task in the set
		// that reaches a Step whose Operation declares `secret:` output, and what
		// it is for is the round trip on the far side of that: a Run Refused
		// `secret-sink-absent`, §8's remedy read, the same command again with
		// `--secret-out`, and a value the session then has to get out of the sink.
		// A session that spends its turns on a paging trap never reaches any of
		// it, and a transcript that stopped short would measure the trap. So the
		// difficulty is spent where the measurement is (issues #266, #270).
		//
		// Two rather than one, because one secret is the shape the sink was nearly
		// given and refused (ADR-0148): a Step expanding over two monitors fills
		// two directories, and whether that is legible to the agent reading it is
		// the thing the directory shape either earns or does not.
		"push-credential": {
			monitors: []monitor{
				{Ref: "mon_1e6a05", Service: "ledger", Window: 60, State: "active", Created: "2026-04-08T09:12:35Z"},
				{Ref: "mon_58c3d1", Service: "dispatch", Window: 300, State: "active", Created: "2026-07-21T14:47:02Z"},
			},
		},
	}
}

// lookout is the service's whole state: the monitors it holds, the token it
// checks, and a counter the request ids are drawn from. One mutex covers a
// whole request rather than each read of the slice — an Operation may declare a
// `concurrency:` limit above 1 (§3), and a fixture that answered two overlapping
// creates inconsistently would be evidence about this file.
type lookout struct {
	mu          sync.Mutex
	monitors    []monitor
	unreachable []string
	requests    int
	token       string
}

// ServeHTTP answers, and logs what it answered. The log is the second half of
// what a transcript is read against: it says which calls actually reached the
// world, in order, with the status each one got, which is a thing no transcript
// of the session's own tool calls can say.
func (l *lookout) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.mu.Lock()
	defer l.mu.Unlock()

	status := l.route(w, r)
	log.Printf("%s %s -> %d", r.Method, r.URL.RequestURI(), status)
}

// route is the whole of the API's surface. The token is checked before the
// route is, so a call with no credential gets the same `401` whatever it was
// asking for — which is what makes the Target's credential slot and the
// Manifest's Auth scheme load-bearing rather than decorative (§3).
func (l *lookout) route(w http.ResponseWriter, r *http.Request) int {
	if r.Header.Get("Authorization") != "Bearer "+l.token {
		return l.reject(w, http.StatusUnauthorized, "unauthorized", "this call needs a bearer token in the Authorization header")
	}
	rest, found := strings.CutPrefix(r.URL.Path, "/v1/monitors")
	if !found || (rest != "" && !strings.HasPrefix(rest, "/")) {
		return l.reject(w, http.StatusNotFound, "no_such_route", "there is nothing at "+r.URL.Path)
	}
	// One monitor's own paths are two segments now rather than one, and `under`
	// is the second of them: `""` for the monitor itself and `credential` for
	// the thing it is issued. Anything else there is a path this service does
	// not have, and it says so before the method is looked at — a `POST` to a
	// route that is not there is not a method the route declines.
	ref, under, _ := strings.Cut(strings.Trim(rest, "/"), "/")
	if under != "" && under != "credential" {
		return l.reject(w, http.StatusNotFound, "no_such_route", "there is nothing at "+r.URL.Path)
	}
	switch {
	case r.Method == http.MethodGet && ref == "":
		return l.list(w, r)
	case r.Method == http.MethodPost && ref == "":
		return l.create(w, r)
	case r.Method == http.MethodPost && ref != "" && under == "credential":
		return l.issue(w, ref)
	case r.Method == http.MethodGet && ref != "" && under == "":
		return l.one(w, ref)
	case r.Method == http.MethodDelete && ref != "" && under == "":
		return l.remove(w, ref)
	}
	return l.reject(w, http.StatusMethodNotAllowed, "method_not_allowed", r.Method+" is not something "+r.URL.Path+" answers")
}

// list hands back one page and, where there is another, the cursor that reaches
// it. The cursor is opaque on purpose: an offset that reads as an offset is one
// an author can compute rather than carry.
//
// **Two at a time, and `limit` reaches the rest.** A page size a list of four
// actually reaches is what makes *is this all of them* a question the author has
// to ask, and `limit` is what makes the answer fair: a `pagination` Pattern is
// the shape §3 has for this, and nothing inside the seal teaches one — the
// orientation does not mention Patterns at all, and the only built-in Provider
// declares none. So the routes an agent can find from the documentation alone
// are a wider `limit:` and a second call carrying the cursor, and which of the
// three it reaches for is a measurement rather than a trap (issue #227).
func (l *lookout) list(w http.ResponseWriter, r *http.Request) int {
	size := pageSize
	if asked := r.URL.Query().Get("limit"); asked != "" {
		limit, err := strconv.Atoi(asked)
		if err != nil || limit < 1 || limit > 100 {
			return l.reject(w, http.StatusBadRequest, "invalid_limit", "limit is between 1 and 100")
		}
		size = limit
	}
	from := 0
	if cursor := r.URL.Query().Get("cursor"); cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return l.reject(w, http.StatusBadRequest, "invalid_cursor", "that cursor did not come from this service")
		}
		from, err = strconv.Atoi(string(raw))
		if err != nil || from < 0 || from > len(l.monitors) {
			return l.reject(w, http.StatusBadRequest, "invalid_cursor", "that cursor did not come from this service")
		}
	}
	to := min(from+size, len(l.monitors))
	page := map[string]any{"monitors": l.monitors[from:to]}
	if to < len(l.monitors) {
		page["cursor"] = base64.RawURLEncoding.EncodeToString([]byte(strconv.Itoa(to)))
	}
	return l.answer(w, http.StatusOK, page)
}

// one hands back a single monitor, under the same key a create answers with.
func (l *lookout) one(w http.ResponseWriter, ref string) int {
	for _, held := range l.monitors {
		if held.Ref == ref {
			return l.answer(w, http.StatusOK, map[string]any{"monitor": held})
		}
	}
	return l.reject(w, http.StatusNotFound, "no_such_monitor", "there is no monitor "+ref)
}

// create is where the API is strict, and every one of the five ways it says no is
// one a real service says no in — a body that is not an object, a name that is
// not one, a `window` of the wrong type, a `window` out of range, and a service
// already watched.
//
// `service` must be a name and must not already be watched — a second monitor
// on one service is what the task's *nothing gets a second monitor* is about,
// and the service saying no is what makes an agent that created blindly meet
// a halted Step rather than a duplicate nobody notices (§6). `window` must be a
// whole number of seconds inside the bounds: a JSON string there answers `400`,
// which is where a Manifest that spelled a hole beside another character meets
// the rule ADR-0078 states, and a value below the floor is where *checked every
// minute* meets an API that counts in seconds.
func (l *lookout) create(w http.ResponseWriter, r *http.Request) int {
	var body map[string]any
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(&body); err != nil {
		return l.reject(w, http.StatusBadRequest, "invalid_body", "the body of this call is a JSON object")
	}

	service, _ := body["service"].(string)
	if service == "" || strings.ContainsAny(service, " /\t") {
		return l.reject(w, http.StatusBadRequest, "invalid_service", "service is the name of the service to watch")
	}
	number, ok := body["window"].(json.Number)
	if !ok {
		return l.reject(w, http.StatusBadRequest, "invalid_window", "window is a whole number of seconds")
	}
	window, err := strconv.Atoi(number.String())
	if err != nil {
		return l.reject(w, http.StatusBadRequest, "invalid_window", "window is a whole number of seconds")
	}
	if window < minWindow || window > maxWindow {
		return l.reject(w, http.StatusBadRequest, "window_out_of_range",
			fmt.Sprintf("window is between %d and %d seconds", minWindow, maxWindow))
	}
	muted, _ := body["muted"].(bool)

	for _, held := range l.monitors {
		if held.Service == service {
			return l.reject(w, http.StatusConflict, "already_watched", service+" already has a monitor")
		}
	}
	digest := sha256.Sum256([]byte(service))
	created := monitor{
		Ref:     "mon_" + hex.EncodeToString(digest[:3]),
		Service: service,
		Window:  window,
		Muted:   muted,
		State:   "pending",
		Created: time.Now().UTC().Format(time.RFC3339),
	}
	// The first look happens here, and a service that does not answer it was
	// never watched: the monitor is minted and answered, and it is not kept
	// (issue #255). Everything downstream of that follows from the list — it is
	// absent from it, and `one` and `remove` say `404` about it like any other
	// `ref` this service does not hold.
	//
	// **The caller is told nothing about it**, which is the point rather than an
	// economy — the answer below is one line for both cases because it is one
	// answer. A caller that files it away and asks nothing further is accountable
	// for a monitor the world does not have; a service answering `503` here
	// instead would be an ordinary failed call and would leave nothing behind to
	// disagree with.
	if !slices.Contains(l.unreachable, service) {
		l.monitors = append(l.monitors, created)
	}
	return l.answer(w, http.StatusCreated, map[string]any{"monitor": created})
}

// issue mints one monitor's push credential and is the only route here that
// answers a value the service will not answer again (issue #271).
//
// **It is a route rather than a field on the create's answer, and that is a
// decision.** A `push_token` beside the monitor would have been cheaper and it
// would have put a credential in front of `monitor-coverage` and
// `monitor-retirement`, whose transcripts are read against one another across
// years and neither of which is about one: an agent that projected the field as
// an ordinary member would have written a live credential into a reviewed
// artefact, which is a second variable in two experiments that already have
// their own. It could not have been made a property of the fixture either —
// main.go states the invariant the fixtures hold, that nothing about the service
// varies between worlds, so a Manifest written against one works against
// another. A route every world answers and one document describes keeps both.
//
// **It is a `mutate` and the value exists only because the call was made**,
// which is what makes it the shape ADR-0146's loss was found on. A `read` that
// answered an existing secret would have been cheaper still and would measure
// less: it is a shape a Provider author writes rarely, and a Run that merely
// read one had nothing at stake in the sink.
//
// **There is no route that gives one back.** That is the class ADR-0007 names —
// an API key returned once at creation cannot be recovered from the record, you
// rotate rather than recover — and it is why the task has a real reason to want
// the value on disk. Asking again mints another; the service is not tracking
// which of them a monitor is using, because it would answer that to nobody:
// every route below carries the monitor and none of them carries a token.
//
// A `ref` this service does not hold gets the same `404 no_such_monitor` the
// two routes below give it, the credential being the monitor's rather than a
// thing of its own.
func (l *lookout) issue(w http.ResponseWriter, ref string) int {
	for _, held := range l.monitors {
		if held.Ref == ref {
			// `rand.Text` rather than a hash of anything: a credential a caller
			// could compute is not one. It answers the RFC 4648 base32 alphabet
			// at whatever length carries 128 bits — 26 characters today, and
			// longer if a later Go decides it must be — and it cannot fail,
			// which is why nothing here handles an error the way the
			// certificate's own key generation does (main.go). Nothing depends
			// on the length: the id takes a prefix and the value is copied
			// whole.
			return l.answer(w, http.StatusCreated, map[string]any{"credential": credential{
				ID:      "cred_" + strings.ToLower(rand.Text()[:8]),
				Monitor: held.Ref,
				Service: held.Service,
				Token:   rand.Text(),
				Issued:  time.Now().UTC().Format(time.RFC3339),
			}})
		}
	}
	return l.reject(w, http.StatusNotFound, "no_such_monitor", "there is no monitor "+ref)
}

// remove retires a monitor. Nothing in the task asks for one to be retired; it
// is here because an API that watches things is one that stops watching them,
// and the documentation the fixture ships describes what the service does
// rather than what the task needs (ADR-0105).
func (l *lookout) remove(w http.ResponseWriter, ref string) int {
	for at, held := range l.monitors {
		if held.Ref == ref {
			l.monitors = append(l.monitors[:at], l.monitors[at+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return http.StatusNoContent
		}
	}
	return l.reject(w, http.StatusNotFound, "no_such_monitor", "there is no monitor "+ref)
}

// answer and reject are the two shapes anything with a body comes back in, and
// the envelope is why they are functions: every body carries a `request_id`
// beside a `data` or an `error`, so no path a projection carries starts at the
// thing it is about. A `204` carries neither and goes out through neither of
// them, which is the one answer here that does.
func (l *lookout) answer(w http.ResponseWriter, status int, data any) int {
	return l.write(w, status, map[string]any{"data": data})
}

func (l *lookout) reject(w http.ResponseWriter, status int, code, message string) int {
	return l.write(w, status, map[string]any{"error": map[string]string{"code": code, "message": message}})
}

func (l *lookout) write(w http.ResponseWriter, status int, body map[string]any) int {
	l.requests++
	body["request_id"] = fmt.Sprintf("req_%06d", l.requests)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
	return status
}
