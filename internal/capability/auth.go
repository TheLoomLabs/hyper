package capability

import (
	"encoding/base64"

	"gopkg.in/yaml.v3"
)

// The two Auth schemes §12 closes, and the whole of what one does to a request
// (§3, §12, ADR-0031, issue #137).
//
// **An Auth scheme is a header and a placement, never a protocol.** Nothing
// here fetches, exchanges, refreshes or signs: a scheme decorates a request
// `hyper` was already making, and that is the sentence that fixes the
// membership at two rather than leaving it to accumulate.
//
// The **position is the scheme's** and a Manifest never chooses it. That is
// what closure buys: a credential is suppressed by the position it occupies
// rather than by scanning a rendering for something that looks like one
// (ADR-0007). A Manifest supplies the parameters, a Target declaration names
// the environment variable each of the scheme's slots resolves from, and
// `hyper` writes the header — so a Provider author never handles a secret and
// no artefact ever holds one.

// The two schemes, spelled as a Manifest's `auth:` names them. They are
// constants here as well as in internal/artefact because the two packages ask
// different questions of the same closed set: that one refuses a Manifest
// naming neither, and this one fills the one it named.
const (
	// SchemeHeader is `header:` — parameters `name:` and `prefix:`, and one
	// slot, `token`.
	SchemeHeader = "header"
	// SchemeBasic is `basic:` — no parameters, and two slots, `username`
	// and `password`.
	SchemeBasic = "basic"
)

// The credential slots each scheme owns. Neither a Manifest nor a Target
// declaration invents one — a Provider author can no more mint a slot than mint
// an `error_code` (§12, ADR-0004).
const (
	slotToken    = "token"
	slotUsername = "username"
	slotPassword = "password"
)

// basicHeader is the header `basic:` owns. It is not a parameter: the scheme
// takes none, and the position being the scheme's is the whole of what makes a
// credential suppressible by position (§12).
const basicHeader = "Authorization"

// Auth is the scheme a Manifest names and the parameters it supplied: which
// header the credential lands in, and what goes in front of it.
//
// Scheme is "" where the Manifest declares no `auth:` at all, which is a
// Provider that sends no credential — an uptime check against a public host —
// and is what `local` is. Absence is not a third member of the set: a scheme is
// a way of authenticating a request, and not authenticating one is not a way of
// doing it (§12).
type Auth struct {
	// Scheme is SchemeHeader, SchemeBasic, or "" where none was declared.
	Scheme string
	// Name is `header:`'s `name:` parameter, and Prefix its `prefix:`,
	// which is optional and absent meaning empty. Both are empty under
	// `basic:`, which takes no parameters at all.
	Name, Prefix string
}

// Slots is the credential slots this scheme requires, in §12's own order, and
// none where the Manifest declared no scheme.
//
// It is the scheme's list and never the Target declaration's: a Target may
// carry more slots than any one Provider needs, which is what lets one
// declaration serve a `header:` Provider and a `basic:` Provider at once (§3).
func (a Auth) Slots() []string {
	switch a.Scheme {
	case SchemeHeader:
		return []string{slotToken}
	case SchemeBasic:
		return []string{slotUsername, slotPassword}
	}
	return nil
}

// Credential composes the header this scheme sends from the values its slots
// resolved to, and answers the empty Credential where the Manifest declared no
// scheme.
//
// A slot the mapping does not hold composes into the empty string rather than
// declining, because declining here would be a second reading of a question
// §6's credential pass already answered: presence is resolved once, before Step
// 1, and every unfilled slot Refuses there — `credential-absent` where the
// environment does not hold the variable, `credential-empty` where it holds it
// and sets it to nothing. By the time a request is being built there is nothing
// left to find.
//
// That second code is what keeps this composition honest. A slot resolving to
// the empty string used to reach here and compose into a header that was present
// and blank — `Bearer ` with nothing after it — which the endpoint answered `401`
// and hyper recorded as the world resisting. What had happened is that the
// invocation was never ready, and the gate says so now (§6, §9, ADR-0145).
func (a Auth) Credential(slots map[string]string) Credential {
	switch a.Scheme {
	case SchemeHeader:
		return Credential{name: a.Name, value: a.Prefix + slots[slotToken]}
	case SchemeBasic:
		pair := slots[slotUsername] + ":" + slots[slotPassword]
		return Credential{name: basicHeader, value: "Basic " + base64.StdEncoding.EncodeToString([]byte(pair))}
	}
	return Credential{}
}

// Credential is one request's credential, composed: the header `hyper` writes,
// and what it writes into it.
//
// Every member is unexported and there is no accessor, no String and no
// MarshalJSON. It is handed to Perform and reaches the wire, and there is no
// route by which it reaches a file, a row, a rendering or a log line — which is
// ADR-0007 held by the shape of the value rather than by every surface
// remembering. It is deliberately **not** a member of Call: a Call is what a
// caller may hold, compare and describe, and a credential is neither.
type Credential struct{ name, value string }

// declared says the credential is one to send. The zero Credential is a
// Provider that names no scheme, and writing an empty header for it would be a
// request no artefact describes.
func (c Credential) declared() bool { return c.name != "" }

// ReadAuth reads a Manifest's `auth:` block off its own root: the scheme it
// names, and the parameters it supplied.
//
// It judges nothing and drops what it cannot read, which is the rule every
// reader in this package and in internal/artefact follows: an `auth:` naming
// neither of §12's two schemes, or naming a header `hyper` computes for itself,
// is `check`'s to report and never a performer's to guess at (§4, ADR-0064).
// A Run re-runs `check` in full before its first Step, so nothing that reaches
// here has gone unreviewed (§6).
func ReadAuth(manifest *yaml.Node) Auth {
	block := mappingValue(manifest, "auth")
	if block == nil || block.Kind != yaml.MappingNode {
		return Auth{}
	}
	if header := mappingValue(block, SchemeHeader); header != nil {
		return Auth{
			Scheme: SchemeHeader,
			Name:   scalar(mappingValue(header, "name")),
			Prefix: scalar(mappingValue(header, "prefix")),
		}
	}
	if mappingValue(block, SchemeBasic) != nil {
		return Auth{Scheme: SchemeBasic}
	}
	return Auth{}
}
