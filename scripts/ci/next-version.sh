#!/usr/bin/env bash
set -euo pipefail

# Prints the next release version as GitHub step outputs:
#   name=X.Y.Z   code=<build number>
# The latest vX.Y.Z tag is incremented: BUMP (patch|minor|major) when set,
# otherwise [release:major] or [release:minor] in COMMIT_MESSAGES, else patch.
# Without any tag the first release is 0.1.0. A commit that already carries a
# release tag keeps its version, so re-running a release publishes the same one.

TAG_PATTERN='v[0-9]*.[0-9]*.[0-9]*'

existing="$(git tag --points-at HEAD --list "$TAG_PATTERN" | sort -V | tail -n 1)"
latest="$(git tag --list "$TAG_PATTERN" --sort=-v:refname | head -n 1)"

if [[ -n "$existing" ]]; then
  version="${existing#v}"
elif [[ -z "$latest" ]]; then
  version="0.1.0"
else
  bump="${BUMP:-}"
  if [[ -z "$bump" ]]; then
    case "${COMMIT_MESSAGES:-}" in
      *"[release:major]"*) bump=major ;;
      *"[release:minor]"*) bump=minor ;;
      *) bump=patch ;;
    esac
  fi
  IFS=. read -r major minor patch <<< "${latest#v}"
  case "$bump" in
    major) major=$((major + 1)); minor=0; patch=0 ;;
    minor) minor=$((minor + 1)); patch=0 ;;
    patch) patch=$((patch + 1)) ;;
    *) echo "Unknown bump: $bump" >&2; exit 2 ;;
  esac
  version="$major.$minor.$patch"
fi

IFS=. read -r major minor patch <<< "$version"
# Android versionCode and CFBundleVersion must grow with every release.
if (( minor >= 1000 || patch >= 1000 )); then
  echo "Minor and patch must stay below 1000 to fit the build number" >&2
  exit 2
fi
echo "name=$version"
echo "code=$((major * 1000000 + minor * 1000 + patch))"
