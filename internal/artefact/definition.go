// This file is the Definition's own schema, the checks that read it against
// itself — kind: against definitions/, definition: against the file's
// basename, and the two keys' own shape (§3, §4, issue #93) — and the first
// checks in this package that read more than one artefact at once: whether
// provider: and every targets: member name an artefact this repository
// holds, whether a destroy: member names an Operation the bound Provider
// declares, and the two checks that need a (Definition, Target) pair rather
// than either artefact alone — a Target outside the bound Provider's class,
// a Capability the Target's declaration does not grant, and the Target's
// credential slots not covering the bound Provider's Auth scheme, the
// twelfth shape of manifest-inconsistent left open by #92 (§4, §5).
package artefact

import (
	"fmt"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/TheLoomLabs/hyper/internal/problem"
	"github.com/TheLoomLabs/hyper/internal/schema"
)

// CodeArtefactAbsent is the code a name an artefact writes for one of this
// repository's own artefacts earns where it resolves to nothing — a
// Definition's provider: or a targets: member here, and, as those artefacts
// arrive, a Step's definition: and a nested invocation's procedure: (§4,
// ADR-0064). The row carries the file and line the name was written on and
// the path hyper looked for, so the two edits it points at are fix the name
// and write that file.
const CodeArtefactAbsent = "artefact-absent"

// CodeReferenceUnresolvable is the code the same fault earns where the
// namespace is what an artefact declares rather than what the repository
// holds — a Definition's destroy: member against the Operations its bound
// Provider declares, here, and, elsewhere, a Step's operation:, a field: at
// either Record root, and the step: half of a reference (§3, §4). A missing
// artefact is one the reader may have to write; a missing member is a key
// inside an artefact that already exists, and where that artefact is a
// built-in or somebody else's Extension it is not theirs to write at all.
const CodeReferenceUnresolvable = "reference-unresolvable"

// CodeDefinitionKindsMixed is the code read in kinds: earns for standing
// beside mutate, or beside a destroy: claim naming any Operation: a
// Definition observes or it effects, never both (§3, §4, ADR-0032). It
// reads one file and needs no Target, which is why it is checked before any
// resolution below.
const CodeDefinitionKindsMixed = "definition-kinds-mixed"

// CodeTargetClassMismatch is the code a Definition naming a Target outside
// its Provider's declared class: earns (§3, §4) — a Target class only ever
// rejects a mismatch and never expands a Definition's reach.
const CodeTargetClassMismatch = "target-class-mismatch"

// CodeCapabilityNotGranted is the code a Capability the bound Provider's
// Operations require and the bound Target's declaration does not grant
// earns, checked per (Definition, Target) pair — a Target declaration is
// written without knowing which Provider will bind it, so the question can
// only be asked once a binding exists (§3, §4).
const CodeCapabilityNotGranted = "capability-not-granted"

// KindDefinition is the one kind: value a file in definitions/ may carry
// (§12's kind table).
const KindDefinition = "definition"

// DefinitionDeclaration is a Definition's own schema (§3): the definition:
// this file's name is checked against, the provider: it is a named,
// authority-scoped use of, the kinds: it claims for read and mutate —
// destroy is not a member, granularity following severity so a destroy
// claim names Operations instead — the destroy: Operations it claims by
// name, and the targets: it may bind, named literally rather than by class
// or tag. additionalProperties: false is forced rather than authored
// (§12), so a sixth key is unknown-key wherever it appears. A Definition
// carries no argument value of its own — those belong to the Step (§3).
var DefinitionDeclaration = schema.Schema{
	Type: schema.Object,
	Properties: []schema.Property{
		{Name: "kind", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "definition", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "provider", Required: true, Schema: schema.Schema{Type: schema.String}},
		{Name: "kinds", Required: false, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String, Enum: []string{"read", "mutate"}},
		}},
		{Name: "destroy", Required: false, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String},
		}},
		{Name: "targets", Required: true, Schema: schema.Schema{
			Type:  schema.Array,
			Items: &schema.Schema{Type: schema.String},
		}},
	},
}

