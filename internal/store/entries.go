package store

import (
	"cmp"
	"errors"
	"fmt"
	"iter"
	"slices"
	"strings"
	"time"
)

// The Journal, read (§7, issue #132). Which entries the branch holds, how each
// one ended, what became of each of its Steps, and how to find the last Run in
// which a given Step did something.
//
// **Nothing here is defined over the branch's commits.** Append-only makes a
// year-old Run a read of the tip like any other, and Provenance names revisions
// on the *code* branch and none on this one — so a Journal answer is a fact
// about the files a listing found and never about how or when they arrived.
//
// **No local index and no derived state.** §7 permits state under `.git/hyper/`
// that makes a Head lookup or a backward scan faster and states that no answer
// depends on one existing; this milestone builds none, so the scan is a scan:
// one listing of the branch, and the files of each entry it actually visits.
//
// Every file is read for what it says about itself, and the path is held to
// what the file builds rather than the other way round — an entry whose
// run.json does not build the directory it sits in is a directory `hyper` did
// not write, which is the same answer readListing gives one prefix over.

// Account is how a Journal entry ended: which of the four the files present
// under its directory make it.
//
// It is a classification and never a member. There is no state key to leave
// stale and no growing file to rewrite, because closing has two *forms* rather
// than two writes to one path (§7, ADR-0011).
type Account int

const (
	// AccountOpen is an entry holding no account at all — neither an
	// outcome.json its own Run wrote nor a closing write another Run wrote
	// — and that absence is the whole representation. The Run may be in
	// flight or its process may be gone; hyper never guesses which, and
	// neither does this reader.
	AccountOpen Account = iota + 1
	// AccountOwn is outcome.json alone: the Run gave an account of its own
	// end, and the entry's outcome is that file's.
	AccountOwn
	// AccountReaped is one or more closing writes and no outcome.json. The
	// Run really did not come back: the entry is failed, and the close
	// instant is the earliest inference among the closers.
	AccountReaped
	// AccountContested is both, and it is what a reap of a Run that was
	// alive after all leaves behind. The entry's outcome is the owner's
	// observation, the inference stays true of the Run that drew it, and
	// both files stand — hyper picks no side between two accounts of what
	// the world did, and holding both is what keeps this from being that
	// (§7, ADR-0076).
	AccountContested
)

// String names the classification in §7's own prose, for a diagnostic and for
// nothing else. It is not a rendering: what a surface prints is §8's, stated
// there and built there, and a caller reaching for these words to print is
// reaching past the section that owns them.
func (a Account) String() string {
	switch a {
	case AccountOpen:
		return "open"
	case AccountOwn:
		return "closed by its own Run"
	case AccountReaped:
		return "reaped"
	case AccountContested:
		return "contested"
	}
	return "no account"
}

// ErrUnreadable is a file under a Journal entry that would not decode: written
// in a shape above this binary's ceiling, or holding bytes no decoder here
// accepts (ADR-0028).
//
// It is named so that a caller can tell it from the other way a read of the
// Journal stops. A file that will not decode is a **file** this binary cannot
// read, and §6 puts a gate one place in a Run's order whose whole job is to
// report exactly that over the Journal whole (Readable). A path that disagrees
// with the file standing at it is something else — a directory `hyper` did not
// write — and no gate reports it, so nothing may treat it as tolerable.
var ErrUnreadable = errors.New("a file under this Journal entry would not decode")

// Closer is one closing write as the reader answers it: another Run's inference
// about this entry, and the Run that drew it.
//
// The Run comes from the file's name and not from a member — a closing write
// carries none naming its author, its path being that member (§7, ADR-0076) —
// which is why the two are one value here rather than a ClosedBy a caller has
// to carry an id beside.
type Closer struct {
	ClosedBy
	// Run is the Run that wrote this file: the one making the claim.
	Run RunID
}

// Entry is one Journal entry as a listing answers it: what its own run.json
// says, the account its own Run gave where it gave one, and every inference
// another Run drew about it.
//
// It carries no Step file. Reading an entry whole is this and Dispositions
// together — a listing of a year of Runs holds one of these each and opens the
// Step files of the entry a caller went on to ask about, which is the same
// split Version and RecordVersion are two shapes for (§7).
//
// It carries no account *field* either: how the entry ended is Account's, and
// deriving it from the files present is what leaves nothing to go stale.
type Entry struct {
	// RunFile is what the entry's own run.json holds, `dry_run` among it —
	// exposed on every entry so that each of the four consumers of Journal
	// evidence can filter rehearsals out, and filtered here on none of
	// their behalves (§7, ADR-0001).
	RunFile
	// Owner is the account the entry's own Run gave of its end, and the
	// zero value where it gave none. An outcome.json is written by the Run
	// whose entry it is and by no other.
	Owner OutcomeFile
	// Closers is every closing write the entry holds, earliest inference
	// first. All of them stand and none is discarded, however many landed —
	// and the ordering is the rule rather than a second lookup, the account
	// of a reaped entry being drawn from the earliest (§7).
	Closers []Closer
}

