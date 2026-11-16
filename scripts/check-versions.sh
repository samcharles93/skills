#!/bin/sh
# Version alignment gate. Every plugin manifest and every skill VERSION file
# must carry the same version, so a published plugin never advertises a
# version its skills do not have. Run directly, or from the hooks in
# .githooks/ (enable with: git config core.hooksPath .githooks).
#
#   scripts/check-versions.sh              check the working tree
#   scripts/check-versions.sh --tag v2.2.0 check the working tree and a tag name

set -eu

root=$(git rev-parse --show-toplevel)
cd "$root"

die() {
	printf 'version check: %s\n' "$*" >&2
	exit 1
}

json_version() {
	grep -m1 '"version"' "$1" | sed 's/.*: *"\([^"]*\)".*/\1/'
}

claude=$(json_version .claude-plugin/plugin.json)
codex=$(json_version .codex-plugin/plugin.json)

[ -n "$claude" ] || die "no version found in .claude-plugin/plugin.json"
[ -n "$codex" ] || die "no version found in .codex-plugin/plugin.json"
[ "$claude" = "$codex" ] ||
	die ".claude-plugin/plugin.json is $claude but .codex-plugin/plugin.json is $codex"

for file in skills/*/VERSION; do
	[ -e "$file" ] || continue
	version=$(sed 's/^version *//; s/^v//' "$file")
	[ "$version" = "$claude" ] ||
		die "$file is $version but the plugin manifests are $claude"
done

if [ "${1:-}" = --tag ]; then
	tag=${2:-}
	case "$tag" in
	"v$claude" | "v$claude"-*) ;;
	*) die "tag '$tag' does not match version v$claude" ;;
	esac
fi

printf 'version check: all manifests and skill VERSION files at %s\n' "$claude"
