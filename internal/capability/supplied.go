package capability

import (
	"bytes"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
)

// A supplied response object: §12's object read out of JSON a caller handed
// `hyper` rather than assembled from a call `hyper` made (issue #230, ADR-0108).
//
// **Nothing here performs anything.** That is the whole of why it is
// admissible: the response did not arrive through `hyper`, so there is no
// credential in a request, no host reached, no wire to render that `hyper`
// fetched, and nothing for ADR-0017's argument to bite on. What is left is the
// half a Manifest author cannot get from any other client — whether the paths
// their `record:` block writes address anything in the object a projection
// actually reads from (§3, ADR-0040).
//
// It is read against the same closed member set the assembled object carries
// and built in the same member order, because it is the same object. An author
// who supplied `data` at the top level wrote a path root that no Capability
// has, and being told so here is the cheapest that fact ever gets: the
// alternative is a projection that resolves against a shape `hyper` never
// builds, which is a Manifest that works on this page and on nothing else.

// The two Capabilities a response object belongs to, named as §12 names them.
// A supplied object is read against the one the Operation's request block
// declares, and never against a Capability the caller chose.
const (
	CapabilityHTTP  = "http"
	CapabilityShell = "shell"
)

// suppliedMember is one member of a response object as a caller may spell it:
// its name, whether the object is refused without it, and what shape its value
// must take.
type suppliedMember struct {
	name string
	// required marks the one member of each object that is a fact about
	// the **call** rather than about the answer — `host` for `http` and
	// `command` for `shell`. It is the member that survives where nothing
	// came back at all (§12, ADR-0050), so an object without it is one no
	// call could have produced, and a projection reading `$.host` against
	// it would go quiet for a reason the real call would never have.
	required bool
	// read turns the supplied JSON into the value the assembled object
	// holds at that position, so that a path resolving here resolves
	// against the same Go value it would resolve against after a call.
	read func(json.RawMessage) (any, error)
}

// httpMembers and shellMembers are §12's two objects, in §12's own order, which
// is the order they are built and rendered in. The order is the table's rather
// than the caller's: a supplied object is the object, and two authors spelling
// one response with their keys two ways read one rendering back.
var httpMembers = []suppliedMember{
	{name: MemberHost, required: true, read: readText},
	{name: MemberStatus, read: readInteger},
	{name: MemberHeaders, read: readHeaders},
	{name: MemberBody, read: readBodyValue},
	{name: MemberTLS, read: readTLS},
}

var shellMembers = []suppliedMember{
	{name: MemberCommand, required: true, read: readText},
	{name: MemberExitCode, read: readInteger},
	{name: MemberStdout, read: readText},
	{name: MemberStderr, read: readText},
}

// tlsMembers is the `tls` member's own four, read the same way one level down.
// It is a nested object rather than a mapping of anything, so an unknown member
// here is refused exactly as one at the top level is.
var tlsMembers = []suppliedMember{
	{name: MemberNotAfter, read: readText},
	{name: MemberDaysLeft, read: readInteger},
	{name: MemberSubject, read: readText},
	{name: MemberIssuer, read: readText},
}

// ReadObject reads a supplied response object for one Capability, and answers
// the fault a caller is owed where it cannot.
//
// Every member is optional but one. A response that carried no body, no
// headers and no certificate is an ordinary answer — a site that is down
// answers with a status and nothing else — and absence is a value a projection
// reads (§12). What the object may not do is carry a member no Capability has,
// or hold one at a shape the assembled object never holds: both are a path
// root that would resolve here and nowhere else.
func ReadObject(capability string, supplied []byte) (Object, error) {
	members, named := objectMembers(capability)
	if !named {
		return nil, fmt.Errorf("no response object belongs to the %s Capability", capability)
	}
	return readObject("", members, supplied)
}

// objectMembers is the member table one Capability's object is read against.
func objectMembers(capability string) ([]suppliedMember, bool) {
	switch capability {
	case CapabilityHTTP:
		return httpMembers, true
	case CapabilityShell:
		return shellMembers, true
	default:
		return nil, false
	}
}

// readObject reads one object against a member table: the top-level response
// object with an empty prefix, and the `tls` member under its own name.
//
// The members come back in the **table's** order rather than the JSON's, and
// the ones the JSON left out do not come back at all — the ordinary absence
// rule, which is what makes *resolved to nothing* and *resolved to null* two
// answers a projection tells apart (§7, §12).
func readObject(prefix string, members []suppliedMember, supplied []byte) (Object, error) {
	held, err := readMapping(prefix, supplied)
	if err != nil {
		return nil, err
	}

	// Sorted, because a map has no order to read: an object carrying two
	// members no response object has would otherwise be reported against
	// whichever key came first, and one file would answer two ways.
	for _, name := range slices.Sorted(maps.Keys(held)) {
		if !slices.ContainsFunc(members, func(m suppliedMember) bool { return m.name == name }) {
			return nil, fmt.Errorf("%s%s: no response object carries that member — the members are %s",
				prefix, name, spelledMembers(members))
		}
	}

	object := Object{}
	for _, member := range members {
		raw, carried := held[member.name]
		if !carried {
			if member.required {
				return nil, fmt.Errorf("%s%s: the response object always carries it, so an object without one is an object no call could have produced",
					prefix, member.name)
			}
			continue
		}
		value, err := member.read(raw)
		if err != nil {
			return nil, fmt.Errorf("%s%s: %w", prefix, member.name, err)
		}
		object = append(object, Member{Name: member.name, Value: value})
	}
	return object, nil
}

