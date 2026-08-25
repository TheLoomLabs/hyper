# The worked example, byte for byte

`retire-preview-envs.yml` is §10's *The generated workflow* — the fenced block
of `docs/spec/11-cadence-and-projection.md`, copied out with nothing added and
nothing dropped. It is the independent source of truth `Generate` is held to:
a generated workflow is verified by regeneration (§10's check), so a byte that
differs is a `projection-stale` on every repository at once, and the only
comparand that can catch a drift in the generator is the section the generator
was written from.

Its own inputs are the ones the section states around it — the Procedure
`retire-preview-envs`, the Cadence `0 3 * * 1`, a Procedure that effects, the
one credential slot variable `STAGING_TOKEN`, and the version and digest the
example pins. They are written out in `workflow_test.go` rather than here,
where they would be a second file to keep in step.
