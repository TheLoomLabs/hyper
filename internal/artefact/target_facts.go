package artefact

import "gopkg.in/yaml.v3"

// CredentialSlot is one member of a Target declaration's auth: mapping as a
// surface reports it: the slot the repository named, and the environment
// variable that slot resolves from. The variable's name and never its value —
// nothing hyper writes has ever held a secret, and the pair exists so that a
// declaration carrying slots for more than one scheme says which fills what
// (§3, §9, ADR-0007).
//
// Env is empty where the slot names no variable, which is a slot whose value is
// not the mapping whose sole key is env: that §4 fixes. The slot is still a
// slot the declaration carries, so it is still reported; what is wrong with it
// is credential-slot-malformed, which is check's to name (ADR-0064).
type CredentialSlot struct {
	Slot string
	Env  string
}

// TargetFacts is what a Target declaration grants, in the shape a surface
// reports it: ordered lists rather than the membership sets TargetInfo carries.
//
// The two readings are one artefact read for two questions. A check asks *does
// this Target grant that host*, which is a set; a row states *what does this
// Target grant*, which is an enumeration and is answered in the declaration's
// own order — a grant silently reduced or re-sorted is not the grant the
// reviewer has open beside it (§3, ADR-0024, ADR-0029).
//
// Hosts is nil where the declaration enumerates no host, which is a Target
// granting no http: hosts: is the one key here a declaration may leave out, and
// the ordinary absence rule is what a reader reads off it (§4, §7).
type TargetFacts struct {
	Hosts        []string
	Kinds        []string
	Capabilities []string
	Credentials  []CredentialSlot
}

// ReadTargetFacts reads those four facts off a Target declaration's own root.
// It judges none of them: a declaration that names a Kind outside the closed
// set, or a host that is not a hostname, states what it states here and earns
// its problem from check (ADR-0064). What it does drop is what it cannot read —
// a list member that is not a plain scalar has no value to report.
func ReadTargetFacts(root *yaml.Node) TargetFacts {
	fields := topLevelFields(root, "hosts", "kinds", "capabilities", "auth")
	return TargetFacts{
		Hosts:        scalarSequence(fields["hosts"]),
		Kinds:        scalarSequence(fields["kinds"]),
		Capabilities: scalarSequence(fields["capabilities"]),
		Credentials:  credentialSlots(fields["auth"]),
	}
}

// scalarSequence is a sequence's plain scalars in the sequence's own order, nil
// where the key is absent or is not a sequence at all.
func scalarSequence(val *yaml.Node) []string {
	if val == nil || val.Kind != yaml.SequenceNode {
		return nil
	}
	var members []string
	for _, item := range val.Content {
		if item.Kind == yaml.ScalarNode {
			members = append(members, item.Value)
		}
	}
	return members
}

// credentialSlots is the auth: mapping's members in the mapping's own order,
// each paired with the variable its env: names. A slot whose value is not a
// mapping carrying a legible env: is carried with no variable rather than
// dropped: the declaration has that slot, and a row that hid it would report a
// Target as having fewer credentials in place than it asks for.
func credentialSlots(authVal *yaml.Node) []CredentialSlot {
	if authVal == nil || authVal.Kind != yaml.MappingNode {
		return nil
	}
	var slots []CredentialSlot
	for i := 0; i+1 < len(authVal.Content); i += 2 {
		key, val := authVal.Content[i], authVal.Content[i+1]
		if key.Kind != yaml.ScalarNode {
			continue
		}
		slots = append(slots, CredentialSlot{Slot: key.Value, Env: envVariable(val)})
	}
	return slots
}

// envVariable is the variable one credential slot resolves from — the value of
// its sole env: key — or "" where the slot's value is not the mapping §4 fixes.
func envVariable(val *yaml.Node) string {
	if val == nil || val.Kind != yaml.MappingNode {
		return ""
	}
	for i := 0; i+1 < len(val.Content); i += 2 {
		key, envVal := val.Content[i], val.Content[i+1]
		if key.Kind == yaml.ScalarNode && key.Value == "env" && envVal.Kind == yaml.ScalarNode {
			return envVal.Value
		}
	}
	return ""
}

// TargetDeclarationName is the name a Target declaration declares for itself,
// or "" where its target: is absent or is not a plain scalar. It is exported
// for ManifestProviderName's own reason: the Target namespace is not the only
// thing folded over that rule — the load folds each name to the declaration it
// came from, so `hyper targets` and a Definition's targets: resolve one name to
// one declaration by construction rather than by two folds agreeing (§9, issue
// #112).
func TargetDeclarationName(root *yaml.Node) string {
	return DeclaredName(root, "target")
}
