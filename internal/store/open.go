package store

import "slices"

// The open entries, as the Run that closes them reads them (§6, §7, ADR-0076,
// issue #154).
//
// It is a reader of its own rather than a filter over Entries because the two
// ask different questions of the branch. Entries reads every entry's account
// whole — the outcome.json its own Run wrote and every closing write another
// Run left — and an **open** entry is by definition one that holds neither, so
// the classification is answered by the listing and the only file there is to
// open is the dead Run's own run.json.
//
// That is a cost and a scope at once. A reap is one listing and the run.json
// files of the entries that turned out to be open, rather than every file of
// every entry the Journal holds; and a Journal file this binary cannot read is
// one that stops the Run at the gate §6 puts one line further on, where it is
// `store-schema-unsupported` naming the path, rather than here, where it would
// be a reap failing over a file no reap was going to write into (§6, gates).

// OpenEntry is one entry holding no account at all, and everything a reaper
// establishes about it before it loads a line of code.
//
// It carries no account members and could not: an entry that held either is not
// open, and a value with a place to put one would be a value a caller could
// read a contest off before the contest existed (§7).
type OpenEntry struct {
	// RunFile is the entry's own run.json: the Procedure the dead Run was
	// performing and the **repository** revision to load it at, which is
	// what makes *which Step was it* derived rather than guessed. It is
	// `repo_revision` and never `procedure_revision` — reconstructing the
	// Step sequence means loading every Procedure the top-level one
	// invokes, which a commit resolves and a blob id cannot (§7).
	RunFile
	// Last is the highest step ordinal the entry holds — the last Step that
	// finished — and zero where the Run wrote no Step file at all, which is
	// a Run that went quiet on Step 1.
	//
	// It is the highest ordinal **present** and never a count of the files.
	// The two agree on every entry `hyper` writes, and any future change
	// that wrote a Step file out of order, or wrote one before its Step
	// concluded, breaks this arithmetic silently (§7, issue #147).
	Last int
}

// OpenEntries answers every entry on the branch holding no account at all,
// newest first.
//
// **It answers every one of them and never a subset.** A rule that reaped some
// would need a criterion, and the only candidates are age and liveness — both
// of them the guess §6 declines. So there is no threshold here, no clock read
// against an entry's `started_at`, and nothing that asks whether a process is
// alive (§6, ADR-0076). This is where that rule is *enforced*, by there being
// nothing here to enforce it against.
//
// It answers ErrUnreadable where a file it opened would not decode, and an
// ordinary error where the branch itself could not be read or where an entry
// does not sit at the path its own run.json builds. The two are told apart
// because only the first is a condition a Run's own gates go on to report
// (§6, entries.go).
//
// The order is Entries' own at this grain: the instant each entry's own
// run.json carries, ties broken by the Run id. Every entry it answers is
// reaped, so the order decides nothing about **which** — what it decides is
// that two reads of one branch answer the same sequence, which is what keeps
// the closing writes of one reap a set a case can state.
func (s *Store) OpenEntries() ([]OpenEntry, error) {
	partitions, err := s.partitions()
	if err != nil {
		return nil, err
	}

	var groups []group
	for _, partition := range partitions {
		for _, held := range partition {
			if accounted(held) {
				continue
			}
			groups = append(groups, held)
		}
	}

	// One file per open entry, read in one batch: an entry that turned out
	// to be open holds exactly one file the classification did not already
	// answer from its path, and that is its own run.json. An entry holding
	// none contributes no blob and is refused by decodeEntry below, which is
	// where that sentence is stated for every reader of the Journal.
	accounts := make([]group, len(groups))
	var blobs []string
	for i, held := range groups {
		accounts[i] = group{dir: held.dir, run: held.run, account: runFileOf(held)}
		blobs = append(blobs, blobsOf(accounts[i].account)...)
	}
	contents, err := s.repo.readBlobs(blobs)
	if err != nil {
		return nil, err
	}

	open := make([]OpenEntry, len(groups))
	read := 0
	for i, held := range groups {
		account := accounts[i]
		entry, err := decodeEntry(account, contents[read:read+len(account.account)])
		if err != nil {
			return nil, err
		}
		read += len(account.account)
		open[i] = OpenEntry{RunFile: entry.RunFile, Last: highestStep(held)}
	}
	slices.SortFunc(open, func(a, b OpenEntry) int { return newest(a.RunFile, b.RunFile) })
	return open, nil
}

// accounted says the entry holds an account of either form — an outcome.json
// its own Run wrote, or a closing write another Run wrote.
//
// It is answered off the **listing** and opens nothing, which is what §7's *the
// absence is the whole representation* means read from this end: there is no
// state key to consult and no file whose contents decide it, so a closed entry
// costs a reaper the path of the file that closed it and no bytes at all.
func accounted(held group) bool {
	return slices.ContainsFunc(held.account, func(file entryFile) bool {
		return file.Form == FormOutcome || file.Form == FormClosedBy
	})
}

// runFileOf is the entry's own run.json as the listing found it, and nothing
// where the listing found none. It answers a slice rather than a file and a
// flag because what it builds is a group's `account`, which is the shape
// decodeEntry reads.
func runFileOf(held group) []entryFile {
	for _, file := range held.account {
		if file.Form == FormRun {
			return []entryFile{file}
		}
	}
	return nil
}

// highestStep is the highest ordinal the entry's steps/ directory holds, and
// zero where it holds none. It is read off the listing: the ordinal is in the
// path, and a Step file's own `step` is held to building that path wherever one
// is opened (§7, §12, dispositionsOf).
func highestStep(held group) int {
	highest := 0
	for _, file := range held.steps {
		highest = max(highest, file.Step)
	}
	return highest
}
