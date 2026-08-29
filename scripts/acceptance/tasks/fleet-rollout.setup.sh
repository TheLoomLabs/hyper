#!/usr/bin/env bash
# The three machines the rollout task names, each with the version it is
# currently on, and the version they are all supposed to be on.
#
# Two of them are behind, and behind by different amounts, so *what was each
# machine on* is three different answers rather than one repeated. The versions
# carry the trailing newline a file on a machine has, which is the shape a
# projected `stdout` actually arrives in.
#
# **What the Store is the only copy of, and what it is not.** Once the rollout
# has run every file under `fleet/` reads `1.4.2` and the working tree stops
# saying what any of them was on. That is not the last copy: this runs before
# `run.sh` commits, so the starting versions are in that commit, and an agent
# that reads `fleet/` before it runs has seen them anyway. What exists nowhere
# but the Store is the **account of the rollout** — which Records it minted,
# which of them it did rather than merely saw, and under what — and the task
# asks for that as well, which is what keeps *ran it* and *read what it did*
# two claims a transcript can be scored on separately.
set -euo pipefail
repo=${1:?usage: fleet-rollout.setup.sh <repository>}

mkdir -p "$repo/fleet"
printf '1.4.2\n' >"$repo/fleet/wanted"
while read -r machine version; do
	mkdir -p "$repo/fleet/$machine"
	printf '%s\n' "$version" >"$repo/fleet/$machine/installed"
done <<-VERSIONS
	web-01 1.4.2
	web-02 1.3.9
	db-01  1.2.0
VERSIONS
