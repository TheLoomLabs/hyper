# One Cadence on the hour, and one past it

`repo/` is the repository §10's two facts are goldened against (issue #175).
Two facts render beside every gloss — that scheduled runs happen on the default
branch only, and, where the minute field selects `0`, that `:00` is the
executor's busiest minute — and the second of them is the one a page can be
wrong about in either direction.

Every other Procedure in the corpora declares a Cadence whose minute field is
`0`, `*` or a step over the whole span, so every one of them lands on the hour
and no checked-in page shows the hour-boundary fact **not** rendering. This
repository is the pair that does:

```
procedures/on-the-hour.yaml     cadence: "0 3 * * 1"   both facts render
procedures/past-the-hour.yaml   cadence: "30 4 * * *"  the default-branch fact alone
```

The two artefacts are identical but for that line, which is what makes the two
pages a difference of one fact rather than a difference of two repositories.
The rest of the repository is the least that carries a Procedure with a Step:
one Manifest, one Target declaration, one Definition, and a Repository
declaration whose digest is the placeholder every offline corpus writes.

Neither fact is a problem with the artefact. Neither page carries an
`error_code`, both exit 0, and neither fact reaches the `artefact` row — §9
closes it at the gloss's three parts, and a consumer derives both from the
`cadence` and the `phrase` it already has (§8, §10). The `--json` twin of the
first case is what holds that last claim: it is the page carrying **both**
facts, and its row carries neither.