// owner answers the account the entry's own Run gave of its end, and whether it
// gave one at all.
//
// It is the test three of the answers below turn on, and they turn on it rather
// than on the classification because they are one rule: **the entry's outcome
// is the owner's wherever one exists** — on a contested entry included, an
// outcome.json being its own Run's observation and a closing write another
// Run's inference drawn from a silence (§7).
//
// An outcome.json carrying no outcome cannot be written, so the empty triple is
// the file's absence rather than a value it holds.
func (e Entry) owner() (OutcomeFile, bool) {
	return e.Owner, e.Owner.Outcome != ""
}

// reaped answers whether another Run drew an inference about this entry.
func (e Entry) reaped() bool { return len(e.Closers) > 0 }

// Account answers how this entry ended, from the files present under it and
// from nothing else.
func (e Entry) Account() Account {
	_, owned := e.owner()
	switch {
	case owned && e.reaped():
		return AccountContested
	case owned:
		return AccountOwn
	case e.reaped():
		return AccountReaped
	}
	return AccountOpen
}

// Outcome answers the entry's outcome, and whether the entry has one at all.
//
// **It is the owner's wherever one exists** — on a contested entry included:
// an outcome.json is its own Run's observation and a closing write is another
// Run's inference drawn from a silence, and where the two disagree the
// observation is what happened (§7). A reaped entry is `failed`. An open entry
// has none, and nothing here infers one.
func (e Entry) Outcome() (Outcome, bool) {
	if account, gave := e.owner(); gave {
		return account.Outcome, true
	}
	if e.reaped() {
		return OutcomeFailed, true
	}
	return "", false
}

// Ended answers when the entry closed, and whether it closed at all.
//
// On a reaped entry it is the **earliest** `ended_at` among the closers — the
// first inference, later ones adding nothing but their own existence — and that
// instant is on the closing Run's clock, which is why Duration answers nothing
// there (§7).
func (e Entry) Ended() (time.Time, bool) {
	if account, gave := e.owner(); gave {
		return account.EndedAt, true
	}
	if e.reaped() {
		return e.Closers[0].EndedAt, true
	}
	return time.Time{}, false
}

// Duration answers how long the Run took, and whether one derives at all.
//
// It derives inside one entry or not at all. Every file stamps the instant it
// was written and no duration is stored anywhere, so this is a subtraction —
// and on a **reaped** entry the two instants come from two clocks, the closing
// Run's and the dead Run's, which is the cross-entry subtraction §7 forbids
// wearing one entry's directory. **The entry's account being a closing write is
// what says so**, and there is no second flag: the fact and the flag arrive
// together here, which is what keeps them from coming apart in each rendering
// that would otherwise rediscover it.
//
// A contested entry derives one normally. There the account is the owner's,
// written on the owner's clock inside the owner's entry, and the closing write
// beside it is not an endpoint of anything.
func (e Entry) Duration() (time.Duration, bool) {
	account, gave := e.owner()
	if !gave {
		return 0, false
	}
	return account.EndedAt.Sub(e.StartedAt), true
}

// inference answers what the entry's closing writes record about the Step the
// dead Run went quiet on, and whether any Run drew one.
//
// It is the earliest closer's and no other. Where several landed they are one
// inference restated — the first, later ones adding nothing but their own
// existence — and all of them stand in Closers regardless (§7).
func (e Entry) inference() (StepFile, bool) {
	if !e.reaped() {
		return StepFile{}, false
	}
	return e.Closers[0].Reading(), true
}

