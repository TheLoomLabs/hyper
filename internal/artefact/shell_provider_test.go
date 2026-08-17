package artefact

import "testing"

// TestCheckBuiltinShellProvider_IsClean is issue #91's acceptance criterion
// that the built-in shell Provider's bytes check clean with no exemption
// (§12).
func TestCheckBuiltinShellProvider_IsClean(t *testing.T) {
	mustNone(t, CheckBuiltinShellProvider())
}
