# A `method:` that is not an HTTP token is refused where it is written

An `http:` block's `method:` is one HTTP `token` — RFC 9110's `1*tchar` — and a literal that is not one
is `manifest-inconsistent` at `<op>.http.method`. The grammar is the line and the registry is not: a
verb no registry names, and a verb written in lower case, both check clean. Issue
[#285](https://github.com/TheLoomLabs/hyper/issues/285).

Nothing moves for an artefact that checks clean: no artefact key, no closed-set member, no new
`error_code`, no wire member, no rendering, and no Run-time behaviour anywhere. **One `check` row
appears for an artefact that does not.**

## What the state was

[ADR-0155](0155-a-requests-method-is-a-literal.md) closed the hole in `method:` and named this as a
stated limit rather than fixing it: *a `method:` that is a literal but not an HTTP token is left where
it is.* That was the right split — a value arriving from outside the artefact and a malformed literal
are different faults — and this record decides the second.

`method: "GET "` checked clean, and both halves of what a Run then did were wrong, each in its own way.
Both were driven against the binary through `cli.Main` and the golden harness's own dialler.

**On a `read` the Run completes, exit 0, and writes a Record.** `Call.request` fails at
`http.NewRequestWithContext` — a space is not `tchar`, so the token never reaches the wire — and
`Perform` answers the object it answers for a host that never replied: `host` and nothing else
(`internal/capability/http.go`). That object's doc comment is exact about what it means, and it is
correct for the case it was written for: *where no response arrived at all it is host and nothing else,
which is the answer a `read` records rather than a failure it halts on* (§6, ADR-0050). This is not
that case. The Record written for `method: "GET "` was byte for byte the Record written for a host that
was down — `status:` absent, `days_left:` absent — and the Run reported success over it. **A Manifest
fault and a silent host were the same Record.**

**On a `mutate` the disposition was right and the reason was wrong.** The Run Refused
`attempted-world-untouched`, which is true: nothing left. The sentence beside it — *no response arrived
from `api.cloudflare.com`, so the request never left and the world is untouched* — pointed the operator
at a host that was standing and answering, for a fault on line 14 of the Manifest. The Journal entry
carried `"answered": [{"host": "api.cloudflare.com"}]` and no `error_code`. The disposition was reached
from the response object one layer up rather than from `NeverSent`, which is false here: only the
marking dialler marks, and a request that failed before the dialler ran was never marked
(`internal/capability/sent.go`).

**The corpus had not decided this; it inherited it.** Every fixture in the tree writes a well-formed
method, which is the same reason #279's half went uncaught. §3 said `method:` is a literal verb and said
nothing about which literals; §12's positions decide where a value may *come from* and have nothing to
say about one written in place.

## The decision

**A value knowably wrong offline is refused offline, at the key it is written in.** That is
`checkPathDelimiters`' argument one key over (ADR-0107, [#229](https://github.com/TheLoomLabs/hyper/issues/229)),
and it transfers whole: a `?` in `path:` and a space in `method:` are both text that `hyper` will carry
to a place that will not accept it, and in both the file, the Operation and the key are known before
anything runs.

It is `manifest-inconsistent` rather than a code of its own on the same shape argument the other twelve
carry: one file, one Operation, one key, and an edit the row can name. A code of its own would tell a
reader these refusals differ in kind when they differ only in which key.

**The token grammar is the line, and a known verb is not.** HTTP's method space is extensible, and a
`check` reading IANA's registry would refuse a correct Manifest written against an API that ships its
own verb — `PURGE`, `MKCALENDAR`, an API's own word. `token` is what `net/http` will accept, which is
the fault actually being reported; anything narrower would be `check` having an opinion about the API
rather than about the file. **Case is not read either.** `get` is a token, is a different method from
`GET` to a server that compares them, and is the author's to write.

**A hole is not token text.** `{` and `}` are not `tchar`, so a `method: "{verb}"` read as a literal
would draw a second row on the line `checkMethodHole` has already cited for one fault — and eliding the
hole first, which is what `checkPathDelimiters` does, would leave the empty string, which is not a token
either. So a value carrying a hole is left to the check that refuses the position. `check`'s answer for
that Manifest is the two rows ADR-0155 decided and not three.

ADR-0155 is what makes this cheap. `method:` is now a literal with no hole to elide, so there is nothing
to work around and no second reading to keep in step.

## What was considered

- **Fence the Go error into a Refusal with an `error_code` at Run time.** Rejected for the reason
  ADR-0155 rejected it for its own half: it names the symptom. The Manifest is knowably wrong offline,
  and a Run that Refuses where `check` was clean is the arrangement §4 and ADR-0061 both exist to avoid.
  It would also have to be written twice — once for the `read` that records and once for the `mutate`
  that Refuses — where the check is written once.
- **Repair `Perform` instead, so a request that never became a request is distinguishable from a host
  that never answered.** Rejected as the wrong layer for this fault, though it is a real distinction:
  the `read` recording a silent host is correct behaviour for a silent host, and the defect is that a
  Manifest fault reached it. With `check` refusing the Manifest, no artefact in the tree can produce
  that arrival. What remains is the *wording* the `mutate` Refusal uses, which is `Perform`'s to fix if
  a second way of reaching it is ever found; nothing found one.
- **Check the verb against IANA's method registry.** Rejected above. It refuses correct Manifests.
- **Require upper case.** Rejected with it, and for a sharper reason: HTTP method names are
  case-sensitive, so `get` and `GET` are two methods and a `check` folding them would be asserting
  something about the server.
- **Give it a code of its own —** `method-malformed`. Rejected on the shape argument above.

## Consequences

- **One `check` row on a Manifest that used to pass**, at `<op>.http.method`, quoting the literal — a
  trailing space is invisible unquoted, and it is the fault an author is most likely to have written.
  `internal/cli/testdata/check/a-method-that-is-not-a-token` is the golden: one Provider, the malformed
  `read`, and a `PURGE` `mutate` beside it that draws nothing, which is the registry fence stated as a
  fixture rather than as a sentence.
- **`manifest-inconsistent` counts thirteen decidable-from-one-Manifest shapes** where it counted
  twelve. The closed set of `error_code`s is untouched, so §12's set and the coverage test do not move.
- **An empty `method:` is refused by this check** rather than by the schema, which types the key
  `string` and says nothing about length. `token` is `1*tchar`, so the empty string fails the same
  reading, in the same row, with no second rule.
- **No sealed run is owed.** The change is enforced rather than taught in
  [`docs/agents/acceptance-re-runs.md`](../agents/acceptance-re-runs.md)'s sense: `check` now Refuses,
  and the golden and the package cases hold it. No clause of the orientation moves, no existing row is
  reworded, and no task in `scripts/acceptance/tasks/` writes a malformed method — the row is new text,
  and nothing an agent already reads has changed under it.
