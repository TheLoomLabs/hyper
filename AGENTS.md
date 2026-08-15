## Agent skills

### Issue tracker

Issues live as GitHub Issues in `TheLoomLabs/hyper`, managed via the `gh` CLI. See `docs/agents/issue-tracker.md`.

### Triage labels

Default canonical labels (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context layout — root `CONTEXT.md` + `docs/adr/`. See `docs/agents/domain.md`.

## The spec, and building from it

`docs/spec/` is the specification — what `hyper` does, in fourteen sections.
`docs/adr/` is why. Together with `CONTEXT.md` they are about 270k tokens, so **no
session reads them whole**.

Building from the spec goes one milestone at a time. **Read
`docs/build/milestones.md` first** — it names the eleven strata, what each one
reads, what each one delivers, and the session-per-phase protocol. Do not run
`/to-spec` or `/to-tickets` against `docs/spec/` in one pass; it does not fit, and
the spec's sections are layers where a ticket is a slice through all of them.
