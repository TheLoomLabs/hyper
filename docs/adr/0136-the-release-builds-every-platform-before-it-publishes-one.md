# The release builds every platform before it publishes one

`scripts/release.sh` runs in two passes. The first compiles every platform into a staging directory
`mktemp` made; the second writes the archives, and `checksums.txt` after them:

    staging=$(mktemp -d)
    trap 'rm -rf "$staging"' EXIT

    for platform in "${platforms[@]}"; do
        …
        go build … -o "$staging/$platform/hyper" ./cmd/hyper
    done

    for platform in "${platforms[@]}"; do
        tar -czf "$outdir/hyper-$version-$platform.tar.gz" -C "$staging/$platform" hyper
    done

It used to build one platform and immediately archive it, four times. That closes
[#261](https://github.com/TheLoomLabs/hyper/issues/261).

## What was wrong

Three of the four archives `v0.0.1-alpha` published report `commit 85244dd…-dirty`, and the release
was cut from a clean tree. `internal/version`'s `commit()` appends the suffix where Go stamped
`vcs.modified`, so that *a hash on the page is never a claim about bytes edited after it* — and on
three of four archives the page made the opposite claim. It was found by
[#247](https://github.com/TheLoomLabs/hyper/issues/247), which ran all four archives on machines that
match them, and read into
[ADR-0133](0133-three-archives-nobody-had-run-carried-and-the-release-stamps-three-of-four-dirty.md).

**The release wrote its own dirt.** Go stamps `vcs.modified` from `git status` in the module root,
`release.yml` runs `bash scripts/release.sh "$VERSION" dist` **from the repository root**, and this
repository has no `.gitignore`. The first pass through the loop wrote
`dist/hyper-0.0.1-alpha-x86_64-linux.tar.gz` into the checkout; every build after it was stamped
against a tree that file had modified. It is the ordering of a loop and nothing else — the platform
that goes first is clean, and the three laptop archives are dirty because they are not first.

## Why the repair goes in the ordering

The loop's two halves are independent — nothing a build needs comes from an archive — so separating
them costs nothing and removes the mechanism outright: **no build in the first pass can see another
build's output, because there is none yet.** That is what keeps `dist` inside the checkout where
`release.yml` and `docs/build/releasing.md` both put it. Measured from a clean clone, all four
archives are now stamped `vcs.modified=false`; before the change the same invocation gave `false` for
`x86_64-linux` and `true` for the other three.

**It is not a claim that the output directory is harmless wherever it points.** A release cut into a
directory a previous one left archives in starts from a tree that is already modified, and all four
builds are stamped `true` — which is the honest answer and is why the repair is stated as an ordering
between the builds rather than as a property of `$outdir`. An *empty* directory is not a
modification, `git status` reporting nothing for one, which is why `mkdir -p "$outdir"` before the
first build costs nothing.

Three repairs were available and each fixes less:

- **A `.gitignore` naming `dist/`.** It changes what `git status` reports rather than what the
  release does, and the script's contract takes *any* output directory — `release.sh 1.4.0 out`
  dirties the tree exactly as before. It also buys the fault a second way to hide: a real edit under
  a path the file names would stop being reported too. The repository having no `.gitignore` at all
  is a small property worth keeping, every file here being one somebody reviewed.
- **Building into `$RUNNER_TEMP` from `release.yml`.** It repairs the workflow and not the script, so
  the invocation `docs/build/releasing.md` hands a person still publishes a dirty second archive, and
  a case could only see the difference by running the workflow. The property belongs to the thing
  that builds.
- **`-buildvcs=false`.** It deletes the fact rather than making it true: `hyper version`'s second
  line would report `unknown`, and identifying the bytes is the whole job of that page (§9, #103).

Two smaller things follow from the same ordering and are not the reason for it. A platform name with
no build defined for it now exits `2` before any archive exists, where it used to leave a partial
publication behind; and the staging directory holds four binaries at once — about 80 MB, on a runner
that has just checked out a Go toolchain.

## What fences it

`TestRelease_EveryArchiveOfOneReleaseIsStampedFromACleanTree`, and the two properties that make it
able to fail are both deliberate.

**It builds more than one platform.** Every other case in `cmd/hyper/release_test.go` builds one, and
one platform is always the first one — which is why a suite that ran on the tree `85244dd` was tagged
from was green while the release it published carried three false pages.

**It publishes into the working tree.** Handed a `t.TempDir()` outside the checkout there is no dirt
to see. The case makes a directory of its own in the repository root, of the kind `dist` is; an empty
directory is not a modification, `git status` reporting nothing for one, so the tree the builds are
stamped against is the tree the case found.

**And it requires a clean one.** On a tree that already carries edits every build is stamped
`vcs.modified=true` honestly and the ordering makes no difference, so the case cannot see the fault
there. That is a property of the checkout rather than of the machine, so it goes through
`unavailable` ([ADR-0123](0123-the-suite-is-run-by-a-machine-and-a-prepared-machine-may-not-skip.md)):
a laptop mid-change skips and says why, and a runner fails — `actions/checkout` supplies exactly one
tree, a release is cut from it, and this is the suite `publish` waits on.

## What it does not do

**The published `0.0.1-alpha` archives are left alone.** They are what they are; the repair is
visible in the next release and not retroactively, which is the shape #247 asked for and what
ADR-0133 already recorded as a standing cost.

**Nothing downstream is affected, because nothing downstream reads it.** `vcs.modified` reaches
`version.Facts.Modified`, which reaches `Page()`'s second line and nothing else. It is not
`repo_dirty`, which is Provenance about the **user's** repository and is computed per Run (§8). What
the fault cost was exactly the page §9 sends an operator to when they want to know what they are
running, telling them something false about it — and that is what this returns.