// Dispositions is what became of the Steps of one entry: the records the entry
// holds, and the entry itself.
//
// The two are one value because the seventh Disposition is read from an
// absence, and an absence means one thing inside a closed entry and something
// else inside an open one. A slice of Step files alone cannot answer *what
// became of this Step*, and a caller holding the two apart is one join away
// from answering it wrong.
type Dispositions struct {
	// Entry is the entry these records were read from.
	Entry Entry
	// Steps is what the entry recorded about its Steps, in the Run's own
	// written order — the Step files it holds, and, where a reaper closed
	// it, the reading its earliest closing write carries beside them.
	//
	// A record is not always a file: a reaped entry's account of the Step
	// the dead Run went quiet on is a closing write, and it is here in the
	// shape a Step file records one because §8 reads Dispositions
	// generically across all seven values (§7, ClosedBy.Reading).
	//
	// There is **one record per Step**, and where a contested entry holds
	// two accounts of one Step the record is the owner's: an outcome.json
	// and the Step files beside it are the Run's own observations, and a
	// closing write is another Run's inference drawn from a silence. The
	// inference is not removed — it stands in the entry's Closers, where
	// §7 puts it and where nothing here touches it — it is simply not a
	// second account of what became of one Step.
	Steps []StepFile
}

// Of answers what became of the Step authored under id, and whether the entry
// says anything about it at all.
//
// It reads the Disposition from the Step's own record where the entry holds
// one, and answers *never reached* where the entry is **closed** and holds
// none — the seventh value, borne by no file, which is what keeps a forty-Step
// Procedure that halted at Step 3 from writing thirty-seven files saying that
// nothing happened.
//
// It answers **nothing at all** for a Step absent from an **open** entry. There
// the absence means something different — the Step may be running, or the Run's
// process may be gone — and guessing between them is exactly what §7 forbids.
func (d Dispositions) Of(id string) (Disposition, bool) {
	for _, step := range d.Steps {
		if matches(step, id) {
			return step.Disposition, true
		}
	}
	if id == "" || d.Entry.Account() == AccountOpen {
		// The empty string is the absence of an id rather than an id
		// (matches), so it is not a Step this entry never reached
		// either — and in an open entry the absence of a record means
		// something other than *never reached*, which is the guess §7
		// forbids.
		return "", false
	}
	return DispositionNeverReached, true
}

// matches answers whether a Step record is the record of the Step authored
// under id.
//
// The empty string matches nothing. A Step file always carries its authored id
// and a closing write carries one only where the dead Run's revision resolved
// it, so the empty string is the absence of an id rather than an id to match on
// — and matching it would answer every reaper that could not resolve one (§7).
func matches(step StepFile, id string) bool { return id != "" && step.ID == id }

// Evidence is what one Run's entry holds about one Step, as the backward scan
// answers it: the record of the Step, and the entry it sits in.
//
// The word is §6's own — *run-once refuses on evidence rather than on suspicion,
// and the evidence is what the Journal holds for that Step* — and it is what
// both of the scan's consumers are asking the Journal for.
//
// The entry travels with the record because both of them need it. Whether a
// rehearsal counts is the consumer's and is a fact about the Run rather than
// about the Step, and the identity digest's comparand is a Run rather than a
// file (§6, §7, ADR-0001, ADR-0055).
type Evidence struct {
	// Entry is the entry the record sits in.
	Entry Entry
	// Step is what that entry recorded about the Step — its own file, or
	// the reading a closing write carries where that is the only record
	// there is.
	Step StepFile
}

// Comparable answers whether this evidence stands as a record of the Step
// authored under path in procedure — which is the filter **both** readings of a
// Step's identity set apply, and the reason it is stated here rather than at
// either of them (§6, §7, ADR-0001, ADR-0055).
//
// The walk itself filters nothing: Scan matches on the authored id and on
// nothing else, and states that which entries a reading keeps is its own. What
// the two readings keep is the same three facts, and a difference between them
// would be a Run writing a digest against one set and a reader resolving it
// against another.
//
// **A rehearsal is out.** An entry a dry-run wrote is evidence that a rehearsal
// happened and evidence of nothing else, and every consumer of Journal evidence
// filters it out (§7, ADR-0001).
//
// **So is another Procedure's entry.** An authored id is unique inside one
// Procedure and says nothing across two, so a `status` Step in `watch-status`
// and a `status` Step in `watch-many` are two Steps that would otherwise share
// a digest — each reading the other's set as its own.
//
// **And so is another invocation chain's.** One Run holds the Steps of every
// Procedure it invokes, so two nested Procedures may each declare a `status`
// and both be Steps of one Run — told apart by the `path` their files carry
// beside that id (§7).
func (e Evidence) Comparable(procedure, path string) bool {
	return !e.Entry.DryRun && e.Entry.Procedure == procedure && e.Step.Path == path
}

