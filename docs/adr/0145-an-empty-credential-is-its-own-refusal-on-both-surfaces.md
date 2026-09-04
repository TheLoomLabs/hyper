# An empty credential is its own Refusal, on both surfaces

The credential pass reads a variable three-valued. The environment does not hold it
(`credential-absent`), it holds it and sets it to the empty string (`credential-empty`), or it fills it
and the slot resolves. Both Refusals exit `77`. `targets` and the MCP `targets` tool report the same
three, as one word out of `absent`, `empty`, `set`, in a member named `presence` that replaces the
boolean `present`.

We chose this because the empty string passed the gate and the world was blamed for it. Presence was the
whole question and the value was not judged, so a variable exported to nothing resolved, `Auth.Credential`
composed `Bearer ` with nothing after it, and the endpoint answered `401`. That landed as a failure at
`1` — the world resisting — where what had happened is that the invocation was never ready, which is a
Refusal at `77`. On an effectful Procedure the Steps before the first authenticated one had already run
by the time the `401` came back.

An empty credential is not a typo anyone makes twice; it is what an upstream produces. A `$(op read …)`
that returned nothing, a CI secret never set on the fork, a `vault kv get` against the wrong path —
every one of them exports an empty string, and every one is a different fix from *you forgot to export
it*. The old message was not wrong, it was absent: the gate passed, so there was no message.

**The rule issue #112 bought is kept rather than abandoned.** The gate and `targets` ask one question,
so what that column says is what the Run will do. Splitting `present` in two places at once is what
keeps that true: a third state in the gate alone would have left `targets` reporting `present: true` for
a slot the gate Refuses, which is the invariant inverted rather than refined.

**It is a Refusal and not an unauthenticated request.** A declaration naming an Auth scheme has declared
that it authenticates, and letting an empty string silently downgrade that is a thing that works in
development and reaches a public endpoint in production. Unauthenticated is already expressible in the
artefact, where a reviewer sees it: a Provider naming no scheme sends no credential.

Nothing here reads a credential. The one thing taken off a value is whether it has any characters at
all — no length beyond zero, no shape, no plausibility, no scan (ADR-0007). Whether a credential works
is the endpoint's business; whether one was supplied is `hyper`'s.

## Considered options

- **Three-state in the gate, boolean on the surfaces.** Cheaper, and it breaks the thing issue #112 was
  for. Rejected: `targets` would report `present: true` for a slot the Run will decline, and an agent
  reading `targets` over MCP is exactly the consumer that cannot tell the difference by squinting at a
  terminal.
- **One code, two messages.** Rejected on §12's own test for splitting, the one that made
  `cadence-run-once` and `cadence-secret-output` two codes: a reader handed `credential-absent` for a
  variable that *is* exported checks the export, finds it, and is out of moves. §8 holds one remedy per
  code and the two remedies are not the same act — one names the wrappers that export a variable, the
  other the readers that export an empty one — so a single code could only ever have offered the wrong
  one to whichever half it was not written for.
- **Empty is meaningful — send the request unauthenticated.** Rejected above, and recorded here so it is
  not re-proposed.
- **Keep `present` and add an `empty` boolean beside it.** Non-breaking, and rejected: it spells one
  three-valued fact as four states of which *not present but empty* is unreachable, and it leaves
  `present: true` standing for a slot the gate Refuses — the same inversion the second option was
  rejected for, one member over.
- **Judge the value further — a minimum length, a shape per scheme.** Rejected on ADR-0007's ground.
  Zero characters is not a judgement about a credential's contents; it is the observation that there are
  none. Anything past it is `hyper` deciding whether a secret looks right, which is the endpoint's call
  and needs the value to make it.

## Consequences

- **`presence` replaces `present` on both wires, and that is breaking.** The member was renamed rather
  than re-typed so a client written against the boolean finds no `present` at all, rather than reading
  the string `"absent"` as true. The MCP tool's output schema names the closed three.
- **§12 gains a set and grows another by one.** *Credential presence* is a closed three of its own,
  declared once in §12 and cited by §6 and §9 rather than restated by either, and carried in code by
  `internal/presence` — one declaration site, which is what stops the gate and the column from reading
  one variable two ways. It is a leaf package importing nothing rather than three constants beside the
  Auth schemes, because `hyper targets` is fenced to imports that prove it reaches no network, no Store
  and no invocation, and `internal/capability` is how an invocation reaches the world.
- **`error_code` grows by one, to fifty-two.** `credential-empty` cites the same `env:` line
  `credential-absent` does, carries the same `auth.<slot>` field, reports every unfilled slot at once
  out of the same pass, and renders §8's named remedy rather than an `EDIT ONE OF` table. The
  act-on-the-environment remediation class now holds two members and two notes.
- **A `401` still means what it meant.** What changed is which `401`s can happen: one earned by a
  credential the gate accepted is the world resisting, and stays a failure at `1`. Nothing about where
  the process runs enters either decision.
- **A Run that would have failed part-way now Refuses before Step 1.** An effectful Procedure whose
  first authenticated Step was third no longer runs the first two. That is the point, and it is also a
  behaviour change for anyone whose pipeline had come to rely on the prefix running.
- **This is a taught repair and it owes a sealed run, deferred.** The gate half is enforced — the
  goldens hold both codes and the schema holds the closed three — but the shape of a tool's structured
  output is named *taught* by `docs/agents/acceptance-re-runs.md`, and a repair that is both is taught:
  nothing in the suite fails if an agent reads `presence: "empty"` and does the wrong thing with it. The
  run it owes is **`monitor-coverage`**, the task whose Target declaration carries the one credential
  slot in the set (`LOOKOUT_API_TOKEN`). It is not being bought here. It is also a gap in the task set
  rather than only an unspent run: no task exports that variable empty, so no run of the set as it
  stands would put an agent in front of the third state at all, and fencing that is one task file.
