# The Comparison names its own failure and does not fail the job

The projected workflow's second invocation is

    printf '```\n' >> "$GITHUB_STEP_SUMMARY"
    set +e
    ./hyper changes <procedure> | tee -a "$GITHUB_STEP_SUMMARY"
    code=${PIPESTATUS[0]}
    set -e
    printf '```\n' >> "$GITHUB_STEP_SUMMARY"
    if [ "$code" -ne 0 ]; then
      printf 'the Comparison did not render (exit %s)\n' "$code" \
        >> "$GITHUB_STEP_SUMMARY"
    fi

which is `writeRun`'s machinery with a different last line. That closes #259.

The step used to stop at the pipeline. A pipeline's exit status is its **last** command's, so `tee`'s
`0` was the step's under `bash -e {0}` whatever `hyper` did, and the step could not fail — which is
how [ADR-0132](0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md)
Claim 6 found it, on a job whose install step had failed and whose `./hyper` therefore did not exist:

    /home/runner/work/_temp/27b51bc1-….sh: line 2: ./hyper: No such file or directory

    6  hyper changes observe: success

On that job it cost nothing to the *status*, the job being red already — though the page carried an
empty fence there too, with nothing on it saying the second invocation had found no binary to run.
**The case that costs is the one where the Run succeeds and the Comparison does not render**: a Store
the clone cannot read, which is the world resisting at `1` (§12). Everything the two invocations share
fails both of them — the pin gate turns both away, and a name that resolves for one resolves for the
other — so what is left to differ is the Store read the second one makes and the first one does not.
There the page carried an empty fence between two backtick lines, the job was green, and the half of
it §10 added the second invocation for was silently missing.

## The question, and the answer

Not *is this a bug* but **should a failed Comparison fail a Run that succeeded?** The answer is **no,
and the page says so**.

**The status stays the Run's**, because the Run is what reached the world. A Run that provisioned,
retired or observed and then finished is not made less finished by a rendering that could not be
produced afterwards, and turning that job red would put the executor's failure email — addressed to
whoever committed the projection (ADR-0005, ADR-0021) — on the tail rather than the dog. It is the
same judgement the `tee` above already makes about a summary the executor drops for being too large:
the log is complete, the page is a convenience copy, and a convenience copy failing is not the Run
failing (§10). It is also the judgement
[ADR-0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md) made one
layer down, where a Comparison that cannot read a revision renders `not-in-clone` and exits `0`
rather than refusing the report of what happened.

**And the page says so**, because the alternative to a red job is not a quiet one. An empty fence is
an absence with nothing naming it, which is the shape §8 refuses everywhere else it appears: an
absence is named, never substituted and never left to be inferred (ADR-0071). The `${PIPESTATUS[0]}`
`writeRun` was already carrying is what makes the sentence possible, so the repair reads as the two
steps finally agreeing about how a pipeline is read, and differing only in what they spend the code
on — the first exits with it, the second writes it down.

## What the sentence is, and what it is not

**It is written by the workflow, not by `hyper`.** The binary is told nothing: it never reads
`$GITHUB_STEP_SUMMARY`, and it did not get a new rendering, a new exit code or a new flag out of this.
The whole of the change is shell in a generated file, which is where every other fact about the
executor lives (§10, §11, ADR-0021). `hyper run <procedure>` writes the same bytes on a laptop as on a
runner, and so does `hyper changes <procedure>`.

**It sits after the closing fence**, because the fence holds what `hyper` rendered and this line is
the page's own note about a rendering that never arrived. Inside the fence it would read as
preformatted output of a command that produced none.

**It says the Comparison did not render, and means it of a rendering that did not arrive whole.** A
`hyper changes` that wrote part of its page and then stopped leaves those bytes inside the fence with
the sentence beneath them, and that is the reading intended rather than a case the wording missed: a
Comparison renders whole or it did not render, which is the rule
[ADR-0059](0059-a-projected-value-renders-whole-or-renders-changed.md) already fixes one layer down,
where a projected value renders in full or not at all and there is no truncated form.

**It carries the exit code and nothing else.** The code is the one diagnostic the step can honestly
reach: a Refusal's own rendering goes to stdout and would have landed inside the fence, and everything
`hyper` writes on stderr is in the Actions log, which is complete and which the page does not copy.
`77`, `75`, `2` and `1` are §12's and are read there; `127` is `bash`'s, and is the case ADR-0132
found. A sentence that guessed at *why* would be the projection editorialising about a binary it
cannot see.

## Considered options

- **`exit $code`, exactly as `writeRun` has it.** Rejected, and it is the closest call here: it is one
  line, it makes the step honest about what it did, and it needs no new bytes to explain itself. It
  loses on what it does to a green Run — a Comparison is a report about a Run that already happened,
  and letting the report's failure decide the Run's status inverts which of the two is the subject.
  The failure email that follows names the Procedure and says nothing about which half of the job
  produced it, so the ambiguity is paid for on the one surface that reaches a phone.
- **Keep the status and document it, changing no bytes.** Rejected. It was the cheapest fix and it
  leaves the empty fence exactly where it was: a doc comment repairs a reader of this repository, and
  the reader who is hurt is looking at a job summary page. The comment is written anyway, beside the
  code it now describes.
- **Write the note inside the fence.** Rejected above — it would claim to be output of a command that
  produced none.
- **Branch on the code and re-run, or fall back to a narrower `changes`.** Rejected. The projection
  carries two invocations and no branching (§10, ADR-0021); a retry is a second policy about a
  rendering, and a fallback would put a *different* Comparison on the page under the same fence, which
  is worse than none.
- **`continue-on-error: true` on the step.** Rejected. It buys the same status by a different route
  and costs more: the job goes green while the *step* is annotated red in the executor's own UI, which
  is a claim about the Run made in a place `hyper` does not write and cannot render, and it still
  leaves the page empty.

## Consequences

- **Every projected file's bytes change, so every repository holding one is `projection-stale` until
  `hyper project` is run again** (§10). The ordinary blast radius of a generator change, stated here
  because the file changing is the price of the behaviour changing.
- **The `changes` step still cannot fail the job, and now that is a decision rather than an
  omission.** It is stated in §10 beside `${PIPESTATUS[0]}`, in `writeChanges`'s doc comment, and
  here — the three places the why is written down.
- **A Run that succeeded over a Comparison that did not render is legible on the page and nowhere
  else.** No exit code distinguishes it, no Journal entry records it, and `hyper` never learns it
  happened: the Comparison is a rendering, and a rendering that failed leaves no trace in the Store
  (ADR-0011). The page is the whole of the evidence, which is the same deal §10 already strikes for
  the Dispositions.
- **This repair is enforced, not taught** (`docs/agents/acceptance-re-runs.md`). The worked example in
  `internal/workflow/testdata/` is compared byte-for-byte, the `project` goldens carry the file whole,
  and a package case holds the step's bytes and asserts it carries no `exit $code`. Nothing here is a
  sentence an agent reads and then decides about, so **no sealed acceptance run is owed**.
