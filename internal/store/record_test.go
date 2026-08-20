package store_test

import (
	"strings"
	"testing"
	"time"

	"github.com/TheLoomLabs/hyper/internal/store"
)

// A Record version is one file holding that version's projected content and its
// metadata together — one artefact that is diffable and canonical at once (§7).
// §7 publishes an ordinary version and a Tombstone in full, and both are
// compared byte for byte here: a version is minted only where the bytes moved,
// so the bytes are the shape (issue #129).

// theProvenance is the whole of it, which only a Record version carries.
var theProvenance = store.Provenance{
	Run: store.RunProvenance{
		HyperVersion:      "1.4.0",
		ProcedureRevision: "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
		RepoRevision:      "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12",
	},
	Step: store.StepProvenance{
		DefinitionRevision: "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
		ManifestDigest:     "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
	},
}

func recordVersion(t *testing.T) store.RecordVersion {
	t.Helper()
	return store.RecordVersion{
		Metadata: store.Metadata{
			Identity:   store.Identity{Target: "cloudflare-prod", Definition: "preview-dns", Name: "preview-42.example.com"},
			RecordType: store.RecordAsset,
			Run:        entryRun(t),
			Step:       1,
			Operation:  "create_dns_record",
			WrittenAt:  time.Date(2026, 8, 6, 9, 41, 14, 221_000_000, time.UTC),
			Provenance: theProvenance,
		},
		Fields: store.Mapping{
			"created_on": store.String("2026-08-06T09:41:14Z"),
			"id":         store.String("372e67954025e0ba6aaa6d586b9e0b59"),
			"name":       store.String("preview-42.example.com"),
		},
	}
}

func TestRecordVersion_IsTheFileSectionSevenPublishes(t *testing.T) {
	const want = `{
  "definition": "preview-dns",
  "fields": {
    "created_on": "2026-08-06T09:41:14Z",
    "id": "372e67954025e0ba6aaa6d586b9e0b59",
    "name": "preview-42.example.com"
  },
  "name": "preview-42.example.com",
  "operation": "create_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "hyper_version": "1.4.0",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "record_type": "asset",
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "step": 1,
  "target": "cloudflare-prod",
  "written_at": "2026-08-06T09:41:14.221Z"
}
`
	if got := string(recordVersion(t).Encode()); got != want {
		t.Errorf("record version:\n%s\nwant:\n%s", got, want)
	}
}

func TestRecordVersion_NestsProjectedContentUnderFields(t *testing.T) {
	// A projected field's name is a Provider author's to choose, so flat
	// would need a reserved list of metadata names to steer around — one
	// that cannot grow safely on a branch no rule may rewrite (ADR-0011).
	// Nested, the two namespaces are disjoint forever.
	version := recordVersion(t)
	version.Fields = store.Mapping{
		"record_type": store.String("a Provider author chose this name"),
		"run_id":      store.String("and this one"),
	}

	const want = `  "fields": {
    "record_type": "a Provider author chose this name",
    "run_id": "and this one"
  },`
	got := string(version.Encode())
	if !strings.Contains(got, want) {
		t.Errorf("record version:\n%s\nwant it to carry:\n%s", got, want)
	}
	if !strings.Contains(got, `"record_type": "asset"`) || !strings.Contains(got, `"run_id": "`+theEntryRunID+`"`) {
		t.Errorf("record version:\n%s\nwant the metadata beside fields untouched", got)
	}
}

func TestRecordVersion_CarriesPathOnlyWhereItWasWrittenThroughANestedInvocation(t *testing.T) {
	version := recordVersion(t)
	if got := string(version.Encode()); strings.Contains(got, `"path"`) {
		t.Errorf("record version:\n%s\nwant a top-level Step's version to carry no path", got)
	}

	version.Path = "retire.probe"
	if got := string(version.Encode()); !strings.Contains(got, `"path": "retire.probe"`) {
		t.Errorf("record version:\n%s\nwant it to carry the invocation chain", got)
	}
}

