# An Expansion is ordered by the name, not the path it is stored at

An `assets:` or `observations:` Expansion resolves in the order of the Record `name` the Store holds,
sorted by Unicode code point. It is not ordered by the percent-encoded path segment §12 builds from that
name to reach a file.

We chose this because the unaided reading is the wrong one and arrives by the shortest route. A series is
found by listing `records/<target>/<definition>/`, and what a directory listing hands back is encoded
segments, already sorted by the filesystem or one `sort` away from it. Ordering them is not a mistake so
much as the path of least resistance, and it is silently different: every unreserved byte is `-` or
above and every escape begins `%`, so escaping drags an escaped character to the *left* of every
unescaped one. `Über-vm` sorts after `zone-a` by name and before it by path; so do `api:8080` against
`api.example.com`, and `vm/1` against `vm.1`.

The difference is a safety fact rather than a tidiness one. §6 makes a `destroy` Expansion strictly
serial so that *which three of the five a halted Run reached* is determinate, and that is the question a
reviewer asks after a Run stopped midway. Under the path reading the answer depends on which bytes of a
name happened to need escaping — a fact about a filesystem that cannot hold a `/`, arriving in the one
place the tool promises two environments reach the same first three.

## Considered options

- **Order by the encoded path segment.** The reading a directory listing produces. Rejected on the
  argument above, and on a second: §7 already sorts an identity set by Unicode code point over these
  same names, so the path reading gives `hyper` two orderings over one thing, differing only where a name
  is awkward.
- **Order by the encoded segment on the ground that it is what an implementation holds anyway.** Rejected
  because it inverts which of the two is derived. The name is the identity, declared by a Manifest and
  projected from a response; the encoding exists so that identity can be a filename, and §12 says as much
  where it states it.
- **Leave the order unstated for `assets:` and `observations:` and rely on the serial rule alone.**
  Rejected because serial execution without a defined order is a determinate count and an indeterminate
  membership: *three of five* with no answer to *which three*, which is the half a reviewer needs.
- **A locale-aware or case-folded collation.** Rejected as a third ordering with an environment in it,
  which is the axis §5 removed. Code point is total over these names already, case-collisions being
  refused by the Store before a second series can exist.

## Consequences

- **The order a reviewer predicts is the order that runs, on both selector forms.** Where the artefact
  states an order — an `over:` `values:` list — the page is the order, and that is the same principle
  arriving from the other side rather than an exception to this one: a percent-escape is in front of
  nobody. §6 states both rules together.
- **`hyper` has one ordering over Record names.** §6's Expansion and §7's identity set sort identically,
  so the two agree for every Store-ranging selector and differ only where a `values:` list is authored
  out of code-point order — which is exactly where the sequence and the set are meant to be different
  things.
- **An implementation may not order a directory listing.** It decodes, or it carries the name beside the
  path. This is the one place the Store's path grammar is not a valid stand-in for the identity, and §12
  says so where the encoding is defined.
