#!/bin/sh
set -eu
dir=$(cd "$(dirname "$0")" && pwd)
ext=
case $(uname -s) in
Linux) os=linux ;;
Darwin) os=darwin ;;
MINGW* | MSYS* | CYGWIN*) os=windows ext=.exe ;;
*) os=$(uname -s | tr '[:upper:]' '[:lower:]') ;;
esac
case $(uname -m) in
x86_64 | amd64) arch=amd64 ;;
aarch64 | arm64) arch=arm64 ;;
*) arch=$(uname -m) ;;
esac
bin="$dir/bin/go-workspace-$os-$arch$ext"
if [ -x "$bin" ]; then
	exec "$bin" "$@"
fi
exec go -C "$dir" run . "$@"
