# A Provider that declares secret output

One Provider, one Target declaration, one repository declaration — the smallest
repository whose Manifest names a `secret:` field, and the sample response that
carries a value in it.

It is a corpus of its own rather than a `secret:` added to `five-artefact-demo`
or a `samples/` added to `run/repo-secret`, and both for the same reason: a
Manifest digest and a repository revision are Provenance, so a file added to a
repository moves the goldens of every case that runs against it.

It is read by `probe/`, and by the four surfaces that render an Operation:
`provider/`, `operation/`, `mcp/operation/` and `review/`, each holding what
ADR-0152 put on them. One repository for the four is the whole point of them
sharing it — the summary clause, the derived block's `secret_fields`, the
gutter's fourth field and the `SECRET` flag are one declaration rendered four
ways, and a copy per case is a fixture edit that reaches one golden and not
another.

`providers/session.yaml` is `run/repo-secret`'s, copied rather than shared:
three projected fields with one of them declared secret is the shape §3's
worked example has, and a Probe against it renders the two that are not.
