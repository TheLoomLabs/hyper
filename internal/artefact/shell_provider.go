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
// every check CheckManifest runs except the two that need a file of its
// own — kind-mismatch has no directory to compare against and
// name-mismatch no basename, the built-in authoring its name outright and
// having no file at all (§3, §11). It is checked like any other Manifest,
// with no exemption: a Provider is data, and data check may not read is an
// advisory analyzer wearing the tool's own badge.
func CheckBuiltinShellProvider() []problem.Problem {
	return checkManifestBody(BuiltinShellProviderPath, builtinShellProviderRoot())
}

// builtinShellProviderRoot parses BuiltinShellProviderYAML and returns its
// document root, shared by CheckBuiltinShellProvider and
// builtinShellProviderInfo (issue #93) so hyper's own bytes are decoded
// once per caller rather than reimplemented at each.
func builtinShellProviderRoot() *yaml.Node {
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
