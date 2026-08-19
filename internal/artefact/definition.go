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

// CodeOpaqueDestroyNotGranted is the code a Definition claiming an opaque
// destroy Operation against a Target whose declaration has not opted into
// opaque-destroy: earns, checked per (Definition, Target) pair on
// CodeCapabilityNotGranted's own rule — the artefact half of the check; the
// credential half is resolved at Run start and belongs to §5 (§4, §5,
// issue #95).
const CodeOpaqueDestroyNotGranted = "opaque-destroy-not-granted"

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
// namespace a destroy: member and a Step's operation: resolve against — the
// credential slot names its Auth scheme requires, nil where it
// authenticates nothing, and RecordFields and SecretFields, the union of
// every Operation's own two sets of the same name (§3, §4, §12, issue #97).
// The union rather than one Operation's own set is what a selector's and a
// condition's field: resolve against: assets:/observations: range over the
// (Definition, Target) Record series a different Operation of the same
// Provider may have written — a destroy Step's own Operation projects no
// record: at all, and its selector still names the fields the mutate that
// created what it destroys projected.
type ProviderInfo struct {
	Class        string
	Capabilities map[string]bool
	Operations   map[string]OperationInfo
	AuthSlots    []string
	RecordFields map[string]bool
	SecretFields map[string]bool
	// Enumerations is the Manifest's own enumerations: block, name to
	// members — the one source a Capability-relevant hole in an
	// Operation's host: may resolve to, and half the candidate set
	// expansion the host grant is checked against (§3, §12, issue #98).
	Enumerations map[string][]string
}

// OperationInfo is what checking a Step against the Operation it binds
// needs, read once per repository pass alongside the rest of ProviderInfo
// rather than reparsed per Step that names it (§3, §4, issue #94): whether
// its request is the shell Capability, the one hyper's own Provider may
// declare and the one whose argv arrives as the input named command, and
// the same fact by which opacity is read rather than declared (§5, §13,
// issue #95); its own Kind — read, mutate or destroy — the two keys and the
// Bound both check against; its Record cardinality, series where its
// record: carries an over: and one otherwise — the fact a reference's step:
// half is refused against (series-reference); the field names its record:
// projects, nil on an Operation with no record: at all — the namespace a
// reference's path: half resolves against; every input its input: schema
// declares, by name; its own Repeatability, "" where undeclared — the fact
// IsRunOnce reads, and the two Cadence rules' own walk with it (issue #96);
// whether its own secret: is present and names at least one field — the
// other fact that same walk reads; and SecretFields, the field names
// secret: itself names — the set a predicate's own field: is checked
// against, nil where secret: is absent (§3, §4, §12, issue #97).
type OperationInfo struct {
	IsShell       bool
	Kind          string
	HasSeries     bool
	RecordFields  map[string]bool
	Inputs        map[string]InputInfo
	Repeatability string
	HasSecret     bool
	SecretFields  map[string]bool
	// HostTemplate is the raw host: scalar an http: block carries — ""
	// on a shell Operation, which has no host: at all — the template
	// whose at-load expansion is the candidate set the bound Target's
	// grant is checked against (§3, ADR-0029, issue #98).
	HostTemplate string
	// HostInput is the raw host-input: scalar — "" where the Operation
	// declares none — naming the one input that carries a whole host
	// where the candidate set and the grant intersect to several, and
	// the position that makes an over: values: list a host list where a
	// Step wires {item: $} into it (§3, §4, issue #98).
	HostInput string
	// Identity is the raw identity: scalar record: carries — "" on a
	// destroy, which projects nothing — read to decide whether an
	// Expansion's members can be shown to project one identity before
	// any call goes out: a template hole or $.command on a shell
	// Operation resolve before the call, and any response path does not
	// (§3, §4, issue #98).
	Identity string
}

// IsOpaqueDestroy reports whether this Operation is the one Step §5's Bound
// rule and its over: requirement both turn on — a destroy Operation whose
// request is opaque, the shell Capability (§4, §5, §13, issue #95).
func (o OperationInfo) IsOpaqueDestroy() bool {
	return o.Kind == "destroy" && o.IsShell
}

