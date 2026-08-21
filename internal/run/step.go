package run

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/TheLoomLabs/hyper/internal/artefact"
	"github.com/TheLoomLabs/hyper/internal/capability"
	"github.com/TheLoomLabs/hyper/internal/projection"
	"github.com/TheLoomLabs/hyper/internal/repository"
	"github.com/TheLoomLabs/hyper/internal/revision"
	"github.com/TheLoomLabs/hyper/internal/schema"
	"github.com/TheLoomLabs/hyper/internal/store"
)

// One Step: the binding resolved, the call made, the response projected, the
// versions the projection moved, and the file that says what became of it (§6,
// §7, issue #136).
//
// What is here is the `read` path and no other Kind's. An effectful Step
// declines before Step 1 — run.go states that and why — so nothing below has to
// carry a Kind it cannot perform, and milestone 6 grows this file rather than
// working around it.

// binding is what a Step is bound to, resolved: every artefact the Step names,
// read once, so that nothing below resolves a name a second time and gets a
// second answer.
//
// They travel together because they are one fact — *what this Step is bound
// to* — and because the Provenance a Step writes is read off three of them: the
// Definition's blob id, the Manifest's digest, and its `origin:` digest where
// it carries one (§7, ADR-0043).
//
// It is not called a Bound. A Bound is the maximum number of Records an
// effectful Step may affect (§5, CONTEXT.md), and this is the Step's binding —
// two words the glossary keeps apart and one letter would not.
type binding struct {
	definition repository.LoadedArtefact
	manifest   repository.LoadedManifest
	provider   artefact.ProviderInfo
	target     artefact.TargetInfo
	operation  artefact.OperationInfo
}

// perform runs one Step and answers what became of it.
//
// The error it answers is what halted the Run, and the Step it answers beside
// it stands whether or not there is one: a Step that reached a Disposition
// wrote its file, and a halted Run leaves what it did (§6, ADR-0011). A zero
// Step is a Step that reached none — which in this milestone is only a Step
// whose own binding would not resolve, `check` having not yet re-run at Run
// start to have refused it (issue #137).
func (r run) perform(position int, authored artefact.Step) (Step, error) {
	bound, err := resolve(r.request.Repository, authored)
	if err != nil {
		return Step{}, err
	}
	provenance := store.StepProvenance{
		DefinitionRevision: revision.Blob(bound.definition.Bytes),
		ManifestDigest:     artefact.ManifestDigest(bound.manifest.Bytes),
		OriginDigest:       artefact.ReadManifestFacts(bound.manifest.Root).OriginDigest,
	}

	started := r.request.Now()
	concluded, err := r.call(bound, authored)
	if err != nil {
		return Step{}, err
	}

	// The version, written only where the bytes moved. An Operation
	// returning what the head version already holds mints nothing, and the
	// canonical encoding is what makes that an exact test (§7, ADR-0030).
	identity := store.Identity{Target: authored.Target, Definition: authored.Definition, Name: concluded.name}
	version := store.RecordVersion{
		Metadata: store.Metadata{
			Identity:   identity,
			RecordType: store.RecordObservation,
			Run:        r.id,
			Step:       position,
			Operation:  authored.Operation,
			WrittenAt:  r.request.Now(),
			Provenance: store.Provenance{Run: r.provenance, Step: provenance},
		},
		Fields: concluded.fields,
	}
	moved, err := mints(r.request.Store, version)
	if err != nil {
		return Step{}, err
	}
	if moved {
		if err := r.request.Store.Append([]store.Write{{
			Path:    store.RecordPath(identity, r.id, position),
			Content: version.Encode(),
		}}, fmt.Sprintf("Record %s/%s/%s at run %s step %d", identity.Target, identity.Definition, identity.Name, r.id, position)); err != nil {
			return Step{}, err
		}
	}

	// The identity set, and the digest it is written against: the last Run
	// of this Procedure in which this Step carried one, which is never
	// simply the previous Run (§7, ADR-0055).
	previous, err := r.previousDigest(authored.ID)
	if err != nil {
		return Step{}, err
	}
	// A Step carrying no selector concludes about the one Record it would
	// write, which is a set of one (§6). The set is built before it is
	// written because the count is read off it here and cannot be read back
	// off what is written: an entry whose digest did not move carries no
	// members at all (§7, §8, ADR-0030).
	names := store.Names([]string{concluded.name})
	identities := store.Concluded(names, previous)

	step := Step{
		Position:    position,
		ID:          authored.ID,
		Kind:        store.Kind(bound.operation.Kind),
		Disposition: store.DispositionRan,
		Records:     len(names),
		Concluded:   true,
		Provenance:  provenance,
	}

	file := store.StepFile{
		Step: position,
		StepCode: store.StepCode{
			ID:         authored.ID,
			Definition: authored.Definition,
			Operation:  authored.Operation,
			Provider:   bound.manifest.Name,
			Target:     authored.Target,
			Kind:       step.Kind,
		},
		Disposition: step.Disposition,
		StartedAt:   started,
		EndedAt:     r.request.Now(),
		Provenance:  provenance,
		Identities:  identities,
	}
	if err := r.request.Store.Append([]store.Write{{
		Path:    r.entry.StepPath(position),
		Content: file.Encode(),
	}}, fmt.Sprintf("Step %d %s of run %s: %s", position, authored.ID, r.id, step.Disposition)); err != nil {
		return Step{}, err
	}
	return step, nil
}

