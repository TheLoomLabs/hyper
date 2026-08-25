# What `project` wrote, and what it left alone

Nineteen cases, each with a repository of its own, and every one of them holds
a `tree.golden` (issue #177). That is the corpus's own rule and the reason it
exists: the two text streams say what the command **reported**, and only the
tree says what it **did** — a case checking its stdout alone would pass on a
command that printed the right table and wrote nothing, or wrote it somewhere
else.

Each case carries its own `repo/` rather than sharing one. A `tree.golden` is
read against the working tree the case was driven in, so a shared repository
would be a golden with no case to belong to — which is the rule
`store.golden` already keeps one branch over, and the harness enforces both.

## The repository the projecting cases are built from

Seven files: a Repository declaration, a Manifest declaring `header:` auth over
one `read` Operation and one `destroy` Operation, a Target declaring the one
credential slot that scheme reaches, a `read` Definition and an effectful one,
and up to two Procedures.

```
procedures/retire-preview-dns.yaml   cadence: "0 3 * * 1"      a destroy Step
procedures/list-preview-dns.yaml     cadence: "15,45 * * * *"  one read Step
```

The two are what make one criterion visible on the page and the other in the
tree. On the page they gloss differently — `0 3 * * 1` lands on the hour and
carries §10's second fact, `15,45 * * * *` does not — and in the tree they differ
in the **concurrency block** and in nothing else that matters: a Procedure whose
every reachable Step is a `read` takes the shared lock and reaps nothing, so
serialising it would starve a thirty-minute cadence behind a Monday-morning
retirement. Both bind one Target under one scheme, so both carry the same one
`env:` key.

## The cases

| Case | What it holds |
| --- | --- |
| `writes-the-workflow` | one Procedure, no `.github/` at all: the directory is created and one file lands |
| `two-procedures-one-read-only` | both Procedures: two files, differing in the concurrency block |
| `a-dropped-cadence-loses-its-file` | both files stand, one Procedure has dropped its `cadence:`; the same command rewrites one and removes the other |
| `an-unclaimed-workflow-is-removed` | a `hyper-*.yml` naming no Procedure, beside a hand-written `release.yml` the namespace does not own |
| `re-projection-is-byte-identical` | the projection already current: nothing to say and nothing to change |
| `nothing-to-project` | a Procedure that declares no recurrence, and no file standing |
| `a-repository-that-does-not-check` | `cadence-malformed`: `check`'s own problem table, exit `1`, and the stale file still there |
| `usage-positional` | `project <procedure>` — there is no per-Procedure projection to name |
| `usage-unknown-flag` | the three globals and no fourth |
| `usage-dry-run` | `--dry-run` is `run`'s and no other command's: the diff `project` writes is the rehearsal |
| `version-pin-mismatch` | the gate, before the load and long before the first write |

The last five are the criterion no page can state on its own: **a refusal or a
failure before the first write leaves the tree byte-identical.** Each of them
stands a `hyper-nightly.yml` or a stale generated file in the namespace —
exactly the file a `project` that ran would have rewritten or taken away — and
its `tree.golden` is that file, unchanged.

## What the `--json` twins are for

Eight of the cases have one — every case that emits a row, and two that emit
none. Two claims live there and nowhere else: the
`workflow` row carries the gloss's **parts** and never the composed line, and a
removed file's row carries `path` and — where the Procedure still exists —
`procedure`, and no `cadence`, `phrase` or `rate` at all. §10's two facts reach
neither row: they are derived from `cadence` and `phrase`, which are already on
the wire, so a consumer derives them exactly as the page does.
