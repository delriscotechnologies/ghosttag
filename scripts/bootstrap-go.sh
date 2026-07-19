#!/usr/bin/env bash
set -euo pipefail

readonly GO_VERSION="1.26.5"
readonly GO_ARCHIVE="go${GO_VERSION}.linux-amd64.tar.gz"
readonly GO_SHA256="5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053"

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
archive_path="${project_root}/.tmp/downloads/${GO_ARCHIVE}"
toolchain_path="${project_root}/.tools/go"

if [[ -x "${toolchain_path}/bin/go" ]]; then
  "${toolchain_path}/bin/go" version
  exit 0
fi

mkdir -p "${project_root}/.tmp/downloads" "${project_root}/.tools"
curl --fail --location --continue-at - --output "${archive_path}" \
  "https://go.dev/dl/${GO_ARCHIVE}"

printf '%s  %s\n' "${GO_SHA256}" "${archive_path}" | sha256sum --check --status
tar -xzf "${archive_path}" -C "${project_root}/.tools"
"${toolchain_path}/bin/go" version
