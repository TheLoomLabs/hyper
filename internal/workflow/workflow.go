// Package workflow is the file `hyper project` writes, as a pure function:
// the facts one Procedure's workflow is derived from in, its exact bytes out
// (§10, §11, issue #176).
//
// **Nothing here writes a file.** It opens none, reaches no network, starts no
// subprocess and reads no clock — which is what lets the generator and §10's
// projection check be *one function called from two places* rather than two
// renderings that can drift. Who writes, and whether anything is written at
// all, is internal/cli's; what the check compares against is these same bytes.
//
// **It is not called `projection`.** That name is taken, by §12's path grammar
// resolved against a Capability's response object — a different thing wearing
// the same word, which §12 says out loud about the two `projection` error
// codes, and one package name cannot carry both.
//
// **Nothing about the executor is a parameter.** §11 closes the compiled-in
// constants at four and they are this package's own: the runner, the checkout,
// the release artefact URL and the checksums file URL. There is no file, flag
// or environment variable that reaches them (ADR-0014, ADR-0046), and the
// version is the only variable in either URL — the platform is not one.
//
// **The job summary is shell in this file and nothing else.** The binary those
// bytes invoke is told nothing: it never reads `$GITHUB_STEP_SUMMARY`, and
// detecting it would be the is-CI axis the safety model deleted wearing a
// helpful costume. `hyper run <procedure>` writes the same bytes on a laptop as
// on a runner (ADR-0021).
package workflow

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// The four constants §11 closes the set at, compiled in because there is
// nowhere else they could live: there is no configuration file to put them in
// (ADR-0014), the Repository declaration admits only facts that govern every
// Run and belong to no artefact (ADR-0020), and regeneration reaches no
// network — so the binary is the only thing left that could hold them, and it
// does (ADR-0046).
//
// The two URLs are written here as §11 states them whole:
//
//	https://github.com/TheLoomLabs/hyper/releases/download/v<version>/hyper-<version>-x86_64-linux.tar.gz
//	https://github.com/TheLoomLabs/hyper/releases/download/v<version>/checksums.txt
//
// composed of the tag they share and the file each names, so *the checksums
// file is under the same tag* is a fact the code states rather than one two
// literals happen to agree on.
const (
	// runner is the image the job runs on. It supplies the `bash`, `curl`,
	// `tar` and `sha256sum` the install step runs and the `git` the deepen
	// step before it runs, and §11 states that exemption.
	runner = "ubuntu-24.04"
	// checkout is the one action the projection names, pinned by commit
	// and never by a tag, a tag being a mutable pointer for exactly the
	// reason a release tag is.
	checkout = "actions/checkout@08c6903cd8c0fde910a37f88322edcfb5dd907a8"
	// releaseTag is where one release's files live, the version its only
	// variable.
	releaseTag = "https://github.com/TheLoomLabs/hyper/releases/download/v%s"
	// artefactFile is what the runner fetches, for the platform `runner`
	// names — the platform appearing in the filename as it does in the
	// path, and being the same compiled-in fact `runs-on` is.
	artefactFile = "hyper-%s-x86_64-linux.tar.gz"
	// checksumsFile is what `project` reads once, attended, to freeze the
	// digest. Nothing in the generated file fetches it: the digest it
	// froze is literal in the file, and nothing is resolved at run time.
	checksumsFile = "checksums.txt"
)

// storeGroup is the concurrency group an effectful Procedure's workflow takes:
// one group for the repository rather than one per Procedure, standing in for
// the single-store lock §6 states across two runners that share no filesystem
// (§10, ADR-0005, ADR-0006).
const storeGroup = "hyper-store"

// ArtefactURL is the release artefact for version, and ArtefactName is the
// filename alone — what a line of the checksums file names, and what `project`
// looks for there. They are one template read two ways rather than two
// templates, so no reading of it can name a file the other does not fetch.
func ArtefactURL(version string) string {
	return fmt.Sprintf(releaseTag, version) + "/" + ArtefactName(version)
}

