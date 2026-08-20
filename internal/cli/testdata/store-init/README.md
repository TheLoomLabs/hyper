# `hyper store init`

The record's first command, and the first command in the tool that writes
anything at all (issue #126). Every case here drives `hyper store init` — or
the noun group around it — through `cli.Main` from its own `argv`, and asserts
the two streams, the exit code, and the branch the run left behind.

The corpus is named for the command as a caller types it. `store` is §9's one
noun group, so this command's name is two words, and `store-init/` is that name
with the space out of it. The cases about the group's own grammar sit here too:
`hyper store` with no verb is not `hyper store init`, but it is about nothing
else.

## What the command decides, and the case for each answer

`store init` asks two questions — is the branch here, and is it on `origin` —
and there are four answers between them. Every one has a case, and each carries
a `store.golden` because the two text streams say what the command *reported*
and only the tree says what it *did* (`golden_fixture_test.go` states the
format).

- `creates-the-store/` — neither side holds it and there is no remote at all.
  The tracer bullet: a parentless commit whose tree holds `STORE.md` and nothing
  else, no push, exit `0`.
- `creates-and-pushes/` — neither side holds it and `origin` is wired. Its two
  goldens are identical, which is the whole of what the push is for: a Store
  that exists only on the laptop that ran `init` refuses every scheduled Run
  forever (§7).
- `store-already-local/` — the branch is here and there is no remote, so there
  is nothing left to do. Nothing is minted, nothing is written, and its
  `store.golden` is **byte-identical** to `creates-the-store/`'s — which is the
  assertion, `STORE.md` being written once and a second `init` that rewrote it
  being the one rewrite append-only forbids (§7, §12, ADR-0011). Its `store/` is
  seeded with exactly the bytes `internal/store` writes, so the two goldens are
  comparable at all.
- `pushes-a-store-the-remote-lacks/` — the branch is here and `origin` has none.
  Nothing is created and the branch goes out, which is the one combination that
  renders a `pushed` line and no other. It is the state a rejected push leaves
  behind, and this is the way back from it.
- `store-on-origin-alone/` — `hyper-store` on `origin` and nowhere else, which
  is the state every runner's fresh clone is in. It is **fetched** rather than
  re-created: two clones each minting an orphan root produce two histories that
  can never fast-forward into one another. The seeded Store carries a Journal
  file as well as `STORE.md`, which is what makes the fetch visible — a run that
  minted a root instead would render one entry where the golden holds two.

The Journal file under `store-on-origin-alone/remote-store/` is a placeholder
and is meant to be. §7's canonical encoding and its path grammar are milestone
4.3's; what that file is for here is being a second path on the branch.

## The gate

`version-pin-mismatch/` and `version-pin-absent/` Refuse at `77` before any git
subprocess runs, and their `store.golden` is the absent-branch marker: the
branch was not touched, which is the half a stderr comparison cannot show (§11,
ADR-0020). They are the only two cases here that do not pin `1.4.0`.

## The two forms

`creates-and-pushes-json/`, `creates-the-store-json/` and
`store-already-local-json/` are the wire: one `branch` row and the terminal
`result` row, `type` first, `truncated` always `false`. `pushed` is absent
rather than `false` where the command pushed nothing — `creates-the-store-json/`
is that case, its repository having no remote — and `created` is written always,
including `false`, which is a second `init` saying *there was already a Store*
rather than saying nothing.

## The usage errors

`usage-no-verb/`, `usage-unknown-verb/`, `usage-positional/` and
`usage-unknown-flag/` all exit `2` with stdout silent, and the first two are
driven in both modes (`usage-no-verb-json/`, `usage-unknown-verb-json/`): a
usage error is not a path the command takes, it is the command never starting,
so no row stream opens and the rendering goes to stderr (§9). None of them
carries a repository, because none of them needs one — a sub-verb is matched
against a compiled-in set and not against a namespace the repository holds, so
the fault is decided from the argument list alone and before any root is
resolved. `TestGoldenCorpora_StdoutCarriesNothingButTheAnswer` holds the silence
over every case here and everywhere else.

`no-git-repository/` is the fifth fault, and the only one that gets as far as
the gate: `--repo-dir` naming a directory with no `.git` in it. There is no
branch to create and no repository to refuse on behalf of, so it is `2` and not
`77`.

## What is not here

The three rules a case directory cannot state — a dirty working tree left
exactly as it was, a branch a human can `git checkout`, and a push that cannot
complete and is then repaired — are in `../../store_test.go`, driven through the
same entry point. The git-level facts beneath them, including that the fetch is
depth-1 and never filtered and that the commit carries `hyper`'s own identity
and the threaded clock, are `internal/store`'s own.

No case supplies a `now`: the harness's stated instant is enough, no golden
rendering a date.
