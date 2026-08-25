# A ref is a location, and `hyper` names no registry

`install <ref>` takes an **absolute `https://` URL naming the Manifest** — `hyper install
https://providers.example.com/acme/dns.yaml`. There is no registry host in the binary, no namespace
`hyper` owns, and no short name to expand against one: the ref *is* the location. The digest is
published beside the bytes as `checksums.txt` in the ref's own directory, read with `sha256sum`'s
own line grammar; and what `install` writes into the `origin:` block is the ref the caller typed,
across a redirect and regardless of where the bytes came from.

## The gap, and the reading it invites

§11 fixes `install`'s three exit codes, says a ref *names something in a registry's namespace*, and
never says what a ref looks like. §13 says what a registry is beyond *a place bytes and checksums
are published* is not `hyper`'s concern. Between them is a gap an implementer must close before
writing a line of the command, and closing it in a package comment would put the load-bearing half
of distribution somewhere no reader of `docs/adr/` would find it.

The reading a competent implementer reaches unaided is **a name in a namespace** — `acme/dns@1.2.0`,
expanded through a URL template compiled into the binary beside the projection's four constants.
That is arrived at by reasoning from *resolves the ref against a registry* rather than by failing to
read it, which is what makes it worth refusing in writing. Every package manager in existence works
that way, and §11's own sentence about a namespace is what points at it.

## Why the compiled-in host is refused

**It makes `hyper` name a registry, which is the product §13 refuses in as many words.** A name in
a namespace means something only where somebody decides who is `acme` — which is an account, a
publishing command and a promise about availability, the three things *No registry as a product*
enumerates and declines (§11, §13). A host compiled in is not a smaller version of that product; it
is the whole of it, deferred, with `hyper` holding the one name that matters.

**There is nowhere else such a host could live.**
[ADR-0014](0014-hyper-has-no-configuration-files.md) admits no configuration file of any kind, and
the layers that survive it govern presentation only — a registry host decides which bytes enter the
repository, which is the far end of that line.
[ADR-0020](0020-the-hyper-version-is-pinned-by-the-repository.md) admits into the Repository
declaration only facts that govern every Run and belong to no Procedure, Definition, or Target
declaration. So the binary is the only place left, which is the argument
[ADR-0046](0046-the-projections-executor-is-compiled-in-never-authored.md) already ran for the
projection — and it lands differently here. §11 closes that set at **four**, and says so twice: a
script the projection writes is not a fifth constant, and the four are *the complete list of what a
`hyper` release can change in a repository that edited nothing*. A registry host would be a fifth,
and it would be the first one that is a fact about a third party's namespace rather than about the
job the binary generates.

**And the ref is already a location.** The alternative is not *a short name versus a URL*; it is
*one string that says where the bytes are* versus *one string plus a compiled-in rule for turning it
into one*. Nothing in the second is bought except a shorter ref.

## The grammar

Every clause is a parse. Nothing here reaches the network, and that is the property the rule exists
to keep: exit `2` says *you typed it wrong*, and
[ADR-0060](0060-naming-nothing-is-a-usage-error-fetching-nothing-is-not.md) keeps it decidable
without a round trip. A ref is:

- **An absolute URL, scheme `https`, and there is no second one**
  ([ADR-0082](0082-the-scheme-is-https-and-there-is-no-second-one.md)). A bare path, a relative
  reference and `http://` are all outside it, and the transport holds no plaintext dialer to reach
  for anyway.
- **A non-empty host, no userinfo, and a port that is a port.** A port is admitted and carries no
  meaning of its own, a registry being wherever it is served; one that is not a decimal number is
  outside the grammar rather than something to resolve. `https://user:token@host/…` is refused
  because the ref is written into a tracked file and reviewed in a diff, and a credential in it is
  the one place this tool would write a secret down
  ([ADR-0007](0007-hyper-never-stores-a-secret.md)).
- **A path ending in `.yaml`, whose last segment is a legal `providers/` filename** — no `.`, no
  `..`, no path separator, and no percent escape that decodes to one. Two reasons, and they meet:
  the loader reads `providers/*.yaml` and nothing else, so a ref ending otherwise names bytes that
  would land in the tree and never be read; and `install` is the one command that writes a path
  derived from a string a caller typed, so a segment that is not a filename is a traversal.
- **No query and no fragment.** A ref carrying either is one two callers can type differently for
  one set of bytes — and this ref is recorded, rendered by `provider <name>` (§9, §11), and is what
  a later `install` is typed from. A fragment names a position inside a document where a Manifest is
  a whole file, and a query string is where a signed URL would put a token into a tracked file.

Anything outside it is §9's usage error: exit `2`, no `error_code`, no row stream, the rendering to
stderr — the shape the other eight positionals already take (ADR-0060). Where the grammar ends is
where the network begins, and everything the network answers is `1` or `77`.

**Which of the three a given failure earns is not fixed here.** ADR-0060 decided that, and nothing
in this ADR moves it: this one is about what a ref *is*, and that one is about what naming nothing
costs.

## The digest is published beside the bytes

`install` performs two reads and both are `GET`: the ref, and **`checksums.txt` in the ref's own
directory** — the ref with its last path segment replaced. **The Manifest is read first**, and the
order is not arbitrary: it is the read that establishes the registry answers at all, and a ref that
names nothing is the common case, where reading the checksums first spends a request proving a file
exists beside a file that does not. The digest is the line naming the ref's basename, in whichever
of `sha256sum`'s two spellings the publisher used: two spaces for a text read, ` *` for a binary
one.

