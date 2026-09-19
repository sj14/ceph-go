#!/bin/sh
set -eu

readonly ceph_repository=https://github.com/ceph/ceph
readonly releases_url=https://raw.githubusercontent.com/ceph/ceph/main/doc/releases/releases.yml

version=$(
  curl -fsSL "$releases_url" |
    awk '
      /^releases:/ { in_releases = 1; next }
      /^development:/ { in_releases = 0 }
      in_releases && $1 == "-" && $2 == "version:" {
        gsub(/[\047\"]/, "", $3)
        print $3
      }
    ' |
    awk -F. '
      NR == 1 || $1 > major ||
      ($1 == major && $2 > minor) ||
      ($1 == major && $2 == minor && $3 > patch) {
        version = $0
        major = $1
        minor = $2
        patch = $3
      }
      END { print version }
    '
)

if [ -z "$version" ]; then
  echo 'could not determine the latest stable Ceph release' >&2
  exit 1
fi

release_type=$(curl -fsSL "$ceph_repository/raw/v${version}/src/ceph_release" | sed -n '3p')
if [ "$release_type" != stable ]; then
  echo "v${version} declares release type '${release_type}', expected 'stable'" >&2
  exit 1
fi

printf '%s\n' "$version"