// ArtefactName is the release artefact's filename for version.
func ArtefactName(version string) string {
	return fmt.Sprintf(artefactFile, version)
}

// ChecksumsURL is the checksums file under the same tag — the one network read
// `project` makes, and the fourth of §11's constants.
func ChecksumsURL(version string) string {
	return fmt.Sprintf(releaseTag, version) + "/" + checksumsFile
}

// Facts is everything one workflow file says, and the whole of what Generate
// is given: the file is a function of these and of the four constants above,
// and of nothing else.
//
// Every member is derived from something somebody wrote and reviewed — which
// is the division §11 draws through the file's content, the constants being
// what it says about the world outside both the binary and the repository.
type Facts struct {
	// Procedure is the Procedure's name, verbatim. It is the workflow's
	// name and its job's, that string being the subject line of the
	// executor's own failure email and the only part of a failure visible
	// on a phone (§10, ADR-0005, ADR-0021).
	Procedure string
	// Cadence is the recurrence exactly as the artefact declared it. It is
	// the whole of `on:` — the reviewed artefact declares a recurrence and
	// no second occasion for a Run to start, and a Run a person starts is
	// started from a laptop and records that Trigger (§7, §10).
	Cadence string
	// Effects is whether the Procedure reaches, to any depth, a Step that
	// is not a `read`. It decides the concurrency block and nothing else: a
	// Procedure whose every reachable Step is `read` takes the shared lock
	// and reaps nothing, and serialising it would starve a five-minute
	// cadence behind a forty-minute provision (§10).
	Effects bool
	// Variables is the environment variable names the credential slots this
	// Procedure's bindings require resolve from — the executor secrets the
	// `run` step needs, each named exactly as the variable, because the
	// runtime binary resolves an environment variable on a runner exactly
	// as it does on a laptop (§10, ADR-0007).
	//
	// In any order and with repeats: Generate orders them by Unicode code
	// point and writes each once, the block being a function of the
	// repository rather than of the walk that found them. Two Definitions
	// binding one Target under one scheme reach one slot and one variable,
	// and a mapping may not carry a key twice.
	//
	// Empty writes no `env:` key at all rather than an empty mapping: an
	// absent block is the ordinary absence rule, and an empty one asserts a
	// lookup that happened and found nothing (§10).
	Variables []string
	// Version is the `hyper` version to install — the pin, appearing in the
	// four places §11 counts: the header comment, the install step's own
	// name, the release tag and the artefact's filename.
	Version string
	// Digest is the frozen checksum of that artefact, as `hyper.yaml`
	// spells it — the algorithm inline, so a reviewer reads which one
	// produced it rather than trusting the file's silence (§3). The
	// checksum line takes the hex alone and this package writes it that
	// way, which is one split in one place rather than one every caller
	// repeats.
	Digest string
}

// summary is the executor's own job summary file, as the shell names it. It
// appears only in this file's bytes: the binary is told nothing about it.
const summary = `"$GITHUB_STEP_SUMMARY"`

// fence is one line of the Markdown fence the two invocations' output is
// written between, so the Step table and the Comparison land on the job page
// as preformatted text rather than as Markdown the page would re-flow.
const fence = "          printf '```\\n' >> " + summary + "\n"

