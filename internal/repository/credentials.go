package repository

import (
	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// CredentialSlots is the credential slots one (Definition, Target) pair's
// binding requires, resolved against the repository: the file the Target
// declaration was read from, and one entry per slot the bound Provider's
// scheme names that the declaration carries.
//
// It is a method on the load rather than a walk each caller writes because two
// commands ask it and they ask for different halves of one answer. A Run
// resolves the **values** and Refuses where the environment holds none, citing
// the declaration's file and the slot's own line (§6, §12); `project` writes the
// **names** into the generated workflow's `env:` block, an executor secret per
// slot (§10). A second walk would be the day the job's block and the Run's gate
// disagree about which variables a Procedure needs, which is a Run that Refuses
// on a runner and passes on a laptop.
//
// **It is the scheme's slots and never the declaration's whole `auth:`.** A
// Target may carry slots for a scheme this binding never uses, and writing those
// into a job would put a secret on the runner that no Step could reach (§10,
// ADR-0007); the order is the scheme's own, which is §12's.
//
// Two shapes contribute nothing and neither is reported here, both being
// `check`'s: a Target whose slots do not cover the bound Provider's scheme is
// `manifest-inconsistent`, and a slot naming no variable is
// `credential-slot-malformed` (§4, ADR-0064). A pair whose Definition, Provider
// or Target does not resolve contributes nothing for the same reason.
func (l Loaded) CredentialSlots(pair store.Pair) (file string, slots []artefact.CredentialSlot) {
	declaration, held := l.TargetDeclaration(pair.Target)
	if !held {
		return "", nil
	}
	declared := artefact.ReadTargetFacts(declaration.Root).Credentials

	for _, slot := range l.authOf(pair.Definition).Slots() {
		for _, named := range declared {
			if named.Slot == slot {
				slots = append(slots, named)
				break
			}
		}
	}
	return declaration.Path, slots
}

// authOf is the Auth scheme the Definition's Provider names, and the empty
// scheme — which requires no slot at all — where the binding does not resolve
// far enough to say.
func (l Loaded) authOf(definition string) capability.Auth {
	info, declared := l.Definitions[definition]
	if !declared {
		return capability.Auth{}
	}
	manifest, published := l.Manifests[info.ProviderName]
	if !published {
		return capability.Auth{}
	}
	return capability.ReadAuth(manifest.Root)
}
