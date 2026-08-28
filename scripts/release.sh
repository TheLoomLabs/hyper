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
# that stamped nothing would publish a binary reporting `unknown` under a tag
# that names a version.
stamp="github.com/TheLoomLabs/hyper/internal/version.Version"

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
mkdir -p "$outdir"
outdir=$(cd "$outdir" && pwd)

for platform in "${platforms[@]}"; do
	case $platform in
	x86_64-linux) goos=linux goarch=amd64 ;;
	aarch64-linux) goos=linux goarch=arm64 ;;
	x86_64-darwin) goos=darwin goarch=amd64 ;;
	aarch64-darwin) goos=darwin goarch=arm64 ;;
	*)
		echo "release.sh: no build is defined for $platform" >&2
		exit 2
		;;
	esac

	# CGO off and -trimpath: the binary is fetched by a runner that shares no
	# libc with the machine that built it, and a path from this filesystem is
	# not a fact about the release.
	staging=$(mktemp -d)
	trap 'rm -rf "$staging"' EXIT
	CGO_ENABLED=0 GOOS=$goos GOARCH=$goarch go build \
		-C "$root" \
		-trimpath \
		-ldflags "-X $stamp=$version" \
		-o "$staging/hyper" \
		./cmd/hyper
	tar -czf "$outdir/hyper-$version-$platform.tar.gz" -C "$staging" hyper
	rm -rf "$staging"
	trap - EXIT
done

# `sha256sum` from inside the directory, so every line names a bare filename —
# the name the artefact is published under, and the name `project` looks for.
(cd "$outdir" && sha256sum "hyper-$version"-*.tar.gz >checksums.txt)