// resolve reads every artefact the Step names, and answers the one fault a Step
// can have that this milestone reaches: a name that resolves to nothing.
//
// It is a fault of the artefacts rather than of the world, and `check` re-run
// at Run start is what turns it into the Refusal §6 states — which is issue
// #137's. Until then it halts the Run, which is the honest answer: the Step
// cannot be performed, and reporting it as anything other than a stop would be
// this milestone claiming a Refusal it does not render.
func resolve(loaded repository.Loaded, authored artefact.Step) (binding, error) {
	// The two halves of the Definition namespace: what the name resolves to,
	// and the file it was read from. They are folded from one walk, so one
	// answering and the other not is a repository nobody could have loaded
	// (issue #121) — hence one test over both.
	info, declared := loaded.Definitions[authored.Definition]
	definition, held := loaded.Definition(authored.Definition)
	if !declared || !held {
		return binding{}, fmt.Errorf("step %s names definition %s, which resolves to nothing — hyper check reports it", authored.ID, authored.Definition)
	}
	manifest, published := loaded.Manifests[info.ProviderName]
	provider, exposed := loaded.Providers[info.ProviderName]
	if !published || !exposed {
		return binding{}, fmt.Errorf("step %s binds definition %s, whose provider %s resolves to nothing — hyper check reports it", authored.ID, authored.Definition, info.ProviderName)
	}
	target, granted := loaded.Targets[authored.Target]
	if !granted {
		return binding{}, fmt.Errorf("step %s names target %s, which resolves to nothing — hyper check reports it", authored.ID, authored.Target)
	}
	operation, declares := provider.Operations[authored.Operation]
	if !declares {
		return binding{}, fmt.Errorf("step %s names operation %s, which %s declares nothing of that name — hyper check reports it", authored.ID, authored.Operation, info.ProviderName)
	}
	return binding{definition: definition, manifest: manifest, provider: provider, target: target, operation: operation}, nil
}

// conclusion is what one call concluded about one Record: the identity it
// projected, and the fields under it.
type conclusion struct {
	name   string
	fields store.Mapping
}