func TestRecordVersion_WritesTheSecretMarkerInThePositionTheValueWouldOccupy(t *testing.T) {
	// No digest, no length, no sibling list of what was suppressed, so no
	// secret reaches the Store at all (ADR-0007). The marker is a constant,
	// which is what keeps *a version is minted where the bytes moved*
	// honest: a rotated secret writes these same bytes.
	version := recordVersion(t)
	version.Fields = store.Mapping{"api_key": store.Secret(store.String("hunter2"))}

	got := string(version.Encode())
	if !strings.Contains(got, `"api_key": "<secret>"`) {
		t.Errorf("record version:\n%s\nwant the marker in the value's position", got)
	}
	if strings.Contains(got, "hunter2") {
		t.Errorf("record version:\n%s\nwant the secret nowhere in it", got)
	}
}

// A Tombstone is an ordinary version of the series, and the four things it
// carries are three ordinary keys and one marker (§7, ADR-0011).

func TestTombstone_IsTheFileSectionSevenPublishes(t *testing.T) {
	version := store.RecordVersion{Metadata: store.Metadata{
		Identity:   store.Identity{Target: "cloudflare-prod", Definition: "preview-dns", Name: "5b2d84f16c0a39e7d5182bfa604c7e93"},
		RecordType: store.RecordAsset,
		Run:        entryRun(t),
		Step:       3,
		Operation:  "delete_dns_record",
		WrittenAt:  time.Date(2026, 8, 6, 9, 43, 36, 512_000_000, time.UTC),
		Provenance: theProvenance,
		Tombstone:  true,
	}}

	const want = `{
  "definition": "preview-dns",
  "name": "5b2d84f16c0a39e7d5182bfa604c7e93",
  "operation": "delete_dns_record",
  "provenance": {
    "definition_revision": "4d7e118c9a03f5b26e1d84a70c3f9b52d6081e4a",
    "hyper_version": "1.4.0",
    "manifest_digest": "sha256:9c1f0b7e3a2d54867f1b0c93ae42d715c806fb39e5a70d24c1938bf5027ea6d1",
    "procedure_revision": "2f81ac4b6e05d3971c8a4f2b0e63d75a91c4e087",
    "repo_revision": "88bc402f71d3e6a95c0428be1f7d3a09c5e64b12"
  },
  "record_type": "asset",
  "run_id": "01991e21-3c9f-7b04-9d18-5c7e2a94f083",
  "schema_version": 1,
  "step": 3,
  "target": "cloudflare-prod",
  "tombstone": true,
  "written_at": "2026-08-06T09:43:36.512Z"
}
`
	got := string(version.Encode())
	if got != want {
		t.Errorf("tombstone:\n%s\nwant:\n%s", got, want)
	}
	// The one version whose fields can be missing for no other reason, so
	// the absence needs no marker beside it.
	if strings.Contains(got, "fields") {
		t.Errorf("tombstone:\n%s\nwant no fields key and nothing in its place", got)
	}
}

func TestTombstone_CarriesThePreviousHeadsFieldsCopiedForward(t *testing.T) {
	// The fields were projected by some earlier Operation and the operation
	// names the one that destroyed it — the one place in the Store those
	// two keys describe different calls (§7).
	version := recordVersion(t)
	version.Operation = "delete_dns_record"
	version.Step = 3
	version.Tombstone = true

	got := string(version.Encode())
	for _, want := range []string{
		`"tombstone": true`,
		`"operation": "delete_dns_record"`,
		`"run_id": "` + theEntryRunID + `"`,
		`"step": 3`,
		`"id": "372e67954025e0ba6aaa6d586b9e0b59"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tombstone:\n%s\nwant it to carry %s", got, want)
		}
	}
}

func TestRecordVersion_CarriesNoTombstoneMarkerWhereItIsNotOne(t *testing.T) {
	if got := string(recordVersion(t).Encode()); strings.Contains(got, "tombstone") {
		t.Errorf("record version:\n%s\nwant no tombstone key at all", got)
	}
}
