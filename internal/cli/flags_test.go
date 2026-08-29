package cli

import (
	"reflect"
	"strings"
	"testing"
	"unicode"
)

// TestParameters_EveryMemberIsSpelledByTheMessageThatNamesTheNamespace is the
// fence issue #215 asks for: the second line of an unknown flag is composed
// from the parameters value the parser already holds, so a parameter §9 adds
// reaches that message by the edit that adds it and not by a second one.
//
// It reads the members off the type and holds `spelled` to them, and it holds
// each spelling to its member's own name — a member is named for its flag, in
// Go's spelling of the same words, so `secretOut` is `--secret-out` and nothing
// else. A member added and left unspelled fails here, and so does a spelling
// that drifted from the member it is the name of.
func TestParameters_EveryMemberIsSpelledByTheMessageThatNamesTheNamespace(t *testing.T) {
	every := everyParameter()

	var want []string
	for i, members := 0, reflect.TypeOf(every); i < members.NumField(); i++ {
		want = append(want, "--"+kebab(members.Field(i).Name))
	}
	if got := every.spelled(); !reflect.DeepEqual(got, want) {
		t.Errorf("the parameters spell\n  %q,\nwant one flag per member, named for it\n  %q", got, want)
	}
}

// everyParameter is every parameter taken at once, which is a value no command
// passes and exactly what the cases here ask about: the question is what the
// message can spell, not what any one command takes.
//
// It is written out rather than filled in by reflection because a package may
// not set its own unexported members that way — which is what makes it half of
// the fence above: a member added to the type and not to this literal is one
// `spelled` will not name.
func everyParameter() parameters {
	return parameters{
		limit:      defaultListLimit,
		since:      true,
		procedure:  true,
		target:     true,
		outcome:    true,
		definition: true,
		name:       true,
		history:    true,
		between:    true,
		kind:       true,
		input:      true,
		response:   true,
		secretOut:  true,
		dryRun:     true,
		expansion:  true,
	}
}

// TestParameters_EverySpellingIsOneItsCommandActuallyTakes is the other half of
// the fence, and it is the half that catches a rename. The case above holds
// `spelled` to the members; this holds it to the **parser**, which spells each
// flag a second time in the arm that reads it — so a flag renamed in one place
// and not the other would otherwise ship a message naming a flag the command
// refuses.
//
// Ten of the fifteen are held against parseArgs directly: whatever it makes of
// the value handed with them, none of them may come back *unknown*. The other
// five never reach it, their commands taking them off the argument list first,
// so each is held against the splitter that does the taking (flags.go,
// issue #215).
func TestParameters_EverySpellingIsOneItsCommandActuallyTakes(t *testing.T) {
	// The five the loop never sees, each with the value its own command
	// would carry: what is asserted is that the splitter consumed the flag
	// rather than leaving it for parseArgs to call unknown.
	upstream := map[string]func() bool{
		"--input": func() bool { _, rest, _ := splitInputs([]string{"--input", "host=example.com"}); return len(rest) == 0 },
		"--response": func() bool {
			_, rest, _ := splitResponse([]string{"--response", "samples/create.json"})
			return len(rest) == 0
		},
		"--secret-out": func() bool {
			_, rest, _ := splitSecretOut([]string{"--secret-out", "/tmp/token"})
			return len(rest) == 0
		},
		"--dry-run":   func() bool { _, rest := splitDryRun([]string{"--dry-run"}); return len(rest) == 0 },
		"--expansion": func() bool { _, rest := splitExpansion([]string{"--expansion"}); return len(rest) == 0 },
	}

	for _, flag := range everyParameter().spelled() {
		t.Run(flag, func(t *testing.T) {
			if taken, offTheLineFirst := upstream[flag]; offTheLineFirst {
				if !taken() {
					t.Errorf("%s is spelled by the message and its own command does not take it off the line", flag)
				}
				return
			}
			// The value is whatever the flag makes of `1`, and no
			// case here cares: a bad value is a message of its own,
			// and the one message this may not write is the one
			// saying the flag does not exist.
			var stderr strings.Builder
			parseArgs("runs", []string{flag, "1"}, everyParameter(), environment(nil), streams{stderr: &stderr})
			if strings.Contains(stderr.String(), "unknown flag") {
				t.Errorf("%s is spelled by the message and refused by the parser: %s", flag, stderr.String())
			}
		})
	}
}

// kebab is a Go member name in the spelling a flag carries: `secretOut` is
// `secret-out`. It is the test's own and not the package's — what the package
// holds is the flags written out, and this is the rule the case above reads
// them against.
func kebab(name string) string {
	var b strings.Builder
	for _, r := range name {
		if unicode.IsUpper(r) {
			b.WriteRune('-')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

// TestParseArgs_AnUnknownFlagNamesTheNamespaceItWasResolvedAgainst is the
// message itself, at the seam that writes it: the flag that resolved to
// nothing on the first line, and the namespace it was resolved against on the
// second.
//
// The two cases are the two shapes: a command with parameters of its own, and a
// command with none. `--since` on a command that does not take it is
// since_test.go's, beside the gate that refuses it, and the sixteen commands'
// exact messages are the corpus's.
func TestParseArgs_AnUnknownFlagNamesTheNamespaceItWasResolvedAgainst(t *testing.T) {
	for _, c := range []struct {
		name    string
		command string
		takes   parameters
		args    []string
		want    string
	}{
		{
			name:    "a command with parameters of its own",
			command: "changes",
			takes:   changesParameters,
			args:    []string{"--help"},
			want: "hyper changes: unknown flag --help\n" +
				"  changes takes --limit, --since, --target, --between and --kind, past --json, --repo-dir and --no-color\n",
		},
		{
			name:    "a command with none",
			command: "compact",
			takes:   parameters{limit: takesNoLimit},
			args:    []string{"--help"},
			want: "hyper compact: unknown flag --help\n" +
				"  compact takes no flags of its own, past --json, --repo-dir and --no-color\n",
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stderr strings.Builder
			_, _, code := parseArgs(c.command, c.args, c.takes, environment(nil), streams{stderr: &stderr})
			if code != ExitUsage {
				t.Errorf("code = %d, want %d", code, ExitUsage)
			}
			if got := stderr.String(); got != c.want {
				t.Errorf("stderr is\n%s\nwant\n%s", got, c.want)
			}
		})
	}
}

// TestParseArgs_ABareHyphenIsStillAPositional is the boundary of the decision
// above it. A single-hyphen token is a name resolved against the flag
// namespace, because §9 has no single-hyphen flag for one to be — but `-` on
// its own is a whole file name in every shell anybody has used, and nothing is
// being resolved when it is typed.
func TestParseArgs_ABareHyphenIsStillAPositional(t *testing.T) {
	var stderr strings.Builder
	parsed, _, code := parseArgs("check", []string{"-"}, parameters{limit: takesNoLimit}, environment(nil), streams{stderr: &stderr})
	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr = %q", code, stderr.String())
	}
	if len(parsed.positional) != 1 || parsed.positional[0] != "-" {
		t.Errorf("the positionals are %q, want the bare hyphen among them", parsed.positional)
	}
}