// call makes the Step's one call and projects the answer.
//
// **A `read` never halts on what came back.** Whatever the status, the response
// object §12 states is what the projection reads, so a host that answered
// nothing records an Observation whose status has gone quiet rather than
// stopping the Run (§6, ADR-0050). What still halts it is the projection, and
// in this milestone that is the identity path alone — the recorded fields are
// an absence a version simply does not carry, and the rest of what a projection
// that does not resolve does is issue #144's.
func (r run) call(bound binding, authored artefact.Step) (conclusion, error) {
	if bound.operation.Identity == "" {
		return conclusion{}, fmt.Errorf("step %s binds %s %s, whose record: declares no identity, so hyper cannot say which Record a call would be holding — hyper check reports it",
			authored.ID, bound.manifest.Name, authored.Operation)
	}
	inputs, err := arguments(bound.operation, authored)
	if err != nil {
		return conclusion{}, err
	}

	reach := artefact.ResolveHost(bound.provider, bound.operation, bound.target,
		bound.operation.SuppliedHost(inputs))
	if reach.Reach != artefact.ReachGranted {
		return conclusion{}, fmt.Errorf("step %s reaches no host %s grants — hyper check reports it", authored.ID, authored.Target)
	}

	declaration := artefact.OperationNode(bound.manifest.Root, authored.Operation)
	declared, legible := capability.ReadRequest(declaration)
	if !legible {
		return conclusion{}, fmt.Errorf("step %s binds %s %s, which declares no legible http: block — hyper check reports what is wrong with it",
			authored.ID, bound.manifest.Name, authored.Operation)
	}
	built, err := declared.Build(reach.Host, inputs)
	if err != nil {
		return conclusion{}, err
	}

	detail := artefact.ReadOperationDetail(bound.manifest.Root, authored.Operation)
	ctx, cancel := capability.Deadline(context.Background(), detail.DeadlineSeconds)
	defer cancel()

	// The error beside the object is narration's and is deliberately
	// dropped here: no member of the response object says what went wrong,
	// that being the catch-all bucket ADR-0017 closed, and a `read` records
	// the absence as the answer it is (§6, §12, ADR-0050).
	// The instant handed to the call is the **Run's** start and not a fresh
	// read: it is what a certificate's remaining life is counted from, so
	// two Steps of one Run that reach one host record one `days_left`, and
	// nothing a later Step does moves what an earlier one recorded
	// (ADR-0034).
	response, _ := built.Perform(ctx, r.request.Dial, r.started)

	name, resolved := identityOf(bound.operation, inputs, response)
	if !resolved {
		return conclusion{}, fmt.Errorf("step %s: the identity path %s did not resolve against what came back, so hyper cannot say which Record it is holding",
			authored.ID, bound.operation.Identity)
	}
	return conclusion{name: name, fields: projected(bound.operation, projection.Read(declaration).Project(response))}, nil
}

// arguments reads the Step's `args:` against the Operation's declared input
// schema **at that position** rather than by what the value looks like
// (ADR-0081) — the same rule a Probe's `--input` is read under, and the same
// rule §4 checks the authored value against offline.
//
// Every declared input is supplied, there being no null and no key-omission
// syntax (§3, ADR-0081), so an input nothing filled is a fault rather than a
// hole left open.
func arguments(operation artefact.OperationInfo, authored artefact.Step) (map[string]schema.Scalar, error) {
	read := make(map[string]schema.Scalar, len(authored.Args))
	for _, name := range slices.Sorted(keys(operation.Inputs)) {
		node := authored.Args[name]
		if node == nil {
			return nil, fmt.Errorf("step %s supplies no %s, which %s declares — hyper check reports it", authored.ID, name, authored.Operation)
		}
		declared := operation.Inputs[name]
		value, reads := schema.ReadScalar(schema.Type(declared.Type), node.Value)
		if !reads {
			return nil, fmt.Errorf("step %s writes %s: %s, which does not read as the %s %s declares it",
				authored.ID, name, node.Value, declared.Type, authored.Operation)
		}
		read[name] = value
	}
	return read, nil
}

