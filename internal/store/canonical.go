package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"
	"time"
)

// A Value is a decoded JSON value: a mapping, an array, a string, a number or a
// boolean, and nothing else. There is no null — a field's presence is a fact
// stated by a predicate operator and never by a nullable value (§12), and a
// field a file does not carry is absent from it with nothing standing in its
// place (§7).
//
// The set is closed by the unexported method: no type outside this package can
// be a Value, which is what makes Encode total. An encoder that could be handed
// something it did not understand would have to answer with an error on the one
// path where a caller has nothing useful to do with one — the bytes are already
// being written into a git blob by then.
type Value interface {
	// write appends this value's canonical bytes to dst. The caller has
	// already written whatever stands to the left of the opening token, so
	// write never indents its own first character; indent is the depth the
	// closing token sits at and its own lines nest one deeper.
	write(dst []byte, indent int) []byte
}

// String is a JSON string.
type String string

// Mapping is a JSON object. Its keys are written in Unicode code point order
// rather than in any order a caller could arrange, so that two writers of one
// shape agree on the bytes without agreeing on anything else (§7).
type Mapping map[string]Value

// Number is a JSON number, held as the literal text it was decoded from and
// re-emitted rather than round-tripped through a float where it need not be: an
// integer past a float64's exact range is a Record identity on plenty of
// upstreams, and one that moved under a re-encode would mint a version on every
// Run.
//
// The zero Number is zero, so a Number that reached the encoder unset writes a
// number rather than nothing at all — otherwise the one way a total encoder
// could produce bytes that are not JSON.
type Number struct{ literal string }

// Timestamp is an instant, written as the one timestamp form the Store holds:
// RFC 3339, UTC, Z mandatory, milliseconds always to three digits. It is a
// Value of its own rather than a String a caller formatted, because the width
// being fixed is what makes lexicographic order over a timestamp chronological
// order — the fact the Head derivation rests on — and a caller formatting it
// itself is a caller that can format it differently.
type Timestamp time.Time

// Bool is a JSON boolean.
type Bool bool

// Array is a JSON array. It writes one element per line at the same indent, so
// a set that gains a member gains a line and a git diff of it names what moved
// rather than reporting that one long line changed (§7).
type Array []Value

// SecretMarker stands where a value a Manifest declared secret would have gone:
// no digest, no length, no sibling list of what was suppressed, so no secret
// reaches the Store at all (§7, ADR-0007).
//
// It is a constant, which is what keeps "a version is minted only where the
// bytes moved" honest: a rotated secret writes these same bytes and correctly
// mints nothing.
const SecretMarker = "<secret>"

// Secret is the value SecretMarker is written in place of. It takes the secret
// and returns a value that has forgotten it — there is no field to read it back
// out of and no method that returns it — so a value routed through here cannot
// reach the bytes by some later path that only meant to be helpful.
//
// Which fields are secret is a Manifest's fact and arrives here as an input
// rather than a derivation. The encoder holds the constant and nothing else.
func Secret(Value) Value { return secret{} }

// secret is Secret's marker: a value carrying nothing, because carrying the
// secret is the thing it exists not to do.
type secret struct{}

// Always marks a value written even where the absence rule would drop the key
// holding it. §7 names three keys that earn it and each is argued for on its own
// terms: dry_run, always and including false, because a reader that takes its
// absence for false permanently refuses every run-once Step in the Procedure it
// rehearsed; members, whenever the identity digest moved and the empty list
// included, because absence there already means the digest did not move; and
// expanded_to, whenever a selector exists and the empty list included, because
// an Expansion that resolved to nothing is not a Step with no selector.
//
// The encoder holds the rule and not the three names. Which key is an exception
// is a fact about the shape being written, and a list of names compiled in here
// would be a second place for a shape to disagree with itself.
func Always(v Value) Value { return always{v} }

// always is Always's marker. It carries the value rather than replacing it, so
// what an exception key writes is what it would have written anyway — the
// exception is to the dropping, never to the encoding.
type always struct{ Value }

