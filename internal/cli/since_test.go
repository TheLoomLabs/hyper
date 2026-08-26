package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

// The one `--since` parser (§9, §12, ADR-0060, ADR-0065, issue #162).
//
// Three of §9's Inspection commands take a timestamp — `runs`, `records
// --history` and `changes` — and none of them exists yet. What is here is the
// reading itself, decided once so that the three tickets that reach for it
// inherit one spelling, one validation and one refusal instead of writing three
// that agree until they do not.
//
// The corpus cannot drive any of it: a golden case is an argv against a
// command, and no command takes the flag. This is the seam the parser is
// exercised at, exactly as the Trigger's fallbacks are exercised beside it.

// sinceCommand is what the messages below are written in the name of. It is one
// of the three §9 gives the flag to, and it is a string here rather than a
// dispatched command because parseArgs takes the name as a parameter for
// precisely this reason: one parser, and a caller still reads a message in
// their own command's words.
const sinceCommand = "runs"

func TestParseArgs_ReadsAnRFC3339Since(t *testing.T) {
	for name, c := range map[string]struct {
		args []string
		want string
	}{
		"the spelling every instant in the record is written in": {
			args: []string{"--since", "2026-08-04T09:12:00Z"},
			want: "2026-08-04T09:12:00Z",
		},
		"the equals spelling, which every other valued flag takes": {
			args: []string{"--since=2026-08-04T09:12:00Z"},
			want: "2026-08-04T09:12:00Z",
		},
		"a fractional second, which the Journal's own timestamps carry": {
			args: []string{"--since", "2026-08-04T09:12:00.221Z"},
			want: "2026-08-04T09:12:00.221Z",
		},
		"an offset, read as the instant it names": {
			args: []string{"--since", "2026-08-04T11:12:00+02:00"},
			want: "2026-08-04T09:12:00Z",
		},
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			parsed, _, code := parseArgs(sinceCommand, c.args, parameters{limit: defaultListLimit, since: true}, environment(nil), streams{stderr: &stderr})

			if code != 0 {
				t.Fatalf("parseArgs() = %d, want 0; stderr said %q", code, stderr.String())
			}
			if !parsed.sinceNamed {
				t.Errorf("sinceNamed is false on an argument list that named --since")
			}
			if got := parsed.since.Format(time.RFC3339Nano); got != c.want {
				t.Errorf("since = %s, want %s", got, c.want)
			}
			if got := parsed.since.Location(); got != time.UTC {
				t.Errorf("since is in %v, want UTC: an offset names an instant, and the record spells one way", got)
			}
		})
	}
}

// TestParseArgs_RefusesASinceThatIsNotATimestamp is the flag's whole
// validation. A window is named by an instant, and a value that is not one
// names no window at all — so it is refused where it is read, rather than
// carried into a command that would have to decide for itself what a caller
// meant by *yesterday*.
//
// Every one of these is a usage error at exit 2 **carrying no error_code**: an
// error_code names a check that declined an artefact, and a value typed at a
// command line is not one (ADR-0060, and probe's own inputs one file over).
func TestParseArgs_RefusesASinceThatIsNotATimestamp(t *testing.T) {
	for name, value := range map[string]string{
		"a date with no time in it":        "2026-08-04",
		"a date-time with no offset":       "2026-08-04T09:12:00",
		"a space where the T belongs":      "2026-08-04 09:12:00Z",
		"a month that does not exist":      "2026-13-04T09:12:00Z",
		"a word":                           "yesterday",
		"a duration, which is not a point": "7d",
		"nothing at all":                   "",
	} {
		t.Run(name, func(t *testing.T) {
			var stderr bytes.Buffer
			parsed, _, code := parseArgs(sinceCommand, []string{"--since", value}, parameters{limit: defaultListLimit, since: true}, environment(nil), streams{stderr: &stderr})

			if code != ExitUsage {
				t.Fatalf("parseArgs() = %d, want %d", code, ExitUsage)
			}
			if parsed.sinceNamed {
				t.Errorf("sinceNamed is true on a value that was refused")
			}
			said := stderr.String()
			if !strings.HasPrefix(said, "hyper "+sinceCommand+": --since ") {
				t.Errorf("stderr = %q, want it to open in the command's own name and name the flag", said)
			}
			if !strings.Contains(said, "RFC 3339") {
				t.Errorf("stderr = %q, want it to name the spelling it wanted", said)
			}
			if strings.Contains(said, "error_code") {
				t.Errorf("stderr = %q, want no error_code: nothing was reviewed to decline", said)
			}
		})
	}
}

