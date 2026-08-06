# Extensions are data, not code

A Provider is a Manifest and nothing else. There is no extension programming language, no plugin
binary, no sandboxed module — an extension is a declaration of an API surface, and the only code that
ever runs is `hyper`'s own. Every effect an Operation has on the world is performed by `hyper` on the
Manifest's behalf: it makes the request, it holds the credential, it applies the pagination and the
retry, it writes the Record.

We chose this because `hyper` exists to answer a supply-chain question, and every design that keeps
third-party code answers it only partially. Swamp demonstrates the failure precisely: extensions load
into the privileged isolate with no declared permissions, their dependencies are inlined into the
bundle on the author's machine so they never reach the consumer's lockfile, and the safety analyzer
reads a `.ts` while a prebuilt `.js` executes. Sandboxing fixes the last of those and none of the
others. Removing the code fixes all of them at once — there is no transitive graph, no build step, no
gap between the reviewed artefact and the executed one, because the artefact is the whole extension.
The second reason is that `hyper` is an AI-authoring tool: a Manifest can be checked statically,
offline, without credentials, which gives an agent a correctness oracle that a program does not have.

## Considered options

- **Native Go plugins.** Disqualified outright: Linux and macOS only, requires CGO and dynamic
  linking, and demands the consumer's toolchain and entire dependency graph match the author's
  byte-for-byte. It destroys the single static cross-compiled binary and provides no isolation.
- **Subprocess over stdio or gRPC.** Pleasant to author, but the boundary is fiction. A child process
  inherits full ambient authority and there is no portable way to remove it — seccomp and Landlock
  are Linux, `sandbox-exec` is deprecated macOS, Windows has neither. A capability declaration over
  that boundary is advisory, which is the failure we are here to avoid.
- **WASM via a pure-Go runtime.** The only candidate with a real, portable, deny-by-default boundary,
  and it was the working answer for most of the design session. It was eliminated on the merits once
  the capability model was tightened: with no filesystem, no `exec`, no access to secrets, and no
  network beyond what `hyper` fetches, a module's entire residual power is arbitrary computation over
  data `hyper` already holds. Every concrete case for it — bespoke signing, complex pagination,
  binary protocols — turned out to argue for growing `hyper`'s own primitives instead, and growing
  `hyper` is reviewed once by everyone where a module is reviewed separately by each consumer.

## Consequences

- **Built-in Providers and Extensions are the same object.** Both are Manifests; the only difference
  is who authored and distributed them, and that a built-in may declare Capabilities the grant model
  never extends to an extension — `Opaque` shell execution being the one that matters. A third party
  can never ship a Provider that runs commands on your machine.
- **The set of Capabilities is closed and owned by `hyper`.** It is exactly the set of effects
  `hyper` can perform, not an open vocabulary an author can extend. There is no "some extension will
  need a permission we did not anticipate": if `hyper` cannot do it, nothing names it.
- **Over-declaration is a static error, not a smell.** Because an Operation is data, the Capabilities
  it requires can be derived and compared against the declaration. They must match exactly rather
  than the declaration merely containing the derivation. This does not soften the rule that Kinds are
  authored rather than derived — a Kind is a claim about the world and genuinely underivable, while a
  Capability is mechanical.
- **The ceiling is a wall, not a slope.** When a Provider needs something `hyper` lacks, no amount of
  cleverness routes around it: the Provider is unwritable until `hyper` grows the primitive and
  ships. The failure mode moves from *the author wrote subtly dangerous code* to *the author could not
  write this Provider at all*. This is the accepted price, and the closed set will feel too small
  early.
- **A registry can exist and it makes no safety claim.** Distribution is a separate concern precisely
  because there is nothing to vet: `hyper` verifies that fetched bytes match a published digest, never
  that an extension is benign. The honest promise is not "this extension was checked" but "a hostile
  extension can reach nothing you did not grant, and you can read everything it can reach."
