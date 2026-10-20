#!/usr/bin/env bash
set -euo pipefail

fail() { printf '%s\n' "$*" >&2; exit 1; }
[[ $# -le 1 ]] || fail "usage: setup_linux_toolchain.sh [temporary-bin-directory]"
[[ "$(uname -s)" == Linux && "$(uname -m)" == x86_64 ]] ||
  fail "Linux toolchain provisioning requires Ubuntu 24.04 x86_64"
[[ -r /etc/os-release ]] || fail "cannot identify /etc/os-release"
source /etc/os-release
[[ "$ID" == ubuntu && "$VERSION_ID" == 24.04 ]] ||
  fail "Linux toolchain reference requires Ubuntu 24.04; found ${PRETTY_NAME:-unknown}"

packages=(gcc-13 clang-18 libc6-dev binutils libclang-rt-18-dev)
missing=()
for package in "${packages[@]}"; do
  if [[ "$(dpkg-query -W -f='${Status}' "$package" 2>/dev/null || true)" != 'install ok installed' ]]; then
    missing+=("$package")
  fi
done
if [[ ${#missing[@]} -gt 0 ]]; then
  privilege=()
  if [[ $EUID -ne 0 ]]; then privilege=(sudo); fi
  "${privilege[@]}" apt-get update
  "${privilege[@]}" apt-get install -y --no-install-recommends "${missing[@]}"
fi
[[ -x /usr/bin/gcc-13 && -x /usr/bin/clang-18 ]] ||
  fail "versioned compilers are missing; reinstall gcc-13 and clang-18"

if [[ $# -eq 1 ]]; then
  bin=$1
  [[ "$bin" = /* && "$bin" != *$'\n'* ]] || fail "bin directory must be an absolute path without newlines"
  mkdir -p -- "$bin"
else
  bin=$(mktemp -d "${TMPDIR:-/tmp}/fairyc-toolchain.XXXXXX")
fi
ln -sfn -- /usr/bin/gcc-13 "$bin/gcc"
ln -sfn -- /usr/bin/clang-18 "$bin/clang"
if [[ -n ${GITHUB_PATH:-} ]]; then
  printf '%s\n' "$bin" >> "$GITHUB_PATH"
fi
printf 'Toolchain ready. Add %s to PATH for this shell.\n' "$bin"
"$bin/gcc" --version
"$bin/clang" --version
getconf GNU_LIBC_VERSION
