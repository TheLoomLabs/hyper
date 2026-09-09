# `docs/images/`

The mark, in the four cuts it is used in. Nothing else.

An image is none of the five reviewed artefacts. It carries no authority, no
`check` counts it, and nothing about a Run reads it — the same argument
[ADR-0095](../adr/0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)
made for `AGENTS.md`. It lives here rather than at the repository root for that
reason: the root is where the one artefact with no directory sits, and a file
with no authority does not belong beside it.

## This directory has held three things, and holds the third

It began as hand-written SVG: a hero drawn as linework, on the argument that a
repository whose claim is that reading the artefact is sufficient should not
hold a blob nobody can read.

**That argument was right about artefacts and wrong about that file.** What it
bought was a hero that could be diffed and a hero that looked like a diagram —
and the thing it was diagramming was `hyper`'s own `review` screen, quoted
verbatim, which made the image a claim about a rendering that went stale the
moment the rendering moved.

So the hero became a photograph: a precision optical instrument on a steel
frame, five machined apertures held in line, a beam entering the widest and
emerging from the last as the word. It cost what a photograph costs — nobody
could review it by reading it — and that was accepted, because the image
asserted nothing about the tool that could become false.

**Both are now gone, and the directory holds linework again.** What is left is a
mark rather than a picture, and that distinction is the whole of why the first
argument now holds. A hero *depicts* something, and a depiction can go stale. A
mark depicts nothing and asserts nothing, so the cost that sank the SVG hero is
not a cost it pays, while the benefit is intact: every cut is hand-authored, one
element per line, with no filter, no gradient and no embedded raster, so a `git
diff` can speak about a change to it.

**What went with them is stated rather than argued away.** The README opens on a
title now, not on an image — a reader gets no two-second impression before
deciding whether to keep scrolling, and that was the photograph's entire job.
GitHub's link-preview card falls back to whatever GitHub generates, which is not
a decision anybody here made. And the array itself is no longer pictured
anywhere: the mark is one plate of it, and the other four survive as a sentence.

## The mark

**One of the five apertures, seen along the beam.** The array is five machined
plates held in line, each aperture narrower than the one before; the mark is a
single plate head-on, where the *reading* is — the machined rim, its twenty-four
graduations, six blades closed onto a hexagonal opening, and the beam a point at
the centre. The blades are the six sides of that opening, each run outward until
it meets the rim, which is how an iris actually closes and why the opening needs
no outline of its own.

That array is `hyper`'s shape. An agent could do anything. The Manifest declares
the Capabilities it needs; the Target declaration grants a subset; the Bound caps
how far a Step may reach; the review approves what is left. **What touches the
world is only what every one of them permitted** — and each plate is an object
with graduations engraved on it, which is the other half of the claim: the
narrowing is not hidden in a service, it is a thing you can pick up and read.

The mark keeps the second half and gives up the first. A single aperture is
legible at the sizes a mark has to survive and says *a thing you can read*; it
cannot say *five of them, in this order*. The array drawn side-on was the other
candidate and it carries the sequence, which is the better argument — and it
becomes an illegible smear below about 48 px, which is where a mark spends most
of its life.

The beam is not stopped by the plates. Nothing here is about refusal for its own
sake; it is about arriving *exact*.

### Which cut

`logo.svg` is primary, tuned for a dark ground. `logo-light.svg` is the same
geometry retuned for paper — the blades darken, and the beam's core goes
*deeper* than its halo rather than brighter, because on a light ground that is
what reads as hot. `logo-mono.svg` takes its colour from wherever it is placed
and drops the halo, an opacity wash in a single ink being only a smudge.

`logo-small.svg` is the cut for **32 px and below**. The graduations are
sub-pixel there and render as grain, so they come off and everything left
thickens to hold its edge. It is the same mark with less of it, not a different
one — and reaching for it above 32 px means the mark is too small on that
surface rather than wrong.

The README pairs the first two in a `<picture>`, so the mark follows the
reader's GitHub theme.

## The files

| file | size | where it is used |
|---|---|---|
| `logo.svg` | 256×256 | the mark, on a dark ground |
| `logo-light.svg` | 256×256 | the mark, on a light ground |
| `logo-mono.svg` | 256×256 | one ink, inheriting `currentColor` |
| `logo-small.svg` | 256×256 | the reduced cut, 32 px and below |

The four cuts share one geometry: the same rim, the same six blades, the same
opening, differing only in colour and — in `logo-small.svg` — in what is left
out. **Nothing enforces that.** They were emitted together from one description,
but that description is not in the tree and there is no Go or bash in this
repository it could have been written in; the files themselves are the source,
which is why each is one element per line with the construction named in
comments. A change to the mark is four edits, and the check that they still
agree is a person looking.

## The social preview

GitHub still serves whatever card was last uploaded to it, and deleting the file
that produced one does not clear it — there is no API for the setting, so `gh`
can neither read it nor unset it. It is *Settings → General → Social preview*, by
hand, and a file put there must stay under 1 MB.

## Diagrams

This directory holds none. The structural ones are mermaid, written in the README
itself — GitHub renders them natively, and they are reviewed as text. The
argument at the top of this file still holds for those: a diagram makes a claim,
and a claim should be readable.