// omitted answers whether the absence rule drops a key holding this value: a
// mapping or an array that would be written empty, and nothing else.
//
// It is recursive through a mapping, because a mapping whose every key was
// dropped is itself the empty mapping the rule is about — "would be" rather than
// "is". It is deliberately not recursive through an array: the rule is stated
// over a key, and dropping an element would move every element after it.
func omitted(v Value) bool {
	switch v := v.(type) {
	case Mapping:
		return len(v.written()) == 0
	case Array:
		return len(v) == 0
	default:
		return false
	}
}

// ParseNumber reads a JSON number literal. It is the door a decoded value comes
// through, and the grammar it checks is JSON's own rather than Go's: strconv
// accepts `0x1p3`, `Inf` and `1_000`, none of which a JSON decoder ever handed
// anybody, and all of which would reach the Store as something else.
//
// It answers an error where the literal is not a JSON number, and where it is
// one no float64 can hold: `1e400` is a number hyper cannot write down, and
// saying so at the door is better than writing `+Inf` onto a branch.
func ParseNumber(literal string) (Number, error) {
	if !isJSONNumber(literal) {
		return Number{}, fmt.Errorf("%q is not a JSON number", literal)
	}
	if _, err := strconv.ParseFloat(literal, 64); err != nil {
		return Number{}, fmt.Errorf("%q is not a number a float64 can hold: %w", literal, err)
	}
	return Number{literal: literal}, nil
}

// Int is a Number from an integer hyper counted itself — a schema version, a
// Step's position — where there is no literal to parse and no error to answer.
func Int(i int64) Number {
	return Number{literal: strconv.FormatInt(i, 10)}
}

// Encode writes a value in §7's canonical encoding: UTF-8, LF endings, a
// trailing LF, two-space indent, keys sorted by Unicode code point, and no
// trailing whitespace on any line.
//
// The encoding is a property of a value and a file is the case where the value
// is the whole file (ADR-0079). A value encoded on its own is encoded exactly as
// it would be were it that file's whole content — an array alone opens at no
// indent and writes its elements two spaces in — which is what makes the
// identity digest computable at all.
//
// This is not §8's row stream. That wire is compact, keyed in the renderer's
// order and hashed by nobody; it lives in internal/render and neither encoding
// reaches for the other.
func Encode(v Value) []byte {
	return append(v.write(nil, 0), '\n')
}

