# `docs/adr/`

Why `hyper` is the way it is, including the options that lost. The specification in
[`docs/spec/`](../spec/) says what `hyper` does; these say why, what it cost, and what was
measured. Where a record and the spec disagree the spec is right — a record is a decision at a
moment, and the moment does not come back.

**This corpus holds 141 records**, numbered `0001`–`0141` with no gaps. They are chronological:
the number is the order they were written in and carries no other meaning. Nothing here is
superseded by status — a later record that revises an earlier one says so in its own text, and
the earlier one stays as it was written.

Two genres share the numbering. Most records are **decisions** — a rule fixed, with the
alternatives it beat. A growing minority are **field reports**: what happened the first time
something met the world, written up because the measurement is the finding.
[ADR-0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md)
and
[ADR-0133](0133-three-archives-nobody-had-run-carried-and-the-release-stamps-three-of-four-dirty.md)
are the shape.

## Start here

Twelve that carry the most of the design. Read in this order they are roughly an argument.

- **[0001](0001-no-bypass-for-safety-refusals.md)** — Safety refusals have no bypass flag. The
  first rule, and the one every other guardrail leans on.
- **[0004](0004-extensions-are-data-not-code.md)** — Extensions are data, not code. Why reviewing
  a Manifest is reviewing the whole of what will run.
- **[0010](0010-hyper-has-no-plan.md)** — `hyper` has no plan. The thesis's sharpest omission.
- **[0006](0006-the-record-travels-in-the-repository.md)** — The record travels in the
  repository. Where the Store is, and why it is not a service.
- **[0011](0011-the-store-is-append-only.md)** — The Store is append-only.
- **[0025](0025-the-domain-model-splits-by-what-a-reviewer-must-tell-apart.md)** — The domain
  model splits by what a reviewer must tell apart. Why the nouns are the nouns.
- **[0036](0036-every-run-is-a-run-of-a-procedure.md)** — Every Run is a Run of a Procedure. The
  reason there is no ad-hoc invocation.
- **[0014](0014-hyper-has-no-configuration-files.md)** — `hyper` has no configuration files.
- **[0020](0020-the-hyper-version-is-pinned-by-the-repository.md)** — The `hyper` version is
  pinned by the repository. The gate every command but one stands behind.
- **[0019](0019-hyper-never-updates-itself.md)** — `hyper` never updates itself.
- **[0093](0093-orientation-is-a-handshake-field-and-hyper-writes-no-file-to-carry-it.md)** —
  Orientation is a handshake field, as
  **[0095](0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)**
  amends it. How an agent arrives knowing what to do.
- **[0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md)** — The
  acceptance harness is sealed. Why a transcript that read the spec proves nothing.

## Reading paths

Ways in, by subject. **Not a partition** — a record can sit on more than one path and some sit on
none. The list below is the whole corpus and the only complete thing on this page.

**The thesis, and what it refuses**
[0001](0001-no-bypass-for-safety-refusals.md) ·
[0009](0009-a-probe-is-not-a-run.md) ·
[0010](0010-hyper-has-no-plan.md) ·
[0013](0013-hyper-has-no-query-language.md) ·
[0014](0014-hyper-has-no-configuration-files.md) ·
[0015](0015-the-cli-never-prompts.md) ·
[0016](0016-hyper-has-no-telemetry.md) ·
[0019](0019-hyper-never-updates-itself.md) ·
[0021](0021-hyper-never-speaks-first.md) ·
[0036](0036-every-run-is-a-run-of-a-procedure.md)

**The authoring format**
[0002](0002-a-procedure-is-a-sequence-not-a-graph.md) ·
[0008](0008-a-procedure-is-fully-bound.md) ·
[0022](0022-the-authoring-format-has-no-expression-language.md) ·
[0023](0023-the-authoring-format-is-a-strict-yaml-subset.md) ·
[0051](0051-a-shell-command-is-an-argv-list-with-a-literal-head.md) ·
[0078](0078-a-body-literal-is-typed-by-its-spelling-and-a-hole-by-its-input.md) ·
[0079](0079-the-canonical-encoding-is-a-property-of-a-value.md) ·
[0081](0081-a-value-is-read-against-the-schema-at-its-position.md) ·
[0117](0117-a-closed-key-set-is-stated-as-closed-and-an-invocation-is-a-fixed-block.md)