// readMapping decodes one JSON object, holding its numbers as their literals
// for the reason parseBody does: an integer past a float64's exact range is a
// Record identity on plenty of upstreams, and one that moved under a re-encode
// would be a different Record here than it is after a call.
func readMapping(prefix string, supplied []byte) (map[string]json.RawMessage, error) {
	decoder := json.NewDecoder(bytes.NewReader(supplied))
	decoder.UseNumber()

	var held map[string]json.RawMessage
	if err := decoder.Decode(&held); err != nil {
		return nil, fmt.Errorf("%sis not a JSON object: %w", positioned(prefix), err)
	}
	if decoder.More() {
		return nil, fmt.Errorf("%scarries a second value after the object, and a response object is one object", positioned(prefix))
	}
	if held == nil {
		return nil, fmt.Errorf("%sis null, and a response object is an object", positioned(prefix))
	}
	return held, nil
}

// positioned is how a fault about the object as a whole names it: by the member
// it sits under, or as *the supplied response* where it is the top level.
func positioned(prefix string) string {
	if prefix == "" {
		return "the supplied response "
	}
	return strings.TrimSuffix(prefix, ".") + " "
}

// spelledMembers is the member table written out, which is what a caller who
// named something else is told. It is the namespace and never a near miss, on
// the rule every name in this tool resolves under (ADR-0047).
func spelledMembers(members []suppliedMember) string {
	names := make([]string, 0, len(members))
	for _, member := range members {
		names = append(names, member.name)
	}
	return strings.Join(names, ", ")
}

// readText is a member the object holds as a string.
func readText(raw json.RawMessage) (any, error) {
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return nil, fmt.Errorf("the object holds a string here")
	}
	return text, nil
}

// readInteger is a member the object holds as an int. `status`, `exit_code` and
// `days_left` are counts and codes rather than measurements, and a projection
// comparing one against a predicate's operand compares integers (§12).
func readInteger(raw json.RawMessage) (any, error) {
	// It decodes into `any` rather than into a json.Number, because the
	// decoder reads a quoted number into one: `"200"` would arrive as the
	// int 200, and a supplied object would then hold at that position what
	// no call ever puts there. What is read here is the JSON's own type,
	// which is the rule a value is read by everywhere in this tool
	// (ADR-0081).
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var held any
	if err := decoder.Decode(&held); err != nil {
		return nil, fmt.Errorf("the object holds an integer here")
	}
	number, isNumber := held.(json.Number)
	if !isNumber {
		return nil, fmt.Errorf("the object holds an integer here")
	}
	whole, err := number.Int64()
	if err != nil {
		return nil, fmt.Errorf("the object holds an integer here, and %s is not one", number)
	}
	return int(whole), nil
}

// readHeaders is the `headers` member: a mapping of header name to value, one
// value per name.
//
// The names are lowered exactly as they are off the wire, because that is what
// makes one path mean one thing: a header name is case-insensitive on the wire
// and a path is exact (§12). An author who supplied `Content-Type` gets the
// member their path reaches rather than one it does not, which is the same
// lowering a call would have done to the same bytes.
func readHeaders(raw json.RawMessage) (any, error) {
	var supplied map[string]string
	if err := json.Unmarshal(raw, &supplied); err != nil {
		return nil, fmt.Errorf("the object holds a mapping of header name to value here, one value per name")
	}

	lowered := make(map[string]string, len(supplied))
	// claimed is which spelling took each lowered name, so the fault below
	// can name both. The walk is over the sorted names because a map has no
	// order to read, and a fault naming whichever key came first would be one
	// two runs of one file report two ways.
	claimed := make(map[string]string, len(supplied))
	for _, name := range slices.Sorted(maps.Keys(supplied)) {
		// Two spellings that lower onto one name are refused rather than
		// collapsed. A call cannot produce the pair — the wire's own rule
		// joins repeated field lines into one value before the object is
		// built (§12) — so a file carrying both is a file no response could
		// have been, and taking either would make what a path resolves to
		// depend on which key a map handed over first.
		key := strings.ToLower(name)
		if first, taken := claimed[key]; taken {
			return nil, fmt.Errorf("%s and %s are one header name lowered, and the object holds one value per name",
				first, name)
		}
		claimed[key] = name
		lowered[key] = supplied[name]
	}
	return lowered, nil
}

// readBodyValue is the `body` member: whatever the response's JSON parsed to,
// carried through as it was supplied. It is the one position with no shape of
// its own — a body is an object on most APIs, a list on some, and a bare scalar
// on a few, and what a path reads off each is the grammar's business (§12).
func readBodyValue(raw json.RawMessage) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var body any
	if err := decoder.Decode(&body); err != nil {
		return nil, fmt.Errorf("the object holds the parsed JSON body here")
	}
	return body, nil
}

// readTLS is the `tls` member: the four facts a peer certificate carries, read
// against their own table one level down.
func readTLS(raw json.RawMessage) (any, error) {
	return readObject(MemberTLS+".", tlsMembers, raw)
}