func TestParseArgs_SinceRequiresAValue(t *testing.T) {
	var stderr bytes.Buffer
	_, _, code := parseArgs(sinceCommand, []string{"--since"}, parameters{limit: defaultListLimit, since: true}, environment(nil), streams{stderr: &stderr})

	if code != ExitUsage {
		t.Fatalf("parseArgs() = %d, want %d", code, ExitUsage)
	}
	if got, want := stderr.String(), "hyper "+sinceCommand+": --since requires a value\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

// TestParseArgs_ACommandThatTakesNoSinceRefusesTheFlag is the gate. The flag is
// §9's on three commands and on no other: a listing over a namespace has no
// time axis for a window to cut, and offering it there would be a parameter
// that narrows nothing.
func TestParseArgs_ACommandThatTakesNoSinceRefusesTheFlag(t *testing.T) {
	var stderr bytes.Buffer
	_, _, code := parseArgs("providers", []string{"--since", "2026-08-04T09:12:00Z"}, parameters{limit: defaultListLimit}, environment(nil), streams{stderr: &stderr})

	if code != ExitUsage {
		t.Fatalf("parseArgs() = %d, want %d", code, ExitUsage)
	}
	if got, want := stderr.String(), "hyper providers: unknown flag --since\n"; got != want {
		t.Errorf("stderr = %q, want %q", got, want)
	}
}

// TestParseArgs_AnUnnamedSinceIsNoWindow is the difference between the flag and
// --limit beside it: a limit left off has a default, and a window left off has
// none. An unnamed --since is the absence of a lower bound, which is what
// sinceNamed says and what the zero instant on its own could not — the zero
// instant is a point in the year 1, and a command reading it as a bound would
// be filtering rather than not filtering.
func TestParseArgs_AnUnnamedSinceIsNoWindow(t *testing.T) {
	var stderr bytes.Buffer
	parsed, _, code := parseArgs(sinceCommand, []string{"--limit", "3"}, parameters{limit: defaultListLimit, since: true}, environment(nil), streams{stderr: &stderr})

	if code != 0 {
		t.Fatalf("parseArgs() = %d, want 0; stderr said %q", code, stderr.String())
	}
	if parsed.sinceNamed {
		t.Errorf("sinceNamed is true on an argument list that named no --since")
	}
	if !parsed.since.IsZero() {
		t.Errorf("since = %s, want the zero instant", parsed.since)
	}
}

// TestParseArgs_SinceAfterTheDoubleHyphenIsAPositional holds the flag to the
// rule every other one already follows: "--" ends the flags, so a positional
// beginning with a hyphen is reachable.
func TestParseArgs_SinceAfterTheDoubleHyphenIsAPositional(t *testing.T) {
	var stderr bytes.Buffer
	parsed, _, code := parseArgs(sinceCommand, []string{"--", "--since"}, parameters{limit: defaultListLimit, since: true}, environment(nil), streams{stderr: &stderr})

	if code != 0 {
		t.Fatalf("parseArgs() = %d, want 0; stderr said %q", code, stderr.String())
	}
	if parsed.sinceNamed {
		t.Errorf("sinceNamed is true on a --since that stood after the double hyphen")
	}
	if got, want := parsed.positional, []string{"--since"}; len(got) != 1 || got[0] != want[0] {
		t.Errorf("positional = %q, want %q", got, want)
	}
}