// ProviderInfo is what checking a Definition against its bound Provider
// needs, read once per repository pass rather than reparsed per Definition
// that binds it (§4, §5): the class its Definitions may bind, the
// Capabilities its Operations require, the Operations it declares — the
// namespace a destroy: member and a Step's operation: resolve against — and
// the credential slot names its Auth scheme requires, nil where it
// authenticates nothing.
type ProviderInfo struct {
	Class        string
	Capabilities map[string]bool
	Operations   map[string]OperationInfo
	AuthSlots    []string
}

// OperationInfo is what checking a Step against the Operation it binds
// needs, read once per repository pass alongside the rest of ProviderInfo
// rather than reparsed per Step that names it (§3, §4, issue #94): whether
// its request is the shell Capability, the one hyper's own Provider may
// declare and the one whose argv arrives as the input named command; its
// Record cardinality, series where its record: carries an over: and one
// otherwise — the fact a reference's step: half is refused against
// (series-reference); the field names its record: projects, nil on an
// Operation with no record: at all — the namespace a reference's path: half
// resolves against; and every input its input: schema declares, by name.
type OperationInfo struct {
	IsShell      bool
	HasSeries    bool
	RecordFields map[string]bool
	Inputs       map[string]InputInfo
}

// InputInfo is what checking a Step's args: value against one Operation
// input needs: the type: it declares, read the way an argument's own type
// is — object and array are the two a reference may never fill (§3, §4) —
// and the enum: its value is checked against, nil where the input declares
// none.
type InputInfo struct {
	Type string
	Enum []string
}

// TargetInfo is what checking a Definition against a Target it binds needs:
// the class it declares, the Capabilities it grants, and the credential
// slot names its auth: mapping supplies.
type TargetInfo struct {
	Class        string
	Capabilities map[string]bool
	AuthSlots    map[string]bool
}

// ProviderIndex maps a Provider's own name — a built-in or a providers/
// file's provider: — to what a Definition binding it is checked against.
// It is the namespace a Definition's provider: resolves against (§4).
type ProviderIndex map[string]ProviderInfo

// TargetIndex maps a Target declaration's own target: to what a Definition
// binding it is checked against. It is the namespace a Definition's
// targets: member resolves against (§4).
type TargetIndex map[string]TargetInfo