// IsRunOnce reports whether this Operation is the effectful default the
// Cadence walk refuses at any depth (§4, §5, issue #96): its repeatability:
// is omitted and its Kind is mutate or destroy — the one combination with
// no spelling of its own, run-once having none. A read's undeclared
// Repeatability is always repeatable rather than run-once (§12), so a read
// is never run-once whatever its Repeatability reads.
func (o OperationInfo) IsRunOnce() bool {
	return o.Repeatability == "" && (o.Kind == "mutate" || o.Kind == "destroy")
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
// the class it declares, the Capabilities it grants, the credential slot
// names its auth: mapping supplies, the Kinds it accepts — the grant half of
// the two keys, a Step's bound Definition supplying the claim — whether
// it has opted into opaque-destroy: (§3, §4, §5, issue #95), and the granted
// host set its hosts: enumerates — the comparand both host-not-granted
// comparisons run against (§3, §4, ADR-0024, ADR-0029, issue #98).
type TargetInfo struct {
	Class         string
	Capabilities  map[string]bool
	AuthSlots     map[string]bool
	Kinds         map[string]bool
	OpaqueDestroy bool
	Hosts         map[string]bool
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
		name := ManifestProviderName(root)
		if name == "" {
			continue
		}
		idx[name] = providerInfoFromManifest(root)
	}
	return idx
}

// ManifestProviderName is the name a Manifest declares for itself, or "" where
// its provider: is absent or is not a plain scalar. It is exported because the
// Provider namespace is not the only thing folded over that rule: `hyper
// providers` writes one row per member of that namespace and needs the bytes
// each name loaded from, which the index does not carry (§9, issue #111) — and
// two folds of one rule written twice is where the day comes that a name is in
// the namespace and not on the list.
func ManifestProviderName(root *yaml.Node) string {
	return DeclaredName(root, "provider")
}

// BuildTargetIndex adds one entry per targets/ root whose target: is a
// legible scalar, on BuildProviderIndex's own rule. The name is read through
// TargetDeclarationName, which is the same rule the load folds each name to its
// declaration by: the namespace a targets: resolves against and the list `hyper
// targets` writes cannot disagree about which declaration a name means (issue
// #112).
func BuildTargetIndex(declarationRoots []*yaml.Node) TargetIndex {
	idx := TargetIndex{}
	for _, root := range declarationRoots {
		name := TargetDeclarationName(root)
		if name == "" {
			continue
		}
		idx[name] = targetInfoFromDeclaration(root)
	}
	return idx
}

// DefinitionInfo is what checking a Step against the Definition it binds
// needs, read once per repository pass rather than reparsed per Step that
// names it (§3, §4, §5, issue #95): the provider: it names, unresolved — a
// Step's operation: and args: are checked against a second lookup, into
// ProviderIndex, once ProviderName has resolved; the Kinds it claims via
// kinds: — read and/or mutate, the claim half of the two keys; the
// Operations it claims for destroy:, by name — the same set both
// operation-not-claimed and the destroy half of ClaimsKind read; and the
// targets: it claims, resolved against TargetIndex and keyed by name — the
// namespace a Step's target: resolves against and nothing wider (§4), and
// the source of the Kinds a Step's bound Target grants.
type DefinitionInfo struct {
	ProviderName string
	Kinds        map[string]bool
	Destroy      map[string]bool
	Targets      map[string]TargetInfo
}

// ClaimsKind reports whether this Definition's own claim covers kind —
// membership in kinds: for read and mutate, and a non-empty destroy: for
// destroy, granularity following severity the same way the Step-level check
// against it does (§4, §5, issue #95).
func (d DefinitionInfo) ClaimsKind(kind string) bool {
	if kind == "destroy" {
		return len(d.Destroy) > 0
	}
	return d.Kinds[kind]
}

// DefinitionIndex maps a definitions/ file's own definition: to what a Step
// binding it is checked against — a Step's definition: resolves against
// this namespace (§3, §4, issue #94).
type DefinitionIndex map[string]DefinitionInfo

