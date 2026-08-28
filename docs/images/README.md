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
like anything else here**: no raster, no external font, no script, no generated
output checked in. `git diff` over it says what moved.

`hero-light.svg` and `hero-dark.svg` differ **only** in five colour values.
Every coordinate is identical, so an edit made to one and not the other shows up
as a diff rather than as a hero that is subtly wrong in one theme. `README.md`
selects between them with `<picture>` and `prefers-color-scheme`; the `<img>`
fallback is the light file, which is what a renderer that understands neither
will show.

This directory holds no diagrams. The structural ones are mermaid, written in
the README itself — GitHub renders them natively, and they are reviewed as text.
