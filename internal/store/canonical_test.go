package store_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// §7 publishes two identity digests and prints the exact bytes one of them is
// taken over. They are the acceptance test for the whole encoding rather than
// for the digest alone: they fail loudly on an escaped umlaut, a wrong indent, a
// missing trailing LF, a locale-dependent sort or a code-point order got
// backwards (ADR-0079, issue #127).
//
// The values here are §7's own, copied from the spec rather than computed the
// way the code computes them, and both were reproduced under `sha256sum` while
// #124 was written.

func TestIdentityDigestOfThePublishedSet(t *testing.T) {
	const want = "sha256:a118a517431e241eac83559919ae969346bf5a3bf6e06c6db3e636f378fcdf12"

	// Deliberately not in sorted order: the digest is a fact about the set,
	// never about the order a response happened to arrive in.
	got := store.IdentityDigest([]string{"ci-x86", "über-vm", "ci-macos", "ci-riscv"})

	if got != want {
		t.Errorf("identity digest = %q, want %q", got, want)
	}
}

func TestIdentityDigestOfTheEmptySet(t *testing.T) {
	const want = "sha256:37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570"

	if got := store.IdentityDigest(nil); got != want {
		t.Errorf("identity digest of the empty set = %q, want %q", got, want)
	}
}

// The encoding, rule by rule. §7 states it in full because it is not
// presentation — a version is minted only where the bytes moved — so every rule
// is asserted over bytes rather than over a decoded shape.

