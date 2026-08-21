package artefact

import "github.com/TheLoomLabs/hyper/internal/schema"

// One invocation reaches one host, and this file is the whole of how that host
// is decided (§3, §9, ADR-0029, ADR-0042, issue #135, issue #136).
//
// It lives here because reach is this package's subject, and it has two callers
// that are one rule. A Probe names no Definition and no Step, so nothing in the
// repository authorised the call — and the one thing it does not escape is the
// grant, which comes from an artefact even where no artefact named the
// Operation. A Run's Step is the reviewed case of the same walk. The three
// steps it takes are the three procedure.go already checks a Step's binding
// against: the candidate set the Operation's `host:` template expands to, the
// `hosts:` the Target declares, and their intersection. A second reading of
// them at either surface is a second place for the reach rule to drift.
//
// What is deliberately not here is the rendering. Which of these answers is a
// Refusal, which is a usage error, and what either says to a reader is
// internal/cli's, on the same line every check in this package draws: this
// answers what the artefacts decided, and the surface says it.

// Reach is what resolving an invocation's host answered.
type Reach int

const (
	// ReachGranted: the call reaches Host, which the grant admits.
	ReachGranted Reach = iota
	// ReachNotGranted: nothing the call could reach is granted.
	// `host-not-granted` (§4, ADR-0042).
	ReachNotGranted
	// ReachUndecidable: the candidate set and the grant intersect to
	// several hosts under an Operation naming no `host-input:`, so which
	// host the request would reach is not decided by anything. `check`
	// reports this as `manifest-inconsistent` wherever a Step binds the
	// Operation (§4), so a Run reaches it only where `check` has not run;
	// a Probe is a binding no artefact wrote, and reaches it always.
	ReachUndecidable
	// ReachIllegible: the `host:` template writes a hole naming neither
	// `from-target` nor a declared enumeration, which `check` has already
	// refused as `hole-illegal`. There is nothing to expand and so nothing
	// to decide (ADR-0064).
	ReachIllegible
)

// HostReach is where a call goes, and why it goes nowhere.
//
// Host is the host the call reaches under ReachGranted, and under
// ReachNotGranted the host it *would* have reached — the value the input
// carried, or the first candidate the template expanded to — so a Refusal can
// name what was asked for. It is "" where nothing named one at all, which is
// `{from-target}` against a grant with nothing in it: the grant is the only
// thing that could have named a host, and it named none.
//
// Granted is how many hosts the candidate set and the grant intersected to, and
// is read only under ReachUndecidable, that being the count above one.
type HostReach struct {
	Reach   Reach
	Host    string
	Granted int
}

// SuppliedHost is the value of the input the Operation's `host-input:` names,
// and "" where it names none — the whole of what the resolution below needs
// from an invocation's arguments (§3, ADR-0029).
//
// It is here rather than at either caller because the reading is the same
// reading at both: a Probe's `--input` and a Step's `args:` are one map of
// resolved inputs, and which of them carries a host is the Operation's fact.
func (o OperationInfo) SuppliedHost(inputs map[string]schema.Scalar) string {
	if o.HostInput == "" {
		return ""
	}
	return inputs[o.HostInput].Text()
}

// ResolveHost walks ADR-0029's three steps at invocation rather than at
// load. suppliedHost is the value of the input the Operation's `host-input:`
// names, and is ignored where it names none — which is where a Probe's
// `--input` and a Step's `args:` arrive at one function: both supply the
// Operation's inputs, and neither can name a host the grant does not hold.
//
// The two arms are §3's own, and they do not consult the same thing. Where the
// Operation declares `host-input:`, the input **always carries a whole host**
// and *the value that input carries is checked for grant membership like any
// other* — an enumeration hole being a compact way of writing a large candidate
// set rather than a second thing filled at Run time, and the candidate set
// having done its work at load, where the grant was checked against it. Where
// it declares none, the intersection decides and `hyper` fills the one host it
// resolved to.
func ResolveHost(provider ProviderInfo, operation OperationInfo, target TargetInfo, suppliedHost string) HostReach {
	if operation.HostInput != "" {
		if target.Hosts[suppliedHost] {
			return HostReach{Reach: ReachGranted, Host: suppliedHost}
		}
		return HostReach{Reach: ReachNotGranted, Host: suppliedHost}
	}

	candidates, expanded := expandHostTemplate(operation.HostTemplate, provider.Enumerations, target.Hosts)
	if !expanded {
		return HostReach{Reach: ReachIllegible}
	}

	var granted []string
	for _, candidate := range candidates {
		if target.Hosts[candidate] {
			granted = append(granted, candidate)
		}
	}
	switch {
	case len(granted) == 1:
		return HostReach{Reach: ReachGranted, Host: granted[0]}
	case len(granted) > 1:
		return HostReach{Reach: ReachUndecidable, Granted: len(granted)}
	case len(candidates) > 0:
		return HostReach{Reach: ReachNotGranted, Host: candidates[0]}
	default:
		return HostReach{Reach: ReachNotGranted}
	}
}
