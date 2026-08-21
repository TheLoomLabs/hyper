package artefact

// A Probe binds `local` and reaches one host, and this file is the whole of
// how that host is decided (§9, ADR-0029, ADR-0042, issue #135).
//
// It lives here because reach is this package's subject. `hyper probe` names no
// Definition and no Step, so nothing in the repository authorised the call —
// and the one thing it does not escape is the grant, which comes from an
// artefact even where no artefact named the Operation. The three steps it walks
// are the three procedure.go already checks a Step's binding against: the
// candidate set the Operation's `host:` template expands to, the `hosts:` the
// Target declares, and their intersection. A second reading of them at the
// surface is a second place for the reach rule to drift.
//
// What is deliberately not here is the rendering. Which of these answers is a
// Refusal, which is a usage error, and what either says to a reader is
// internal/cli's, on the same line every check in this package draws: this
// answers what the artefacts decided, and the surface says it.

// Reach is what resolving a Probe's host answered.
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
	// Operation (§4) — the fault is decidable only at a binding, and a
	// Probe is a binding no artefact wrote.
	ReachUndecidable
	// ReachIllegible: the `host:` template writes a hole naming neither
	// `from-target` nor a declared enumeration, which `check` has already
	// refused as `hole-illegal`. There is nothing to expand and so nothing
	// to decide (ADR-0064).
	ReachIllegible
)

// ProbeHost is where a Probe's call goes, and why it goes nowhere.
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
type ProbeHost struct {
	Reach   Reach
	Host    string
	Granted int
}

// ResolveProbeHost walks ADR-0029's three steps at invocation rather than at
// load. suppliedHost is the value of the input the Operation's `host-input:`
// names, and is ignored where it names none.
//
// The two arms are §3's own, and they do not consult the same thing. Where the
// Operation declares `host-input:`, the input **always carries a whole host**
// and *the value that input carries is checked for grant membership like any
// other* — an enumeration hole being a compact way of writing a large candidate
// set rather than a second thing filled at Run time, and the candidate set
// having done its work at load, where the grant was checked against it. Where
// it declares none, the intersection decides and `hyper` fills the one host it
// resolved to.
func ResolveProbeHost(provider ProviderInfo, operation OperationInfo, target TargetInfo, suppliedHost string) ProbeHost {
	if operation.HostInput != "" {
		if target.Hosts[suppliedHost] {
			return ProbeHost{Reach: ReachGranted, Host: suppliedHost}
		}
		return ProbeHost{Reach: ReachNotGranted, Host: suppliedHost}
	}

	candidates, expanded := expandHostTemplate(operation.HostTemplate, provider.Enumerations, target.Hosts)
	if !expanded {
		return ProbeHost{Reach: ReachIllegible}
	}

	var granted []string
	for _, candidate := range candidates {
		if target.Hosts[candidate] {
			granted = append(granted, candidate)
		}
	}
	switch {
	case len(granted) == 1:
		return ProbeHost{Reach: ReachGranted, Host: granted[0]}
	case len(granted) > 1:
		return ProbeHost{Reach: ReachUndecidable, Granted: len(granted)}
	case len(candidates) > 0:
		return ProbeHost{Reach: ReachNotGranted, Host: candidates[0]}
	default:
		return ProbeHost{Reach: ReachNotGranted}
	}
}
