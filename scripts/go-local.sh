#!/usr/bin/env bash
set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

export GOROOT="${project_root}/.tools/go"
export PATH="${GOROOT}/bin:${PATH}"
export GOBIN="${project_root}/.tools/bin"
export GOCACHE="${project_root}/.cache/go-build"
export GOMODCACHE="${project_root}/.cache/go-mod"
export GOTMPDIR="${project_root}/.tmp"
export TMPDIR="${project_root}/.tmp"
export GOTELEMETRY="off"
export GOTOOLCHAIN="local"

mkdir -p "${GOBIN}" "${GOCACHE}" "${GOMODCACHE}" "${GOTMPDIR}" "${TMPDIR}"
exec "${GOROOT}/bin/go" "$@"
