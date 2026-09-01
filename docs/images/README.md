# `docs/images/`

The README's hero, and the card GitHub shows in a link preview.

An image is none of the five reviewed artefacts. It carries no authority, no
`check` counts it, and nothing about a Run reads it — the same argument
[ADR-0095](../adr/0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)
made for `AGENTS.md`. It lives here rather than at the repository root for that
reason: the root is where the one artefact with no directory sits, and a file
with no authority does not belong beside it.

## This directory used to hold no raster, and now it holds nothing else

The earlier hero was hand-written SVG, and this file argued for that at length:
a repository whose claim is that reading the artefact is sufficient should not
hold a blob nobody can read, so the hero was linework `git diff` could speak
about.

**That argument was right about artefacts and wrong about this file.** What it
bought was a hero that could be diffed and a hero that looked like a diagram —
and the thing it was diagramming was `hyper`'s own `review` screen, quoted
verbatim, which made the image a claim about a rendering that went stale the
moment the rendering moved. The SVG discipline was being applied to the one file
in the repository that is not an argument and is not read: it is what somebody
sees for two seconds before deciding whether to keep scrolling.

So the hero is now a photograph, and the honest consequence is that **nobody can
review it by reading it.** That is a real cost and it is accepted here rather
than argued away. It is bounded: the image asserts nothing about the tool that
could become false, so a reader is never misled by a hero that drifted — the
worst it can do is be a picture somebody dislikes.

## What it depicts

A precision optical instrument on a steel frame: **five machined apertures held
in line, each narrower than the one before.** A single beam enters the widest and
passes through all five; what emerges from the last is the word.

That is `hyper`'s shape. An agent could do anything. The Manifest declares the
Capabilities it needs; the Target declaration grants a subset; the Bound caps how
far a Step may reach; the review approves what is left. **What touches the world
is only what every one of them permitted** — and each plate is an object with
graduations engraved on it, which is the other half of the claim: the narrowing
is not hidden in a service, it is a thing you can pick up and read.

The beam is not stopped by the plates. Nothing here is about refusal for its own
sake; it is about arriving *exact*.

## The files

| file | size | where it is used |
|---|---|---|
| `hero.jpg` | 1280×490 | the README's first element |
| `social-preview.jpg` | 1280×640 | GitHub's link-preview card |

Both are crops of one master, and **the master is not in this repository** — it
is a generated image kept outside the tree, so re-cropping needs it rather than
being derivable from what is here. That is the second cost of the format change,
stated so nobody goes looking.

JPEG rather than PNG: the content is photographic, and at quality 92 with no
chroma subsampling it is visually identical to the PNG at a sixth of the bytes.
The social card must stay under GitHub's 1 MB limit, which the PNG very nearly
was not.

## Uploading the social preview

There is no API for it — the REST API exposes no social preview, so `gh` cannot
set it. It is *Settings → General → Social preview → Edit*, by hand, and the file
must stay under 1 MB.

## Diagrams

This directory holds none. The structural ones are mermaid, written in the README
itself — GitHub renders them natively, and they are reviewed as text. The
argument at the top of this file still holds for those: a diagram makes a claim,
and a claim should be readable.
