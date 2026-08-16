package pin

import "testing"

func TestCheck_Absent(t *testing.T) {
	got := Check(false, nil, "1.4.0")
	if !got.Refused || got.Code != CodeAbsent {
		t.Fatalf("Check() = %+v, want refused version-pin-absent", got)
	}
}

func TestCheck_UnreadablePinIsAbsent(t *testing.T) {
	got := Check(true, []byte("version:\n"), "1.4.0")
	if !got.Refused || got.Code != CodeAbsent {
		t.Fatalf("Check() = %+v, want refused version-pin-absent", got)
	}
}

func TestCheck_Match(t *testing.T) {
	got := Check(true, []byte("kind: repository-declaration\nversion: 1.4.0\n"), "1.4.0")
	if got.Refused {
		t.Fatalf("Check() = %+v, want not refused", got)
	}
}

func TestCheck_MismatchBinaryNewer(t *testing.T) {
	got := Check(true, []byte("version: 1.3.0\n"), "1.4.0")
	if !got.Refused || got.Code != CodeMismatch {
		t.Fatalf("Check() = %+v, want refused version-pin-mismatch", got)
	}
}

func TestCheck_MismatchBinaryOlder(t *testing.T) {
	got := Check(true, []byte("version: 1.5.0\n"), "1.4.0")
	if !got.Refused || got.Code != CodeMismatch {
		t.Fatalf("Check() = %+v, want refused version-pin-mismatch", got)
	}
}

func TestCheck_ComparisonIsExactNotSemver(t *testing.T) {
	// §11: no range, no minimum, no compatible-release operator. A patch
	// difference is a mismatch like any other.
	got := Check(true, []byte("version: 1.4.1\n"), "1.4.0")
	if !got.Refused || got.Code != CodeMismatch {
		t.Fatalf("Check() = %+v, want refused version-pin-mismatch on a patch difference", got)
	}
}
