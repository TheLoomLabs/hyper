# A browser sets the attribute, and the shell runs what Finder offers to delete

**The macOS download path has been walked with a browser.** #247 (ADR-0133) established what the two
published archives are signed with and that a quarantine attribute *written by hand* stopped neither
binary — on machines no browser had touched, with an attribute `xattr -w` had invented. This session
had three browsers fetch the published `v0.0.1-alpha` archives on both macOS runners, unpack them
the way Finder unpacks them, and run the result twice: from a shell and through `open`.

**The two ways of running it disagree, and that is the whole answer.** `./hyper version` from a
Terminal prints its page and exits `0` with the attribute still attached. Double-clicked — which is
what `open` does — the same file raises *"hyper" Not Opened · Apple could not verify "hyper" is free
of malware that may harm your Mac or compromise your privacy*, over two buttons, **Done** and **Move
to Trash**. The README tells a person to use a shell, and that is the path that works. Issue #262.

## What ran, and where

Both macOS runners, dispatched from the fixture repository
[`TheLoomLabs/hyper-runner-fixture`](https://github.com/TheLoomLabs/hyper-runner-fixture) — #246's,
reused a third time — on 2026-09-02, against the published `v0.0.1-alpha`.

| machine | what it is | image |
|---|---|---|
| `macos-15-intel` | macOS 15.7.9 `24G830`, `x86_64` | `macos-15` 20260824.0482.1 |
| `macos-15` | macOS 15.7.7 `24G720`, `arm64` | `macos-15-arm64` 20260727.0256.1 |

Both report `spctl --status` → *assessments enabled*, both carry a live `WindowServer` and an `Aqua`
session owned by `runner`, and `screencapture` answers on both — which is what made a browser walk
possible on a runner at all, and what #247 had not established. `LSQuarantine` and Safari's
`AutoOpenSafeDownloads` are unset on both, so both take their defaults.

**A log cannot hold a dialog, so the job photographs the screen.** Every claim below about what a
person *sees* is a frame in `dev/hyper-sessions/hyper-262-evidence/` (ADR-0130), and every claim
about what a machine *says* is a line of a `walk-*.log` beside it.

## Claim 1 — a browser download sets the attribute, and all three set the same one

`curl` sets nothing; #247 confirmed that and it is still true. A browser sets this:

    Chrome    com.apple.quarantine: 0081;6a986782;Chrome;2AA1062A-CB81-433D-BE29-2F75DA6C54E1
    Firefox   com.apple.quarantine: 0081;6a98678b;Firefox;867EC493-5E32-4237-9232-A6751DC0A9FC
    Safari    com.apple.quarantine: 0081;6a9867ad;Safari;0D4F52F9-0117-4C47-8696-632D6A6B7556

**#247's hand-written attribute was the right shape.** It guessed `0081;<timestamp>;Safari;<uuid>`
and that is what all three browsers write, flags word included — so the mechanism ADR-0133 measured
was the mechanism, and what it was missing was not the attribute but everything downstream of it.

Chrome's and Firefox's archives also carry `com.apple.metadata:kMDItemWhereFroms`, and the URL in it
is `release-assets.githubusercontent.com`, not `github.com`: the release redirects, and what is
recorded is where the bytes came from. Only the quarantine attribute survives onto the extracted
binary; `kMDItemWhereFroms` does not, so **the file that gets refused carries no record of where it
came from**.

## Claim 2 — Safari asks first, and names a host the README never mentions

Before Safari downloads anything it asks, per site:

> **Do you want to allow downloads on "release-assets.githubusercontent.com"?**
> You can change which websites can download files in the Websites section of Safari Settings.
> *Cancel · Allow*

Photographed on both architectures. **The site it names is the redirect target.** A person following
the README has typed or clicked a `github.com` URL, and the sheet asks them to trust a hostname that
appears nowhere in the README, in this repository, or in the release page they came from.

**Answering it is the only way Safari writes the file**, and answering it is exactly what a machine
turned out to be bad at. A synthetic return through System Events opened the download on the Intel
machine, every time, at the first attempt. On the arm64 machine it never did: one press in run
33665965202 and three — at 16s, 32s and 48s — in run 33667291715, `osascript` reporting success each
time, and nothing ever landed in `~/Downloads`. Whatever the cause, **Safari downloaded on one
machine because a keystroke landed and on the other it did not** — the same brittleness Claim 5 runs
aground on. Chrome and Firefox ask nothing and downloaded on both.

## Claim 3 — Archive Utility unpacks it and carries the attribute onto the binary

`open`ing the `.tar.gz` is what a double-click does, and Archive Utility is what answers — it is the
frontmost application in the frame taken straight afterwards. It extracts `hyper` with its executable
bit, and it propagates the attribute unchanged:

    -rwxr-xr-x@  1 runner  staff  15815632  hyper
    com.apple.quarantine: 0081;6a986782;Chrome;2AA1062A-CB81-433D-BE29-2F75DA6C54E1

Byte-identical to the archive's, on both machines. **This settles what ADR-0133 left open**: it had
found `tar -xzf` carrying the attribute on the Intel runner and dropping it on the Apple Silicon one,
and called two macOS 15.7 machines disagreeing about propagation the gap the whole story ran
through. They do not disagree about Archive Utility, which is the unpacker a person actually gets.

Safari, with `AutoOpenSafeDownloads` at its default, unpacks one layer before anybody asks it to:
what lands in `~/Downloads` is `hyper-0.0.1-alpha-x86_64-darwin.tar`, quarantined, and the `.tar.gz`
is gone. Whether a second `open` on that `.tar` reaches the same binary was not walked — Chrome
downloaded first and the rest of the job read Chrome's archive.

## Claim 4 — the shell runs it, Finder refuses it, and `spctl` describes neither

With the attribute still attached, from a Terminal:

    $ ./hyper version
    hyper 0.0.1-alpha
    commit  85244dd1703f92c75f7c0915927ef5341479954f-dirty
    …
    exit 0

Through `open`, the same file, the same second:

> **"hyper" Not Opened**
> Apple could not verify "hyper" is free of malware that may harm your Mac or compromise your privacy.
> *Done · Move to Trash*

On the arm64 machine **Move to Trash is the highlighted default button**. The dialog is modal and
`open` never returns; the job bounds it at 45s and photographs it.

`spctl --assess --type execute` answers `rejected` on both machines — before the download, after it,
with the attribute and without it — so **the assessment is not the thing that decides**, which is the
half of this ADR-0133 got right for the wrong reason. What decides is whether the launch goes through
LaunchServices: a binary `exec`ed by a shell is not assessed, and one opened by Finder is.

## Claim 5 — whether `xattr -d` clears that dialog is *not* established, and three runs say why

`xattr -d com.apple.quarantine ./hyper` is the line ADR-0133 found `releasing.md` recommending for a
mechanism nobody had measured. **It is still unmeasured, and this session failed at it three times
in the same way.** The failure is recorded here rather than smoothed over, because the intermediate
readings each looked like an answer.

The step removes the attribute and `open`s the binary again. Both times it got:

    _LSOpenURLsWithCompletionHandler() failed with error -10673 … open exit 1

which reads as *Finder refuses it whatever you strip off it*. It is not that. **The sheet raised by
the first `open` was still on screen**, in the frame taken immediately afterwards, in every run — so
every reading after it was taken behind a live modal.

- Run [33666463613](https://github.com/TheLoomLabs/hyper-runner-fixture/actions/runs/33666463613)
  tried `pkill -x Finder` first. The sheet is CoreServicesUIAgent's, not Finder's, and killing Finder
  dismissed nothing.
- Run [33666832760](https://github.com/TheLoomLabs/hyper-runner-fixture/actions/runs/33666832760)
  pressed **escape** through System Events — escape rather than the default button, because the
  highlighted default is *Move to Trash* and a job that pressed return would delete what it is
  measuring. `osascript` reported no error and the sheet did not close.
- Run [33667291715](https://github.com/TheLoomLabs/hyper-runner-fixture/actions/runs/33667291715)
  added a control — `int main(void){return 0;}`, compiled on the machine, downloaded by nothing —
  to find out whether `-10673` was about the download at all. It gives the same `-10673`. **And it
  disambiguates nothing**, because it was `open`ed behind the same undismissed sheet.

So the honest reading of `-10673` is *a second launch was requested while a modal for the first was
still pending*, and neither the repair nor the control has been measured. **What stands is Claim 4**:
with the attribute attached, `open` raises the dialog. Whether removing it takes the dialog away is a
question this session opened and did not close, and the documents say nothing about it.

**A synthetic keystroke is not a click**, which is the general lesson. Three of this session's
findings are dialogs, and the one thing it could never do to a dialog is answer it.

## Claim 6 — the signature ADR-0133 needed two Macs to read is four words of a load command

`codesign -dv` on the arm64 runner says `flags=0x20002(adhoc,linker-signed)`. The same number, read
off a cross-build on the Linux machine this session ran from, by walking `LC_CODE_SIGNATURE` to the
SuperBlob to the CodeDirectory:

    aarch64-darwin   CodeDirectory flags=0x20002   (CS_ADHOC | CS_LINKER_SIGNED)
    x86_64-darwin    no LC_CODE_SIGNATURE          (not signed at all)

This is ADR-0133's own reading rather than a second one that agrees with it, and it needs no Mac.
That matters because the decision below makes the asymmetry something two documents assert where a
person downloads, and **the asymmetry is the toolchain's rather than this repository's**:
`release.sh` passes no signing flag and holds no identity, so a Go release that began signing
`darwin/amd64`, or stopped signing `darwin/arm64`, would leave both documents describing a
publication that had changed underneath them.

`TestRelease_TheMacOSArchivesCarrySignaturesNobodyIssued` fails there now. It asserts nothing about
notarisation, which is not in the bytes: what a file can carry is a stapled ticket, and *nobody
notarised these* is a fact about a release process that has no identity in it at all.

## Claim 7 — a `go install` at a version stamps a version and no commit

Not the ticket's question, and the reason it is here: **the person this ticket is about probably has
Go.** `hyper` is a developer's tool distributed as one Go binary, and the README's second install
section is the one a Mac user is most likely to take — which turns out to be the path with no
Gatekeeper story at all. A binary the toolchain built locally was never written by an app that calls
LaunchServices, so no attribute is set, nothing propagates, and `open` has nothing to refuse.
**Quarantine is a property of the download, not of the binary.**

What that path does have is a gap nothing documented:

    $ go install -ldflags "-X …/internal/version.Version=0.0.1-alpha" \
        github.com/TheLoomLabs/hyper/cmd/hyper@v0.0.1-alpha
    $ hyper version
    hyper 0.0.1-alpha
    commit  unknown
    built   unknown

Go writes `vcs.revision`, `vcs.time` and `vcs.modified` from the repository a build's source sits in.
Module mode builds from the module cache, which holds the zip the proxy served and no `.git` at all,
so there is nothing for `-buildvcs` to read and **no flag can change it**. The same source built from
a clone stamps all three. Nothing downstream cares — the pin gate compares the first line and nothing
else, so such a binary installs, checks and projects — and what it costs is the one page ADR-0133
found `-dirty` costing: an operator asking what they are running is told the build has no commit,
which is true, and is §9's rendering of a fact the build did not stamp (#103).

`releasing.md` said *`go install` stamps like any other build when it is given `-ldflags`*, which is
right about the version and silent about the two facts under it. Corrected in place.

## What this does not establish

- **SIP is still disabled.** Both runner images ship it off, no workflow can turn it on, and it was
  ADR-0133's first named confound. It survives. What has changed is that everything *around* it is
  now measured on a real download rather than on an invented attribute, and a Mac somebody owns
  would be closing one variable rather than the whole question.
- **One macOS version, still.** 15.7.9 and 15.7.7. Nothing ran on macOS 26 or macOS 14. The
  `macos-15` arm64 label also served two different images inside one hour — `20260829.0321.1`
  (15.7.9) on the first dispatch and `20260727.0256.1` (15.7.7) on the five after it — so *the arm64
  runner* names a pool rather than a machine, and the Safari sheet was photographed on both members
  of it.
- **Nobody clicked anything, and one sheet would not take a keystroke.** The Safari download prompt
  was answered with a synthetic return and did open the download; the Gatekeeper sheet ignored a
  synthetic escape and stayed up, which is Claim 5 and the reason the repair is unmeasured. What a
  person with a mouse sees past that dialog is the one thing left on this ticket.
- **`hyper install` never ran, and no Target reached the network.** As in ADR-0133.

## The decision this records

**The macOS archives stay published, stay unsigned, and the consequence is stated where a person
downloads.** That is the second of the three the ticket named, and the other two are declined for
reasons this session is now able to give rather than guess at.

*Signing and notarising* would put an Apple Developer Program membership, a Developer ID certificate
and a long-lived signing secret into a release process that currently holds no identity at all — for
a tool whose install instruction is a shell command, whose runner platform is Linux, and whose macOS
users are being pointed at a source build anyway. It buys the Finder path, which is not the path
anything here tells anyone to take.

*Dropping the archives* is what ADR-0133 declined when all four platforms turned out to execute, and
nothing here revisits that. They run. What refuses them is a launch mechanism, not the bytes.

**What follows is one obligation: the README says what happens, in the place where somebody is about
to make it happen.** Both documents now carry the shell/Finder split, the *Not Opened* dialog by
name, and Safari's prompt with the hostname it actually names. Neither carries the `xattr -d` line,
and that is Claim 5 rather than an oversight: `releasing.md` recommended it for a mechanism nobody
had measured, and it is still nobody's measurement. **A repair a document names is a claim the
document makes**, and the honest state of this one is that the dialog is real, the shell is the way
past it, and the repair is untested.