// identityOf is the name the Record is held under: what the Operation's
// `identity:` resolved to.
//
// It reads from whichever of the two roots the Manifest wrote, which is decided
// by the spelling and by nothing else (§3): a template fills from the resolved
// inputs before the call, and a `$`-rooted path resolves against the response
// after it. Which of the two a Manifest declares is what decides whether an
// identity collision Refuses at Expansion or halts the Run (§6, ADR-0072).
func identityOf(operation artefact.OperationInfo, inputs map[string]schema.Scalar, response capability.Object) (string, bool) {
	declared := operation.Identity
	if declared == "" {
		return "", false
	}
	if !strings.HasPrefix(declared, "$") {
		filled, err := capability.Fill("identity:", declared, inputs)
		return filled, err == nil && filled != ""
	}

	value, resolved := projection.Resolve(declared, response)
	if !resolved {
		return "", false
	}
	name, isText := value.(string)
	if !isText {
		name = projection.Text(value)
	}
	return name, name != ""
}

// projected is what the version's `fields` holds: every field that resolved, in
// the Store's own value types, with the ones the Manifest declares `secret:`
// written as the constant marker in the position the value would occupy (§7,
// ADR-0007).
//
// A field whose path resolved to nothing is not here at all — absence is the
// answer, and the field not being written is what carries it (§6, §12).
func projected(operation artefact.OperationInfo, fields projection.Fields) store.Mapping {
	mapping := store.Mapping{}
	for _, field := range fields {
		value, holdable := stored(field.Value)
		if !holdable {
			continue
		}
		if operation.SecretFields[field.Name] {
			value = store.Secret(value)
		}
		mapping[field.Name] = value
	}
	return mapping
}

// mints says whether this version's content differs from what the series' head
// version already holds — *the bytes moved*, made an exact test by the
// canonical encoding rather than an approximate one (§7).
//
// It compares the content and never the whole file. A version restates the Run,
// the Step and the instant that wrote it, so two files of one unchanged Record
// differ in every case and a comparison of them would mint a version on every
// Run — which is precisely the reading ADR-0030 exists to refuse.
func mints(held *store.Store, version store.RecordVersion) (bool, error) {
	head, standing, err := held.Head(version.Identity)
	if err != nil {
		return false, err
	}
	if !standing {
		return true, nil
	}
	previous, err := held.Read(head)
	if err != nil {
		return false, err
	}
	if previous.RecordType != version.RecordType || previous.Tombstone != version.Tombstone {
		return true, nil
	}
	return !slices.Equal(store.Encode(previous.Fields), store.Encode(version.Fields)), nil
}

// previousDigest is the identity digest this Step carried in the last Run **of
// this Procedure** in which it carried one, and "" where there is no such Run —
// a Step's first, and a Step whose authored id moved, which is a different Step
// with no digest behind it (§7, ADR-0055).
//
// It is a backward walk over the Journal's date partitions, and it terminates
// at the first record it finds carrying a set: three of §12's seven
// Dispositions carry none and a fourth writes no file, so the comparand is the
// last Run that carried one rather than the previous Run.
//
// Two filters stand between the scan and the answer, and both are this
// consumer's rather than the walk's — internal/store's Scan matches on the
// authored id and on nothing else, and states that which entries a reading
// keeps is its own.
//
// **A rehearsal is filtered out.** An entry a dry-run wrote is evidence that a
// rehearsal happened and evidence of nothing else, and every consumer of
// Journal evidence filters it out (§7, ADR-0001).
//
// **So is another Procedure's entry.** An authored id is unique inside one
// Procedure and says nothing across two, so a `status` Step in `watch-status`
// and a `status` Step in `watch-many` are two Steps that would otherwise share
// a digest — and each would then read the other's set as its own, writing no
// members while the digest never moved. §7's rule is *the last Run in which
// that Step carried a set*, and a Step belongs to the Procedure that wrote it.
func (r run) previousDigest(id string) (string, error) {
	for evidence, err := range r.request.Store.Scan(id) {
		if err != nil {
			return "", err
		}
		if evidence.Entry.RunFile.DryRun || evidence.Entry.RunFile.Procedure != r.request.Procedure {
			continue
		}
		if digest := evidence.Step.Identities.Digest; digest != "" {
			return digest, nil
		}
	}
	return "", nil
}