// BuildProviderIndex starts from the built-in Providers — today, shell
// alone — and adds one entry per providers/ root whose provider: is a
// legible scalar. A root whose provider: is absent or malformed has already
// earned its own schema-mismatch and contributes nothing to the namespace
// other artefacts resolve against — the same rule ADR-0064 states for a
// file that will not parse at all, applied to a file that parses but names
// itself badly.
func BuildProviderIndex(manifestRoots []*yaml.Node) ProviderIndex {
	idx := ProviderIndex{"shell": builtinShellProviderInfo()}
	for _, root := range manifestRoots {
		nameVal := topLevelFields(root, "provider")["provider"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		idx[nameVal.Value] = providerInfoFromManifest(root)
	}
	return idx
}

// BuildTargetIndex adds one entry per targets/ root whose target: is a
// legible scalar, on BuildProviderIndex's own rule.
func BuildTargetIndex(declarationRoots []*yaml.Node) TargetIndex {
	idx := TargetIndex{}
	for _, root := range declarationRoots {
		nameVal := topLevelFields(root, "target")["target"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		idx[nameVal.Value] = targetInfoFromDeclaration(root)
	}
	return idx
}

// DefinitionIndex maps a definitions/ file's own definition: to the
// provider: it names, unresolved — a Step's definition: resolves against
// this namespace, and what its operation: and args: are checked against is
// reached by a second lookup, into ProviderIndex, once the Definition's own
// provider: has resolved (§3, §4, issue #94).
type DefinitionIndex map[string]string

// BuildDefinitionIndex adds one entry per definitions/ root whose
// definition: is a legible scalar, on BuildProviderIndex's own rule. An
// entry whose provider: is absent or illegible carries "" — CheckDefinition
// has already named that fault on the Definition's own line, and a Step
// naming this Definition resolves no Operation against an empty provider
// name, which is reference-unresolvable on the same rule as any other name
// that does not resolve.
func BuildDefinitionIndex(definitionRoots []*yaml.Node) DefinitionIndex {
	idx := DefinitionIndex{}
	for _, root := range definitionRoots {
		fields := topLevelFields(root, "definition", "provider")
		nameVal := fields["definition"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		providerName := ""
		if providerVal := fields["provider"]; providerVal != nil && providerVal.Kind == yaml.ScalarNode {
			providerName = providerVal.Value
		}
		idx[nameVal.Value] = providerName
	}
	return idx
}

// providerInfoFromManifest reads the four facts CheckDefinition needs off a
// Manifest's own root: class:, capabilities:, the name set of operations:,
// and the credential slots its auth: scheme requires.
func providerInfoFromManifest(root *yaml.Node) ProviderInfo {
	fields := topLevelFields(root, "class", "capabilities", "operations")
	info := ProviderInfo{Capabilities: map[string]bool{}, Operations: map[string]OperationInfo{}}
	if classVal := fields["class"]; classVal != nil && classVal.Kind == yaml.ScalarNode {
		info.Class = classVal.Value
	}
	if capsVal := fields["capabilities"]; capsVal != nil && capsVal.Kind == yaml.SequenceNode {
		for _, item := range capsVal.Content {
			if item.Kind == yaml.ScalarNode {
				info.Capabilities[item.Value] = true
			}
		}
	}
	if opsVal := fields["operations"]; opsVal != nil && opsVal.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(opsVal.Content); i += 2 {
			key, opNode := opsVal.Content[i], opsVal.Content[i+1]
			if key.Kind == yaml.ScalarNode {
				info.Operations[key.Value] = operationInfoFromNode(opNode)
			}
		}
	}
	info.AuthSlots = authSlotNames(root)
	return info
}

// targetInfoFromDeclaration reads the three facts CheckDefinition needs off
// a Target declaration's own root: class:, capabilities:, and the
// credential slot names its auth: mapping's own keys are.
func targetInfoFromDeclaration(root *yaml.Node) TargetInfo {
	fields := topLevelFields(root, "class", "capabilities", "auth")
	info := TargetInfo{Capabilities: map[string]bool{}, AuthSlots: map[string]bool{}}
	if classVal := fields["class"]; classVal != nil && classVal.Kind == yaml.ScalarNode {
		info.Class = classVal.Value
	}
	if capsVal := fields["capabilities"]; capsVal != nil && capsVal.Kind == yaml.SequenceNode {
		for _, item := range capsVal.Content {
			if item.Kind == yaml.ScalarNode {
				info.Capabilities[item.Value] = true
			}
		}
	}
	if authVal := fields["auth"]; authVal != nil && authVal.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(authVal.Content); i += 2 {
			if key := authVal.Content[i]; key.Kind == yaml.ScalarNode {
				info.AuthSlots[key.Value] = true
			}
		}
	}
	return info
}

// authSlotNames reads root's auth: scheme and returns the credential slot
// names it requires — header:'s one slot, token, or basic:'s two, username
// and password — nil where auth: is absent, or present but too malformed to
// say which of the two closed schemes it names (§3, §12) — the same "no
// scheme, no fault added here" rule checkAuth's own schema check already
// covers.
func authSlotNames(root *yaml.Node) []string {
	authVal := topLevelFields(root, "auth")["auth"]
	if authVal == nil || authVal.Kind != yaml.MappingNode {
		return nil
	}
	fields := topLevelFields(authVal, "header", "basic")
	switch {
	case fields["header"] != nil:
		return []string{"token"}
	case fields["basic"] != nil:
		return []string{"username", "password"}
	default:
		return nil
	}
}

// builtinShellProviderInfo reads the built-in shell Provider's own compiled
// Manifest the same way providerInfoFromManifest reads any other, so the
// one Provider hyper ships is a member of ProviderIndex on exactly the same
// footing as an Extension (§3, §11).
func builtinShellProviderInfo() ProviderInfo {
	return providerInfoFromManifest(builtinShellProviderRoot())
}

// CheckDefinition validates a definitions/ file's already-parsed root
// against DefinitionDeclaration and every check that reads a Definition on
// its own (§3, §4, issue #93): kind: against definitions/, definition:
// against the file's basename, and the two-keys rule. providers and targets
// are the repository-wide namespaces provider:, targets: members and
// destroy: members resolve against, and the per-pair checks that need a
// binding. root is nil where the file parsed to no document at all; the
// schema check still runs and reports every required key the file never
// supplied.
func CheckDefinition(file string, root *yaml.Node, providers ProviderIndex, targets TargetIndex) []problem.Problem {
	problems := schema.Check(root, DefinitionDeclaration, file)
	problems = append(problems, checkKind(file, root, KindDefinition)...)
	problems = append(problems, checkName(file, root, "definition")...)
	problems = append(problems, checkDefinitionKindsMixed(file, root)...)

	fields := topLevelFields(root, "provider", "targets", "destroy")
	providerName, providerOK := resolveScalar(fields["provider"])
	var provider ProviderInfo
	haveProvider := false
	if providerOK {
		provider, haveProvider = providers[providerName]
		if !haveProvider {
			problems = append(problems, problem.Problem{
				File: file, Line: fields["provider"].Line, Column: fields["provider"].Column, Field: "provider",
				ErrorCode: CodeArtefactAbsent,
				Message:   fmt.Sprintf("provider: %s resolves to nothing — no built-in Provider and no providers/%s.yaml", providerName, providerName),
			})
		}
	}

	resolved := resolveTargets(file, fields["targets"], targets, &problems)

	if haveProvider {
		problems = append(problems, checkDestroyResolution(file, fields["destroy"], provider)...)
		for _, rt := range resolved {
			problems = append(problems, checkDefinitionTargetPair(file, rt, provider)...)
		}
	}
	return problems
}

// checkDefinitionKindsMixed reports definition-kinds-mixed on read standing
// in kinds: beside mutate, or beside a destroy: claim naming any Operation
// (§3, §4, ADR-0032). It says nothing where kinds: is absent or not a
// sequence — the schema check has already named that fault.
func checkDefinitionKindsMixed(file string, root *yaml.Node) []problem.Problem {
	fields := topLevelFields(root, "kinds", "destroy")
	kindsVal := fields["kinds"]
	if kindsVal == nil || kindsVal.Kind != yaml.SequenceNode {
		return nil
	}
	var readVal, mutateVal *yaml.Node
	for _, item := range kindsVal.Content {
		if item.Kind != yaml.ScalarNode {
			continue
		}
		switch item.Value {
		case "read":
			readVal = item
		case "mutate":
			mutateVal = item
		}
	}
	if readVal == nil {
		return nil
	}

	destroyVal := fields["destroy"]
	claimsDestroy := destroyVal != nil && destroyVal.Kind == yaml.SequenceNode && len(destroyVal.Content) > 0
	if mutateVal == nil && !claimsDestroy {
		return nil
	}
	return []problem.Problem{{
		File: file, Line: readVal.Line, Column: readVal.Column, Field: "kinds",
		ErrorCode: CodeDefinitionKindsMixed,
		Message:   "read stands in kinds: beside mutate or a destroy: claim — a Definition observes or it effects, never both",
	}}
}

// resolveScalar reads val as the name it names, where val is present and a
// plain scalar — the shape a legible reference to another artefact always
// takes. ok is false where val is nil or not a scalar, in which case the
// schema check has already named the fault and no resolution check adds a
// second row for the same line.
func resolveScalar(val *yaml.Node) (name string, ok bool) {
	if val == nil || val.Kind != yaml.ScalarNode {
		return "", false
	}
	return val.Value, true
}

// resolveTargets reads targetsVal's members against targets, reporting
// artefact-absent for one naming nothing there and returning the rest
// already paired with the TargetInfo CheckDefinition's own pair checks read
// (§4). It says nothing where targetsVal is absent or not a sequence — the
// schema check has already named that fault.
func resolveTargets(file string, targetsVal *yaml.Node, targets TargetIndex, problems *[]problem.Problem) []resolvedTarget {
	if targetsVal == nil || targetsVal.Kind != yaml.SequenceNode {
		return nil
	}
	var resolved []resolvedTarget
	for i, item := range targetsVal.Content {
		name, ok := resolveScalar(item)
		if !ok {
			continue
		}
		info, found := targets[name]
		if !found {
			*problems = append(*problems, problem.Problem{
				File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("targets[%d]", i),
				ErrorCode: CodeArtefactAbsent,
				Message:   fmt.Sprintf("targets: %s resolves to nothing — no targets/%s.yaml", name, name),
			})
			continue
		}
		resolved = append(resolved, resolvedTarget{name: name, info: info, node: item, index: i})
	}
	return resolved
}

// resolvedTarget pairs one targets: member's own node with the name it
// resolved to and the TargetInfo BuildTargetIndex read for it, threading
// the node through so the two pair checks below still point a reader at the
// line the binding was authored on.
type resolvedTarget struct {
	name  string
	info  TargetInfo
	node  *yaml.Node
	index int
}

// checkDestroyResolution reports reference-unresolvable on a destroy:
// member naming no Operation the bound Provider declares — the Manifest's
// own namespace rather than the repository's, and a different fault from
// operation-not-claimed, which is a Step reaching an Operation that exists
// and this Definition did not claim (§3, §4). It is called only where the
// Provider itself resolved; an unresolved provider: has already earned
// artefact-absent and there is no Operation namespace left to check against.
func checkDestroyResolution(file string, destroyVal *yaml.Node, provider ProviderInfo) []problem.Problem {
	if destroyVal == nil || destroyVal.Kind != yaml.SequenceNode {
		return nil
	}
	var problems []problem.Problem
	for i, item := range destroyVal.Content {
		name, ok := resolveScalar(item)
		if !ok {
			continue
		}
		if _, declared := provider.Operations[name]; declared {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: item.Line, Column: item.Column, Field: fmt.Sprintf("destroy[%d]", i),
			ErrorCode: CodeReferenceUnresolvable,
			Message:   fmt.Sprintf("destroy: %s names no Operation the bound Provider declares", name),
		})
	}
	return problems
}

