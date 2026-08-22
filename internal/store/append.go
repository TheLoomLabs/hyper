package store

// The write half of the branch (issue #136): what a Run puts on the Store, one
// commit at a time, and the push it sends them with.
//
// It is two calls rather than one because §7 separates them and §6 spends the
// separation: **a write is committed as the call that produced it confirms —
// one commit per confirmed write — and pushed after every effectful Step, a
// read-only Run's pushes batching to its end** (ADR-0006). So committing and
// sending are two acts with two rhythms, and a Run decides the second by its
// own Kinds rather than by which method it happened to call.
//
// What is here is the appending and not what is appended: which paths a Run
// writes and what goes in them are §7's five shapes, encoded one file over. And
// nothing here decides what a failure means — a push exhausted is
// ErrPushExhausted, and whether that is a Run that lost the Store or a command
// reporting that the world resisted is the caller's (§9, §12, ADR-0061).

// Write is one file to put on the branch: the path §12's grammar built, and the
// bytes §7's canonical encoding produced.
//
// The two travel together and neither is derived from the other. A path is
// lossy — an over-long identity segment is truncated and suffixed — so the file
// restates its own identity, and a writer that built one from the other would
// be the second representation the Store is written to avoid (§7, §12).
type Write struct {
	Path    string
	Content []byte
}

// Append puts the files on the branch as **one commit** and pushes nothing.
//
// Every path must be one the branch does not already hold. That is append-only
// arriving at the one call that could break it (§7, ADR-0011): every Store path
// carries the id of the Run that wrote it, so two Runs cannot write one path
// and one Run writing one path twice is hyper's own arithmetic being wrong —
// which this answers the way every other impossible path does, by refusing to
// produce the bytes rather than by handing a caller an error to decide about.
//
// A write of no files writes no commit and leaves the branch byte-identical,
// which is the ordinary case rather than an edge: an Operation returning what
// the head version already holds mints nothing (§7), and a Step whose every
// Record came back unchanged is a Step with nothing to commit.
//
// message is what `git log` on the branch says about the write. The branch is
// hyper's account of the world and its commits are part of the account a human
// reads there (§7, §13), so the caller names the act rather than this supplying
// a constant that would describe every write the same way.
//
// The branch is moved with its old value named, which is update-ref's own
// guard: a branch that moved between the read and the write is a second writer,
// and overwriting it is the act append-only forbids arriving at the ref. The
// handle is then pointed at what it wrote, so a read taken through it
// afterwards answers about the branch that now stands — which is what lets a
// Step read the Head of a series an earlier Step of the same Run wrote (§6).
func (s *Store) Append(writes []Write, message string) error {
	if len(writes) == 0 {
		return nil
	}

	held, err := s.repo.listTree(s.commit, "")
	if err != nil {
		return err
	}
	standing := make(map[string]bool, len(held))
	for _, entry := range held {
		standing[entry.path] = true
	}

	operations := make([]pathOperation, len(writes))
	for i, write := range writes {
		if standing[write.Path] {
			impossible("%s is already on the branch: no file in the Store is ever rewritten, and every Store path carries the id of the Run that wrote it (§7, ADR-0011, ADR-0076)", write.Path)
		}
		standing[write.Path] = true

		blob, err := s.repo.writeBlob(write.Content)
		if err != nil {
			return err
		}
		operations[i] = pathOperation{path: write.Path, blob: blob}
	}

	tree, err := s.repo.applyOnto(s.commit, operations)
	if err != nil {
		return err
	}
	commit, err := s.repo.commitOnto(tree, s.commit, message)
	if err != nil {
		return err
	}
	if err := s.repo.moveRef(Ref, commit, s.commit); err != nil {
		return err
	}
	s.commit = commit
	return nil
}

// Publish sends the branch to the remote, re-applying onto a tip that moved and
// retrying, three times, after which it answers ErrPushExhausted (§7).
//
// It is the same push every other write in this package goes out on, exported
// here because a Run decides *when* rather than *how*: a read-only Run batches
// its pushes to its end and an effectful one pushes after every Step, and both
// are this call at two rhythms (§6, ADR-0006).
//
// A repository with no remote configured publishes nothing and answers no
// error. What was written stands locally either way, which is the sentence §7
// makes true of every push that did not land.
//
// **The handle is pointed at what the branch now holds, whichever way the push
// went.** A push the remote moved under re-applies this clone's unpushed
// commits onto the fetched tip, which moves the branch out from under the
// commit this handle was holding — and an effectful Run goes on writing after
// its pushes rather than stopping at one, so an Append that built on the old
// commit would be refused by the ref guard and lose the file it was writing
// (§7, ADR-0076, push.go).
//
// The re-resolution's own failure is answered only where the push did not fail
// first: what a caller acts on is the push, and a second error about the ref it
// left behind would name the wrong cause.
func (s *Store) Publish() error {
	published := s.repo.publish()
	tip, err := s.repo.resolveRef(Ref)
	if err != nil {
		if published == nil {
			return err
		}
		return published
	}
	s.commit = tip
	return published
}