// Generate is one Procedure's workflow file, whole.
//
// The sections are §10's own, in its order, and every one of them carries one
// fact for its own reason. Two of them are conditional and neither is
// conditional on anything about the executor: the concurrency block, on the
// Procedure effecting, and the `env:` block, on its bindings requiring a slot.
func Generate(facts Facts) []byte {
	var b strings.Builder

	fmt.Fprintf(&b, "# generated by hyper %s — edits are overwritten by `hyper project`\n", facts.Version)
	fmt.Fprintf(&b, "name: %s\n", scalar(facts.Procedure))

	b.WriteString("\non:\n  schedule:\n")
	fmt.Fprintf(&b, "    - cron: %s\n", quoted(facts.Cadence))

	b.WriteString("\npermissions:\n  contents: write\n")

	if facts.Effects {
		b.WriteString("\nconcurrency:\n")
		fmt.Fprintf(&b, "  group: %s\n", storeGroup)
		b.WriteString("  cancel-in-progress: false\n")
	}

	b.WriteString("\njobs:\n  run:\n")
	fmt.Fprintf(&b, "    runs-on: %s\n", runner)
	b.WriteString("    steps:\n")

	// The checkout leaves the token behind, which is what `hyper` fetches
	// and pushes the Store branch with — `persist-credentials: true`
	// written out rather than inherited, a byte-exact file being no place
	// to rest silently on a default that belongs to somebody else's
	// release cycle (§10, §11, ADR-0007).
	fmt.Fprintf(&b, "      - uses: %s\n", checkout)
	b.WriteString("        with:\n          persist-credentials: true\n")

	writeDeepen(&b)
	writeInstall(&b, facts)
	writeRun(&b, facts)
	writeChanges(&b, facts)

	return []byte(b.String())
}

// writeDeepen writes the step that makes the Comparison legible on a runner.
// `actions/checkout` defaults to one commit and `hyper changes` reads bytes at
// the baseline Run's revisions, so on a shallow clone eight of the nine code
// classes and the whole line count go unread on every window that had
// something in it (§8, §10, ADR-0071, ADR-0086).
//
// The guard is `.git/shallow` and there is no `|| true`: `--unshallow` errors
// on a repository that is already complete, which a self-hosted or pre-warmed
// runner may hand it, so the test is what keeps the step total while leaving a
// real failure fatal. It deepens the code branch and not the Store — no
// `fetch-depth:` is written, that being *all history for all branches and
// tags* and so a fetch of the Store branch on every scheduled Run of every
// Procedure (§10, ADR-0074).
func writeDeepen(b *strings.Builder) {
	b.WriteString("\n      - name: deepen the checkout\n")
	b.WriteString("        run: |\n")
	b.WriteString("          if [ -f .git/shallow ]; then git fetch --unshallow; fi\n")
}

// writeInstall writes the version, the URL and the digest literally, with
// nothing resolved at run time — the pin and its verification in one reviewed
// file (§11, ADR-0020). The binary the runner fetches compares itself against
// the same pin before it reads a second file, so a workflow left behind by an
// older projection Refuses rather than acting (§9).
func writeInstall(b *strings.Builder, facts Facts) {
	fmt.Fprintf(b, "\n      - name: %s\n", scalar("install hyper "+facts.Version))
	b.WriteString("        run: |\n")
	b.WriteString("          curl -fsSL -o hyper.tar.gz \\\n")
	fmt.Fprintf(b, "            %s\n", ArtefactURL(facts.Version))
	fmt.Fprintf(b, "          echo '%s  hyper.tar.gz' \\\n", hexOf(facts.Digest))
	b.WriteString("            | sha256sum -c -\n")
	b.WriteString("          tar -xzf hyper.tar.gz\n")
}

// writeRun writes the first of the two invocations, fenced onto the job
// summary. `${PIPESTATUS[0]}` is what fails the job — the status the step
// exits with is `hyper`'s rather than `tee`'s, which is what makes a Refusal, a
// failure and Store contention land as a red job carrying the code §12 fixes —
// and the `set +e` around it exists so the closing fence is written before the
// step exits (§10).
func writeRun(b *strings.Builder, facts Facts) {
	fmt.Fprintf(b, "\n      - name: %s\n", scalar("hyper run "+facts.Procedure))
	writeEnv(b, facts.Variables)
	b.WriteString("        run: |\n")
	b.WriteString(fence)
	b.WriteString("          set +e\n")
	fmt.Fprintf(b, "          ./hyper run %s | tee -a %s\n", facts.Procedure, summary)
	b.WriteString("          code=${PIPESTATUS[0]}\n")
	b.WriteString("          set -e\n")
	b.WriteString(fence)
	b.WriteString("          exit $code\n")
}

