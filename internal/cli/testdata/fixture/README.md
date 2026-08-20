# Six cases about the repository they are standing in

This corpus is the git fixture as a fixture (issue #125). Every case here
exercises `check`, and none of them is about `check`: what each one drives is a
path through `resolveRepoRoot` or the harness behind it that no case could
reach while a fixture repository was a plain directory reached by `--repo-dir`.

`../exemption/README.md` states the constraint this corpus lifts — *a checked-in
fixture cannot carry a `.git`, and a walk from here would climb straight past
`repo/` and resolve hyper's own repository*. So a case that wants a git root
asks for one, and `golden_fixture_test.go` copies its `repo/` into a temp
directory, `git init`s it and commits it whole before the command runs. Nothing
is written back: the copy, its branches, the bare origin and git's own
configuration all live under `t.TempDir()`, and `TestMain` weighs `testdata/`
before the suite and after it.

- `root-found-by-walking-up/` and `root-found-from-a-subdirectory/` — the case
  names no repository at all, so the root is the one §9's walk finds. The second
  stands one directory down, which is where an operator usually is. Their
  goldens cannot tell you the walk ran — a spliced `--repo-dir` naming the same
  root writes the same line — so
  `TestGoldenFixture_AFindRootCaseNamesNoRepository` holds the argv instead.
- `no-git-root/` — the other end of the same walk, and the only case here that
  materialises nothing: a working directory with no git root above it, exiting
  `2` on the message that names both globals.
- `no-store-branch/` — a git repository whose Store was never created. Its
  `store.golden` is the stated marker, which is the answer `store-absent` will
  be read off (§7).
- `a-seeded-store/` — `store/` becomes `refs/heads/hyper-store` before the
  command runs, built as a parentless commit. Its `store.golden` is the tree:
  sorted paths, each under a header line naming it and its length.
- `a-store-on-origin-alone/` — `hyper-store` on `origin` and nowhere else, which
  is the state every runner's fresh clone is in and the one `store init` must
  not mint a second root against. Its two goldens are the pair: the marker
  locally, the tree on the remote.

The Store content here is a placeholder and is meant to be. §7's encoding, its
path grammar and `STORE.md`'s prose are milestone 4.2 and 4.3's; what these
files are for is proving that the bytes a case seeds are the bytes the branch
holds, in sorted order, and that an absent branch and an empty one are two
different answers.

**Neither branch golden renders a commit id, a tree id, an author or a date.**
Nothing `hyper` answers about the record is defined over the branch's commits
(§7, ADR-0074), so a golden that pinned one would hold the implementation to a
fact the specification does not state — and would move on every machine whose
fixture clock did. The tree is what §7 calls authoritative, and the tree is what
is rendered.

A case here supplies no `version`: they all pin `1.4.0`, which is the version
the harness drives a case with where it names none.