**Authority, blast radius and Refusal**
[0024](0024-local-grants-enumerated-hosts-like-any-other-target.md) ·
[0027](0027-expansion-is-scoped-by-kind-not-by-record-type.md) ·
[0029](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md) ·
[0032](0032-a-definition-observes-or-effects-never-both.md) ·
[0033](0033-a-destroy-by-literal-identifier-opens-the-series-it-ends.md) ·
[0035](0035-a-predicate-that-cannot-decide-refuses.md) ·
[0041](0041-local-is-authored-by-the-repository-like-any-other-target.md) ·
[0042](0042-a-probe-is-bounded-by-the-grant-it-binds.md) ·
[0053](0053-an-opaque-destroy-names-its-population.md) ·
[0069](0069-authority-is-one-relation-read-from-whichever-end-the-artefact-supplies.md) ·
[0085](0085-the-selector-a-destroy-carries-is-required-by-the-kind-not-by-the-capability.md) ·
[0103](0103-a-grant-is-the-targets-and-the-narrower-one-is-a-second-declaration.md) ·
[0121](0121-an-opaque-mutate-is-unbounded-and-no-number-clears-the-flag.md)

**Providers, and what ships in the binary**
[0004](0004-extensions-are-data-not-code.md) ·
[0031](0031-an-auth-scheme-is-a-header-and-a-placement-never-a-protocol.md) ·
[0039](0039-hyper-ships-a-provider-only-where-nobody-else-could-write-it.md) ·
[0040](0040-an-http-response-is-an-object-hyper-builds.md) ·
[0052](0052-a-commands-stdout-is-text-never-a-parsed-object.md) ·
[0073](0073-a-providers-origin-is-where-its-bytes-load-from-never-whether-it-claimed-an-upstream.md) ·
[0082](0082-the-scheme-is-https-and-there-is-no-second-one.md) ·
[0087](0087-a-ref-is-a-location-and-hyper-names-no-registry.md) ·
[0106](0106-a-manifest-is-writable-from-the-surface-and-both-costs-were-paid-at-the-world.md) ·
[0107](0107-a-query-string-in-path-is-refused-where-it-is-written.md)

**Execution**
[0003](0003-a-run-is-never-resumed.md) ·
[0018](0018-retry-only-follows-a-failure-that-provably-preceded-the-request.md) ·
[0034](0034-a-predicate-is-evaluated-against-one-instant-fixed-at-the-runs-start.md) ·
[0037](0037-an-operations-kind-fixes-which-repeatability-values-it-may-declare.md) ·
[0045](0045-a-concurrency-limit-is-a-reads-and-is-one-unless-declared.md) ·
[0050](0050-a-status-is-an-answer-not-an-error.md) ·
[0056](0056-skip-if-recorded-tests-a-record-not-a-step.md) ·
[0061](0061-a-refusal-belongs-to-the-run-not-to-the-step.md) ·
[0062](0062-a-request-that-never-left-is-a-disposition-of-its-own.md) ·
[0072](0072-a-guardrail-that-declines-after-a-call-is-a-halt.md) ·
[0091](0091-a-rehearsals-withheld-step-is-a-member-on-that-steps-row.md) ·
[0111](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md) ·
[0116](0116-a-requirement-halts-and-claims-nothing-to-do-it.md) ·
[0122](0122-a-requirement-roots-at-any-projected-field-and-the-value-goes-on-the-line.md)

**The record: Store, Journal, Provenance**
[0006](0006-the-record-travels-in-the-repository.md) ·
[0011](0011-the-store-is-append-only.md) ·
[0012](0012-deleting-a-definition-abandons-its-assets.md) ·
[0028](0028-a-store-files-schema-version-is-its-own.md) ·
[0030](0030-a-steps-identity-set-holds-what-it-concluded-not-what-it-wrote.md) ·
[0043](0043-a-provenance-member-is-written-where-it-has-exactly-one-value.md) ·
[0048](0048-provenance-names-the-top-level-procedures-revision.md) ·
[0070](0070-an-expansions-members-are-one-record-identity-each.md) ·
[0074](0074-the-store-branch-is-fetched-shallow-and-whole.md) ·
[0075](0075-hyper-never-checks-the-store-out.md) ·
[0076](0076-every-store-path-carries-the-id-of-the-run-that-wrote-it.md) ·
[0084](0084-a-version-carrying-no-fields-is-not-a-tombstones-alone.md) ·
[0119](0119-a-revision-resolves-where-the-artefact-was-committed-and-the-loop-commits.md)

