# A query string in `path:` is refused where it is written

**A `path:` carrying a `?` or a `#` is `manifest-inconsistent`, decided offline from the one file.**
`path:` is written as text and `hyper` percent-encodes it, so neither character can mean there what it
means in a URL: a `?` is escaped to `%3F` inside the path and opens no query, and a `#` is escaped to
`%23` and opens no fragment, a fragment being a client-side construct no request ever carries. The row
cites the `path:` line and names `query:`, which is the key beside it that a query string belongs in.

It earns no code of its own. It is the twelfth shape of `manifest-inconsistent` decidable from the
Manifest alone, and it is the plainest instance of what those shapes share: one file, one Operation, and
**two adjacent keys**, the value in the first and its home in the second.

The cost is a limit, and §13 states it: a path *segment* holding a literal `?` or `#` is not writable
inline any more. It stays reachable through a hole, at what an input costs — every one an Operation
declares is supplied by every Step, so the constant becomes an argument.

## What was wrong (issue #229)

`internal/capability/http.go` builds the address as `url.URL{Scheme, Host, Path}`, and `URL.String()`
escapes the path component — correctly, a raw `?` terminating a path in RFC 3986. So

```yaml
http:
  method: GET
  host: "{from-target}"
  path: /v1/monitors?limit=100
```

goes out as `GET /v1/monitors%3Flimit=100`, and `hyper check` answered `checked 10 artefacts: no
problems found`.

**The fault was paid for with a call against the world, and the surface that reported it named the far
end of the mistake.** The endpoint answered `404`, so the collection path found nothing and the Step
reported `{"disposition":"ran","records":[],"projection_failed_path":"$.body.data.monitors"}` — true,
and not the defect. An author reading that row goes looking at the projection, which is the one part of
the Manifest that was right. On an effectful Operation the same slip sends a `POST` to a path that does
not exist and the Step halts on the non-`2xx` (§6): cheaper to read, and it still costs the call.

The evidence is the sealed acceptance run of 2026-08-29, the first run of `monitor-coverage`
(ADR-0105, ADR-0106) — the task that asks an agent to author a Manifest. The fixture endpoint's own log
records what actually arrived:

```
16:01:24 GET /v1/monitors%3Flimit=100 -> 404
16:01:53 GET /v1/monitors -> 200
16:02:36 GET /v1/monitors?limit=100 -> 200
```

Three calls, a diagnostic Operation the session had to author and then delete, and about four minutes
of a five-minute run. The session's own handback names it as one of two things it had to find out by
running rather than by reading.

The fixture's own API document is not changed and should not be. It writes *`?limit=` takes anything
from 1 to 100*, which is how a real API document writes a query parameter, and an author reading it
reaches for the `?` because that is the character the vendor put in front of them. What closes the gap
is the check, not a friendlier world.

This is what §4's claim is for. *A Manifest is data, so what it claims can be checked against what its
own Operations require, with nothing but the file itself* — and the offline check is what gives an agent
a correctness oracle a program does not have. The fault is decidable with no Target, no credential and
no network, from two adjacent lines.

## Why it joins `manifest-inconsistent` rather than earning a code

The twelve share one name because each points a reader at one file, one Operation, and two adjacent
keys, and none of them needs a discrimination a reader would act on differently. This one is the same
shape and the same remedy: the row already says which line to edit and which key to move the value to,
and a code of its own would add a member to §12's closed set that carries no information the message
does not.

The counter-argument is that the other twelve are a Manifest *disagreeing with itself* — two
declarations that cannot both be true — while this is one value being wrong on its own, which is closer
to `header-reserved`'s shape (a name the tool holds). It is rejected on what the author actually did: an
Operation that writes a query into `path:` while `query:` sits beside it unused has written one fact
into two places' worth of keys and chosen the wrong one. The disagreement is between the value and the
position, which is `host-input:`-names-no-input's shape exactly.

## `#`, decided with it

**The same fault, refused with the `?`, and for a nearer reason.** A fragment is never transmitted at
all — it is the one part of a URL that never reaches a server — so a `#` in `path:` is not a construct
that went into the wrong key; it is one that has no key anywhere, and it reaches the wire as `%23`
inside a path segment. Leaving it legal would mean refusing the character an author can at least move
and admitting the character an author cannot use at all.

**One row, and the `?` wins.** A path carrying both, `/v1/monitors?limit=100#current`, is one mistake:
in a URL the `?` opens the query and everything after it — the `#` included — is inside what the author
wrote as a query string. A second row would name a second fault that is not there, on the same line, and
the edit that fixes the first fixes both.

## What an author who needs a literal `?` writes

