#!/usr/bin/env bash
# The three log directories the snapshot task names, with something in each.
set -euo pipefail
repo=${1:?usage: snapshot-lifecycle.setup.sh <repository>}
for dir in app web db; do
	mkdir -p "$repo/logs/$dir"
	printf 'started\nserved 200\nstopped\n' >"$repo/logs/$dir/current.log"
done