**Review, Comparison and rendering**
[0026](0026-the-gutter-annotates-a-table-aggregates-and-one-surface-editorialises.md) ·
[0047](0047-an-id-a-human-retypes-renders-whole.md) ·
[0049](0049-the-rendered-ordinal-is-a-position-not-an-identity.md) ·
[0054](0054-a-flag-renders-in-line-order-never-in-severity-order.md) ·
[0055](0055-a-steps-identity-digest-is-compared-against-the-last-run-that-carried-one.md) ·
[0057](0057-a-review-renders-the-working-tree-and-the-range-is-the-gutters-supply.md) ·
[0058](0058-a-comparison-row-is-a-record-read-at-its-endpoints.md) ·
[0059](0059-a-projected-value-renders-whole-or-renders-changed.md) ·
[0063](0063-a-gloss-is-a-notation-and-the-header-is-the-reviews-fourth-rendering.md) ·
[0065](0065-a-read-command-orders-on-the-axis-it-ranges-over-and-time-runs-newest-first.md) ·
[0067](0067-a-range-is-anchored-by-whether-the-artefact-carries-a-revision-not-by-its-kind.md) ·
[0080](0080-a-code-fact-renders-whole-in-the-shape-it-was-written.md) ·
[0086](0086-a-code-fact-is-read-where-it-is-authored.md) ·
[0135](0135-the-comparison-names-its-own-failure-and-does-not-fail-the-job.md)

**The two surfaces: CLI and MCP**
[0015](0015-the-cli-never-prompts.md) ·
[0060](0060-naming-nothing-is-a-usage-error-fetching-nothing-is-not.md) ·
[0088](0088-the-server-is-started-by-mcp-and-it-stands-outside-the-tree.md) ·
[0089](0089-a-path-argument-is-read-against-the-repository-never-against-the-callers-directory.md) ·
[0090](0090-run-takes-a-procedures-name-and-review-takes-two-forms-because-the-commands-differ.md) ·
[0092](0092-a-cancelled-call-drains-and-the-server-catches-no-signal.md) ·
[0094](0094-the-argument-less-invocation-writes-the-tree-and-there-is-no-help.md) ·
[0097](0097-a-checks-rows-travel-in-the-text-block.md) ·
[0098](0098-an-unknown-flag-names-the-flags-that-command-takes.md) ·
[0100](0100-a-reviews-page-travels-in-the-structured-content.md) ·
[0102](0102-a-tool-that-declined-answers-no-structured-half.md) ·
[0113](0113-a-listing-over-the-record-says-where-the-record-is.md)

**Orientation: what an agent is told**
[0093](0093-orientation-is-a-handshake-field-and-hyper-writes-no-file-to-carry-it.md) ·
[0095](0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md) ·
[0096](0096-the-shape-carried-whole-is-the-effectful-one-and-the-example-is-not-the-acceptance-task.md) ·
[0101](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md) ·
[0118](0118-the-envelope-cannot-be-trapped-now-that-the-orientation-states-it.md) ·
[0120](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md) ·
[0127](0127-a-remedy-may-not-assert-what-the-answer-could-not-establish.md)

**Cadence and projection**
[0005](0005-cadence-is-declared-and-projected-never-scheduled.md) ·
[0038](0038-a-cadence-and-a-run-once-step-are-refused-together.md) ·
[0046](0046-the-projections-executor-is-compiled-in-never-authored.md) ·
[0066](0066-a-step-is-not-an-interval-and-a-rate-has-no-year-in-it.md) ·
[0077](0077-a-cadence-and-a-secret-producing-step-are-refused-together.md) ·
[0132](0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md) ·
[0134](0134-the-deepen-step-names-one-ref-and-what-deepens-the-code-branch-is-the-clones-own-boundary.md) ·
[0139](0139-the-fixture-delivered-one-tick-in-forty-and-the-gaps-moved-together-across-four-files.md)

**Distribution, versions and the release**
[0019](0019-hyper-never-updates-itself.md) ·
[0020](0020-the-hyper-version-is-pinned-by-the-repository.md) ·
[0128](0128-third-party-bytes-entered-a-repository-for-the-first-time-and-the-ref-recorded-was-the-one-typed.md) ·
[0131](0131-project-wrote-a-digest-for-the-first-time-and-the-network-contributed-one-scalar.md) ·
[0133](0133-three-archives-nobody-had-run-carried-and-the-release-stamps-three-of-four-dirty.md) ·
[0136](0136-the-release-builds-every-platform-before-it-publishes-one.md) ·
[0137](0137-a-browser-sets-the-attribute-and-the-shell-runs-what-finder-offers-to-delete.md) ·
[0138](0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)

