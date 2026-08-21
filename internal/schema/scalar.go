package schema

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// Scalar is one value read against the type declared at its position: not the
// characters it arrived as, but §12's two columns for that type — the text it
// leaves as in a `path:`, `query:`, `headers:` or `command:` position, and the
// JSON token it leaves as in a `body:` one.
//
// It is one value with two renderings rather than two values, because §12
// states one table with two columns and the type decides both: `integer` is
// decimal digits in a path and a JSON number in a body, and nothing about the
// value differs between them. A caller holding one of these has already had
// the reading performed and cannot re-read it differently.
//
// The reading is ADR-0081's, stated once by readsAs above and performed here:
// the characters are read *as* the declared type rather than compared with a
// type of their own, so the quoting YAML required is lexical and `"2592000"`
// and `2592000` are one value at an `integer` position. What this adds to
// readsAs is the answer — a check needs to know only whether a value reads, and
// a Run needs what it read to.
type Scalar struct {
	// text is §12's text column, which is not always the characters that
	// were read: `0755` at an `integer` position is the integer 755 and its
	// text is `755`, and `1.0` at a `number` position is `1`. Canonical
	// here rather than at the sink is what keeps one value from reaching
	// the wire two ways (§7, ADR-0079).
	text string
	// jsonString says the body sink writes this value as a JSON string.
	// It is the type's fact and never the value's: `duration` and
	// `timestamp` are JSON strings carrying digits, and `boolean` is a bare
	// token carrying letters, so nothing about the characters decides it.
	jsonString bool
}

// Text is what the value writes in a `path:`, `query:`, `headers:` or
// `command:` position — §12's second column, where every type is text on the
// wire and there is no other type to carry into.
func (s Scalar) Text() string { return s.text }

// JSON is what the value writes in a `body:` position — §12's first column, as
// one JSON token: a quoted string, or the bare literal a number and a boolean
// are. It is a token rather than a value because a body is serialised compact
// with its keys in the order they were authored (§3), which is not an encoding
// encoding/json can be asked for.
func (s Scalar) JSON() string {
	if !s.jsonString {
		return s.text
	}
	// encoding/json rather than strconv.Quote, whose escaping is Go's
	// rather than JSON's — and with HTML escaping off, which is the rule
	// wherever hyper writes JSON: a value carrying a & or a < reaches the
	// server as it was authored (§8, internal/render).
	var quoted strings.Builder
	enc := json.NewEncoder(&quoted)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s.text); err != nil {
		// Unreachable: a Go string always encodes, invalid UTF-8
		// included, which encoding/json writes as the replacement
		// character rather than refusing.
		return `""`
	}
	return strings.TrimSuffix(quoted.String(), "\n")
}

// ReadScalar reads value against the type declared at its position and answers
// what it read to, or false where the characters will not read as that type —
// which is `schema-mismatch` at every authored position (§4, ADR-0081) and a
// usage error at a Probe's `--input`, where nothing authored declined (§9,
// ADR-0060).
//
// object and array read as nothing at all: both are refused at both sinks,
// both being reached through a hole and a hole filling a scalar position
// (§12, ADR-0078). The one place an `array` input reaches anything is the
// `shell` Capability's argv, which is no sink on that table — `hyper` execs
// the list rather than serialising it.
func ReadScalar(t Type, value string) (Scalar, bool) {
	switch t {
	case String:
		return Scalar{text: value, jsonString: true}, true
	case Duration:
		// The authored form, byte-identical: `14d` renders back as it was
		// written, which is what makes a duration a value the format can
		// compare without normalising (§12).
		if !durationPattern.MatchString(value) {
			return Scalar{}, false
		}
		return Scalar{text: value, jsonString: true}, true
	case Timestamp:
		if !readsAsTimestamp(value) {
			return Scalar{}, false
		}
		return Scalar{text: value, jsonString: true}, true
	case Boolean:
		if !booleanPattern.MatchString(value) {
			return Scalar{}, false
		}
		return Scalar{text: value}, true
	case Integer:
		// The pattern first, so that the exactness below is asked only of
		// a value this position admits: big.Int reads `0x10` and `_1`,
		// which §12's integer is not.
		if !integerPattern.MatchString(value) {
			return Scalar{}, false
		}
		exact, ok := new(big.Int).SetString(value, 10)
		if !ok {
			return Scalar{}, false
		}
		return Scalar{text: exact.String()}, true
	case Number:
		if !numberPattern.MatchString(value) {
			return Scalar{}, false
		}
		text, ok := store.NumberText(value)
		if !ok {
			return Scalar{}, false
		}
		return Scalar{text: text}, true
	default:
		return Scalar{}, false
	}
}
