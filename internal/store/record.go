package store

import "time"

// recordVersionMembers is what every Record version carries. `fields` is not
// among them: a Tombstone opening a series carries none, and that absence is
// guarded where it is argued rather than here (§7, ADR-0033).
var recordVersionMembers = []string{
	"target", "definition", "name", "record_type",
	"run_id", "step", "operation", "written_at", "provenance",
}

// RecordType is what a Record version is a version of: an Observation hyper
// read, or an Asset hyper is accountable for having made (§2, ADR-0025).
type RecordType string

const (
	// RecordObservation is what a read Operation projects.
	RecordObservation RecordType = "observation"
	// RecordAsset is what a mutate Operation projects, and what a destroy
	// Operation Tombstones — a Tombstone's record_type is asset because
	// hyper's effect reached the thing.
	RecordAsset RecordType = "asset"
)

// RecordVersion is one version of one Record: that version's projected content
// and its metadata together, one artefact that is diffable and canonical at
// once (§7).
//
// A version is written only where the bytes moved. An Operation returning what
// the head version already holds mints nothing, and the canonical encoding is
// what makes *the bytes moved* an exact test rather than an approximate one.
//
// There are no binary Records, no streaming writes and no appending inside a
// version: a Record is the projection its Manifest declared, and a blob nobody
// reviews has no business on a branch whose whole point is that it can be read.
type RecordVersion struct {
	Metadata
	// Fields is the projected content, nested under its own key rather than
	// sitting beside the metadata: a projected field's name is a Provider
	// author's to choose, and flat would need a reserved list of metadata
	// names for it to steer around — a list that cannot grow safely on a
	// branch no rule may rewrite. Nested, the two namespaces are disjoint
	// forever and there is no check to state or forget (§7, ADR-0011).
	//
	// A field a Manifest declares secret is Secret here, which writes the
	// marker in the position the value would occupy (ADR-0007). The Store
	// holds nothing else about it, so a decode answers that marker as the
	// string it is — §7's own rule that a projected value reading the same
	// is not a case hyper disambiguates.
	//
	// Nil and empty are one value: a version carrying no projected content
	// at all, and the key is not written. A decode answers nil.
	//
	// Two versions reach that state and neither is a defect. A Tombstone
	// opening the series it ends has no previous Head to copy forward, and
	// the absence means *hyper destroyed this and never observed what it
	// was* (§7, ADR-0033). An ordinary version reaches it where **every**
	// path its Manifest projected resolved to nothing, which is the
	// ordinary field absence §6 states applied to all of them at once: a
	// `shell` command that could not be started at all answers `command`
	// and nothing else, and the built-in Provider projects `exit_code`,
	// `stdout` and `stderr` (§12, issue #142). The two are never confused
	// for each other — `tombstone` is a written marker and not the
	// absence — and what the second says is exactly what it looks like:
	// hyper made the call and read nothing back off it.
	Fields Mapping
}

// Metadata is everything a version says about itself but its content: which
// series it belongs to, what wrote it, when, and whether it is the destruction.
//
// The word is §7's own — *a Record version is one file, holding that version's
// projected content and its metadata together* — and the split here is that
// sentence's two halves made two values.
//
// It is a value of its own because that is the grain the reader answers at.
// Ordering a series and naming a version need every member of this and no byte
// of the content, so a listing of a thousand versions holds a thousand of these
// and reads the content of the one a caller went on to ask for (§7, issue
// #130).
type Metadata struct {
	// Identity is the series this version belongs to, unencoded and in
	// full. It is restated here rather than read back out of the path
	// because the path is lossy — the grammar truncates an over-long
	// segment and suffixes a hash (§12) — and because the working tree must
	// describe itself (ADR-0011).
	Identity Identity
	// RecordType says which of the two this is.
	RecordType RecordType
	// Run, Step and Operation are what wrote this version. On a Tombstone
	// the Operation is the one that destroyed the Asset, which is the one
	// place in the Store it and Fields describe different calls.
	Run       RunID
	Step      int
	Operation string
	// Path is the invocation chain where the Step that wrote this version
	// was reached through a nested Procedure invocation, and empty on a
	// top-level Step — the same member a Step file carries.
	Path string
	// WrittenAt is when this version was written, and on a Tombstone when
	// destruction was confirmed. The Head is derived by ordering a series'
	// versions on it, ties broken by the file name, so nothing in the Store
	// points at the current version and two environments writing one series
	// contend over nothing (§7, ADR-0011).
	WrittenAt time.Time
	// Provenance is the whole of it. A version file saying only *see Run
	// abc* would be unreadable in a browser and in a diff, which is exactly
	// where this field set is read.
	Provenance Provenance
	// Tombstone marks the destruction. A Tombstone is an ordinary version
	// of the series carrying this marker, the previous Head's Fields copied
	// forward, and the Operation, Run and Step every version carries
	// anyway.
	//
	// A Tombstone opening a series carries no Fields at all — a `values:`
	// member named nothing the Store held, so there is no previous Head to
	// copy forward, and the absence means *hyper destroyed this and never
	// observed what it was*. It is the one version whose Fields can be
	// missing for no other reason, so the absence needs no marker beside it
	// (ADR-0033).
	Tombstone bool
}

// Encode writes a Record version in §7's canonical encoding.
func (v RecordVersion) Encode() []byte {
	return file(RecordSchemaVersion, "a Record version", recordVersionMembers, func(m members) {
		m.text("target", v.Identity.Target)
		m.text("definition", v.Identity.Definition)
		m.text("name", v.Identity.Name)
		m.text("record_type", string(v.RecordType))
		m.text("run_id", v.Run.String())
		m.count("step", v.Step)
		m.text("operation", v.Operation)
		m.text("path", v.Path)
		m.at("written_at", v.WrittenAt)
		m.block("provenance", v.Provenance.write)
		m.mark("tombstone", v.Tombstone)
		m.value("fields", v.Fields)
	})
}

// DecodeRecordVersion reads a Record version back to the value it was written
// from, a Tombstone included — a Tombstone being an ordinary version of the
// series and not a shape of its own.
//
// The one thing it does not read back is which fields were suppressed. A secret
// is written as a constant string and the Store holds nothing else about it, so
// a decode answers the marker as the string it is — which is §7's own rule that
// a projected value reading the same is not a case hyper disambiguates.
func DecodeRecordVersion(data []byte) (RecordVersion, error) {
	return decodeFile(data, RecordSchemaVersion, func(r *fields, v *RecordVersion) {
		r.require(recordVersionMembers...)
		v.Identity = Identity{
			Target:     r.text("target"),
			Definition: r.text("definition"),
			Name:       r.text("name"),
		}
		v.RecordType = oneOf(r, "record_type", RecordObservation, RecordAsset)
		v.Run = r.run("run_id")
		v.Operation = r.text("operation")
		v.Path = r.text("path")
		v.WrittenAt = r.at("written_at")
		v.Step = r.position("step")
		v.Tombstone = r.mark("tombstone")
		// `fields` is not required on any version. It is absent on a
		// Tombstone opening the series it ends, and absent on an
		// ordinary version whose every projected path resolved to
		// nothing — a `shell` command that could not be started at all
		// being where the second arrives (§6, §7, issue #142).
		v.Fields = r.nested("fields")

		if provenance := r.block("provenance"); provenance != nil {
			provenance.require(runProvenanceMembers...)
			provenance.require(stepProvenanceMembers...)
			v.Provenance = Provenance{
				Run:  readRunProvenance(provenance),
				Step: readStepProvenance(provenance),
			}
			r.join(provenance, "provenance")
		}
	})
}