**Field reports — the world answering**
[0104](0104-the-acceptance-fixture-ships-a-store.md) ·
[0105](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md) ·
[0109](0109-the-seal-covers-the-output-directory-the-harness-writes.md) ·
[0110](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md) ·
[0112](0112-the-second-run-skipped-what-the-first-did-and-every-revision-it-recorded-is-unresolvable.md) ·
[0124](0124-the-second-change-window-run-declined-the-bound-and-a-taught-repair-owes-a-run.md) ·
[0125](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md) ·
[0126](0126-a-predicate-over-an-expansion-holds-of-all-of-them-and-an-answer-must-name-which.md) ·
[0129](0129-the-destroy-landed-inside-the-seal-and-what-held-the-hand-made-monitors-was-not-the-bound.md) ·
[0130](0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md)

**How this repository is built and held**
[0007](0007-hyper-never-stores-a-secret.md) ·
[0017](0017-the-wire-is-visible-only-where-no-credential-was-used.md) ·
[0025](0025-the-domain-model-splits-by-what-a-reviewer-must-tell-apart.md) ·
[0044](0044-an-expansion-is-ordered-by-the-name-not-the-path.md) ·
[0064](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md) ·
[0068](0068-one-supply-is-stated-once-and-the-member-it-silences-is-not-omitted.md) ·
[0071](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md) ·
[0099](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md) ·
[0114](0114-the-rehearsal-marker-is-the-entrys-and-records-joins-for-it.md) ·
[0115](0115-a-rehearsal-is-the-comparisons-subject-where-a-caller-names-it-as-one.md) ·
[0123](0123-the-suite-is-run-by-a-machine-and-a-prepared-machine-may-not-skip.md) ·
[0140](0140-the-readme-holds-the-first-read-and-a-corpus-it-points-at-carries-an-index.md) ·
[0141](0141-a-releases-body-is-a-file-in-the-tree-the-tag-names.md)

## Every record

In order. `TestDocs_TheADRIndexNamesEveryRecord` requires this list and the directory to be the
same set — every record linked exactly once, every link resolving — so an ADR added without a line
here fails the suite.

