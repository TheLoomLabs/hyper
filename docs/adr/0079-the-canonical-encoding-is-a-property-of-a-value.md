# The canonical encoding is a property of a value, not of a file

§7's canonical encoding is defined over a JSON **value**, and a file is the case where the value is the
whole file. A value encoded on its own is encoded exactly as it would be were it that file's whole
content. An identity digest is therefore `sha256:` over the sorted array written alone — opening at no
indent, one member two spaces in per line, trailing LF — and the empty set is `[]` under the same
section's inline rule.

## The problem this decides

§7 states the encoding as a property of files: *every JSON file in the Store is written in one
encoding*. It then takes a digest over something that is not one. The rules it states — two-space
indent, one array element per line at the same indent, a comma immediately after a value then the line
ending, an empty array inline — were written for a value nested inside a file, where the indent of the
line holding the opening bracket fixes the rest. An array standing alone has no such line.

At least three readings satisfy the prose, and they are three digests of one set:

```
[\n  "a",\n  "b"\n]\n      indented, one per line
[\n"a",\n"b"\n]\n          one per line, level with the brackets
["a","b"]\n                the array as the inline rule writes an empty one
```

The ambiguity is normative rather than an implementation detail because of the sentence beside it: *a
reader recomputes its digest with `sha256sum` over those exact bytes and nothing else*. Under any other
sentence the encoder's habit would be the answer.

**It is the one thing in the corpus that cannot be corrected later.** No file in the Store is ever
rewritten (ADR-0011), in-place migration is impossible (ADR-0028), and the walk that reads a set back
terminates at the Run where that Step first carried one (ADR-0055). A reading adopted after the first
Run is the reading forever, and changing it makes every entry written before the change **unverifiable**
rather than merely stale — which is the failure a schema integer exists to prevent and is exactly the
failure it cannot prevent here, the bytes being the thing hashed.

## The decision

**The encoding is a function of a value.** A file is one case of a value, not the subject of the rules.
An array alone opens at no indent and writes its elements two spaces in; a nested mapping is a mapping
like any other and is never compacted onto one line.

Reading it this way adds no convention. It is what §7's existing rules already produce for `members`
inside a Step file — the bracket at four spaces, the members at six — read at a bracket sitting at zero.
The other two readings each need a sentence that exists nowhere and applies nowhere else.

**The rule is bounded to what the Store holds and what `hyper` hashes.** §8's row stream is a second
encoding — compact, keys in the renderer's order — and is stated there. A rule quantified over every
JSON `hyper` emits would be false of a neighbouring chapter on the day it was written, which is the
fault ADR-0078 found in ADR-0023.

**Every hexadecimal digit `hyper` writes is lowercase**, the one exception being a percent-escape in the
Store path grammar, uppercase because RFC 3986 says so.

## Considered options

- **The inline form, `["a","b"]`.** This is what an implementer reaches unaided: Go's `json.Marshal` on
  a `[]string` returns exactly it, with no trailing LF. Rejected because it contradicts the array rule
  the same section states for every other array in the Store, and adopting it would put two array
  encodings in one chapter distinguished only by whether anyone happens to hash the array.
- **Elements level with the brackets.** Defensible on *one element per line at the same indent* read as
  *the same indent as the array*. Rejected: nothing else in the Store writes an array that way, so it is
  a form that exists only where it cannot be seen — inside a digest and never in a file.
- **Not JSON at all: LF-terminated bare names**, which a reader digests with `sort | sha256sum` and no
  quoting to get wrong. The most attractive of the rejected options, and it fails on hostile input. A
  `name` is a Manifest-declared field of an upstream response (§7), so a name containing an LF makes the
  form ambiguous where JSON quoting makes it exact; and the empty set becomes an empty file, whose digest
  is indistinguishable from a read that returned nothing.
- **The bytes as they sit in the entry** — hash the `members` block at the indent it occupies. Rejected
  twice over: the digest is computed on **every** Run, including the ones where the set did not move and
  no `members` block exists to point at, and it would make a set's identity a function of its nesting
  depth.
- **Stating the bytes at the digest and nowhere else.** The minimal fix, and it leaves the same trap set
  for the next value anybody hashes. §7's encoding section is the one place a reader looks for what a
  value's bytes are, and it currently answers only for files.

## Consequences

- **The hashed bytes are not the bytes in the entry.** `members` sits at four spaces with its elements
  at six; the digest is over the same array at zero. §7 says so in the sentence that makes the
  `sha256sum` promise, because a promise a careful reader executes wrongly on the first attempt is worse
  than no promise.
- **§7 carries the bytes of one small set and its digest**, with a non-ASCII member in it. That member
  pins the two halves prose cannot: an implementation escaping to `\u00fc` produces a plausible digest
  and no signal, and code point order puts the member last, which `LC_ALL=C sort` reproduces exactly.
- **The empty set's digest is a constant and is printed.** `[]` under the inline rule, so
  `sha256:37517e5f3dc66819f61f5a7bb8ace1921282415f10551d2defa5c3eb0985b570`. Printing it discloses
  nothing: a digest that moved to the empty set writes `members` in full beside it (§7).
- **§7's own worked Step file was not in the encoding §7 states**, writing a selector's predicates as
  inline mappings, and is re-rendered — as are the same two lines in the worked scenario, whose digests
  were computed under this reading and stay re-checkable. A projected value that is an object (§12) is
  written the same way, and that is the form *the bytes moved* runs over.
- **§12's path grammar gains one word.** Its truncation suffix said *the first 16 hexadecimal digits*
  with no case, one clause after stating uppercase for a percent-escape. It is path bytes: two writers
  disagreeing give one identity two paths and split a series silently, in a Store where no file is ever
  rewritten. The word is stated locally rather than by cross-reference, a reader implementing a grammar
  being owed it on the page they are reading.
- **Nothing a file holds changed, so no schema version moves and no closed set moves.** An encoding was
  stated, not altered, and every digest already written in the corpus is unaffected.
