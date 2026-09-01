# A remedy may not assert what the answer could not establish

**`hyper project`'s `release-artefact-absent` Refusal named two remedies — *publish a release for
0.0.1-alpha, or install a released hyper* — and both are false against a release that is published
and private.** The Refusal is correct there and stays; the exit code is `77` there and stays. What
changes is one sentence: where the evidence is a status rather than a file, the remedy names the
third possibility it cannot rule out instead of asserting the two it cannot establish. Issue #254,
observed by the session read in
[ADR-0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md).

Nothing about the check moves. `absentAt` sorts the same statuses the same way, a `429` is still exit
`1`, and a verbatim retry against a private release still Refuses identically — which is `77`'s whole
promise and the reason the classification was never the defect.

## What was wrong (issue #254)

`v0.0.1-alpha` is published. This repository is private, so an unauthenticated read of a release asset
answers `404`, and `project`'s read is unauthenticated by construction: it holds no credential, takes
none, and never will ([ADR-0007](0007-hyper-never-stores-a-secret.md)). So the operator setting up the
throwaway repository for ADR-0125's runs was told, by a binary that had itself come out of that very
archive, to publish a release that was already published and to install a released `hyper` they were
already running.

**Neither remedy applied, and the sentence offered nothing that did.** The route out they took was to
hand-write `hyper.yaml` carrying the two values `project` derives — the one act the orientation tells
authors never to perform — and §11 closes the other door on the way past: *a pin already equal to the
binary's version resolves nothing*, so re-running `project` afterwards confirms the hand-written digest
rather than checking it.

This is a defect in what the Refusal *says*, and it is the second of its kind this repository has
found the same way. [ADR-0122](0122-a-requirement-roots-at-any-projected-field-and-the-value-goes-on-the-line.md)
is the other: a surface that is correct, that no test can fail on, and whose cost is a reader's time
and a wrong edit.

## The decision

**The remedy turns on whether the checksums file was read.**

- **It arrived and named no artefact for this version.** The release is readable and its contents were
  seen: this absence was *observed*. The remedy is the two routes it has always named — publish a
  release for this version, or install a binary some release does name.
- **Nothing arrived: the status was `404` or `410`.** This absence was *inferred* from a status that
  three different worlds answer identically — no release under the tag, which is what an unreleased
  binary asks for; no checksums file beside a release that is there; and a release nobody may read
  unauthenticated. The remedy names the two routes above and, between them, *make an existing one
  readable unauthenticated*.

`internal/release`'s `Absent` carries the one bit that parts them and `internal/cli`'s `remedyFor`
writes the sentence, which keeps the shape where the fetch is and the wording where every other
Refusal's wording is. §11 states the fourth shape outright and §12's closed-set entry names it in the
gloss.

## What was considered

**A second, authenticated read to tell them apart.** Rejected on ADR-0007: `hyper` holds no credential
of its own and a `project` that reached for one would be a tool authenticating to a host no artefact
named, on an act whose whole claim is that it is a reviewable fetch of a public file. The distinction
is not worth the thing it would cost, and it is not `hyper`'s to make — the operator knows whether
their own repository is private.

**A code of its own for the private case.** Rejected because `hyper` cannot detect it. A code is a
claim about what happened, and this one would be a guess dressed as an observation — handed to a
reader as the closed set's other members are, every one of which names something the binary saw. The
four shapes are one fact for a reader —
there is no artefact this binary can pin — and the remedy is where a possibility gets named without
being claimed.

**Exit `1` rather than `77`, on the ground that the answer is ambiguous.** Rejected: `77` promises a
verbatim retry Refuses identically, which a private release keeps perfectly. Ambiguity about the
world's *reason* is not the same as instability in the answer, and moving the code would tell a
runner's operator that a private release is a transient fault to be retried.

**Leaving it until this repository is public.** Rejected, and the scope note in #254 is the reason it
was tempting: the message becomes correct for every reader the day the repository opens. But the same
shape reaches every private fork, every vendored release in a company's own repository, and every
release published to a repository whose visibility changed after the tag — none of which this project
controls or can wait out.

## What it costs

**A longer line, and one that names a possibility most readers do not have.** The sentence is now
three clauses where it was two, on a Refusal that fires most often for the plainest reason of all — an
unreleased binary, which is the case the README documents. That is the trade this record is: a
sentence slightly less crisp for every reader, against a sentence that is *wrong* for a whole class of
repository.

**A bit of state on an error type.** `Absent` was a fault string; it now carries whether the file
arrived. That is one field held for one caller's sentence, and it is the narrowest thing that could
distinguish shapes the code already distinguishes internally and was throwing away.

## The run this repair owes, and why it is not bought

**This is a taught repair** — the wording of a Refusal, which nothing in the suite can fail on
([`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md), #250). The goldens hold that
the text changed and no more.

**No task owes a run, and the reason is structural rather than a deferral.** The sealed harness runs
`project` itself in setup, and `scripts/acceptance/run.sh` says why nothing there can reach this code:
the pin and the binary's stamp are one value by construction, so `frozenDigest` resolves nothing and
touches no network. No agent in six sealed sessions has read this sentence or can. Its reader is a
human operator on first use, which is where it was found.

**The measurement it owes is #245's**, the by-hand `hyper project` against the first published tag —
where the same operator, on the same machine, is the reader this sentence is for. Where that run
happens against a repository that is still private, the `404` branch is what it will take, and this
sentence is what it will read; that is the evidence, and it goes into the ADR #245 asks for.