- **0001** · [Safety refusals have no bypass flag](0001-no-bypass-for-safety-refusals.md)
- **0002** · [A Procedure is a sequence, not a graph](0002-a-procedure-is-a-sequence-not-a-graph.md)
- **0003** · [A Run is never resumed](0003-a-run-is-never-resumed.md)
- **0004** · [Extensions are data, not code](0004-extensions-are-data-not-code.md)
- **0005** · [Cadence is declared and projected, never scheduled](0005-cadence-is-declared-and-projected-never-scheduled.md)
- **0006** · [The record travels in the repository](0006-the-record-travels-in-the-repository.md)
- **0007** · [`hyper` never stores a secret](0007-hyper-never-stores-a-secret.md)
- **0008** · [A Procedure is fully bound](0008-a-procedure-is-fully-bound.md)
- **0009** · [A Probe is not a Run](0009-a-probe-is-not-a-run.md)
- **0010** · [`hyper` has no plan](0010-hyper-has-no-plan.md)
- **0011** · [The Store is append-only](0011-the-store-is-append-only.md)
- **0012** · [Deleting a Definition abandons its Assets](0012-deleting-a-definition-abandons-its-assets.md)
- **0013** · [`hyper` has no query language](0013-hyper-has-no-query-language.md)
- **0014** · [`hyper` has no configuration files](0014-hyper-has-no-configuration-files.md)
- **0015** · [The CLI never prompts](0015-the-cli-never-prompts.md)
- **0016** · [`hyper` has no telemetry](0016-hyper-has-no-telemetry.md)
- **0017** · [The wire is visible only where no credential was used](0017-the-wire-is-visible-only-where-no-credential-was-used.md)
- **0018** · [Retry only follows a failure that provably preceded the request](0018-retry-only-follows-a-failure-that-provably-preceded-the-request.md)
- **0019** · [`hyper` never updates itself](0019-hyper-never-updates-itself.md)
- **0020** · [The `hyper` version is pinned by the repository](0020-the-hyper-version-is-pinned-by-the-repository.md)
- **0021** · [`hyper` never speaks first](0021-hyper-never-speaks-first.md)
- **0022** · [The authoring format has no expression language](0022-the-authoring-format-has-no-expression-language.md)
- **0023** · [The authoring format is a strict YAML subset](0023-the-authoring-format-is-a-strict-yaml-subset.md)
- **0024** · [`local` grants enumerated hosts like any other Target](0024-local-grants-enumerated-hosts-like-any-other-target.md)
- **0025** · [The domain model splits by what a reviewer must tell apart](0025-the-domain-model-splits-by-what-a-reviewer-must-tell-apart.md)
- **0026** · [The gutter annotates, a table aggregates, and one surface editorialises](0026-the-gutter-annotates-a-table-aggregates-and-one-surface-editorialises.md)
- **0027** · [Expansion is scoped by Kind, not by Record type](0027-expansion-is-scoped-by-kind-not-by-record-type.md)
- **0028** · [A Store file's schema version is its own, not the branch's](0028-a-store-files-schema-version-is-its-own.md)
- **0029** · [A host is a candidate set, a grant, and their intersection](0029-a-host-is-a-candidate-set-a-grant-and-their-intersection.md)
- **0030** · [A Step's identity set holds what it concluded, not what it wrote](0030-a-steps-identity-set-holds-what-it-concluded-not-what-it-wrote.md)
- **0031** · [An Auth scheme is a header and a placement, never a protocol](0031-an-auth-scheme-is-a-header-and-a-placement-never-a-protocol.md)
- **0032** · [A Definition observes or effects, never both](0032-a-definition-observes-or-effects-never-both.md)
- **0033** · [A destroy by literal identifier opens the series it ends](0033-a-destroy-by-literal-identifier-opens-the-series-it-ends.md)
- **0034** · [A predicate is evaluated against one instant, fixed at the Run's start](0034-a-predicate-is-evaluated-against-one-instant-fixed-at-the-runs-start.md)
- **0035** · [A predicate that cannot decide Refuses](0035-a-predicate-that-cannot-decide-refuses.md)
- **0036** · [Every Run is a Run of a Procedure](0036-every-run-is-a-run-of-a-procedure.md)
- **0037** · [An Operation's Kind fixes which Repeatability values it may declare](0037-an-operations-kind-fixes-which-repeatability-values-it-may-declare.md)
- **0038** · [A Cadence and a run-once Step are refused together](0038-a-cadence-and-a-run-once-step-are-refused-together.md)
- **0039** · [`hyper` ships a Provider only where nobody else could write it](0039-hyper-ships-a-provider-only-where-nobody-else-could-write-it.md)
- **0040** · [An HTTP response is an object `hyper` builds, not the body it returned](0040-an-http-response-is-an-object-hyper-builds.md)
- **0041** · [`local` is authored by the repository like any other Target](0041-local-is-authored-by-the-repository-like-any-other-target.md)
- **0042** · [A Probe is bounded by the grant it binds](0042-a-probe-is-bounded-by-the-grant-it-binds.md)
- **0043** · [A Provenance member is written where it has exactly one value](0043-a-provenance-member-is-written-where-it-has-exactly-one-value.md)
- **0044** · [An Expansion is ordered by the name, not the path it is stored at](0044-an-expansion-is-ordered-by-the-name-not-the-path.md)
- **0045** · [A concurrency limit is a `read`'s, and is one unless declared](0045-a-concurrency-limit-is-a-reads-and-is-one-unless-declared.md)
- **0046** · [The projection's executor is compiled in, never authored](0046-the-projections-executor-is-compiled-in-never-authored.md)
- **0047** · [An id a human retypes renders whole](0047-an-id-a-human-retypes-renders-whole.md)
- **0048** · [Provenance names the top-level Procedure's revision](0048-provenance-names-the-top-level-procedures-revision.md)
- **0049** · [The rendered ordinal is a position, not an identity](0049-the-rendered-ordinal-is-a-position-not-an-identity.md)
- **0050** · [A status is an answer, not an error](0050-a-status-is-an-answer-not-an-error.md)
- **0051** · [A shell command is an argv list with a literal head](0051-a-shell-command-is-an-argv-list-with-a-literal-head.md)
- **0052** · [A command's stdout is text, never a parsed object](0052-a-commands-stdout-is-text-never-a-parsed-object.md)
- **0053** · [An `opaque` `destroy` names its population](0053-an-opaque-destroy-names-its-population.md)
- **0054** · [A flag renders in line order, never in severity order](0054-a-flag-renders-in-line-order-never-in-severity-order.md)
- **0055** · [A Step's identity digest is compared against the last Run that carried one](0055-a-steps-identity-digest-is-compared-against-the-last-run-that-carried-one.md)
- **0056** · [`skip-if-recorded` tests a Record, not a Step](0056-skip-if-recorded-tests-a-record-not-a-step.md)
- **0057** · [A review renders the working tree, and the range is the gutter's supply](0057-a-review-renders-the-working-tree-and-the-range-is-the-gutters-supply.md)
- **0058** · [A Comparison row is a Record read at its endpoints](0058-a-comparison-row-is-a-record-read-at-its-endpoints.md)
- **0059** · [A projected value renders whole or renders `changed`](0059-a-projected-value-renders-whole-or-renders-changed.md)
- **0060** · [Naming nothing is a usage error, fetching nothing is not](0060-naming-nothing-is-a-usage-error-fetching-nothing-is-not.md)
- **0061** · [A Refusal belongs to the Run, not to the Step](0061-a-refusal-belongs-to-the-run-not-to-the-step.md)
- **0062** · [A request that never left is a Disposition of its own](0062-a-request-that-never-left-is-a-disposition-of-its-own.md)
- **0063** · [A gloss is a notation, and the header is the review's fourth rendering](0063-a-gloss-is-a-notation-and-the-header-is-the-reviews-fourth-rendering.md)
- **0064** · [An authored name that resolves to nothing is a check, not a load failure](0064-an-authored-name-that-resolves-to-nothing-is-a-check-not-a-load-failure.md)
- **0065** · [A read command orders on the axis it ranges over, and time runs newest-first](0065-a-read-command-orders-on-the-axis-it-ranges-over-and-time-runs-newest-first.md)
- **0066** · [A step is not an interval, and a rate has no year in it](0066-a-step-is-not-an-interval-and-a-rate-has-no-year-in-it.md)
- **0067** · [A range is anchored by whether the artefact carries a revision, not by its kind](0067-a-range-is-anchored-by-whether-the-artefact-carries-a-revision-not-by-its-kind.md)
- **0068** · [One supply is stated once, and the member it silences is not omitted](0068-one-supply-is-stated-once-and-the-member-it-silences-is-not-omitted.md)
- **0069** · [`AUTHORITY` is one relation read from whichever end the artefact supplies](0069-authority-is-one-relation-read-from-whichever-end-the-artefact-supplies.md)
- **0070** · [An Expansion's members are one Record identity each](0070-an-expansions-members-are-one-record-identity-each.md)
- **0071** · [A missing git object is an absence to name, never a supply to substitute](0071-a-missing-git-object-is-an-absence-to-name-never-a-supply-to-substitute.md)
- **0072** · [A guardrail that declines after a call is a halt, never a Refusal](0072-a-guardrail-that-declines-after-a-call-is-a-halt.md)
- **0073** · [A Provider's origin is where its bytes load from, never whether it claimed an upstream](0073-a-providers-origin-is-where-its-bytes-load-from-never-whether-it-claimed-an-upstream.md)
- **0074** · [The Store branch is fetched shallow and whole](0074-the-store-branch-is-fetched-shallow-and-whole.md)
- **0075** · [`hyper` never checks the Store out](0075-hyper-never-checks-the-store-out.md)
- **0076** · [Every Store path carries the id of the Run that wrote it](0076-every-store-path-carries-the-id-of-the-run-that-wrote-it.md)
- **0077** · [A Cadence and a secret-producing Step are refused together](0077-a-cadence-and-a-secret-producing-step-are-refused-together.md)
- **0078** · [A body literal is typed by its spelling and a hole by its input](0078-a-body-literal-is-typed-by-its-spelling-and-a-hole-by-its-input.md)
- **0079** · [The canonical encoding is a property of a value, not of a file](0079-the-canonical-encoding-is-a-property-of-a-value.md)
- **0080** · [A code fact renders whole, in the shape it was written](0080-a-code-fact-renders-whole-in-the-shape-it-was-written.md)
- **0081** · [A value is read against the schema at its position](0081-a-value-is-read-against-the-schema-at-its-position.md)
- **0082** · [The scheme is `https`, and there is no second one](0082-the-scheme-is-https-and-there-is-no-second-one.md)
- **0083** · [A read-only Run attempts the sync and tolerates its failure](0083-a-read-only-run-attempts-the-sync-and-tolerates-its-failure.md)
- **0084** · [A version carrying no fields is not a Tombstone's alone](0084-a-version-carrying-no-fields-is-not-a-tombstones-alone.md)
- **0085** · [The selector a `destroy` carries is required by the Kind, not by the Capability](0085-the-selector-a-destroy-carries-is-required-by-the-kind-not-by-the-capability.md)
- **0086** · [A code fact is read where it is authored](0086-a-code-fact-is-read-where-it-is-authored.md)
- **0087** · [A ref is a location, and `hyper` names no registry](0087-a-ref-is-a-location-and-hyper-names-no-registry.md)
- **0088** · [The server is started by `mcp`, and it stands outside the tree](0088-the-server-is-started-by-mcp-and-it-stands-outside-the-tree.md)
- **0089** · [A path argument is read against the repository, never against the caller's directory](0089-a-path-argument-is-read-against-the-repository-never-against-the-callers-directory.md)
- **0090** · [`run` takes a Procedure's name, and `review` takes two forms because the commands differ](0090-run-takes-a-procedures-name-and-review-takes-two-forms-because-the-commands-differ.md)
- **0091** · [A rehearsal's withheld Step is a member on that Step's row](0091-a-rehearsals-withheld-step-is-a-member-on-that-steps-row.md)
- **0092** · [A cancelled call drains, and the server catches no signal](0092-a-cancelled-call-drains-and-the-server-catches-no-signal.md)
- **0093** · [Orientation is a handshake field, and `hyper` writes no file to carry it](0093-orientation-is-a-handshake-field-and-hyper-writes-no-file-to-carry-it.md)
- **0094** · [The argument-less invocation writes the tree, and there is no `help`](0094-the-argument-less-invocation-writes-the-tree-and-there-is-no-help.md)
- **0095** · [`project` writes the orientation to `AGENTS.md`, and the handshake is not the only channel](0095-project-writes-the-orientation-to-agents-md-and-the-handshake-is-not-the-only-channel.md)
- **0096** · [The shape carried whole is the effectful one, and the worked example is not the acceptance task](0096-the-shape-carried-whole-is-the-effectful-one-and-the-example-is-not-the-acceptance-task.md)
- **0097** · [A `check`'s rows travel in the `text` block](0097-a-checks-rows-travel-in-the-text-block.md)
- **0098** · [An unknown flag names the flags that command takes](0098-an-unknown-flag-names-the-flags-that-command-takes.md)
- **0099** · [The acceptance harness is sealed, and the foraging was the blind `check`](0099-the-acceptance-harness-is-sealed-and-the-foraging-was-the-blind-check.md)
- **0100** · [A `review`'s page travels in the structured content](0100-a-reviews-page-travels-in-the-structured-content.md)
- **0101** · [A rule the orientation states is stated with its exception](0101-a-rule-the-orientation-states-is-stated-with-its-exception.md)
- **0102** · [A tool that declined answers no structured half](0102-a-tool-that-declined-answers-no-structured-half.md)
- **0103** · [A grant is the Target's, and the narrower one is a second declaration](0103-a-grant-is-the-targets-and-the-narrower-one-is-a-second-declaration.md)
- **0104** · [The acceptance fixture ships a Store](0104-the-acceptance-fixture-ships-a-store.md)
- **0105** · [The acceptance endpoint is a local TLS server, and no artefact trusts it](0105-the-acceptance-endpoint-is-a-local-tls-server-and-no-artefact-trusts-it.md)
- **0106** · [A Manifest is writable from the surface, and both costs were paid at the world](0106-a-manifest-is-writable-from-the-surface-and-both-costs-were-paid-at-the-world.md)
- **0107** · [A query string in `path:` is refused where it is written](0107-a-query-string-in-path-is-refused-where-it-is-written.md)
- **0108** · [A supplied response is not a call, and every rule it lifts was the call's](0108-a-supplied-response-is-not-a-call-and-every-rule-it-lifts-was-the-calls.md)
- **0109** · [The seal covers the output directory the harness writes](0109-the-seal-covers-the-output-directory-the-harness-writes.md)
- **0110** · [A Run is reachable from the surface, and the rehearsal is what recorded the pre-state](0110-a-run-is-reachable-from-the-surface-and-the-rehearsal-is-what-recorded-the-pre-state.md)
- **0111** · [Composition is not conditionable, and the shared check had to claim `mutate` to halt](0111-composition-is-not-conditionable-and-the-shared-check-had-to-claim-mutate-to-halt.md)
- **0112** · [The second Run skipped what the first did, and every revision it recorded is unresolvable](0112-the-second-run-skipped-what-the-first-did-and-every-revision-it-recorded-is-unresolvable.md)
- **0113** · [A listing over the record says where the record is](0113-a-listing-over-the-record-says-where-the-record-is.md)
- **0114** · [The rehearsal marker is the entry's, and `records` joins for it](0114-the-rehearsal-marker-is-the-entrys-and-records-joins-for-it.md)
- **0115** · [A rehearsal is the Comparison's subject where a caller names it as one](0115-a-rehearsal-is-the-comparisons-subject-where-a-caller-names-it-as-one.md)
- **0116** · [A Requirement halts, and claims nothing to do it](0116-a-requirement-halts-and-claims-nothing-to-do-it.md)
- **0117** · [A closed key set is stated as closed, and an invocation is a fixed block](0117-a-closed-key-set-is-stated-as-closed-and-an-invocation-is-a-fixed-block.md)
- **0118** · [The envelope cannot be trapped now that the orientation states it](0118-the-envelope-cannot-be-trapped-now-that-the-orientation-states-it.md)
- **0119** · [A revision resolves where the artefact was committed, and the loop commits](0119-a-revision-resolves-where-the-artefact-was-committed-and-the-loop-commits.md)
- **0120** · [The orientation taught the envelope, and the first Requirement was authored from one sentence](0120-the-orientation-taught-the-envelope-and-the-first-requirement-was-authored-from-one-sentence.md)
- **0121** · [An opaque `mutate` is unbounded, and no number clears the flag](0121-an-opaque-mutate-is-unbounded-and-no-number-clears-the-flag.md)
- **0122** · [A Requirement roots at any projected field, and the value goes on the line](0122-a-requirement-roots-at-any-projected-field-and-the-value-goes-on-the-line.md)
- **0123** · [The suite is run by a machine, and a prepared machine may not skip](0123-the-suite-is-run-by-a-machine-and-a-prepared-machine-may-not-skip.md)
- **0124** · [The second `change-window` run declined the bound, and a taught repair owes a run](0124-the-second-change-window-run-declined-the-bound-and-a-taught-repair-owes-a-run.md)
- **0125** · [The world answered for the first time, and the two `404`s differed only in the Kind](0125-the-world-answered-for-the-first-time-and-the-two-404s-differed-only-in-the-kind.md)
- **0126** · [A predicate over an Expansion holds of all of them, and an answer must name which](0126-a-predicate-over-an-expansion-holds-of-all-of-them-and-an-answer-must-name-which.md)
- **0127** · [A remedy may not assert what the answer could not establish](0127-a-remedy-may-not-assert-what-the-answer-could-not-establish.md)
- **0128** · [Third-party bytes entered a repository for the first time, and the ref recorded was the one typed](0128-third-party-bytes-entered-a-repository-for-the-first-time-and-the-ref-recorded-was-the-one-typed.md)
- **0129** · [The `destroy` landed inside the seal, and what held the hand-made monitors was not the Bound](0129-the-destroy-landed-inside-the-seal-and-what-held-the-hand-made-monitors-was-not-the-bound.md)
- **0130** · [The seal covers the home directory and the session comes back by name](0130-the-seal-covers-the-home-directory-and-the-session-comes-back-by-name.md)
- **0131** · [`project` wrote a digest for the first time, and the network contributed one scalar](0131-project-wrote-a-digest-for-the-first-time-and-the-network-contributed-one-scalar.md)
- **0132** · [The projected job ran on a runner, and the deepen step fetched the Store whole](0132-the-projected-job-ran-on-a-runner-and-the-deepen-step-fetched-the-store-whole.md)
- **0133** · [Three archives nobody had run carried, and the release stamps three of four dirty](0133-three-archives-nobody-had-run-carried-and-the-release-stamps-three-of-four-dirty.md)
- **0134** · [The deepen step names one ref, and what deepens the code branch is the clone's own boundary](0134-the-deepen-step-names-one-ref-and-what-deepens-the-code-branch-is-the-clones-own-boundary.md)
- **0135** · [The Comparison names its own failure and does not fail the job](0135-the-comparison-names-its-own-failure-and-does-not-fail-the-job.md)
- **0136** · [The release builds every platform before it publishes one](0136-the-release-builds-every-platform-before-it-publishes-one.md)
- **0137** · [A browser sets the attribute, and the shell runs what Finder offers to delete](0137-a-browser-sets-the-attribute-and-the-shell-runs-what-finder-offers-to-delete.md)
- **0138** · [A flagless build answers with the version the toolchain recorded](0138-a-flagless-build-answers-with-the-version-the-toolchain-recorded.md)
- **0139** · [The fixture delivered one tick in forty, and the gaps moved together across four files](0139-the-fixture-delivered-one-tick-in-forty-and-the-gaps-moved-together-across-four-files.md)
- **0140** · [The README holds the first read, and a corpus it points at carries an index](0140-the-readme-holds-the-first-read-and-a-corpus-it-points-at-carries-an-index.md)
- **0141** · [A release's body is a file in the tree the tag names](0141-a-releases-body-is-a-file-in-the-tree-the-tag-names.md)
