package capability_test

import (
	"net/http"
	"slices"
	"testing"

	"github.com/TheLoomLabs/hyper/internal/capability"
)

// The two Auth schemes §12 closes, asserted where they end: on a real request
// arriving at a real server (issue #137).
//
// What is held here is the whole of what a scheme does — a header, its name and
// what stands in front of the credential — and the one thing that is not a
// header: that a Provider naming no `auth:` sends nothing at all.

// TestReadAuth_TheSchemeAManifestNamesAndItsParameters reads §12's two members
// off a Manifest, and the absence that is not a third member.
func TestReadAuth_TheSchemeAManifestNamesAndItsParameters(t *testing.T) {
	for _, c := range []struct {
		name   string
		source string
		want   capability.Auth
		slots  []string
	}{{
		name:   "a bearer token",
		source: "auth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n",
		want:   capability.Auth{Scheme: capability.SchemeHeader, Name: "Authorization", Prefix: "Bearer "},
		slots:  []string{"token"},
	}, {
		name:   "an API key, whose prefix is absent meaning empty",
		source: "auth:\n  header: {name: X-Api-Key}\n",
		want:   capability.Auth{Scheme: capability.SchemeHeader, Name: "X-Api-Key"},
		slots:  []string{"token"},
	}, {
		name:   "basic, which takes no parameters and carries an empty mapping",
		source: "auth:\n  basic: {}\n",
		want:   capability.Auth{Scheme: capability.SchemeBasic},
		slots:  []string{"username", "password"},
	}, {
		name:   "no auth: at all, which is a Provider that sends no credential",
		source: "provider: uptime\n",
		want:   capability.Auth{},
		slots:  nil,
	}} {
		t.Run(c.name, func(t *testing.T) {
			read := capability.ReadAuth(operation(t, c.source))
			if read != c.want {
				t.Errorf("ReadAuth = %+v, want %+v", read, c.want)
			}
			if got := read.Slots(); !slices.Equal(got, c.slots) {
				t.Errorf("Slots = %v, want %v — a scheme's slots are the scheme's, and no Manifest invents one", got, c.slots)
			}
		})
	}
}

// TestPerform_TheHeaderSchemeSendsItsDeclaredNameAndPrefix is `header:` on the
// wire: the header the Manifest named, carrying the prefix it declared
// concatenated verbatim in front of the value the environment held.
//
// The prefix is asserted rather than the token alone because that is what the
// one scheme covers three ways — a bearer token, an API key in any header, and
// a vendor's compound token are one placement rather than three schemes (§12).
func TestPerform_TheHeaderSchemeSendsItsDeclaredNameAndPrefix(t *testing.T) {
	var seen http.Header
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) { seen = r.Header.Clone() })

	auth := capability.ReadAuth(operation(t, "auth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n"))
	call := capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}
	if _, err := call.Perform(t.Context(), dial, instant, auth.Credential(map[string]string{"token": "t0ken"})); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if got, want := seen.Get("Authorization"), "Bearer t0ken"; got != want {
		t.Errorf("Authorization = %q, want %q", got, want)
	}
}

// TestPerform_TheBasicSchemeSendsThePositionItOwns is `basic:` on the wire:
// `Authorization: Basic <base64>` over `username:password`.
//
// It is not a `header:` with a prefix, and this case is where that shows: what
// goes out is a composition of two slots that `hyper` encoded, which is exactly
// the work a `header:` scheme would have left a human doing by hand into an
// environment variable that no longer held what the vendor issued (§12).
func TestPerform_TheBasicSchemeSendsThePositionItOwns(t *testing.T) {
	var seen *http.Request
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) { seen = r })

	auth := capability.ReadAuth(operation(t, "auth:\n  basic: {}\n"))
	slots := map[string]string{"username": "ada", "password": "s3cret"}
	call := capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}
	if _, err := call.Perform(t.Context(), dial, instant, auth.Credential(slots)); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	user, password, supplied := seen.BasicAuth()
	if !supplied {
		t.Fatalf("Authorization = %q, want the position basic: owns", seen.Header.Get("Authorization"))
	}
	if user != "ada" || password != "s3cret" {
		t.Errorf("basic auth = %q/%q, want ada/s3cret", user, password)
	}
}

// TestPerform_AProviderNamingNoSchemeSendsNoHeader is the absence: `auth:` is
// optional, and a Provider omitting it sends no credential — which is what an
// uptime check against a public host is (§3, §12).
func TestPerform_AProviderNamingNoSchemeSendsNoHeader(t *testing.T) {
	var seen http.Header
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) { seen = r.Header.Clone() })

	auth := capability.ReadAuth(operation(t, "provider: uptime\n"))
	call := capability.Call{Host: servedHost, Method: http.MethodGet, Path: "/"}
	if _, err := call.Perform(t.Context(), dial, instant, auth.Credential(nil)); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if got := seen.Get("Authorization"); got != "" {
		t.Errorf("Authorization = %q, want no header at all", got)
	}
}

// TestPerform_TheSchemesHeaderWinsOverAnAuthoredOne holds the position against
// an authored `headers:` entry naming it.
//
// A Manifest naming a position its scheme owns is `manifest-inconsistent` and a
// Run re-runs `check` in full before its first Step, so this is unreachable
// from a reviewed repository (§4, §6). It is held anyway and for the reason the
// five reserved headers are dropped rather than trusted: the position is
// `hyper`'s, and what says so is the code rather than the ordering of two Set
// calls.
func TestPerform_TheSchemesHeaderWinsOverAnAuthoredOne(t *testing.T) {
	var seen http.Header
	dial := serve(t, func(w http.ResponseWriter, r *http.Request) { seen = r.Header.Clone() })

	auth := capability.ReadAuth(operation(t, "auth:\n  header: {name: Authorization, prefix: \"Bearer \"}\n"))
	call := capability.Call{
		Host: servedHost, Method: http.MethodGet, Path: "/",
		Headers: []capability.Parameter{{Name: "Authorization", Value: "Bearer authored"}},
	}
	if _, err := call.Perform(t.Context(), dial, instant, auth.Credential(map[string]string{"token": "t0ken"})); err != nil {
		t.Fatalf("Perform: %v", err)
	}

	if got, want := seen.Get("Authorization"), "Bearer t0ken"; got != want {
		t.Errorf("Authorization = %q, want %q — the position is the scheme's and a Manifest never chooses it", got, want)
	}
}
