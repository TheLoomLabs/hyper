# `hyper` ships a Provider only where nobody else could write it

`hyper` embeds one Provider, `shell`, and the rule that fixes the set is a criterion rather than an
inventory: a Provider ships inside the binary only where the Capability it needs is one nobody else
may declare. Every other Provider is installed from a registry or authored locally, and `hyper` is
not in the business of maintaining any of them.

We chose this because a built-in that declares only `http` widens nothing. Built-ins and Extensions
are the same object under the same grammar and the same checks (ADR-0004), so a Provider `hyper`
embeds for convenience is a Provider its consumers could have written themselves — and embedding it
costs three things they do not get back. It ties the Provider's correctness to the release cycle: a
vendor moving a field means a `hyper` release, where a file in `providers/` means an edit and a
review, and no artefact downstream may override what a Manifest declares. It takes a name out of the
namespace permanently, since an Extension may never shadow a built-in (ADR-0004), so every convenience
built-in is a name every consumer is forbidden. And it puts `hyper` in the position of vouching for a
description of somebody else's API, which is the one correctness question the model has never owned.

The reserved Capability is a different matter, and it is the only one. `shell` is granted to no
Extension, so nothing outside the binary can carry it, and without something inside the binary that
does, `opaque` is a trait with no instances and the asymmetry that reserves it guards an empty room.

## Considered options

- **Batteries included** — an HTTP Provider, a DNS Provider, a git Provider. This is the reading an
  implementer reaches unaided, and it is the one every comparable tool takes. It is rejected above on
  what embedding costs rather than on taste; what makes the rejection affordable is that the response
  object (ADR-0040) leaves nothing an ordinary Manifest cannot reach, so the convenience is a file
  somebody writes once rather than a capability they lack.
- **A generic HTTP Provider** — method, path and body arriving as Operation inputs. Rejected on more
  than convenience: it is the typed layer with an escape hatch, one step from the shape this whole
  design replaces. A Manifest whose request is supplied at the Step describes nothing, so the review
  surface renders an Operation name and a URL argument where it is built to render a claim about what
  an Operation does, and `check` has nothing left to check it against.
- **A narrow built-in check Provider** — one `read` Operation fetching a URL and projecting its status
  and certificate. Genuinely not an escape hatch, and the closest call: it is what the corpus's own
  `uptime` example wants. Rejected because ADR-0040 makes it writable by anyone in twenty lines, which
  turns it from a capability into a convenience, and a convenience is what the paragraph above prices.
- **No built-in at all**, and `opaque` and `shell` struck from the model. Seriously considered: it
  would make *nothing reaches the world unreviewed* mechanically bounded rather than reviewed by eye,
  with every effect an HTTP request to a granted host and no escape hatch for anybody, `hyper`
  included. Rejected because request signing is already behind the wall, and with it every hyperscaler
  — so removing the local command too would leave a whole class of user with no route at all rather
  than an opaque one whose words a reviewer reads.

## Consequences

- **The roster and the reserved Capabilities are two closed sets with two growth stories.** Adding a
  built-in that declares only `http` widens nothing an author could not already write; adding one that
  carries a reserved Capability is authority. Only the second is a decision about power, and the
  criterion above means the first never happens.
- **A built-in is forkable in form and not in power.** Its source is readable, and copying it into
  `providers/` and editing it is an ordinary edit — but the copy must be renamed, an Extension being
  unable to shadow a built-in name, and once renamed it may not declare `shell`.
- **The first Provider a repository uses is one somebody wrote.** `hyper` ships no vendor Manifest and
  no starter Manifest, which is the same line ADR-0001 and the MCP surface already draw between what
  `hyper` derives and what a human reviews.
- **The set is stated in §12 in full.** A closed set nobody can read in one place is not visibly
  closed, and this one doubles as the list of names an Extension may never take.