// Entries answers every Journal entry the branch holds, newest first.
//
// The order is each entry's own `started_at`, ties broken by the Run id, which
// is the Head's rule at the Journal's grain: the instant the file carries, and
// the name where two share one. It is not the listing's order and not the
// commits' — a year-old Run is a read of the tip like any other.
//
// It opens every entry's run.json, outcome.json and closing writes, and no Step
// file. That is what makes a listing of a year of Runs one batch read rather
// than one per Step, and Dispositions the door to the rest.
func (s *Store) Entries() ([]Entry, error) {
	partitions, err := s.partitions()
	if err != nil {
		return nil, err
	}

	var groups []group
	for _, partition := range partitions {
		groups = append(groups, partition...)
	}
	visits, err := s.accountsOf(groups)
	if err != nil {
		return nil, err
	}

	entries := make([]Entry, len(visits))
	for i, visit := range visits {
		entries[i] = visit.entry
	}
	slices.SortFunc(entries, func(a, b Entry) int { return newest(a.RunFile, b.RunFile) })
	return entries, nil
}

// newest orders two entries the way every listing of the Journal does: on the
// instant each one's own run.json carries, newest first, ties broken by the Run
// id.
//
// It is §7's Head rule at the Journal's grain — the instant the file carries,
// and the name where two share one — and it is defined over the files rather
// than over the listing, so a date partition being a text order is a
// convenience the walk uses and never the answer it gives.
func newest(a, b RunFile) int {
	return cmp.Or(
		b.StartedAt.Compare(a.StartedAt),
		strings.Compare(b.Run.String(), a.Run.String()),
	)
}

// Entry answers one entry by the id of the Run whose entry it is, and whether
// the branch holds it.
//
// It is the entry alone. Reading one *whole* is this and Dispositions together,
// for the reason Entries opens no Step file: the Step files are the bulk of a
// Journal, and a caller asking which Runs there were is not asking what each of
// their forty Steps did.
func (s *Store) Entry(run RunID) (Entry, bool, error) {
	partitions, err := s.partitions()
	if err != nil {
		return Entry{}, false, err
	}
	for _, partition := range partitions {
		for _, held := range partition {
			if held.run != run {
				continue
			}
			read, err := s.accountsOf([]group{held})
			if err != nil {
				return Entry{}, false, err
			}
			return read[0].entry, true, nil
		}
	}
	return Entry{}, false, nil
}

// Dispositions answers what became of the Steps of one entry: every record it
// holds, in the Run's own written order.
//
// The entry is the one a listing answered, so its directory is built from what
// its own run.json says rather than from a path a caller supplied.
func (s *Store) Dispositions(entry Entry) (Dispositions, error) {
	files, err := s.repo.listTree(s.commit, entryPrefix(entry.At()))
	if err != nil {
		return Dispositions{}, err
	}

	var steps []entryFile
	for _, file := range files {
		parsed, err := ParsePath(file.path)
		if err != nil {
			return Dispositions{}, err
		}
		if parsed.Form == FormStep {
			steps = append(steps, entryFile{treeEntry: file, Path: parsed})
		}
	}
	return s.dispositionsOf(steps, entry)
}

// Scan walks the Journal backward — newest entry first, across every date
// partition — and yields every Run in which the Step authored under id did
// something.
//
// It is the second workload the whole layout exists to serve, and it is a scan:
// no index is kept here or under `.git/hyper/`, so this is one listing of the
// branch and then the files of each entry it visits, in order, until the caller
// stops. **Stopping is what makes it cheap** — a set read off a recent entry
// costs one entry's files and one off an old one costs the entries between —
// and both of its callers stop: run-once Repeatability at the first Run
// recording this Step as *ran* or *attempted, outcome unknown*, and the
// identity digest's comparand at the last Run in which the Step carried a set
// at all (§6, §7, ADR-0055).
//
// Within one entry the records run backward too, the Run's own order reversed,
// so a walk that never leaves the newest entry still runs newest first.
//
// It matches on the **authored id** and on nothing else. A Step whose id moved
// is a different Step with no Run behind it, and it writes its set in full on
// its first Run like any other (ADR-0055).
//
// It filters no entry on any consumer's behalf. A rehearsal is reached like
// every other entry and reported with its `dry_run` marker, because which
// readings exclude one is each of the four consumers', not this walk's (§7,
// ADR-0001).
func (s *Store) Scan(id string) iter.Seq2[Evidence, error] {
	return func(yield func(Evidence, error) bool) {
		if id == "" {
			// It matches nothing (matches), and the branch need not
			// be read to say so.
			return
		}
		partitions, err := s.partitions()
		if err != nil {
			yield(Evidence{}, err)
			return
		}

		for _, partition := range partitions {
			// The partitions arrive newest first and the entries
			// inside one are ordered on the instant each run.json
			// carries, which costs that partition's run.json files
			// and none of its Step files. A day of Runs is the
			// widest that ever is.
			visits, err := s.accountsOf(partition)
			if err != nil {
				yield(Evidence{}, err)
				return
			}
			slices.SortFunc(visits, func(a, b visit) int { return newest(a.entry.RunFile, b.entry.RunFile) })

			for _, visit := range visits {
				dispositions, err := s.dispositionsOf(visit.files.steps, visit.entry)
				if err != nil {
					yield(Evidence{}, err)
					return
				}
				for _, step := range slices.Backward(dispositions.Steps) {
					if !matches(step, id) {
						continue
					}
					if !yield(Evidence{Entry: dispositions.Entry, Step: step}, nil) {
						return
					}
				}
			}
		}
	}
}

