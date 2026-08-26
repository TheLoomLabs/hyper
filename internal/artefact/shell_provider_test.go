package artefact

import "testing"

// TestCheckBuiltinShellProvider_IsClean is issue #91's acceptance criterion
// that the built-in shell Provider's bytes check clean with no exemption
// (§12).
func TestCheckBuiltinShellProvider_IsClean(t *testing.T) {
	mustNone(t, CheckBuiltinShellProvider())
}

// TestCheckBuiltinShellProvider_ReachesNeitherExtensionCode is §11's rule read
// from the side it does not apply to. Both of what an Extension may never be
// are about a Manifest **loaded from providers/**, and the built-in is loaded
// from neither a file nor that directory: it declares the reserved Capability
// and is entitled to, and it holds the name an Extension may not take.
//
// capability-reserved is asserted here rather than only through the clean
// check above because the clean check would go on passing if the rule were
// written into checkManifestBody by mistake and the built-in exempted by a
// branch — the exemption §11 does not have and ADR-0081 refuses. What says the
// built-in is untouched is that the function it runs never asks.
func TestCheckBuiltinShellProvider_ReachesNeitherExtensionCode(t *testing.T) {
	mustNoCode(t, CheckBuiltinShellProvider(), CodeCapabilityReserved)

	// provider-name-collision is verify's, its subject being the Provider
	// namespace rather than a file (issue #185), so what is held here is the
	// predicate both codes read the built-in's own facts through.
	if !IsBuiltinProviderName(BuiltinShellProviderName) {
		t.Errorf("IsBuiltinProviderName(%q) = false, want the built-in's own name in the set", BuiltinShellProviderName)
	}
	if !IsReservedCapability(ReservedCapability) {
		t.Errorf("IsReservedCapability(%q) = false, want the reserved member in the set", ReservedCapability)
	}
	if IsReservedCapability("http") {
		t.Errorf("IsReservedCapability(\"http\") = true, want http unreserved — §12 reserves exactly one of the two")
	}
}
