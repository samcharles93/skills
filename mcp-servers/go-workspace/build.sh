#!/bin/sh
# Rebuilds the shipped go-workspace binaries. Run after any change to the server.
set -eu
cd "$(dirname "$0")"
for target in linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64; do
	os=${target%/*}
	arch=${target#*/}
	ext=
	[ "$os" = windows ] && ext=.exe
	out="bin/go-workspace-$os-$arch$ext"
	CGO_ENABLED=0 GOOS=$os GOARCH=$arch go build -trimpath -ldflags='-s -w' -o "$out" .
	echo "built $out"
done
