# What `project` wrote, and what it left alone

Twenty-three cases, each with a repository of its own, and every one of them
holds a `tree.golden` (issues #177 and #178). That is the corpus's own rule and
the reason it exists: the two text streams say what the command **reported**,
and only the tree says what it **did** — a case checking its stdout alone would
pass on a command that printed the right table and wrote nothing, or wrote it
somewhere else.

A `tree.golden` renders `.github/workflows/` **and `hyper.yaml`**, which are the
two places `project` writes. The declaration is where the pin and the frozen
digest land, and a golden that showed only the workflows would assert half of
one edit.

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
| `the-pin-the-binary-disagrees-with` | the upgrade, seen whole: a declaration pinning `1.3.0` under a `1.4.0` binary, the checksum resolved once and frozen, and the two scalars moved with the comments and `retention:` carried through |
| `a-repository-with-no-pin` | no `hyper.yaml` at all: one is created carrying `kind:`, `version:` and `digest:`, and **no `retention:`** |
| `no-release-under-the-tag` | `release-artefact-absent`: the checksums file answers `404`, which is a tag with no release and a release with no checksums file alike |
| `no-line-for-the-artefact` | the same code's third shape: the file is there and names no artefact for the platform `runs-on` fixes |
| `a-checksum-that-never-arrived` | the connection is refused: exit `1`, the world resisting rather than a check declining, and nothing written |

`a-repository-that-does-not-check`, the three usage cases and the three that
fail at the fetch are together the criterion no page can state on its own: **a
refusal or a failure before the first write leaves the tree byte-identical.** Each of the three that fail at the fetch stands a
repository whose projection is *wanted and absent* and whose pin is one this
binary is not, so a `project` that got past the fetch would have created the
namespace and moved the pin; their `tree.golden` is `no .github/workflows/
directory` and a declaration still pinning `1.3.0`.

## The pin, and the one network read

`project` is the only command in §9's tree that calls no pin gate, because it is
the pin's only writer — which is what the first two cases above drive, one
against a declaration the gate would have Refused and one against no declaration
at all. The corpus-wide guard beside `TestGolden` holds the other half: no case
here Refuses under either pin code, and every other command in the tree has a
case that does.

**Only a version change fetches.** Every other case in this corpus pins the
version the binary is, so its digest is copied out of the declaration and no
connection is made at all — which the harness asserts for free, a case that
dials without a `serve/` entry failing on that alone.

The five cases that do fetch carry `serve/github.com.json`, and what they serve
is what a *server* would: `sha256sum`'s own output under the release tag, a
`404`, or a connection refused. There is no case here for a `503` or a `429`,
and that is deliberate: which statuses are *the artefact is not there* and which
are the world resisting is a classification internal/release owns and its own
cases drive, where what this corpus is for is the two exits it comes out as. The read binds no Target, resolves no credential,
opens no Store and writes no Journal entry — `a-repository-with-no-pin` is the
one that says so, its Target declaring a credential slot no case ever sets and
its repository holding no Store branch at all.

## What the `--json` twins are for

Eight of the cases have one — every case that emitted a row when the `--json`
twins were written, and two that emit none. The cases about the pin carry no
twin: what they are about is the tree and the exit code, and the rows they emit
are shapes the twins already hold. Two claims live there and nowhere else: the
`workflow` row carries the gloss's **parts** and never the composed line, and a
removed file's row carries `path` and — where the Procedure still exists —
`procedure`, and no `cadence`, `phrase` or `rate` at all. §10's two facts reach
neither row: they are derived from `cadence` and `phrase`, which are already on
the wire, so a consumer derives them exactly as the page does.
