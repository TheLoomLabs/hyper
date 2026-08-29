#!/usr/bin/env bash
# The roster the onboarding task names, the three tenants on it, and the two
# templates a tenant is built out of.
#
# **The point of this task is the second Run** (issue #224). Repeatability is
# the hardest thing in the model to hold from the orientation alone — every one
# of its rules is about the time after this one — and a transcript that runs a
# Procedure once cannot say whether an agent holds any of it. So the outcome the
# task asks for cannot be reached by one Run, and the three values are told
# apart by what each of them does on the second:
#
#   skip-if-recorded  the shape that works. Three names skip, `hooli` calls,
#                     the Step is *ran* and its identity set holds all four.
#   run-once          Refuses `run-once-recorded` on what the Journal holds for
#                     the Step, checked at Expansion before the first call. The
#                     value decides per Step and not per Record, so the Refusal
#                     is wholesale and terminal — `hooli` is never reached.
#   repeatable        completes, and copies over three `tenant.conf` files the
#                     task has just said are edited by hand. Nothing refuses it;
#                     the transcript is the only place it shows.
#
# **The Expansion is what makes the per-Record rule visible.** The roster is on
# disk and the population is in the Procedure's `values:` list (§3), so *a fourth
# name arrives* is an edit to the artefact — which is the whole of what makes the
# second Run not a repeat of the first, and which lands in the Comparison's
# `THE CODE MOVED` table as the list growing from three members to four. A task
# whose second Run re-ran one Step would show a skip; this one shows the rule.
#
# **`skel/shared.conf` is the trap, and it is left open.** The task says that
# file is written once and edited in place afterwards, which is a run-once Step;
# and it asks for the whole thing on a 06:00 clock. Those two are
# `cadence-run-once` (§4, ADR-0038), refused at `check` before anything runs, and
# what is authored instead is the split the rule itself describes: the run-once
# Step in a Procedure run by hand, the recurring one carrying the Cadence. An
# agent that leaves them in one Procedure and drops the Cadence meets the same
# family one Run later, `run-once-recorded` taking the second Run down before the
# provisioning Step is reached.
#
# **The clock cannot do the thing the task says it is for, and that is the third
# trap.** The Cadence is asked for so that *a name somebody adds to the roster
# overnight is set up before anyone is awake*, and no Cadence delivers that: the
# population is the Procedure's `values:` list, an `assets:` selector under
# `skip-if-recorded` being `skip-if-recorded-unreachable` (§4), so every
# occurrence re-runs the same authored names and skips every one of them. A name
# reaching the roster still costs an artefact edit and a review, which is this
# tool working rather than failing. The task's last paragraph invites that answer
# without naming it.
#
# Two consequences of the Cadence worth knowing before reading a transcript.
# `check` then wants a projection — `projection-stale`, repaired by `hyper
# project`, which the orientation says in as many words — and the workflow that
# writes carries `hyper.yaml`'s digest, which in this fixture is sixty-four
# zeros. The orientation says that too, one line further on. Neither is a defect
# and both are things an agent may or may not report.
#
# Completed by hand once before it landed: `check` clean, three Runs (the
# bootstrap, then the Procedure twice), `hyper records` naming which Run last
# wrote each of the four, and `hyper changes` rendering one `created` row against
# a `THE CODE MOVED` block holding the grown list.
set -euo pipefail
repo=${1:?usage: tenant-onboarding.setup.sh <repository>}

mkdir -p "$repo/tenants" "$repo/skel"
printf 'acme\nglobex\ninitech\n' >"$repo/tenants/roster"
printf 'region = eu-west-1\nretention = 30d\n' >"$repo/skel/shared.conf"
printf 'plan = standard\nquota_gb = 20\n' >"$repo/skel/tenant.conf"
