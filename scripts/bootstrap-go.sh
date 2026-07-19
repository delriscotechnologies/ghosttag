#!/usr/bin/env bash
set -euo pipefail

readonly GO_VERSION="1.26.5"

case "$(uname -m)" in
  x86_64|amd64)
    readonly GO_ARCH="amd64"
    readonly GO_SHA256="5c2c3b16caefa1d968a94c1daca04a7ca301a496d9b086e17ad77bb81393f053"
    ;;
  aarch64|arm64)
    readonly GO_ARCH="arm64"
    readonly GO_SHA256="fe4789e92b1f33358680864bbe8704289e7bb5fc207d80623c308935bd696d49"
    ;;
  *)
    printf 'Unsupported Linux architecture: %s\n' "$(uname -m)" >&2
    exit 1
    ;;
esac

readonly GO_ARCHIVE="go${GO_VERSION}.linux-${GO_ARCH}.tar.gz"
project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
archive_path="${project_root}/.tmp/downloads/${GO_ARCHIVE}"
download_path="${archive_path}.part"
toolchain_path="${project_root}/.tools/go"

mkdir -p "${project_root}/.tmp/downloads" "${project_root}/.tools"

verify_archive() {
  printf '%s  %s\n' "${GO_SHA256}" "$1" | sha256sum --check --status
}

if [[ -f "${archive_path}" ]] && ! verify_archive "${archive_path}"; then
  rm -f "${archive_path}"
fi

if [[ ! -f "${archive_path}" ]]; then
  # Start from a clean temporary file. A failed or corrupted partial download
  # must not poison later bootstrap attempts.
  rm -f "${download_path}"
  curl --proto '=https' --tlsv1.2 --fail --location --retry 3 \
    --output "${download_path}" \
    "https://go.dev/dl/${GO_ARCHIVE}"
  if ! verify_archive "${download_path}"; then
    rm -f "${download_path}"
    printf 'Go toolchain checksum verification failed.\n' >&2
    exit 1
  fi
  mv -f "${download_path}" "${archive_path}"
fi

# Never trust an existing ignored toolchain directory. Recreate it only from
# the checksum-verified official archive on every bootstrap.
rm -rf "${toolchain_path}"
tar -xzf "${archive_path}" -C "${project_root}/.tools"

actual_version="$("${toolchain_path}/bin/go" version)"
expected_version="go version go${GO_VERSION} linux/${GO_ARCH}"
if [[ "${actual_version}" != "${expected_version}" ]]; then
  rm -rf "${toolchain_path}"
  printf 'Unexpected Go toolchain version: %s\n' "${actual_version}" >&2
  exit 1
fi

printf '%s\n' "${actual_version}"
