# `run` takes a Procedure's name, and `review` takes two forms because the commands differ

`run <procedure>` takes **the name the Procedure declares for itself, and no second form**. A
positional containing `/` or ending `.yaml` is a name that resolves to nothing, exit `2`, with a
sentence naming the grammar this command takes. `review <artefact>` keeps both of its forms. The two
positionals are spelled apart because the two commands are, and the tool schema says of each what
its command does.

## One string said two, and the command took one

`runTool`'s `procedure` argument advertised the pair:

> The Procedure to run: a repository-relative path — one containing / or ending .yaml — or the name
> the artefact declares for itself.

`RunRun` resolves its positional through one lookup, in the load's Procedure index — a `procedures/`
file's own `procedure:` against whether it is there. Nothing on that road reads a path. So `hyper
run procedures/watch-status.yaml` was `2` for a Procedure `hyper run watch-status` ran, and over MCP
that `2` is a protocol error: an agent that believed the schema got a malformed call back for an
artefact that is there and is runnable.

The half-sentence is `reviewTool`'s, one screen up, and on `review` it is true. What §9 says of this
command is that it runs the **named** Procedure, and its tool sketch says only *the Procedure to
run, carrying no target*. Neither half offered a second form. The path form was never this
command's; it was carried across.

That makes it a decision rather than a typo, because the two repairs move in opposite directions —
delete the clause, or build what it promises — and one of them changes what `hyper run` accepts.

## Why the name is the whole of this positional

**`review`'s two forms are two namespaces, not two spellings.** Each reaches what the other cannot.
A name there is matched across all four artefact namespaces at once, so one string can be a
Definition here and a Target declaration there, and the path is what tells the two apart. And an
artefact whose file will not parse declares no name at all, so it is in no namespace and only its
path reaches it — which is where §9's *found and faulty* exits `1` with `check`'s row (ADR-0060,
ADR-0064). Neither property is a fact about `run`. `procedures/` is one namespace, so a Procedure's
name is unambiguous in it; and a Procedure whose file will not parse cannot be run at all, the
Run-start `check` declining the load before Step 1. **A path arm on `run` would reach no Procedure
the name does not already reach**, under a second spelling and two more usage errors — a path
matching no artefact, and a path matching one that is not a Procedure. The one thing it would reach
beyond the name is the file that will not parse, and reaching it buys a Run nothing: on `review`
that is the *found and faulty* rendering, `check`'s row at exit `1`, and `check` is the command that
writes it — on `run` it would be a Procedure the Run-start gate declines whichever way the caller
spelled it.

**The name is what goes in because it is what comes out.** `run.json` records `procedure` as a name;
`changes <procedure>` takes that name back, byte-exact, and its own tool argument has always
described itself accurately as one. ADR-0089 read a positional against the repository so that the
spelling `check`'s rows come back in is the spelling its argument accepts. The same argument lands
here on the other axis: this command's namespace is keyed by names, so the round trip out of a Run
and back into a Comparison is a name, and the positional that opens it is one.

**The ergonomic cost the second form was for is already paid by a rule.** The case against a
name-only positional is an agent holding `procedures/deploy.yaml` — off a `problem` row, off a
review header, off the file it just wrote — having to open the file to find the name. It does not:
`name-mismatch` pins an artefact's `name:` to its file's basename (§4), and §9 already leans on that
rule where it fixes how a name is matched. So `procedures/deploy.yaml` is `deploy` by a rule `check`
enforces, computed from the string in hand with no read at all. What a caller holding a path needs
is that rule stated, and the decline now states it.

## What the decline says, and when it decides

The path-shaped positional earns its own sentence rather than the namespace one, because the
remedies differ exactly as they do for the Definition arm already there:

```
hyper run: "procedures/watch-status.yaml" is a path, and run takes a Procedure's name
  the name is that file's basename without the .yaml; `review` is the command whose positional takes either form
```

It names the string that was typed, the grammar this positional has, and the rule that gets from one
to the other. It performs no lookup and suggests no candidate: §9 refuses a near miss because a
suggestion is a partial name resolved on the caller's behalf, and a human who accepts one has run
something they did not type (ADR-0047). The basename rule is not that — it is a fact about the
repository `check` already holds every artefact to.

**The shape is read after the lookup, not before it.** `resolveArtefact` reads it before, because it
has two namespaces to choose between and a name tried as a path would make which artefact `hyper
review deploy` renders depend on what else is in the repository. Here there is one namespace and
nothing to choose, so the shape picks a sentence for a positional that already resolved to nothing —
which costs no Procedure whose own name happens to end `.yaml` the ability to run. Both commands
read the shape with `isPathForm`: what a path looks like is one fact, and the two differ in what
they do with one.

## Considered options

- **Build the second form**, so the description becomes true and the two artefact-naming positionals
  become one grammar. Rejected on the section above: on `review` the second form reaches artefacts
  the first cannot, and on `run` it would reach the same set twice. It also grows the command's
  usage-error surface by two sentences to buy a spelling the basename rule already supplies. The
  consistency it offers is between two positionals whose commands are not alike — one renders any of
  five artefact kinds, the other runs the one kind that can be run.
- **Take both forms and resolve one against the other**, trying a name first and a path if nothing
  matched. Rejected for ADR-0060's reason at the other command: a fallback makes which artefact a
  string names depend on what else is in the repository, and here that string starts a Run.
- **Correct the description and leave the message alone**, so a path-shaped positional lands on *no
  Procedure named "procedures/watch-status.yaml" in this repository's Procedure namespace*.
  Accurate, and it is the status quo. Rejected: the sentence is true and says nothing to the one
  caller who reaches it, whose fault is a grammar rather than a typo, and this surface's usage
  errors are read by agents that cannot ask a second question.
- **Leave the schema as it is and let the two surfaces disagree.** Rejected outright. A tool builds
  the command line its command would have received and holds no logic of its own (issue #198), so a
  form advertised here and resolved by no command is a promise nothing can keep.

## Consequences

- **`hyper run <path>` is `2` and always was**; what changes is the sentence, and that the schema no
  longer says otherwise. No Procedure that ran before stops running.
- **`run`'s positional now has three declines**, all §9's usage error — exit `2`, no `error_code`,
  no row stream, the rendering on stderr — and each with its own remedy: a name matching nothing, a
  name that is a Definition, and a positional written as a path (ADR-0060, ADR-0036).
- **The corpora drive the form on both surfaces.** Every `run` case named a Procedure by its name
  and none drove the form the schema advertised, which is why nothing caught this.
  The pair `run/a-path-rather-than-a-procedures-name` and its `mcp/run/usage-` twin are one sentence
  on two surfaces, held together by the fence that pairs every declining envelope with the CLI case
  that writes it.
- **§9 now states both positionals' forms where it states either.** It said *the artefact named* of
  `review` and *the named Procedure* of `run`, and left the path form to a clause inside `check`'s
  paragraph — so the one place the two forms were written down together was the tool schema that had
  them wrong.
- **`changes <procedure>` is untouched**, its argument having described itself accurately
  throughout. It is now one of two Procedure-naming positionals that say the same thing rather than
  the only one.