**`url.URL`'s own semantics decide it, and they are already chosen.** `Path` is a decoded path that
`String()` escapes; `RawPath` is a pre-encoded spelling of that same path, and `String()` uses it only
where it is a valid encoding of `Path` — so honouring an authored escape means setting both fields from
one authored string, not swapping one field for the other. `hyper` writes `Path` alone, which makes an
authored `path:` text and the percent-encoding `hyper`'s, and that is the same rule `query:` already
follows — `url.QueryEscape` on every name and value, an author writing the text and never the escape.
Under that rule the raw `?` *was* the spelling of a literal one, and the check takes it.

Reading `path:` as pre-encoded was the alternative and is rejected. It would make an authored `%3F`
mean a literal `?` and buy the escape hatch outright, at two prices and a complication. `path:` would
stop meaning what `query:` means one key over. And a `%` followed by two hex digits would silently stop
being those three characters: `path: /reports/50%20off` goes out as `/reports/50%2520off` today, which a
server reads as `50%20off`, and would go out as `/reports/50%20off`, which it reads as `50 off`.

The complication is the one that decides it. A value filled into a path hole at Run time would be read
as pre-encoded too, so a Step whose argument is `50%20off` would send `50 off` — what an argument means
would depend on characters inside it, in the one position §3 promises is text. That is avoidable, but
only by escaping hole values at the splice and passing the authored spans through, which means `Build`
tracking which characters of the rendered path came from the Manifest and which from an input. It is a
second path grammar inside the filler, carried so that one character can be written inline in a Manifest
where no API this format targets puts one.

So the literal is behind the wall, stated in §13's list with the rest, and the wall has a door in it: a
hole's value arrives at Run time, is read against no path grammar, and is escaped like any other text.
`path: /v1/monitors/{monitor_id}` with the input carrying `a?b` sends `/v1/monitors/a%3Fb`. What is
unwritable is the literal *inline*, not the request.

The door has a toll, and §13 states it with the entry. An input is not a constant: every input an
Operation declares is supplied by every Step that binds it (ADR-0081), and one no position reaches is
`manifest-inconsistent` in its own right — so a Provider author whose endpoint has a fixed `?` in a path
segment cannot hold it in the Manifest at all. It becomes an argument every call site writes out. That
is a real cost and it lands on the Provider author rather than on the Step's, which is the wrong end of
the model; what buys it is that no API this format targets has such a path, and that the mistake the
check decides is one an agent made on the first Manifest it ever authored.

## The fences

`TestCheckManifest_ManifestInconsistentPathCarriesAQueryString` and `…PathCarriesAFragment` hold the two
rows and what each names — `query:` on the first, *fragment* on the second.

`TestCheckManifest_APathCarryingBothDelimitersEarnsOneRow` holds the row count on a path carrying both,
which is the case an author actually writes, and holds the `?` as what the one row names.

`TestCheckManifest_APathHoleIsNotPathText` and `…APathWithAFilledHoleIsClean` are the false-positive
half. The first keeps the check reading the path rather than the names written into it: a hole's text is
a name, checked as a name on the same node by `checkOrdinaryHoles`, so reading it as path text would put
two rows on one line for one fault. The second is the ordinary Manifest, holes and `query:` and all,
checking clean — the fence a check like this is worth nothing without.

`TestCallURL_AFilledHoleCarriesADelimiterPercentEncoded` is the door in the wall, held open in the
package that would close it: a hole filled with `a?b#c` renders `/v1/monitors/a%3Fb%23c`. It is what
makes §13's entry honest, and it fails the day `Call.URL` starts honouring an authored escape.

`check/a-query-string-written-into-path` is the shape as a page, and it is the shape a real Manifest
meets it in: two sibling `read` Operations on one lookout API, identical but for where the limit went —
`list_monitors` writing `?limit=100` into `path:`, `list_incidents` writing `limit: "100"` into `query:`.
One row, exit `1`, and the spelling that works sitting four lines below the one that does not.

## Consequences

- **No `error_code` is added or removed**, and §12's closed set holds the same members. `check` gains a
  thing it fires on, not a thing it says.
- **The count moves in three places**, which is the discipline §12's opening rule asks for in a closed
  set: eleven shapes become twelve in the checker, twelve become thirteen in §4, and §13's ceiling goes
  from twenty-one victims to twenty-two.
- **`path:` is stated for the first time.** §3 named the key and said nothing about what its text means;
  it now says that the text is text, that `hyper` does the escaping, and what that costs.
- **No existing artefact moves.** Nothing in the corpus, the built-in Provider, or the acceptance
  fixtures writes either character into a `path:`, so the only golden this adds is the new case's.
- **The next `monitor-coverage` run is what says whether this worked.** The fault it cost four minutes
  to find is now a row before the first call, and the row names the key.
