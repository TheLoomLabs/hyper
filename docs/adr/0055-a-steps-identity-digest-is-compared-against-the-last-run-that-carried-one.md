# A Step's identity digest is compared against the last Run that carried one

A Step's Disposition carries its identity set as a digest, with the sorted `members` written in full only
where that digest moved (§7). The digest it is compared against is the one **that Step carried in the last
Run of the Procedure that carried one at all** — not the digest in the immediately previous Run, and not
the absence of one there.

The reading an implementer reaches unaided is the previous Run: it is what §7's rule said before this
decision, and it is what a Journal ordered by time hands you. It is wrong, and it is wrong in the
direction that costs a guarantee rather than bytes. Two of the six Dispositions carry no identity set and
a third writes no Step file at all (ADR-0030), so a Step refused on Tuesday holds no digest for
Wednesday's Run to differ from. Treating that absence as a difference writes `members` in full on a Run
where the set did not move — and the presence of `members` is what says that it did: §7 makes their
absence mean *the digest did not move*, and the Comparison reads the identity sets to tell a Record that
vanished from one that did not change (ADR-0030). A false *moved* signal in that position is the hole the
identity set exists to close, reopened one key over, on every Run following a Refusal, a condition-skip or
a halt.

The Step is matched by its authored `id` (§3). An `id` that moved is a different Step, with no digest
anywhere behind it, and it writes its set in full on its first Run like any other new Step.

## Considered options

- **The immediately previous Run of the Procedure.** What §7 said, and the cheapest thing to implement
  against a Journal sorted by time. Rejected above: it writes `members` on Runs where nothing moved,
  which is the one thing their presence is defined to mean.
- **Write `members` unconditionally.** Removes the comparison, and with it every question about what to
  compare against. Rejected because the cost is the one the digest exists to avoid: an unchanged listing of
  five hundred Records would cost the set on every Run, which under a Cadence is the whole branch (§7).
- **Compare against the Step's own last written file, wherever it sits.** The same rule stated in terms of
  files rather than Runs. Rejected because the two are not the same rule: *ran* and *refused* both write a
  Step file and only one of them carries a digest, so the file-shaped statement fails on exactly the case
  this decision is about.

## Consequences

- **The walk that reads a set back is total.** Every entry either holds `members`, or — by holding a
  digest — names a set that an earlier entry holds in full, terminating at the Run where that Step first
  carried one and wrote it against no predecessor. Nothing removes the entries in between: Compaction
  touches interior Observation versions and never a Journal entry (§7). So a set and its count are
  recoverable from any entry, and nothing is stored to make them so.
- **A Step's history of sets is unbroken by the Runs where it did nothing.** A Refusal, a condition-skip
  and a halt leave the comparison where they found it rather than resetting it, which makes what a Cadence
  costs a function of how often the world moves rather than of how often a Run declined.
- **`members` present means the set moved, with no exception a reader has to know about.** That is the
  property the Comparison reads, and it is now the property the Store holds.
