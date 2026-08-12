# A concurrency limit is a `read`'s, and is one unless declared

Concurrency is a function of Kind and is `hyper`'s: a `read` Step's Expansion may run concurrently, an
effectful one runs strictly serially, and there is no authored knob (§6). What that left unstated was
the key itself. **`concurrency:` may be declared only on a `read`, and a Manifest declaring none
declares 1.** Both halves are one sentence read from its two ends — where the limit may be written, and
what its absence means — and both exist because a declaration that governs nothing is the failure mode
§3's per-Operation facts were introduced to avoid.

The reading a competent implementer reaches unaided is that an absent limit means *unbounded*, or means
some sensible-looking default the implementer picks. §6 as it stood said a `read` Step's Expansion *runs
concurrently* and pointed at the limit for how much of it runs at once; meeting an Operation that
declares none, the sentence reads as permission with no ceiling. The corpus supplies the case rather
than inviting one to be imagined: §3's `uptime` Manifest declares no `concurrency:`, and its
`check_http` is the very Operation §8 expands over the hosts a Target grants. Under the unaided reading,
the specification's own worked example dispatches every host at once against whatever is on the other
end.

That is not a performance question. Any number `hyper` invents is a claim about how hard a system may be
hit, made by the one party in the arrangement that has never seen it — and it is a claim no artefact
carries, so it appears in no review, no gutter, and no Comparison. A limit is the Provider author's
measurement or it is a guess, and §6 admits per-Operation facts on exactly the ground that the author
downstream would be guessing at them. Silence buying nothing is the only reading under which the
declaration is what buys the concurrency.

The other half falls out of the same paragraph from the other side. An effectful Expansion is serial by
a rule with no exception and no override, so a `concurrency:` written on a `mutate` or a `destroy`
governs nothing from the moment it is written — and unlike expandability, which is the Step author's
choice, this is decidable from the Manifest alone. A declaration that quietly does nothing is
indistinguishable, on every surface the tool has, from one that did what it says (ADR-0035), so it is
refused where it can be: `manifest-inconsistent`, on the shape ADR-0037 had already given three
neighbours.

## Considered options

- **An absent limit means unbounded.** Rejected above. It is the largest guess available about a system
  nobody described, and it is reached by omission rather than by anyone deciding it.
- **An absent limit means a number `hyper` picks — 4, 8, 10.** Rejected. It has the same defect as
  unbounded with better manners: a figure that looks measured, is not, and appears in no reviewed
  artefact. It would also be the first place in the tool where behaviour reaching the world came from a
  constant with no author.
- **Make `concurrency:` mandatory on every `read`.** Rejected: it forces the one-call `read` to state a
  number that governs nothing, which is the complaint this decision exists to answer, restated as a
  requirement. §3's objection to a field whose only useful value is fixed applies to the commonest Kind
  in the tool.
- **Refuse an explicit `concurrency: 1` as a second spelling of silence.** Rejected. The Repeatability
  precedent does not transfer — run-once is unwritable because it is not in the value set at all, where
  1 is an ordinary member of an integer's. Refusing it would need a rule saying *this one number is
  spelled by absence*, and it would take away the only way a Provider author who has established that an
  API refuses concurrency can say so rather than be read as having not considered it.
- **Accept `concurrency:` on an effectful Operation and ignore it.** Rejected as the status quo the
  decision was opened on: a Manifest fact a reviewer reads as a constraint, constraining nothing, in a
  file whose whole claim is that what it declares is what happens.
- **Let the limit bound a Pattern's calls as well — pages dispatched concurrently under it.** Rejected,
  and it turns out not to be available: all three Patterns are serial by construction (§3), each
  learning whether there is another call to make only from the answer to the one before. Buying
  concurrency there would mean `hyper` speculatively issuing requests no artefact asked for, against an
  API that has not said the next page exists.
- **Mark an inert limit in a rendering rather than refusing one.** Rejected for the `read` case, where
  the required predicate does not exist: whether an Operation is ever expanded over is fixed by a Step's
  `over:`, and nothing visible in a Manifest distinguishes an Operation that will be from one that will
  not.

## Consequences

- **A ninth shape of `manifest-inconsistent`, and no new `error_code`** (§4): a `concurrency:` limit on
  an Operation that is not a `read`. It joins ADR-0037's three as a fourth Kind-disagrees-with-itself
  check, pointing at one file, one Operation, and two adjacent keys.
- **A third per-Operation fact is stated by omission** (§3), beside `repeatability:` and Record
  cardinality. An explicit `concurrency: 1` remains legal and means what the omission means.
- **§6's rule softens from *runs concurrently* to *may run concurrently*.** Concurrency is now something
  a Manifest grants rather than something a Kind confers, which is the shape the two-key check already
  has one level up: the Kind admits it and the declaration supplies it.
- **The limit's boundary is stated rather than inferred** (§6): the members of one `read` Step's
  Expansion in flight at once, and nothing else — not a Pattern, not two Steps (ADR-0002), and where a
  Step carries no `over:` a set of one, which no limit ever written can exceed. A declared limit is a
  fact about the Operation the way its Kind is, live wherever a Step expands it.
- **Because every Pattern is serial, *members in flight* and *requests in flight* are one number.** The
  limit needs no second reading for a paginated `read` expanded over five hundred zones.
- **§9's `derived.concurrency_limit` is always present and always effective** — 1 for a `read` omitting
  the key, 1 for every `mutate` and `destroy`. `derived:` reports what `hyper` computed, so the rule
  about what may be authored is not left to be inferred from a field that came back empty.
- **The built-in `shell` Provider's `read` omits it**, so a command Expansion runs serially (§12). It is
  the criterion that section already states — `hyper` is the Provider author there and knows nothing
  whatever about the command.
- **A named cost, stated in §6 beside the rule rather than in §13.** A `read` over five hundred granted
  hosts whose Manifest omits the limit makes five hundred serial calls. It is not a wall: the remedy is
  one key, written by the author who measured the API.
