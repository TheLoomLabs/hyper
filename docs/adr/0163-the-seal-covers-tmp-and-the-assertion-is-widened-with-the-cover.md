# The seal covers `/tmp`, and the assertion is widened with the cover

**`scripts/acceptance/run.sh` puts a `--tmpfs` over `/tmp` at mode `01777` and binds nothing back on
top of it; the assertion that follows walks `/tmp` alongside `$HOME` and the output directory, and
looks for a regular file named `hyper` alongside the `go.mod`, the `mcp.json` and the `lookout` it
already looked for.** That closes the hole [ADR-0162](0162-the-taught-destroy-fired-and-closed-the-record-and-the-seal-was-not-holding-while-it-did.md)
found and issue [#292](https://github.com/TheLoomLabs/hyper/issues/292) booked, and it closes it in
both halves, because the half that failed silently for two paid runs was the assertion.

The claim [ADR-0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md) states —
**no source checkout, no second binary, no fixture internals, and the one binary that is reachable is
the one the MCP server is** — is held again.

**Read the issue's first requirement against that claim and not past it.** *A sealed `find / -iname
'*hyper*'` comes back empty* is not literally true and never was: `$outdir/bin/hyper` is bound in on
purpose, because the MCP server is a child of the sealed session and `mcp.json` names that path
(ADR-0109). What the requirement asks for, and what holds now, is that the one `hyper` such a `find`
answers with is that one.

## What was under there, in one paragraph

ADR-0162 has the finding and the evidence that nothing was read. In short: `/tmp` was covered by
nothing and searched by nothing, and both sealed runs of 2026-09-09 could reach a second `hyper` —
sixteen megabytes, stamped `0.0.3-alpha`, **the version the fixture pins**, so the version gate that
Refuses a dev build would have let it run — beside ten kilobytes of `docs/spec/` prose, a diff over
the specification and a program whose only purpose is to print the orientation. All of it sat under
`/tmp/claude-1000/<project>/<session>/scratchpad`, the directory the client hands every attended
session on this project.

## The cover is the inversion, not two more entries on a list

Issue [#257](https://github.com/TheLoomLabs/hyper/issues/257)'s argument against a list of what to
hide is that **what a list of that shape cannot cover is the thing that grows**, and the scratchpad
directory is a thing that grows: it is assigned to every attended session on this project, so it
collects exactly the artefacts the seal is about — those being what sessions on this project handle.
Five session directories were sitting in it during the run that found this. The material `$HOME` used
to leak has not stopped being produced; it moved somewhere nothing looked.

So `/tmp` goes the way `$HOME` went in
[ADR-0130](0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md): wholesale,
by a tmpfs, and what the sealed session needs comes back on top by name.

**Here nothing comes back.** The cost issue #292 named for this shape was *finding out what the
client needs there*, and a `--tmpfs` narrows that question to one thing: what the client wants in
`/tmp` it can still make, since the mount is writable and carries `/tmp`'s own mode; what an empty
`/tmp` denies is reading something that was in there *already*, which is the whole of the point.

**What was actually checked is narrower than a sealed session**, and is worth stating as such: the
client starts and answers under `bwrap --bind / / --tmpfs /tmp`, and `--clearenv` means it is handed
no `TMPDIR` pointing anywhere else. A full `--print` run with its MCP child and its shell snapshots
was not bought for this; the next sealed run is the evidence, and the failure mode it would show is a
client that cannot start rather than a quiet one.

**It is laid down before the other two**, because either of them can be inside it — `t.TempDir()`
lands under `/tmp` wherever `TMPDIR` is unset, and the suite drives the harness with both a home
directory and an output directory there. A tmpfs over `/tmp` mounted after one over `$HOME` would
shadow the home directory and everything bound back into it. Two behaviours this rests on were
checked the way ADR-0109 checked its three: **a tmpfs destination inside a tmpfs is created, parents
included**, and **a bind source is still resolved against the old root**, so the operands naming
`$outdir/.empty` and `$outdir/.claude.json` go on working with an output directory inside `/tmp`.

`allow` records a kept path's chain **all the way to `/`** now rather than stopping at `$HOME`. That
was looser than it needed to be while `$HOME` and the output directory were the only walk roots — the
extra entries were ancestors of a walk's own root, which `find` never prints. With `/tmp` a walk root,
a `$HOME` inside it has ancestors the walk does print, and stopping at `$HOME` would leave them
unaccounted for and the seal reading as broken over the two directories it had just built.

## The assertion is widened in the same change, because it is the half that failed

A cover that is never asserted is how this went unnoticed for two runs. The assertion searched
`$HOME /opt /srv /var/tmp` for a `go.mod`, an `mcp.json` and a regular file named `lookout`: **`/tmp`
was not one of its roots, and a second `hyper` was not one of its names.** Either one alone would have
caught what was there, and both are added.

- **The root.** `/tmp` joins the name search and the inventory walk. Inside the seal it is a tmpfs
  holding the mount points the covers below it created, so what it costs is nothing.
- **The name.** A regular file called `hyper`, beside `lookout`. This run's own stamped binary matches
  it and is not a finding — it is on `keep`'s list, as `mcp.json` is, and the list is subtracted
  before anything is concluded. `-type f` is load-bearing, because a *directory* named `hyper` is
  ordinary: a checkout of this project is one, and `.git/hyper/` is where a Run puts the Store's own
  local state (`internal/store/lock.go`), so every repository a Run has touched carries one. The
  Store's branch is `hyper-store` and matches this name nowhere.

**Outside the three inventoried directories it is still names and not an inventory**, which is
ADR-0109's split and is where this repair stops. A `section.md` or a `284.diff` under `/opt`, `/srv`
or `/var/tmp` is silent, as it was before. That is the bargain the names have always been: an
inventory of a directory nobody covered is a list of everything on the machine, and the answer to a
directory that collects this project's material is to cover it, which is what the first half of this
record does.

**The cover and the root are one decision rather than two, and the machine says so.** With the cover
removed and the root kept, the inventory does not report a stray file — it exits non-zero on the
machine's own `systemd-private-*` directories, which are unreadable to us, and the missing `SEALED`
sentinel stops the harness with *the search inside it never ran*. The inventory is only sound over a
`/tmp` that is a tmpfs of ours. That is a fatal answer either way, and the `find` errors on the
terminal say which paths they were, but it is the reason the two halves land together.

## What this is fenced by

`cmd/hyper/acceptance_test.go` now drives the setup half three ways, all through `setUp`:

- **`TestAcceptance_TheSealCoversWhatCollectsInTheScratchpadDirectory`** plants the shapes this ticket
  is about under the real `/tmp` — a `hyper`, a `section.md`, a `284.diff`, an `orient.go`, and beside
  them a `go.mod` naming this module and a `lookout` — and asserts the harness runs. Three of them
  match no search at all, which is the finding: a harness that covered `/tmp` by naming what to hide
  would pass in silence. The other three — the `go.mod`, the `lookout` and the `hyper` this change
  itself added the name for — do match, so a harness that stopped covering `/tmp` fails loudly.
  It plants in the real `/tmp` because the cover names the literal path and no variable moves it —
  which is what separates it from the `$HOME` case beside it, where `HOME` is redirected and nothing
  is left on the machine. A uniquely-named directory, removed on the way out, is the price.
- **`TestAcceptance_ASecondBinaryTheCoversDoNotReachStopsTheHarness`** plants a `hyper` under
  `/var/tmp` — a root the assertion walks and one where the only covers are previous output
  directories, found by their `mcp.json` — and asserts the harness **refuses to start** and names what
  it found. That is the position the second `hyper` was in, and the refusal is the pass.
- **`TestAcceptance_TheSealCoversWhatAnAttendedSessionLeftInTheHomeDirectory`** is unchanged.

Both new cases were run against the repair reverted one half at a time, and each fails on its own
half.

## What was considered

**A `--tmpfs` over `/tmp/claude-1000`, or over the project subdirectory, alone.** Narrower and
cheaper, and refused: it is another entry on the list issue #257 condemned, and worse than most,
because it names a path that belongs to *the client running the session* rather than to this project.
A client that moves its scratchpad leaves a cover that quietly covers nothing, and the thing it stops
covering is the thing that grows.

**Widening the assertion and covering nothing**, which issue #292 called the cheapest and the honest
one. Taken as well, refused as instead. It seals nothing: it converts a silent hole into a harness
that refuses to start until the operator clears the directory — and the directory is the scratchpad of
an attended session on this project, which is where the harness is always run from and which refills
with the working material of the very ticket in hand. The operator would be deleting their own
evidence to buy a run, which is the shape ADR-0109 refused when it chose to *cover* a previous run's
output directory rather than complain about it.

**A `keep` list over `/tmp`, as `$HOME` got.** There is nothing to keep. The list is empty because the
client was checked rather than reasoned about, and an empty list is the strongest form of this cover
rather than an unfinished one.

**An empty directory bound over `/tmp`, like the checkout's parent and the build cache.** Not
available, for ADR-0109's reason one directory over: an output directory and a home directory can live
inside `/tmp`, and `bwrap` creates a destination inside a tmpfs it laid down but not inside a
read-only bind of an empty directory.

**Naming `hyper` in the inventory rather than in the name search.** The inventory takes no names —
that is what makes it survive a task leaving a file nobody thought of (ADR-0109). The names are for
what lies *outside* the inventoried directories, and that is where the leg went.

**Buying a sealed run to confirm the repair.** Not owed and not bought.
`docs/agents/acceptance-re-runs.md`'s rule is that a **taught** repair owes a run and an **enforced**
one does not; nothing an agent reads changes here, and what does change is asserted by the harness's
own fence on every `go test ./cmd/hyper`. The transcripts already collected are not in doubt either:
ADR-0162 checked both, and neither read any of what was reachable.

## Consequences

- **The instruction to clear the scratchpad before buying a run is retired.** `run.sh`'s header,
  `CONTRIBUTING.md` and ADR-0109 carried it; all three now say what the seal covers instead. Leave
  what you like in `/tmp`, as you may in `$HOME`.
- **A second `hyper` outside the covered directories now stops the harness.** A dev build left under
  `/opt`, `/srv` or `/var/tmp` is a run that does not start until the operator moves it, deletes it,
  or finds out why no cover reached it. That is the intended cost of the name and it is the same
  bargain `lookout` has had since ADR-0109.
- **ADR-0109 is amended rather than corrected**, this being a wider seal and not a fact that was wrong
  when it was written — the same treatment ADR-0130 got. The paragraph recording that the claim was
  not held stands, with the amendment beside it.
- **No transcript is invalidated and no conclusion moves.** What was false was the claim, for the two
  runs ADR-0162 names; what those runs read is unchanged.
- **`allow` records to `/`**, and the loosening is now the thing that makes the walk over `/tmp`
  correct.
