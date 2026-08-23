# One repository, one Journal, and a range that opens

`repo/` is the repository the review's range is asserted against (issue #164). It
is small on purpose: six artefacts, one of each kind the range reads plus a
second Procedure that is only ever reached by invocation, so that every arm of
ADR-0067's anchor has exactly one artefact to be read off.

```
hyper.yaml                   the Repository declaration — read by every Run there is
providers/uptime.yaml        the Manifest — read by every Run naming it in `provider`
targets/staging.yaml         the Target declaration — read by every Run naming it in `target`
definitions/heartbeat.yaml   the Definition — read by every Run naming it in `definition`
procedures/watch.yaml        the top-level Procedure — read by the Run whose run.json names it
procedures/probe.yaml        the nested Procedure — read by every Run carrying it in a Step file's `path`
```

`watch` declares a Cadence and `probe` does not, which is the other split this
corpus needs: *last ran* is a member of the gloss and renders where the gloss
does, so the artefact that carries an age and the artefact that carries none are
two Procedures in one repository.

**Two of the six anchor on a revision of their own** — `watch` on
`procedure_revision` and `heartbeat` on the `definition_revision` of the Step
file that named it — and those are git blob ids over the file's own bytes, so a
case may hold them as checked-in constants:

```
5639c68a1e0a79e88a92cfd1153dd40d4febd1cf  procedures/watch.yaml
295fea3b5d37d11f4007541e1721ebcc5fd40030  definitions/heartbeat.yaml
```

Editing either file changes the id, and the case that names it fails saying so.
Run `git hash-object <file>` in this directory to get the new one.

**The other four anchor on a commit**, which no case can hold: a commit id is a
function of the tree, the message, the identity and the dates, and the fixture
builds one at run time. Those cases are driven from
[review_range_test.go](../../review_range_test.go), which materialises the
repository, reads its `HEAD`, and seeds the Journal against it — so nothing in
this directory names a commit and nothing has to be regenerated when one moves.