// BuildDefinitionIndex adds one entry per definitions/ root whose
// definition: is a legible scalar, on BuildProviderIndex's own rule. targets
// is the namespace a targets: member resolves against — the same
// TargetIndex CheckDefinition's own per-pair checks read — so a member that
// does not resolve there contributes nothing to DefinitionInfo.Targets,
// CheckDefinition having already named that fault on the Definition's own
// line (ADR-0064). An entry whose provider: is absent or illegible carries
// ProviderName "" — a Step naming this Definition resolves no Operation
// against an empty provider name, which is reference-unresolvable on the
// same rule as any other name that does not resolve.
func BuildDefinitionIndex(definitionRoots []*yaml.Node, targets TargetIndex) DefinitionIndex {
	idx := DefinitionIndex{}
	for _, root := range definitionRoots {
		fields := topLevelFields(root, "definition", "provider", "kinds", "destroy", "targets")
		nameVal := fields["definition"]
		if nameVal == nil || nameVal.Kind != yaml.ScalarNode {
			continue
		}
		info := DefinitionInfo{Kinds: map[string]bool{}, Destroy: map[string]bool{}, Targets: map[string]TargetInfo{}}
		if providerVal := fields["provider"]; providerVal != nil && providerVal.Kind == yaml.ScalarNode {
			info.ProviderName = providerVal.Value
		}
		if kindsVal := fields["kinds"]; kindsVal != nil && kindsVal.Kind == yaml.SequenceNode {
			for _, item := range kindsVal.Content {
				if item.Kind == yaml.ScalarNode {
					info.Kinds[item.Value] = true
				}
			}
		}
		if destroyVal := fields["destroy"]; destroyVal != nil && destroyVal.Kind == yaml.SequenceNode {
			for _, item := range destroyVal.Content {
				if item.Kind == yaml.ScalarNode {
					info.Destroy[item.Value] = true
				}
			}
		}
		if targetsVal := fields["targets"]; targetsVal != nil && targetsVal.Kind == yaml.SequenceNode {
			for _, item := range targetsVal.Content {
				if item.Kind != yaml.ScalarNode {
					continue
				}
				if t, ok := targets[item.Value]; ok {
					info.Targets[item.Value] = t
				}
			}
		}
		idx[nameVal.Value] = info
	}
	return idx
}

// providerInfoFromManifest reads the six facts CheckDefinition and a
// predicate's own field: resolution need off a Manifest's own root: class:,
// capabilities:, the name set of operations:, the credential slots its
// auth: scheme requires, and RecordFields and SecretFields, each the union
// of every Operation's own set of the same name (§3, §4, §12, issue #97).
func providerInfoFromManifest(root *yaml.Node) ProviderInfo {
	fields := topLevelFields(root, "class", "capabilities", "operations", "enumerations")
	info := ProviderInfo{
		Capabilities: map[string]bool{}, Operations: map[string]OperationInfo{},
		RecordFields: map[string]bool{}, SecretFields: map[string]bool{},
		Enumerations: map[string][]string{},
	}
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
			if key.Kind != yaml.ScalarNode {
				continue
			}
			opInfo := operationInfoFromNode(opNode)
			info.Operations[key.Value] = opInfo
			for name := range opInfo.RecordFields {
				info.RecordFields[name] = true
			}
			for name := range opInfo.SecretFields {
				info.SecretFields[name] = true
			}
		}
	}
	info.AuthSlots = authSlotNames(root)
	if enumVal := fields["enumerations"]; enumVal != nil && enumVal.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(enumVal.Content); i += 2 {
			key, val := enumVal.Content[i], enumVal.Content[i+1]
			if key.Kind != yaml.ScalarNode || val.Kind != yaml.SequenceNode {
				continue
			}
			var members []string
			for _, item := range val.Content {
				if item.Kind == yaml.ScalarNode {
					members = append(members, item.Value)
				}
			}
			info.Enumerations[key.Value] = members
		}
	}
	return info
}

