#!/usr/bin/env bash
#
# The release build: the two files a `hyper` release publishes, produced from a
# version and nothing else (§11, ADR-0020, issue #191).
#
# It is a script rather than lines inside the workflow so that the bytes a tag
# runs are the bytes a case runs and the bytes a human runs. What `project`
# freezes is the checksum published beside the artefact, and a publication that
# drifted from the template the binary holds would be discovered by an author
# whose `hyper project` Refuses `release-artefact-absent` — which is a defect
# this repository has no way to notice for itself unless the release process is
# a thing the tests can invoke.
#
#   scripts/release.sh <version> <output-directory> [platform...]
#
# The version is the release's own, without the leading `v` the tag carries:
# the tag is `v1.4.0` and every filename under it says `1.4.0`, the two being
# the shape §11 states for the release artefact URL. Platforms default to the
# published set, and a caller naming one pays for one build.
#
# What it writes into the output directory is one `.tar.gz` per platform, each
# holding the binary as `hyper` at its root — which is what the install step in
# a generated workflow untars and invokes — and one `checksums.txt` naming them
# all, in `sha256sum`'s own output shape, which is what `hyper project` reads
# one line of (§10, internal/release).
#
# **It writes none of them until every platform has been built**, and that
# ordering is load-bearing rather than tidy. Go stamps `vcs.modified` from
# `git status` in the module root, and the output directory is inside that root
# wherever a release is actually cut — the workflow hands it `dist` and
# docs/build/releasing.md hands a person the same. Archiving as it went put the
# first platform's tarball in the checkout and stamped every build after it from
# a tree nobody had edited, which is what `v0.0.1-alpha` published on three of
# its four archives (issue #261, ADR-0136). What the ordering holds is that no
# build sees another build's output; an output directory a previous run left
# files in is a tree already modified when this script starts, and every build
# is then stamped `true` alike.
set -euo pipefail

version=${1:?usage: release.sh <version> <output-directory> [platform...]}
outdir=${2:?usage: release.sh <version> <output-directory> [platform...]}
platforms=("${@:3}")

if [ "${version#v}" != "$version" ]; then
	echo "release.sh: the version is ${version#v} and the tag is $version; the leading v belongs to the tag alone" >&2
	exit 2
fi

# The published set. It is the release process's business rather than the
# tool's — §11 fixes one platform, the one `runs-on` names and the one the
# projection's compiled-in template fetches, and says that what a release
# publishes beyond that is no property of the binary. The other three are the
# set a laptop can also download — two Linux, two macOS, and no Windows, which
# no `runs-on` here names and nobody has asked for.
if [ ${#platforms[@]} -eq 0 ]; then
	platforms=(x86_64-linux aarch64-linux x86_64-darwin aarch64-darwin)
fi

# The one flag that stamps. It writes internal/version's `Version`, which is a
# `var` for exactly this reason: the linker cannot write a `const`, and a build
# that stamped nothing would publish a binary naming a version nobody chose.
#
# `-X` names the symbol by import path and the linker checks it against nothing,
# so a typo here is ignored in silence. Since #263 it is also invisible in the
# published bytes — a release is cut from a clean checkout at the tag, and the
# module version the binary would fall back to is the same string this flag
# writes. What holds the spelling is
# `TestStamp_TheReleaseScriptNamesTheSymbolThatWorks`, which compares it against
# a symbol the suite builds with and reads back out of a running binary.
stamp="github.com/TheLoomLabs/hyper/internal/version.Version"

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$outdir"
outdir=$(cd "$outdir" && pwd)

# The builds, all of them, and nothing is written to `$outdir` here: no build in
# this loop can see another build's output, which is the ordering the header
# states and the reason for it. The staging directory is `mktemp`'s, outside the
# tree Go stamps from.
staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT

for platform in "${platforms[@]}"; do
	case $platform in
	x86_64-linux) goos=linux goarch=amd64 ;;
	aarch64-linux) goos=linux goarch=arm64 ;;
	x86_64-darwin) goos=darwin goarch=amd64 ;;
	aarch64-darwin) goos=darwin goarch=arm64 ;;
	*)
		# Named before the first archive is written, so a set with a typo
		# in it publishes nothing rather than part of a release.
		echo "release.sh: no build is defined for $platform" >&2
		exit 2
		;;
	esac

	# CGO off and -trimpath: the binary is fetched by a runner that shares no
	# libc with the machine that built it, and a path from this filesystem is
	# not a fact about the release.
	mkdir -p "$staging/$platform"
	CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
		-C "$root" \
		-trimpath \
		-ldflags "-X $stamp=$version" \
		-o "$staging/$platform/hyper" \
		./cmd/hyper
done

# The publication, once every binary exists. Each archive holds the binary as
# `hyper` at its root, which is what the install step in a generated workflow
# untars and invokes.
for platform in "${platforms[@]}"; do
	tar -czf "$outdir/hyper-$version-$platform.tar.gz" -C "$staging/$platform" hyper
done

# `sha256sum` from inside the directory, so every line names a bare filename —
# the name the artefact is published under, and the name `project` looks for.
(cd "$outdir" && sha256sum "hyper-$version"-*.tar.gz >checksums.txt)
