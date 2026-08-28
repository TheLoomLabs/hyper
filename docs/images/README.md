# `docs/images/`

The README's hero, and nothing else.

An image is none of the five reviewed artefacts. It carries no authority, no
`check` counts it, and nothing about a Run reads it — the same argument
[ADR-0095](../adr/0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)
made for `AGENTS.md`. It lives here rather than at the repository root for that
reason: the root is where the one artefact with no directory sits, and a file
with no authority does not belong beside it.

A repository whose claim is that reading the artefact is sufficient should not
hold a blob nobody can read. So the hero is **SVG, hand-written, and reviewed
like anything else here**: no raster, no external font, no script. `git diff`
over it says what moved.

`social-preview.png` is the one raster, and it is here under protest rather than
by preference — GitHub's *Settings → General → Social preview* accepts PNG, JPG
and GIF and **no SVG**, so the card that represents this repository in every
link preview cannot be the format the rest of this directory argues for. The
compromise is that the PNG is never authored: `social-preview.svg` sits beside
it, is what anyone edits, and the PNG is regenerated from it. Review the SVG;
the PNG is output.

## What it depicts

[§0](../spec/01-what-hyper-is.md)'s worked example, abridged: an agent widened a
`destroy` Step's Bound from 3 to 5, and the review renders `DESTROY`, `WIDENED`
and `ENVELOPE` beside the lines that made the claim. **It is a claim, like any
diagram, and nothing checks it** — the wording is §0's verbatim, so if §0's
example changes this file is stale and should be regenerated with it.

The three pills — *no plan*, *no daemon*, *no telemetry* — are
[§13](../spec/14-non-goals-and-honest-limits.md)'s, and are three of the six the
README's own non-goals section states in full.

## Regenerating the social preview

`social-preview.svg` is 1280x640, the 2:1 ratio GitHub crops that card to, and
it is dark-only: a social card is a fixed image and cannot answer a
`prefers-color-scheme` query the way the hero does. Any headless renderer will
do; this is the one used:

```bash
google-chrome --headless --screenshot=docs/images/social-preview.png \
  --window-size=1280,640 --force-device-scale-factor=1 \
  file://$PWD/docs/images/social-preview.svg
```

Uploading it is a manual step and has to be — the REST API exposes no social
preview, so `gh` cannot set it. It is *Settings → General → Social preview →
Edit*, and the file must stay under 1 MB.

## The two hero files

`hero-light.svg` and `hero-dark.svg` are generated from one template and
**differ only in their palette**. Every coordinate is identical, which is
mechanically checkable:

```bash
diff <(sed -E 's/#[0-9a-f]{6}|0\.[0-9]+//g' hero-light.svg) \
     <(sed -E 's/#[0-9a-f]{6}|0\.[0-9]+//g' hero-dark.svg)
```

An edit made to one and not the other shows up there rather than as a hero that
is subtly wrong in one theme. `README.md` selects between them with `<picture>`
and `prefers-color-scheme`, which GitHub wraps in its own `<themed-picture>`;
the `<img>` fallback is the light file, which is what a renderer understanding
neither will show.

This directory holds no diagrams. The structural ones are mermaid, written in
the README itself — GitHub renders them natively, and they are reviewed as text.