// targetInfoFromDeclaration reads the six facts CheckDefinition and the
// Step-level authority checks need off a Target declaration's own root:
// class:, capabilities:, the credential slot names its auth: mapping's own
// keys are, the Kinds it accepts, whether it opts into opaque-destroy:
// (§3, §4, §5, issue #95), and the granted host set hosts: enumerates —
// nil where the declaration grants no http and so carries none (§3,
// issue #98).
//
// Every one of them is a membership question, which is what a check asks: does
// this Target grant that Capability, accept that Kind, supply that slot. What
// the same declaration *states*, in its own order, is ReadTargetFacts's, and
// the two share the scans below rather than each walking the node themselves.
func targetInfoFromDeclaration(root *yaml.Node) TargetInfo {
	fields := topLevelFields(root, "class", "capabilities", "auth", "kinds", "opaque-destroy", "hosts")
	info := TargetInfo{Capabilities: map[string]bool{}, AuthSlots: map[string]bool{}, Kinds: map[string]bool{}}
	if classVal := fields["class"]; classVal != nil && classVal.Kind == yaml.ScalarNode {
		info.Class = classVal.Value
	}
	for _, capability := range scalarSequence(fields["capabilities"]) {
		info.Capabilities[capability] = true
	}
	for _, slot := range credentialSlots(fields["auth"]) {
		info.AuthSlots[slot.Slot] = true
	}
	for _, kind := range scalarSequence(fields["kinds"]) {
		info.Kinds[kind] = true
	}
	info.OpaqueDestroy = grantsOpaqueDestroy(root)
	// hosts: is read as a set here and as an enumeration by ReadTargetFacts,
	// two readings of one key: a check asks whether a candidate is granted,
	// and a row states what the grant is. The set stays nil where the
	// declaration enumerates none, which is what says a Target granting no
	// http has no host grant rather than an empty one.
	if hostsVal := fields["hosts"]; hostsVal != nil && hostsVal.Kind == yaml.SequenceNode {
		info.Hosts = map[string]bool{}
		for _, host := range scalarSequence(hostsVal) {
			info.Hosts[host] = true
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
	fields := topLevelFields(authVal, authSchemes...)
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
	return providerInfoFromManifest(BuiltinShellProviderRoot())
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
		claimsOpaqueDestroy := definitionClaimsOpaqueDestroy(fields["destroy"], provider)
		for _, rt := range resolved {
			problems = append(problems, checkDefinitionTargetPair(file, rt, provider, claimsOpaqueDestroy)...)
		}
	}
	return problems
}

// definitionClaimsOpaqueDestroy reports whether destroyVal names at least
// one Operation among provider's own that is opaque — the shell Capability,
// the one Capability behind an opaque Operation (§5, §13) — which is what
// decides whether opaque-destroy-not-granted applies to a (Definition,
// Target) pair below. It says nothing where destroyVal is absent or not a
// sequence, or where a member does not resolve — checkDestroyResolution has
// already named that fault, and there is no Operation here to read Kind or
// opacity off.
func definitionClaimsOpaqueDestroy(destroyVal *yaml.Node, provider ProviderInfo) bool {
	if destroyVal == nil || destroyVal.Kind != yaml.SequenceNode {
		return false
	}
	for _, item := range destroyVal.Content {
		name, ok := resolveScalar(item)
		if !ok {
			continue
		}
		if op, declared := provider.Operations[name]; declared && op.IsShell {
			return true
		}
	}
	return false
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

// checkDefinitionTargetPair runs the four checks that need a (Definition,
// Target) binding rather than either artefact alone (§3, §4, §5): a Target
// outside the bound Provider's declared class is target-class-mismatch; a
// Capability the Provider's Operations require and the Target does not
// grant is capability-not-granted; a Target whose credential slots do not
// cover the Provider's Auth scheme is manifest-inconsistent — the twelfth
// shape, decidable only once a binding exists; and, where claimsOpaqueDestroy
// is true, a Target that has not opted into opaque-destroy: is
// opaque-destroy-not-granted — the artefact half of the check, the credential
// half needing a Run to resolve and belonging to §5. Each points a reader at
// the targets: member that made the binding, since that is the line whose
// edit — bind a different Target, or widen the one bound — fixes it.
func checkDefinitionTargetPair(file string, rt resolvedTarget, provider ProviderInfo, claimsOpaqueDestroy bool) []problem.Problem {
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

	if claimsOpaqueDestroy && !rt.info.OpaqueDestroy {
		problems = append(problems, problem.Problem{
			File: file, Line: line, Column: column, Field: field,
			ErrorCode: CodeOpaqueDestroyNotGranted,
			Message:   fmt.Sprintf("%s has not opted into opaque-destroy:, and this Definition claims an opaque destroy Operation", rt.name),
		})
	}
	return problems
}