**That is `internal/release`'s line grammar, and it is the same grammar because it is the same
fact** — a checksum published beside a file, read by a tool that is about to trust the bytes.
`project` already reads one for the release artefact (§11, ADR-0020). One reader, exported once,
rather than two parses of one format that can drift; a digest missed for a space would read as a
registry that published no checksum for a file it published perfectly well.

Stating the convention is also what makes §13's *a place bytes and checksums are published*
something a publisher can satisfy without `hyper` naming them. The whole registry contract is two
files in a directory over TLS.

## What a redirect does, and what is recorded across one

**Redirects are followed**, for `internal/release`'s reason with one word changed: there is no grant
here and no authored host — the URL is compiled in there and typed by the caller here, and neither
is a host any artefact granted — and a fetch that refused one would resolve nothing on any host that
answers a download with a redirect to the store the bytes are actually in. This is where it parts
company with a Capability's call, where a redirect is reach arriving from data and the grant was
checked against one host
([ADR-0029](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md)).

**A redirect to `http://` is refused rather than followed.** The scheme rule is not escapable by the
one mechanism that rewrites a URL after the grammar has passed, and the refusal is structural rather
than a check: the transport is given no plaintext dialer and says so (ADR-0082). It is a fetch that
did not complete and exits `1` — the grammar is what a parse decides and this happened on the wire,
so it sits beside the timeout rather than beside the typo (ADR-0060).

**The recorded ref is what was typed.** What lands in the `origin:` block is the coordinate the
caller wrote, never the destination their infrastructure happened to answer with. A recorded ref
naming a redirect target is a coordinate the publisher never published — it can stop resolving the
day that redirect is retired, while the ref that survives it keeps working — and it would put a fact
about somebody's CDN into a reviewed file. The checksums read is derived from the typed ref for the
same reason. What makes this safe rather than trusting is that bytes are trusted for matching a
published digest and never for the host that answered (ADR-0004).

## Considered options

- **A name in a namespace, resolved through a compiled-in registry host** — `acme/dns@1.2.0`, a
  fifth constant beside the projection's four. The reading reached unaided, and the reason this ADR
  exists. Rejected on the three grounds above, and on one they do not cover: it makes a Provider's
  availability something `hyper` implicitly promises, which §13 declines in as many words.
- **A registry host in the Repository declaration.** Rejected: ADR-0020 admits facts that govern
  every Run, and no Run resolves anything from a registry — `install` is not a Run and reaches no
  Store. It also makes the recorded ref a fragment: a reviewer reading a Manifest's `origin:` block
  could not say where the bytes came from without opening a second file, and a repository that
  edited that file would silently change what an old ref means.
- **A `--registry` flag or a `HYPER_REGISTRY` variable.** Rejected: ADR-0014's surviving layers
  govern presentation and cannot change what reaches the world, and this changes exactly that. It
  would make which bytes an `install` fetched a fact about a shell history rather than about a
  tracked file, and put a second authority axis one command outside the safety model that deleted
  it.
- **The digest inside the ref** — `…/dns.yaml@sha256:…`, or a `#digest=` fragment. Attractive: it
  deletes the second read and the convention with it. Rejected because it moves the digest's source
  from the publisher to whoever pasted the string, which is a weaker claim than the one §11 makes;
  and because the ref would stop being a location — a later re-install would name bytes rather than
  a coordinate, and an ordinary republication would read as a mismatch rather than as something to
  install. The `origin:` block already carries `ref:` and `digest:` as two members (§3), and this
  would be one of them spelled twice.
- **A signature, an attestation, or a registry-side scan instead of a checksums file.** Rejected: it
  is *digest only, never intent* at the wrong end (ADR-0004), and a signature needs a key
  distribution `hyper` would have to name — which is naming a registry, one aisle over.
- **A directory listing, a search, or an index a ref is resolved against.** Rejected: it is the
  discovery surface §13 refuses, and it requires a registry to be something more than a file host.
- **Say nothing and let the implementation choose.** The status quo. It leaves §11's *namespace*
  sentence unresolvable from the corpus and makes the first Provider published against a different
  convention a bug report rather than a wall entry.

## Consequences

- **There is no discovery surface and there never will be one.** Finding a Provider, and living with
  whoever hosts it, are the user's — which §13 already says and this makes concrete rather than
  aspirational: what a user has to be given is a link.
- **A publisher who moves their bytes breaks every ref that named them.** That is the bargain a URL
  has always made, stated rather than glossed. What it does not break is anything that runs: the
  Extension is a tracked file in the repository, a Run resolves nothing from a registry, and a dead
  ref costs a re-install rather than a Run (§11).
- **Any static file host is a registry.** Object storage, a Pages site, a tag's release assets — two
  files in a directory over TLS and nothing else. Nothing has to be built for `hyper` to install
  from, which is what *no registry as a product* buys rather than what it costs.
- **`install`'s three codes stand unchanged, and what this fixes is the boundary of the first**
  (§11, ADR-0060). The grammar is what exit `2` covers, and it covers nothing a round trip decides.
- **The fetch is `internal/release`'s sibling and shares its reader.** The `sha256sum` line grammar
  is exported rather than copied, one format having one parse. What else the two reads share is the
  fetching package's to settle.
- **No closed set moves.** No `error_code` is minted, a ref outside the grammar carrying none
  (ADR-0060); the projection's constants stay at four (§11, ADR-0046); and the `origin:` block keeps
  its two members (§3).
- **`CONTEXT.md` gains no term.** An Extension is *a Provider authored and distributed by someone
  other than `hyper` itself*, however it arrived, and a ref is a string typed at a command rather
  than a domain object — which is the whole content of this decision, seen from the glossary.