func TestEncodeSortsKeysByCodePointAndNestsTwoSpaces(t *testing.T) {
	// Zebra before apple in the literal, and `Z` before `a` in the output:
	// code point order, which is not what a case-folding sort would give.
	got := string(store.Encode(store.Mapping{
		"zebra": store.String("last"),
		"Zebra": store.String("first"),
		"apple": store.Mapping{
			"inner": store.Array{store.String("one"), store.String("two")},
		},
	}))

	const want = `{
  "Zebra": "first",
  "apple": {
    "inner": [
      "one",
      "two"
    ]
  },
  "zebra": "last"
}
`
	if got != want {
		t.Errorf("encoded mapping:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeWritesTheEmptyMappingAndArrayInline(t *testing.T) {
	for name, tc := range map[string]struct {
		value store.Value
		want  string
	}{
		"the empty mapping alone": {store.Mapping{}, "{}\n"},
		"the empty array alone":   {store.Array{}, "[]\n"},
		"an empty mapping as an array element": {
			store.Array{store.Mapping{}, store.Array{}},
			"[\n  {},\n  []\n]\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			if got := string(store.Encode(tc.value)); got != tc.want {
				t.Errorf("encoded = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestEncodeEndsInATrailingLFAndCarriesNoTrailingWhitespace(t *testing.T) {
	encoded := store.Encode(store.Mapping{
		"array":   store.Array{store.String("a")},
		"mapping": store.Mapping{"k": store.String("v")},
	})

	if n := len(encoded); n == 0 || encoded[n-1] != '\n' {
		t.Errorf("encoded value does not end in a trailing LF: %q", encoded)
	}
	if bytes.Contains(encoded, []byte("\r")) {
		t.Errorf("encoded value carries a CR: %q", encoded)
	}
	for i, line := range strings.Split(strings.TrimSuffix(string(encoded), "\n"), "\n") {
		if strings.TrimRight(line, " \t") != line {
			t.Errorf("line %d carries trailing whitespace: %q", i+1, line)
		}
	}
}

// The absence rule. A key whose value would be an empty mapping or an empty
// list is absent rather than written empty (§7) — and three keys are written
// anyway, because absence at each of them already means something else.

func TestEncodeOmitsAKeyWhoseValueWouldBeEmpty(t *testing.T) {
	got := string(store.Encode(store.Mapping{
		"fields":    store.Mapping{},
		"members":   store.Array{},
		"name":      store.String("preview-42.example.com"),
		"nested":    store.Mapping{"gone": store.Array{}},
		"populated": store.Array{store.String("here")},
	}))

	// `nested` goes too: every key it held was dropped, so what would be
	// written in its place is the empty mapping the rule is about.
	const want = `{
  "name": "preview-42.example.com",
  "populated": [
    "here"
  ]
}
`
	if got != want {
		t.Errorf("encoded mapping:\n%s\nwant:\n%s", got, want)
	}
}

func TestEncodeWritesAnAlwaysKeyIncludingItsEmptyForm(t *testing.T) {
	got := string(store.Encode(store.Mapping{
		"dry_run":     store.Always(store.Bool(false)),
		"expanded_to": store.Always(store.Array{}),
		"members":     store.Always(store.Array{}),
		"skipped":     store.Mapping{},
	}))

	const want = `{
  "dry_run": false,
  "expanded_to": [],
  "members": []
}
`
	if got != want {
		t.Errorf("encoded mapping:\n%s\nwant:\n%s", got, want)
	}
}

func TestAnAlwaysKeyKeepsTheMappingHoldingItPresent(t *testing.T) {
	got := string(store.Encode(store.Mapping{
		"step": store.Mapping{"expanded_to": store.Always(store.Array{})},
	}))

	const want = `{
  "step": {
    "expanded_to": []
  }
}
`
	if got != want {
		t.Errorf("encoded mapping:\n%s\nwant:\n%s", got, want)
	}
}

// Escaping. The minimum JSON requires and nothing beyond it, because a
// character escaped where it need not be is a byte two implementations can
// disagree about — and every hexadecimal digit hyper writes is lowercase (§7,
// ADR-0079).

func TestEncodeEscapesTheMinimumJSONRequires(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"the quote and the backslash":                      {"he said \"\\\"", `"he said \"\\\""`},
		"the short forms":                                  {"\b\f\n\r\t", `"\b\f\n\r\t"`},
		"a control character with no short form":           {"\x00\x1f", `"\u0000\u001f"`},
		"the solidus, which JSON does not require escaped": {"a/b", `"a/b"`},
		"the line separator encoding/json escapes":         {"a\u2028b", "\"a\u2028b\""},
		"a character outside ASCII, written as itself":     {"über-vm", `"über-vm"`},
		"an emoji, written as itself":                      {"ci-🚀", `"ci-🚀"`},
	} {
		t.Run(name, func(t *testing.T) {
			want := tc.want + "\n"
			if got := string(store.Encode(store.String(tc.in))); got != want {
				t.Errorf("encoded %q = %s, want %s", tc.in, got, want)
			}
		})
	}
}

func TestEveryHexadecimalDigitTheEncoderWritesIsLowercase(t *testing.T) {
	// U+001A to U+001F are the control characters whose escape carries a
	// hexadecimal digit above nine, which is the only place in the encoding
	// there is a case to get wrong at all.
	encoded := string(store.Encode(store.String("\x1a\x1b\x1c\x1d\x1e\x1f")))

	if want := `"\u001a\u001b\u001c\u001d\u001e\u001f"` + "\n"; encoded != want {
		t.Errorf("encoded = %s, want %s", encoded, want)
	}
	if strings.ContainsAny(encoded, "ABCDEF") {
		t.Errorf("encoded value carries an uppercase hexadecimal digit: %s", encoded)
	}
}

func TestAKeyIsEscapedTheSameWayAValueIs(t *testing.T) {
	got := string(store.Encode(store.Mapping{"a\"b": store.String("v")}))

	if want := "{\n  \"a\\\"b\": \"v\"\n}\n"; want != got {
		t.Errorf("encoded = %q, want %q", got, want)
	}
}

// Numbers. §7 says "the shortest decimal that round-trips", which is
// under-determined at the exponent threshold where 1e+06 is shorter than 1000000
// and both round-trip; #124 resolves it. An integer representable exactly is
// written as its decimal digits, and every other number as ECMAScript's
// Number::toString would write it — the form the browser shows the reviewer §7
// wrote the whole encoding for. It matters beyond taste: a number that
// re-encodes differently on two machines mints a spurious version on every Run.

func number(t *testing.T, literal string) store.Number {
	t.Helper()
	n, err := store.ParseNumber(literal)
	if err != nil {
		t.Fatalf("ParseNumber(%q): %v", literal, err)
	}
	return n
}

func TestEncodeWritesAnExactIntegerAsItsDecimalDigits(t *testing.T) {
	for literal, want := range map[string]string{
		"0":                "0",
		"-0":               "0",
		"-0.0":             "0",
		"0.0":              "0",
		"-42":              "-42",
		"1000000":          "1000000",
		"1.0":              "1",
		"1.0e3":            "1000",
		"1e20":             "100000000000000000000",
		"1E5":              "100000",
		"9007199254740993": "9007199254740993",
	} {
		t.Run(literal, func(t *testing.T) {
			if got := string(store.Encode(number(t, literal))); got != want+"\n" {
				t.Errorf("encoded %s = %q, want %q", literal, got, want)
			}
		})
	}
}

func TestEncodeWritesANonIntegerAsECMAScriptWouldPrintIt(t *testing.T) {
	for literal, want := range map[string]string{
		"1.5":      "1.5",
		"0.1":      "0.1",
		"123.456":  "123.456",
		"-2.25":    "-2.25",
		"0.000001": "0.000001",
		"1e-7":     "1e-7",
		"1e21":     "1e+21",
		"1.5e300":  "1.5e+300",
	} {
		t.Run(literal, func(t *testing.T) {
			if got := string(store.Encode(number(t, literal))); got != want+"\n" {
				t.Errorf("encoded %s = %q, want %q", literal, got, want)
			}
		})
	}
}

// The integer beyond a float64's exact range is why a number is held as its
// literal text rather than round-tripped through a float: 9007199254740993 is
// 2^53 + 1, and a float64 holding it holds 9007199254740992 instead.
func TestANumberIsHeldAsItsLiteralRatherThanThroughAFloat(t *testing.T) {
	const literal = "9007199254740993"

	if float64(9007199254740993) == float64(9007199254740992) {
		// The premise of the case, stated so the case cannot pass by the
		// premise having quietly stopped holding.
		if got := string(store.Encode(number(t, literal))); got != literal+"\n" {
			t.Errorf("encoded %s = %q, want %q", literal, got, literal)
		}
		return
	}
	t.Fatal("2^53 + 1 is representable as a float64 here; the case needs a bigger number")
}

func TestEveryNumberTheEncoderWritesRoundTripsThroughADecode(t *testing.T) {
	for _, literal := range []string{
		"0", "-42", "1000000", "9007199254740993", "1.5", "0.1",
		"0.000001", "1e-7", "1e21", "1.5e300", "1.0e3",
	} {
		t.Run(literal, func(t *testing.T) {
			written := store.Encode(number(t, literal))

			decoder := json.NewDecoder(bytes.NewReader(written))
			decoder.UseNumber()
			var decoded json.Number
			if err := decoder.Decode(&decoded); err != nil {
				t.Fatalf("decoding %q: %v", written, err)
			}

			// Re-encoding what the decode returned is the round trip:
			// the second pass must write the first pass's bytes.
			again := store.Encode(number(t, decoded.String()))
			if !bytes.Equal(again, written) {
				t.Errorf("re-encoded %q as %q", written, again)
			}
		})
	}
}

func TestParseNumberRefusesWhatIsNotAJSONNumber(t *testing.T) {
	for _, literal := range []string{
		"", " ", "01", "1.", ".5", "+1", "1e", "1e+", "0x10",
		"Infinity", "NaN", "1_000", "1e400", "--1", "1 2", "1e+-5", "1.2.3",
	} {
		t.Run(literal, func(t *testing.T) {
			if n, err := store.ParseNumber(literal); err == nil {
				t.Errorf("ParseNumber(%q) = %v, want an error", literal, n)
			}
		})
	}
}

func TestIntIsANumberWithoutALiteralToParse(t *testing.T) {
	got := string(store.Encode(store.Mapping{
		"schema_version": store.Int(1),
		"step":           store.Int(-3),
	}))

	const want = `{
  "schema_version": 1,
  "step": -3
}
`
	if got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}
}

// Timestamps. RFC 3339, UTC, Z mandatory, milliseconds always to three digits.
// The width is fixed, so lexicographic order over a timestamp is chronological
// order — which is what the Head derivation rests on — and the window in which
// two writers fall through to the file-name tie-break is a thousandth of what
// whole seconds would leave (§7).

func TestEncodeWritesATimestampAtThreeFractionalDigits(t *testing.T) {
	for name, tc := range map[string]struct {
		instant time.Time
		want    string
	}{
		"an instant with no fraction at all": {
			time.Date(2026, time.August, 6, 9, 41, 14, 0, time.UTC),
			`"2026-08-06T09:41:14.000Z"`,
		},
		"an instant with exactly milliseconds": {
			time.Date(2026, time.August, 6, 9, 41, 14, 221_000_000, time.UTC),
			`"2026-08-06T09:41:14.221Z"`,
		},
		"an instant with more than milliseconds, cut rather than rounded": {
			time.Date(2026, time.August, 6, 9, 41, 14, 221_987_654, time.UTC),
			`"2026-08-06T09:41:14.221Z"`,
		},
		"an instant in another zone, moved to UTC": {
			time.Date(2026, time.August, 6, 11, 41, 14, 221_000_000, time.FixedZone("CEST", 2*60*60)),
			`"2026-08-06T09:41:14.221Z"`,
		},
	} {
		t.Run(name, func(t *testing.T) {
			got := string(store.Encode(store.Timestamp(tc.instant)))
			if got != tc.want+"\n" {
				t.Errorf("encoded = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestLexicographicOrderOverATimestampIsChronologicalOrder(t *testing.T) {
	earlier := store.Encode(store.Timestamp(time.Date(2026, time.August, 6, 9, 41, 14, 9_000_000, time.UTC)))
	later := store.Encode(store.Timestamp(time.Date(2026, time.August, 6, 9, 41, 14, 90_000_000, time.UTC)))

	// .009 against .090: the case a fraction written at its natural width
	// would order backwards, nine sorting after ninety.
	if string(earlier) >= string(later) {
		t.Errorf("%s does not sort before %s", earlier, later)
	}
}

// The secret marker. A field a Manifest declares secret is written as the
// constant string "<secret>" in the position the value would occupy — no digest,
// no length, no sibling list of what was suppressed (§7, ADR-0007). Because it
// is a constant, a rotated secret writes identical bytes and correctly mints no
// version, which is what keeps the byte comparison honest.

func TestASecretIsWrittenAsTheConstantMarker(t *testing.T) {
	got := string(store.Encode(store.Mapping{
		"fields": store.Mapping{
			"api_token": store.Secret(store.String("s3cr3t-value")),
			"name":      store.String("preview-42.example.com"),
		},
	}))

	const want = `{
  "fields": {
    "api_token": "<secret>",
    "name": "preview-42.example.com"
  }
}
`
	if got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}
	if strings.Contains(got, "s3cr3t") {
		t.Errorf("the secret reached the bytes: %s", got)
	}
}

func TestTwoDifferentSecretsWriteIdenticalBytes(t *testing.T) {
	rotated := store.Encode(store.Secret(store.String("the-new-token")))
	previous := store.Encode(store.Secret(store.String("the-old-token")))

	if !bytes.Equal(rotated, previous) {
		t.Errorf("a rotated secret moved the bytes: %q against %q", rotated, previous)
	}
	if want := `"<secret>"` + "\n"; string(rotated) != want {
		t.Errorf("encoded = %s, want %s", rotated, want)
	}
}

func TestASecretSuppressesAWholeStructureRatherThanDescribingIt(t *testing.T) {
	got := string(store.Encode(store.Secret(store.Mapping{
		"key_id": store.String("kid-7"),
		"pem":    store.String("-----BEGIN PRIVATE KEY-----"),
	})))

	if want := `"<secret>"` + "\n"; got != want {
		t.Errorf("encoded = %s, want %s", got, want)
	}
}

// ADR-0079's equality, which is the load-bearing one: the encoding is a function
// of a value, and a file is one case of a value rather than the subject of the
// rules. A value encoded on its own is encoded exactly as it would be were it
// that file's whole content, which is what makes the identity digest computable
// at all — so it is a test rather than a remark.

// insideAFile encodes v as the only member of a mapping, then takes v's own
// bytes back out and shifts them left by the indent the key gave them. What
// comes back is what the nested form claims about the alone form.
func insideAFile(t *testing.T, v store.Value) string {
	t.Helper()

	const key = "\"held\": "
	file := string(store.Encode(store.Mapping{"held": v}))

	body, ok := strings.CutPrefix(file, "{\n  "+key)
	if !ok {
		t.Fatalf("the enclosing mapping is not shaped as expected:\n%s", file)
	}
	body, ok = strings.CutSuffix(body, "\n}\n")
	if !ok {
		t.Fatalf("the enclosing mapping is not shaped as expected:\n%s", file)
	}

	lines := strings.Split(body, "\n")
	for i := 1; i < len(lines); i++ {
		shifted, ok := strings.CutPrefix(lines[i], "  ")
		if !ok {
			t.Fatalf("line %d of the nested value is not indented:\n%s", i+1, file)
		}
		lines[i] = shifted
	}
	return strings.Join(lines, "\n") + "\n"
}

// The empty forms are absent from this table on purpose: the absence rule drops
// the key holding one, so there is no nested form to compare an alone form
// against. Their equality is asserted where it is observable instead — as array
// elements, in the case above that writes {} and [] inline.
func TestAValueAloneIsTheValueAsAWholeFile(t *testing.T) {
	for name, value := range map[string]store.Value{
		"a mapping": store.Mapping{
			"definition": store.String("preview-dns"),
			"step":       store.Int(1),
		},
		"an array": store.Array{
			store.String("ci-macos"),
			store.String("über-vm"),
		},
		"a nested mixture": store.Mapping{
			"dry_run": store.Always(store.Bool(false)),
			"members": store.Array{
				store.String("ci-x86"),
				store.Mapping{"nested": store.Array{store.Int(2)}},
			},
			"selector": store.Mapping{
				"predicates": store.Array{
					store.Mapping{"field": store.String("state"), "is": store.String("live")},
				},
			},
			"written_at": store.Timestamp(theInstant),
		},
	} {
		t.Run(name, func(t *testing.T) {
			alone := string(store.Encode(value))
			if nested := insideAFile(t, value); nested != alone {
				t.Errorf("the value inside a file is\n%s\nand alone it is\n%s", nested, alone)
			}
		})
	}
}

// §7 prints the exact bytes one identity digest is taken over. Asserting them
// beside the digest is what makes a failing digest diagnosable: the digest says
// something moved, and this says what.
func TestTheArrayTheDigestIsTakenOverIsWrittenAsSevenPrintsIt(t *testing.T) {
	got := string(store.Encode(store.Array{
		store.String("ci-macos"),
		store.String("ci-riscv"),
		store.String("ci-x86"),
		store.String("über-vm"),
	}))

	const want = `[
  "ci-macos",
  "ci-riscv",
  "ci-x86",
  "über-vm"
]
`
	if got != want {
		t.Errorf("encoded:\n%s\nwant:\n%s", got, want)
	}
}