// entryFile is one file a listing found under an entry: where it sits, and what
// its path says about itself.
type entryFile struct {
	treeEntry
	Path
}

// group is one entry's files as a listing found them, split by what the reader
// opens them for: the three shapes an entry's *account* is classified over, and
// the Step files, which a listing of the Journal never opens.
type group struct {
	dir     string
	run     RunID
	account []entryFile
	steps   []entryFile
}

// partition is one date partition: the entries under it, in the order a listing
// found them — which is the path's, and so the partition's own.
type partition []group

// visit is one entry as the reader holds it: the files a listing found under
// it, and the account read off them.
//
// The two are one value rather than two slices lined up by position, because
// they are read together and walked together and a pairing kept by index is one
// insertion away from being wrong.
type visit struct {
	files group
	entry Entry
}

// partitions lists the whole Journal once and groups it: by date partition,
// newest first, and by entry within each.
//
// The listing is the one place the branch is enumerated, and everything above
// walks what it answers. A partition order is a text order over
// `<yyyy>/<mm>/<dd>`, which is a date order because the segments are
// zero-padded and fixed-width — so the walk crosses a day, a month and a year
// boundary without any of the three being a case.
func (s *Store) partitions() ([]partition, error) {
	files, err := s.repo.listTree(s.commit, journalPrefix)
	if err != nil {
		return nil, err
	}

	var order []string
	held := map[string][]group{}
	seen := map[string]int{}
	for _, file := range files {
		parsed, err := ParsePath(file.path)
		if err != nil {
			return nil, err
		}

		at, found := seen[parsed.Dir]
		if !found {
			at = len(held[parsed.Partition])
			seen[parsed.Dir] = at
			if len(held[parsed.Partition]) == 0 {
				order = append(order, parsed.Partition)
			}
			held[parsed.Partition] = append(held[parsed.Partition], group{dir: parsed.Dir, run: parsed.Entry})
		}
		entry := &held[parsed.Partition][at]

		if parsed.Form == FormStep {
			entry.steps = append(entry.steps, entryFile{treeEntry: file, Path: parsed})
			continue
		}
		entry.account = append(entry.account, entryFile{treeEntry: file, Path: parsed})
	}

	slices.Sort(order)
	slices.Reverse(order)
	partitions := make([]partition, len(order))
	for i, at := range order {
		partitions[i] = held[at]
	}
	return partitions, nil
}

// accountsOf reads the account of every entry handed, in one batch read over
// every file the classification is defined on, and answers each beside the
// files it was read from.
//
// One read for the whole set rather than one per entry is what keeps a listing
// of a year of Runs a cost in bytes rather than in processes — the same trade
// readListing makes one prefix over.
func (s *Store) accountsOf(groups []group) ([]visit, error) {
	var blobs []string
	for _, held := range groups {
		blobs = append(blobs, blobsOf(held.account)...)
	}
	contents, err := s.repo.readBlobs(blobs)
	if err != nil {
		return nil, err
	}

	visits := make([]visit, len(groups))
	read := 0
	for i, held := range groups {
		entry, err := decodeEntry(held, contents[read:read+len(held.account)])
		if err != nil {
			return nil, err
		}
		visits[i] = visit{files: held, entry: entry}
		read += len(held.account)
	}
	return visits, nil
}

