# A Refusal belongs to the Run, not to the Step

A Refusal is held on `outcome.json` and nowhere else, under `refusal` — an ordered array of at least
one member, each member a `check` problem plus the coordinates a Run adds. The Step file carries none
of it. A Run that could not sync the Store is not a Refusal at all: it is `failed` at `75`, beside the
lock and the unrebaseable push.

§7 put the Refusal in the Step file, and the reading is natural because the archetype is: `bound-exceeded`
halts Step 3, so the check that declined, the Bound it declared, the count it observed and the line to
edit all sit beside a Step that exists. Generalise from that one case and a Refusal looks like a
Disposition with extra fields.

The archetype is the minority. A Run begins with `check` re-run in full with nothing skipped (§6), so
**every one of §4's twenty-eight static codes declines a Run before Step 1** — beside `credential-absent`,
`cadence-run-once` and `store-schema-unsupported`. Thirty-odd of the forty-five members of the closed
set have no Step to attach to. Only four require a Step to have been reached at all. Designing the
home for a Refusal around `bound-exceeded` is designing it around the exception.

Nor is a per-Step home the right shape even where a Step exists. A Refusal is terminal: a Run has at most
one, ever. A per-Run singleton in a per-Step file is a category error that the pre-Step cases merely
expose — and `outcome.json` is the file that already says the Run refused, so the reader holding the
outcome is holding the explanation with no second lookup and no second place to check. The wire said
this before the Store did: §8's `refusal` is its own row type carrying an optional `step`, and the
`error_code` rides on the `outcome` row. Neither is a step row.

What survives on the Step file is what actually happened to that Step — `disposition: "refused"`, its
selector, what it expanded to. `step` inside the refusal object is an **artefact coordinate and never an
execution fact**: `bound-missing` cites `steps[2]` in a Run where nothing was reached, and the Step it
names may have no file in the entry at all.

The second half follows from taking the pre-Step population seriously. `store-unsynced` was the one
member citing no artefact and no line, and it was also the one member for which `77`'s promise — *a
verbatim retry will refuse identically* — is false: a Run that could not reach the remote succeeds on the
same retry five minutes later. `75` already means *a Run that lost the Store*, to the lock or to a push
it could not rebase through, and a sync it could not complete is the third member of that list. The line
between the two codes is not severity but **whether an act is required**: past a `77` lies an edit, a
`store init`, a `project`, a newer binary, or a variable set in the environment, and until somebody
performs one nothing changes; past a `75` lies time. Moving it out fixes the exit-code contract and
removes the only Refusal with nothing to cite in the same act.

## Considered options

- **A Step file for a Step that never ran.** Rejected: it needs a `step` position for checks that may
  have none, a seventh thing for §12's six Dispositions not to describe, and `started_at`/`ended_at`
  fields `hyper` would have to invent for a Step that was never begun — the exact fabrication the reaper
  rule already refuses (§7).
- **The Refusal on `outcome.json` *in addition to* the Step file.** Rejected: two homes for one fact
  means every reader checks both and the two can disagree, which is the failure §7 avoids everywhere
  else by storing no exit code, no duration and no Head marker.
- **A fourth Journal file.** Rejected: a fifth schema integer (ADR-0028) and a second file written at the
  same instant as `outcome.json`, on every path where `outcome.json` already says `refused`. Splitting an
  outcome from its explanation across two files makes the reader that knows *what* open another file for
  *why*.
- **A stored head beside the array** — `{"error_code": …, "problems": [ … ]}`. Rejected: the head is the
  first member's code, derivable, and a stored derivation can drift from what it heads.
- **One `refusal` row carrying an array on the wire.** Rejected: the stream would become the one surface
  that nests, and `select(.type=="refusal")` would stop returning one problem per line.
- **Keeping `store-unsynced` as a Refusal that writes nothing, like the bootstrap `store-absent`.**
  Rejected: it leaves `77` telling a caller not to retry the one condition retrying fixes. `store-absent`
  earns `77` because `store init` is an act; a network blip is not.

## Consequences

- **`outcome.json` carries `refusal`**, an ordered array of at least one member: `error_code`, `file`,
  `line`, `field`, `message`, plus `step`/`step_id` where a Step is cited and `declared`/`observed` only
  where the check compared two values. No new file, no fifth schema integer, no seventh Disposition.
- **A Refusal and a `check` problem are one shape** — one thing arriving through two commands, differing
  only in exit code, `1` against `77`. The key is `file` rather than `artefact`, the word `hyper` reserves
  for the five reviewed artefacts: `projection-stale` cites a generated workflow and
  `store-schema-unsupported` cites a Store file. The `remediation` row was renamed with it, matching the
  `FILE` column the `EDIT ONE OF` table already draws.
- **The array has more than one member only where the phase evaluates many checks together** — the
  Run-start `check` and the credential pass — and exactly one everywhere else, a Refusal being terminal.
  Every member renders; the terminal line and the `outcome` row name the first.
- **Run start has a stated order**: pin gate → locate Store → write and push `run.json` → Store schema →
  `check` in full → resolve credentials → Step 1. The push *is* the effectful Run's sync, one reach at
  the remote rather than two, and it comes first so that every gate below declines into an entry that
  exists and has reached the remote — which is what makes an unattended `credential-absent` readable at
  all.
- **`store-unsynced` leaves the `error_code` set**, 46 → 45, and §7's four Store codes become three. A
  Run that lost the Store at its Run-start sync is `failed`, exit `75`, and the terminal line still names
  its id: an entry *was* written, locally, and it goes out with the next Run that syncs.
- **The Refusal rendering degrades by part.** No Step reached, no Step table — replaced by
  `nothing ran. no step was reached.`, because an absence cannot carry the most important fact on the
  page. The remediation table appears only where an artefact edit is the way past; the three other
  remedies — a command, a newer binary, an act on the environment — go in the `=` notes, which may name
  the remedying command verbatim.
- **`store-schema-unsupported` renders no caret**, its file being evidence rather than an artefact
  (ADR-0011), and it is tested at Run start over the Journal and the Record heads under the
  (Definition, Target) pairs the Procedure makes — the same scoping sentence the credential pass uses.
  A per-read test would give one code two phases, and §7 derives the phase from the code.
- **The terminal line's `show` pointer is earned by truncation, not by the outcome.** Where the page is
  complete it falls back to `refused · exit 77 · run <id>`.