// checkDefinitionTargetPair runs the three checks that need a (Definition,
// Target) binding rather than either artefact alone (§3, §4, §5): a Target
// outside the bound Provider's declared class is target-class-mismatch; a
// Capability the Provider's Operations require and the Target does not
// grant is capability-not-granted; and a Target whose credential slots do
// not cover the Provider's Auth scheme is manifest-inconsistent — the
// twelfth shape, decidable only once a binding exists. Each points a reader
// at the targets: member that made the binding, since that is the line
// whose edit — bind a different Target, or widen the one bound — fixes it.
func checkDefinitionTargetPair(file string, rt resolvedTarget, provider ProviderInfo) []problem.Problem {
	var problems []problem.Problem
	line, column := rt.node.Line, rt.node.Column
	field := fmt.Sprintf("targets[%d]", rt.index)

	if provider.Class != "" && rt.info.Class != "" && provider.Class != rt.info.Class {
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeTargetClassMismatch,
			Message:   fmt.Sprintf("%s is class %s, and the bound Provider's declared class is %s", rt.name, rt.info.Class, provider.Class),
		})
	}

	capabilities := make([]string, 0, len(provider.Capabilities))
	for capability := range provider.Capabilities {
		capabilities = append(capabilities, capability)
	}
	sort.Strings(capabilities)
	for _, capability := range capabilities {
		if rt.info.Capabilities[capability] {
			continue
		}
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeCapabilityNotGranted,
			Message:   fmt.Sprintf("%s does not grant %s, and the bound Provider's Operations require it", rt.name, capability),
		})
	}

	var missingSlots []string
	for _, slot := range provider.AuthSlots {
		if !rt.info.AuthSlots[slot] {
			missingSlots = append(missingSlots, slot)
		}
	}
	if len(missingSlots) > 0 {
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeManifestInconsistent,
			Message:   fmt.Sprintf("%s's credential slots do not cover the bound Provider's Auth scheme — missing %s", rt.name, strings.Join(missingSlots, ", ")),
		})
	}
	return problems
}