// IdentityDigest is the digest of a set of names: sha256: over the canonical
// encoding of the sorted array, trailing LF included — the array as it would be
// written alone, at no indent, and never as it sits inside a Journal entry where
// members carries four spaces of it (§7, ADR-0079).
//
// Sorting is by Unicode code point, the same rule the encoding already uses for
// keys and §6 uses for an Expansion, which makes the digest a fact about the set
// rather than about the order a response happened to arrive in.
//
// A name repeated is one name: the argument carries a set, Go having no type
// that says so, and a duplicate that reached the digest would give one set two
// digests — which is a spurious version minted on the next Run, the exact
// failure the digest exists to prevent.
func IdentityDigest(names []string) string {
	sorted := slices.Compact(slices.Sorted(slices.Values(names)))

	array := make(Array, len(sorted))
	for i, name := range sorted {
		array[i] = String(name)
	}

	// sha256: is inline rather than shared with internal/artefact's
	// ManifestDigest: the Store sits beneath the layer that loads artefacts
	// and does not import it, and two lines of agreement cost less than the
	// edge that would remove them (§7, ADR-0047).
	sum := sha256.Sum256(Encode(array))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func (s String) write(dst []byte, _ int) []byte {
	return quote(dst, string(s))
}

func (n Number) write(dst []byte, _ int) []byte {
	return append(dst, n.text()...)
}

// text is the number as the Store writes it. §7 says "the shortest decimal that
// round-trips", which is under-determined at the exponent threshold where
// `1e+06` is shorter than `1000000` and both round-trip. #124 resolves it: an
// integer is written as its decimal digits, exactly and at whatever width it
// takes, and every other number as ECMAScript's Number::toString would write it
// — the form the browser shows the reviewer §7 wrote the whole encoding for.
//
// The two agree wherever both could apply: toString of 1e3 is `1000` and of 1.0
// is `1`. So this is one convention with the exactness kept where a float64
// would lose it, rather than two conventions meeting at a seam.
func (n Number) text() string {
	if n.literal == "" {
		return "0"
	}
	if exact, ok := new(big.Int).SetString(n.literal, 10); ok {
		return exact.String()
	}
	value, err := strconv.ParseFloat(n.literal, 64)
	if err != nil {
		// Unreachable: every Number was built from a literal that parsed.
		return "0"
	}
	return ecmaScriptString(value)
}

// ecmaScriptString writes a finite float64 the way ECMAScript's Number::toString
// does: the shortest round-tripping digits, placed by that specification's own
// thresholds — plainly while the decimal exponent is within (-7, 21], and with
// an exponent outside it.
//
// No strconv format agrees. 'g' switches to an exponent at a different threshold
// and pads it to two digits, so it writes `1e-07` where ECMAScript writes
// `1e-7`; on a branch where a version is minted where the bytes moved, one byte
// is a version.
//
// Zero is answered first and without its sign, ECMAScript's own step: -0 is a
// float64 the shortest-digits form writes as `-0`, which the placement below
// would then punctuate into `-.0`.
func ecmaScriptString(x float64) string {
	if x == 0 {
		return "0"
	}
	if x < 0 {
		return "-" + ecmaScriptString(-x)
	}

	// FormatFloat with 'e' and precision -1 is the shortest round-tripping
	// digit string, which is exactly the s and k the specification's step
	// asks for; its exponent is one less than the specification's n.
	mantissa, exponent, _ := strings.Cut(strconv.FormatFloat(x, 'e', -1, 64), "e")
	power, _ := strconv.Atoi(exponent)
	digits := strings.Replace(mantissa, ".", "", 1)
	k, n := len(digits), power+1

	switch {
	case k <= n && n <= 21:
		return digits + strings.Repeat("0", n-k)
	case 0 < n && n <= 21:
		return digits[:n] + "." + digits[n:]
	case -6 < n && n <= 0:
		return "0." + strings.Repeat("0", -n) + digits
	case k == 1:
		return digits + exponentSuffix(n-1)
	default:
		return digits[:1] + "." + digits[1:] + exponentSuffix(n-1)
	}
}

// exponentSuffix writes an exponent the way ECMAScript does: the sign is always
// written, and the digits are as narrow as the value, so it is `e+21` and `e-7`
// rather than the two-digit form every printf in sight would give.
func exponentSuffix(power int) string {
	if power < 0 {
		return "e-" + strconv.Itoa(-power)
	}
	return "e+" + strconv.Itoa(power)
}

// isJSONNumber answers whether a literal is a number under JSON's grammar and no
// other: an optional minus, an integer part carrying no leading zero, an
// optional fraction of at least one digit, and an optional exponent of at least
// one.
func isJSONNumber(literal string) bool {
	rest := strings.TrimPrefix(literal, "-")

	whole, rest := leadingDigits(rest)
	if whole == "" || (len(whole) > 1 && whole[0] == '0') {
		return false
	}

	if after, found := strings.CutPrefix(rest, "."); found {
		fraction, remainder := leadingDigits(after)
		if fraction == "" {
			return false
		}
		rest = remainder
	}

	if len(rest) > 0 && (rest[0] == 'e' || rest[0] == 'E') {
		after := rest[1:]
		if len(after) > 0 && (after[0] == '+' || after[0] == '-') {
			after = after[1:]
		}
		exponent, remainder := leadingDigits(after)
		if exponent == "" {
			return false
		}
		rest = remainder
	}

	return rest == ""
}

// leadingDigits cuts the leading run of ASCII digits from s.
func leadingDigits(s string) (digits, rest string) {
	i := 0
	for i < len(s) && '0' <= s[i] && s[i] <= '9' {
		i++
	}
	return s[:i], s[i:]
}

func (t Timestamp) write(dst []byte, _ int) []byte {
	// The fraction is cut rather than rounded, which is what Format does and
	// what the fixed width wants: a rounded 23:59:59.9999 would name the
	// next day, and a timestamp in the Store is when something happened.
	return quote(dst, time.Time(t).UTC().Format("2006-01-02T15:04:05.000")+"Z")
}

func (secret) write(dst []byte, _ int) []byte {
	return quote(dst, SecretMarker)
}

func (b Bool) write(dst []byte, _ int) []byte {
	if b {
		return append(dst, "true"...)
	}
	return append(dst, "false"...)
}

// written is the mapping's keys in the order they go out: sorted by Unicode code
// point, and only those the absence rule keeps.
func (m Mapping) written() []string {
	keys := make([]string, 0, len(m))
	for key, value := range m {
		if !omitted(value) {
			keys = append(keys, key)
		}
	}
	slices.Sort(keys)
	return keys
}

func (m Mapping) write(dst []byte, indent int) []byte {
	keys := m.written()
	if len(keys) == 0 {
		return append(dst, '{', '}')
	}
	dst = append(dst, '{', '\n')
	for i, key := range keys {
		dst = indentBy(dst, indent+1)
		dst = quote(dst, key)
		dst = append(dst, ':', ' ')
		dst = m[key].write(dst, indent+1)
		if i < len(keys)-1 {
			dst = append(dst, ',')
		}
		dst = append(dst, '\n')
	}
	dst = indentBy(dst, indent)
	return append(dst, '}')
}

func (a Array) write(dst []byte, indent int) []byte {
	if len(a) == 0 {
		return append(dst, '[', ']')
	}
	dst = append(dst, '[', '\n')
	for i, element := range a {
		dst = indentBy(dst, indent+1)
		dst = element.write(dst, indent+1)
		if i < len(a)-1 {
			dst = append(dst, ',')
		}
		dst = append(dst, '\n')
	}
	dst = indentBy(dst, indent)
	return append(dst, ']')
}

// indentBy writes one line's leading whitespace: two spaces per level, and
// nothing at all at level zero, so no line the encoder writes can carry trailing
// whitespace by having been indented and then left empty.
func indentBy(dst []byte, level int) []byte {
	for range level {
		dst = append(dst, ' ', ' ')
	}
	return dst
}

// lowerHex is the digit set every hexadecimal escape the encoder writes is drawn
// from. Every hexadecimal digit hyper writes is lowercase (§7, ADR-0079); the
// one exception in the corpus is a percent-escape in the Store path grammar,
// which RFC 3986 makes uppercase and which is not JSON.
const lowerHex = "0123456789abcdef"

// quote writes a JSON string. Escaping is the minimum JSON requires — the quote,
// the backslash and the control characters, using the short form where one
// exists — and every character outside ASCII is written as itself in UTF-8
// rather than as an escape, so a Record whose name carries an umlaut is legible
// in a browser and hashes as what it reads as (§7).
//
// It walks bytes rather than runes precisely so that it cannot rewrite one: a
// UTF-8 byte is passed through untouched, which is what keeps the bytes it
// hashes the bytes it was handed.
func quote(dst []byte, s string) []byte {
	dst = append(dst, '"')
	for i := range len(s) {
		switch c := s[i]; c {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\b':
			dst = append(dst, '\\', 'b')
		case '\f':
			dst = append(dst, '\\', 'f')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			if c < 0x20 {
				dst = append(dst, '\\', 'u', '0', '0', lowerHex[c>>4], lowerHex[c&0xf])
				continue
			}
			dst = append(dst, c)
		}
	}
	return append(dst, '"')
}
