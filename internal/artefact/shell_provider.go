package artefact

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
)

// BuiltinShellProviderPath is the pseudo-path §9 renders for the built-in
// shell Provider's Manifest, <built-in>/shell — not a File a check row may
// ever cite, the built-in having no file to open, but a label these
// internals thread through the same functions a providers/ file's checks
// use (§3, §9, §11).
const BuiltinShellProviderPath = "<built-in>/shell"

// BuiltinShellProviderName is the name the compiled-in Manifest below declares
// for itself, and the one member of §12's built-in Provider set.
//
// It is spelled once because three things read it: the Provider namespace,
// which starts from it; the fold, which declines an Extension that takes it;
// and the check that names the taking (§11, §12, IsBuiltinProviderName).
const BuiltinShellProviderName = "shell"

// IsBuiltinProviderName says whether name is a built-in Provider's, which is
// §12's built-in set read as the thing it doubles as: **the list of names no
// Extension may take** (provider-name-collision, §11).
//
// One member today, and the set grows only where the reserved half of the
// Capability set grows — hyper ships a Provider only where the Capability it
// needs is one nobody else may declare (ADR-0039). It is enumerated against the
// binary's own constants rather than derived from the loaded repository because
// that is what the set is: a fact about what this binary compiles in, answerable
// before any tree is walked.
//
// **It is one predicate because two readers of the set would be two answers.**
// The fold declines a colliding Manifest and the check names it, and a name the
// fold declined and the check said nothing about is a file that vanished from
// the namespace with no row to explain it.
func IsBuiltinProviderName(name string) bool {
	return name == BuiltinShellProviderName
}

// ReservedCapability is the one member of §12's Capability set that is
// reserved to the Providers hyper ships: shell, the Capability behind an
// opaque Operation.
//
// It is spelled apart from BuiltinShellProviderName above, which is the same
// four letters standing for a different fact — one is a Provider's name and
// the other a Capability's, and a Manifest may take the second while renaming
// itself past the first, which is the whole of what a fork of the built-in is
// (§11).
const ReservedCapability = "shell"

// IsReservedCapability says whether name is a Capability no Manifest loaded
// from providers/ may hold, declared or derived (capability-reserved, §11).
//
// **One member, and http is not it.** §12 closes the Capability set at two and
// reserves exactly one: http describes what it does and shell cannot describe
// anything, so what an Operation cannot describe and who may write one are one
// fact (ADR-0004). A third party can never ship a Provider that runs commands
// on your machine, and that sentence is this predicate.
//
// **It is closed by the same criterion that closes the built-in Provider set,
// and neither grows without the other.** hyper ships a Provider only where the
// Capability it needs is one nobody else may declare (ADR-0039), so a new
// reserved member is a new built-in and a new built-in is a new reserved
// member — which is why this predicate stands beside IsBuiltinProviderName
// rather than in the file that reads capabilities: off a Manifest.
func IsReservedCapability(name string) bool {
	return name == ReservedCapability
}

// BuiltinShellProviderYAML is hyper's own shell Provider, compiled into the
// binary exactly as §12 states it: six Operations, Kind crossed with the
// Repeatability values each Kind may declare, sharing one request — an
// empty shell: block, the argv arriving as the Operation input named
// command in a Step's args: — and, on the four that carry one, one
// projection (§3, §12, issue #91).
const BuiltinShellProviderYAML = `kind: provider
provider: shell
schema-version: 1
class: local
capabilities: [shell]
operations:
  read:
    kind: read
    repeatability: repeatable
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  mutate:
    kind: mutate
    repeatability: repeatable
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  mutate_once:
    kind: mutate
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  mutate_skip_if_recorded:
    kind: mutate
    repeatability: skip-if-recorded
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
    record:
      identity: $.command
      fields:
        exit_code: $.exit_code
        stdout: $.stdout
        stderr: $.stderr
  destroy:
    kind: destroy
    repeatability: repeatable
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
  destroy_once:
    kind: destroy
    deadline: 1h
    shell: {}
    input:
      type: object
      properties:
        command: {type: array, items: {type: string}}
`

// CheckBuiltinShellProvider validates BuiltinShellProviderYAML against
// every check CheckManifest runs except the three that read a Manifest
// against where it was loaded from — kind-mismatch has no directory to
// compare against and name-mismatch no basename, the built-in authoring its
// name outright and having no file at all, and capability-reserved is §11's
// rule about a Manifest in providers/, which this is not (§3, §11). It is
// checked like any other Manifest, with no exemption: a Provider is data,
// and data check may not read is an advisory analyzer wearing the tool's own
// badge.
//
// The three are absent because the function they live in is never called for
// these bytes, rather than because a branch inside it lets them through. An
// exemption would be the thing §11 does not have: the built-in is entitled to
// the Capability for being compiled in, and *compiled in* is the whole of the
// criterion (ADR-0039, ADR-0073).
func CheckBuiltinShellProvider() []problem.Problem {
	return checkManifestBody(BuiltinShellProviderPath, BuiltinShellProviderRoot())
}

// BuiltinShellProviderRoot decodes BuiltinShellProviderYAML and returns its
// document root, so that the one decode of hyper's own bytes is written here
// rather than reimplemented at each caller: CheckBuiltinShellProvider,
// builtinShellProviderInfo (issue #93) and the repository load (issue #109),
// which carries the built-in as an artefact like any other and needs a root to
// carry (ADR-0039). It is exported for the third of those.
//
// Each call decodes afresh, which is why the constant and not the node is what
// the callers share. That is sound only because the subject is a compiled-in
// constant no repository author can touch: two decodes of immutable bytes cannot
// disagree, where two reads of one file could. Nothing here may be reused for an
// artefact that came off disk.
func BuiltinShellProviderRoot() *yaml.Node {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(BuiltinShellProviderYAML), &doc); err != nil {
		// hyper's own bytes, never touched by a repository author — a
		// parse failure here is a bug in hyper, not reviewed text to
		// report a problem row about.
		panic(fmt.Sprintf("artefact: the built-in shell Provider does not parse: %s", err))
	}
	if len(doc.Content) == 0 {
		panic("artefact: the built-in shell Provider parsed to no document")
	}
	return doc.Content[0]
}