// writeChanges writes the second, under `if: always()`. One invocation would
// put the Step narration on the summary page and leave the Comparison — half
// of what there is to see — unrendered; a Refusal already returns its full
// rendering on stdout and exits 77, so the first invocation puts the
// remediation surface on the page with no conditional needed to get it there
// (§8, §9, §10, ADR-0021).
//
// It carries no `env:`: the Comparison reads the repository and the Store, and
// binds nothing that could need a credential.
func writeChanges(b *strings.Builder, facts Facts) {
	fmt.Fprintf(b, "\n      - name: %s\n", scalar("hyper changes "+facts.Procedure))
	b.WriteString("        if: always()\n")
	b.WriteString("        run: |\n")
	b.WriteString(fence)
	fmt.Fprintf(b, "          ./hyper changes %s | tee -a %s\n", facts.Procedure, summary)
	b.WriteString(fence)
}

// writeEnv writes the credential slots the Procedure's bindings require, on
// the `run` step and nowhere else in the file. It is the bindings rather than
// the Target declarations, because a Target may carry slots for a scheme this
// Procedure never uses and writing those into the job would put a secret on
// the runner that no Step could reach (§6, §10, ADR-0007).
//
// An executor holding no secret of a name written here Refuses before the
// first Step (`credential-absent`, §12) rather than failing part-way through
// one, which is why an absent secret needs nothing said about it here.
func writeEnv(b *strings.Builder, variables []string) {
	names := ordered(variables)
	if len(names) == 0 {
		return
	}
	b.WriteString("        env:\n")
	for _, name := range names {
		fmt.Fprintf(b, "          %s: %s\n", scalar(name), scalar("${{ secrets."+name+" }}"))
	}
}