// blobsOf names every file a listing found to git, in the order it found them,
// which is the order the batch read answers in.
func blobsOf(files []entryFile) []string {
	blobs := make([]string, len(files))
	for i, file := range files {
		blobs[i] = file.blob
	}
	return blobs
}

// decodeEntry reads one entry's account: its run.json, the outcome.json its own
// Run wrote where it wrote one, and every closing write another Run left.
//
// The entry's own run.json is held to building the directory it was found in,
// which is the Journal's half of the rule a Record version is read under: a
// file that does not sit where its own contents put it is a file `hyper` did
// not write, and reading it would give two answers about one entry — one of
// them a date nothing filed it under (§7, §12).
func decodeEntry(held group, contents [][]byte) (Entry, error) {
	entry := Entry{}
	found := false
	for i, file := range held.account {
		switch file.Form {
		case FormRun:
			read, err := DecodeRunFile(contents[i])
			if err != nil {
				return Entry{}, unreadable(file.path, err)
			}
			entry.RunFile, found = read, true

		case FormOutcome:
			read, err := DecodeOutcomeFile(contents[i])
			if err != nil {
				return Entry{}, unreadable(file.path, err)
			}
			entry.Owner = read

		case FormClosedBy:
			read, err := DecodeClosedBy(contents[i])
			if err != nil {
				return Entry{}, unreadable(file.path, err)
			}
			entry.Closers = append(entry.Closers, Closer{ClosedBy: read, Run: file.Run})
		}
	}

	if !found {
		return Entry{}, fmt.Errorf("%q holds no run.json: a Journal entry is written at Run start and every file under one sits beside it (§7)", held.dir)
	}
	if built := entry.At().RunPath(); built != held.dir+"/run.json" {
		return Entry{}, fmt.Errorf("%q holds an entry the grammar names %q: an entry sits under the UTC date of the instant its own run.json carries (§12)", held.dir, built)
	}

	// Earliest inference first, so that the closer a reaped entry's account
	// is drawn from is the one at hand rather than a second lookup. The Run
	// id breaks a tie, two inferences at one instant being two files that
	// must still order the same way on two reads.
	slices.SortFunc(entry.Closers, func(a, b Closer) int {
		return cmp.Or(
			a.EndedAt.Compare(b.EndedAt),
			strings.Compare(a.Run.String(), b.Run.String()),
		)
	})
	return entry, nil
}

// unreadable is one file under an entry that would not decode, named by its
// path and carrying ErrUnreadable so that a caller can tell this from a path
// that disagrees with the file standing at it.
func unreadable(path string, err error) error {
	return fmt.Errorf("%q: %w: %w", path, ErrUnreadable, err)
}

// dispositionsOf reads one entry's Step files and puts them in the Run's
// written order, with the reading its earliest closing write carries beside
// them where it holds one.
//
// Each file is held to sitting at the path its own `step` builds, for the
// reason the entry's own run.json is held to building its directory.
func (s *Store) dispositionsOf(files []entryFile, entry Entry) (Dispositions, error) {
	contents, err := s.repo.readBlobs(blobsOf(files))
	if err != nil {
		return Dispositions{}, err
	}

	steps := make([]StepFile, 0, len(files)+1)
	for i, file := range files {
		read, err := DecodeStepFile(contents[i])
		if err != nil {
			return Dispositions{}, unreadable(file.path, err)
		}
		if built := entry.At().StepPath(read.Step); built != file.path {
			return Dispositions{}, fmt.Errorf("%q holds the record of a Step the grammar names %q: a Step file sits at the path its own position builds (§12)", file.path, built)
		}
		steps = append(steps, read)
	}
	// The reaper's reading, and only where the entry holds no file at the
	// position its earliest closing write names. On a **contested** entry
	// the owner may have gone on to record that Step itself, and there the
	// observation is what became of it: the inference is another Run's
	// account of the entry, it stands in Closers where nothing removes it,
	// and it is not a second record of one Step (§7, ADR-0076).
	//
	// One record per position is what makes that rule hold in both
	// directions at once. A reader walking forward and a scan walking
	// backward would otherwise disagree about a contested Step by the
	// direction they went in, which is the sort of ordering the two files
	// exist to keep hyper out of.
	if reading, drawn := entry.inference(); drawn {
		if !slices.ContainsFunc(steps, func(held StepFile) bool { return held.Step == reading.Step }) {
			steps = append(steps, reading)
		}
	}
	slices.SortFunc(steps, func(a, b StepFile) int { return cmp.Compare(a.Step, b.Step) })
	return Dispositions{Entry: entry, Steps: steps}, nil
}
