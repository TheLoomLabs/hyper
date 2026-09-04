# A Provider that declares secret output

One Provider, one Target declaration, one repository declaration — the smallest
repository whose Manifest names a `secret:` field, and the sample response that
carries a value in it.

It is a corpus of its own rather than a `secret:` added to `five-artefact-demo`
or a `samples/` added to `run/repo-secret`, and both for the same reason: a
Manifest digest and a repository revision are Provenance, so a file added to a
repository moves the goldens of every case that runs against it. This one is
read by `probe/` and by nothing else.

`providers/session.yaml` is `run/repo-secret`'s, copied rather than shared:
three projected fields with one of them declared secret is the shape §3's
worked example has, and a Probe against it renders the two that are not.