// ordered is variables by Unicode code point, each once. The block is a
// function of the repository and not of the order a caller's walk found the
// pairs in, and a mapping may not carry a key twice.
//
// The dedup §10 states is on the (Definition, Target) pair's slot, which is
// where the caller resolves one — two Definitions binding one Target under one
// scheme reach one slot. This is the mapping's own rule beneath it: two slots
// resolving from one variable are one key, whatever pairs named them.
func ordered(variables []string) []string {
	held := make(map[string]bool, len(variables))
	names := make([]string, 0, len(variables))
	for _, name := range variables {
		if held[name] {
			continue
		}
		held[name] = true
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// hexOf is the digest as `sha256sum -c -` takes it: the hex alone, the
// algorithm the declaration carries inline stripped off (§3).
//
// A digest carrying no algorithm is written whole. Nothing here judges what it
// was handed: `hyper.yaml`'s schema makes the digest a string and nothing
// narrower, what is wrong with one is `check`'s to report, and a generator
// that second-guessed its input would put two answers on one fact (ADR-0064).
func hexOf(digest string) string {
	if _, hex, found := strings.Cut(digest, ":"); found {
		return hex
	}
	return digest
}

// Quoting is a fixed rule, because byte-exactness makes it one (§10, §11).
//
// **The cron expression is always single-quoted**, as §10's example fixes.
// **Every other derived scalar is written plain where a plain scalar of those
// exact bytes parses back as that exact string, and single-quoted where it does
// not** — which is what keeps a Procedure named `on`, `true` or `12:30` from
// producing a file that means something other than it says.
//
// **What *parses back* means is the union of the two readings, and that is
// wider than YAML 1.2 alone.** §11 names 1.2, and under 1.2's core schema `on`
// is already a string — but the reader these bytes are written for is the
// executor's, which resolves YAML 1.1's wider implicit set, and there `on` is
// `true`. A rule that admitted `name: on` would produce a workflow whose job is
// named `true` on the one machine that matters, which is the defect the rule
// exists to prevent. So a scalar is written plain only where **neither** reading
// types it as anything but a string. The rule stays total, deterministic and
// testable, and it never widens what a plain scalar may mean.
//
// The set is written out here rather than resolved through a YAML library on
// purpose: these bytes are compared byte for byte against a regeneration, so a
// library upgrade that changed one resolution would make every generated
// workflow in every repository `projection-stale` at once.
var resolvesToAType = []*regexp.Regexp{
	// null, and 1.1's `~`.
	regexp.MustCompile(`^(?:~|[nN]ull|NULL)$`),
	// boolean, 1.2's core schema.
	regexp.MustCompile(`^(?:[tT]rue|TRUE|[fF]alse|FALSE)$`),
	// boolean, 1.1's wider set — the reading `on`, `no` and `off` are
	// typed under.
	regexp.MustCompile(`^(?:[yY]|[yY]es|YES|[nN]|[nN]o|NO|[oO]n|ON|[oO]ff|OFF)$`),

	// integer: decimal, octal — 1.1's leading zero and 1.2's `0o` alike —
	// hexadecimal and binary, with 1.1's underscores throughout.
	regexp.MustCompile(`^[-+]?[0-9][0-9_]*$`),
	regexp.MustCompile(`^[-+]?0[oO]?[0-7_]+$`),
	regexp.MustCompile(`^[-+]?0[xX][0-9a-fA-F_]+$`),
	regexp.MustCompile(`^[-+]?0[bB][01_]+$`),
	// sexagesimal, 1.1's — what `12:30` is typed as.
	regexp.MustCompile(`^[-+]?[0-9][0-9_]*(?::[0-5]?[0-9])+$`),
	// float, its two infinities and its not-a-number.
	regexp.MustCompile(`^[-+]?(?:\.[0-9][0-9_]*|[0-9][0-9_]*(?:\.[0-9_]*)?)(?:[eE][-+]?[0-9]+)?$`),
	regexp.MustCompile(`^[-+]?\.(?:inf|Inf|INF)$`),
	regexp.MustCompile(`^\.(?:nan|NaN|NAN)$`),
	// timestamp, 1.1's — a date, and a date with a time after it.
	regexp.MustCompile(`^[0-9]{4}-[0-9]{1,2}-[0-9]{1,2}(?:[Tt ][^\n]*)?$`),
}

// indicators are the characters a plain scalar may not open with. The list is
// YAML's own, read conservatively: a few of them are legal in a first position
// under conditions this rule does not need to know, and quoting one that need
// not have been changes what the file says about nothing.
const indicators = "-?:,[]{}#&*!|>'\"%@`~"

// scalar is one derived value as the file writes it: plain, or single-quoted
// where plain would not parse back as itself.
func scalar(text string) string {
	if plain(text) {
		return text
	}
	return quoted(text)
}

// quoted is text single-quoted, YAML's own doubling escaping any quote it
// carries. It is what the cron expression always takes and what scalar falls
// back to — one spelling of the quoting, so the two cannot part.
func quoted(text string) string {
	return "'" + strings.ReplaceAll(text, "'", "''") + "'"
}

// plain reports whether text may be written as a plain scalar: whether the
// bytes are ones a plain scalar may carry at all, and whether a reader would
// type them as something other than the string they spell.
func plain(text string) bool {
	if text == "" || strings.TrimSpace(text) != text {
		return false
	}
	for _, r := range text {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	// The comparison is over the first byte and that is exact: every
	// indicator is ASCII, and no byte of a multi-byte character is.
	if strings.IndexByte(indicators, text[0]) >= 0 {
		return false
	}
	// `: ` opens a mapping value and ` #` opens a comment, wherever in a
	// plain scalar they fall, and a trailing `:` is the same colon at the
	// end of the line.
	if strings.Contains(text, ": ") || strings.HasSuffix(text, ":") || strings.Contains(text, " #") {
		return false
	}
	for _, typed := range resolvesToAType {
		if typed.MatchString(text) {
			return false
		}
	}
	return true
}
