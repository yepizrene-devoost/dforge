#!/usr/bin/env bash
#
# installer-contract.sh — contract test for scripts/install.sh.
#
# Drives the installer against a local fixture "release": a mock curl serves
# fixture files by URL basename (no network) and records every requested URL,
# while a mock uname drives the OS/arch matrix. The cases assert the exact
# asset names the installer builds from GoReleaser's template, that a good
# archive installs, and that a corrupted archive, a missing checksums.txt or an
# unsupported platform fail hard without touching the installed binary.
#
set -euo pipefail

ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
TMP=$(mktemp -d "${TMPDIR:-/tmp}/dforge-installer-test.XXXXXX")
trap 'rm -rf "$TMP"' EXIT INT TERM HUP
FIXTURE="$TMP/fixture"
BIN="$TMP/mock-bin"
INSTALL="$TMP/install"
LOG="$TMP/curl.log"
HOME_DIR="$TMP/home"
REPO_URL="https://github.com/yepizrene-devoost/dforge"
mkdir -p "$FIXTURE" "$BIN" "$INSTALL" "$HOME_DIR"

fail_case() {
    echo "FAIL: $*" >&2
    exit 1
}

# Mock uname: platform matrix driven by INSTALLER_UNAME_S / INSTALLER_UNAME_M.
cat >"$BIN/uname" <<'EOF'
#!/bin/sh
case "$1" in
  -s) printf '%s\n' "${INSTALLER_UNAME_S:-Linux}" ;;
  -m) printf '%s\n' "${INSTALLER_UNAME_M:-x86_64}" ;;
  *) exit 1 ;;
esac
EOF

# Mock curl: serves INSTALLER_FIXTURE_DIR by URL basename, records each
# requested URL in INSTALLER_CURL_LOG, and answers the /releases/latest
# redirect probe (-w) with INSTALLER_LATEST_REDIRECT instead of copying.
cat >"$BIN/curl" <<'EOF'
#!/bin/sh
out=""
url=""
want_effective=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    -o) out="$2"; shift 2 ;;
    -w) want_effective=1; shift 2 ;;
    -*) shift ;;
    *) url="$1"; shift ;;
  esac
done
[ -n "${INSTALLER_CURL_LOG:-}" ] && printf '%s\n' "$url" >>"$INSTALLER_CURL_LOG"
if [ "$want_effective" = 1 ]; then
  printf '%s' "${INSTALLER_LATEST_REDIRECT:-https://github.com/yepizrene-devoost/dforge/releases/tag/v1.2.3}"
  exit 0
fi
[ -n "$out" ] && [ -n "$url" ] || exit 90
cp "$INSTALLER_FIXTURE_DIR/${url##*/}" "$out"
EOF
chmod +x "$BIN/uname" "$BIN/curl"

sha256() {
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$1" | awk '{ print $1 }'
    else
        shasum -a 256 "$1" | awk '{ print $1 }'
    fi
}

UNAME_S="Linux"
UNAME_M="x86_64"
LATEST_REDIRECT="$REPO_URL/releases/tag/v1.2.3"
DFORGE_VERSION="v1.2.3"

run_installer() {
    PATH="$BIN:$PATH" \
    HOME="$HOME_DIR" \
    DFORGE_INSTALL_DIR="$INSTALL" \
    DFORGE_VERSION="$DFORGE_VERSION" \
    INSTALLER_UNAME_S="$UNAME_S" \
    INSTALLER_UNAME_M="$UNAME_M" \
    INSTALLER_LATEST_REDIRECT="$LATEST_REDIRECT" \
    INSTALLER_CURL_LOG="$LOG" \
    INSTALLER_FIXTURE_DIR="$FIXTURE" \
    sh "$ROOT/scripts/install.sh"
}

# make_archive <asset-name> <payload-content>: one-binary tarball plus a
# checksums.txt naming that asset, GoReleaser's "<sha256>  <name>" shape.
make_archive() {
    _asset="$1"
    _content="$2"
    _payload="$TMP/payload"
    rm -rf "$_payload"
    mkdir -p "$_payload"
    printf '%s\n' "$_content" >"$_payload/dforge"
    chmod +x "$_payload/dforge"
    tar -czf "$FIXTURE/$_asset" -C "$_payload" dforge
    printf '%s  %s\n' "$(sha256 "$FIXTURE/$_asset")" "$_asset" >"$FIXTURE/checksums.txt"
}

expect_failure() {
    # expect_failure <reason> <output-file>: the run must exit non-zero.
    if run_installer >"$2" 2>&1; then
        fail_case "$1"
    fi
}

# --- Asset name construction per OS/arch (GoReleaser's template) -------------

for combo in "Linux:x86_64" "Linux:arm64" "Darwin:x86_64" "Darwin:arm64"; do
    UNAME_S="${combo%%:*}"
    UNAME_M="${combo##*:}"
    asset="dforge_${UNAME_S}_${UNAME_M}.tar.gz"
    make_archive "$asset" "binary for $combo"
    : >"$LOG"
    run_installer >"$TMP/matrix.out" 2>&1 || {
        fail_case "$combo install failed: $(tail -n 3 "$TMP/matrix.out")"
    }
    grep -Fxq "$REPO_URL/releases/download/v1.2.3/$asset" "$LOG" \
        || fail_case "$combo did not request $asset (log: $(tr '\n' ' ' <"$LOG"))"
    grep -Fxq "$REPO_URL/releases/download/v1.2.3/checksums.txt" "$LOG" \
        || fail_case "$combo did not request checksums.txt"
    if grep -Eq 'amd64|i386' "$LOG"; then
        fail_case "$combo requested a non-GoReleaser arch name (log: $(tr '\n' ' ' <"$LOG"))"
    fi
    [ "$(cat "$INSTALL/dforge")" = "binary for $combo" ] \
        || fail_case "$combo installed the wrong payload"
    printf 'ok - asset name %s/%s -> %s\n' "$UNAME_S" "$UNAME_M" "$asset"
done

# The install directory is never on PATH in this fixture, so the run must
# print the PATH hint, and the verified checksum must have been reported.
grep -Fq "is not on your PATH yet" "$TMP/matrix.out" \
    || fail_case "no PATH hint after install"
grep -Fq "Checksum verified" "$TMP/matrix.out" \
    || fail_case "checksum verification was not reported"
printf 'ok - checksum verification and PATH hint\n'

# --- Latest-release resolution (no pinned version) ---------------------------

UNAME_S="Linux"
UNAME_M="x86_64"
make_archive "dforge_Linux_x86_64.tar.gz" "latest binary"
: >"$LOG"
DFORGE_VERSION=""
run_installer >"$TMP/latest.out" 2>&1 || fail_case "latest-release install failed"
[ "$(sed -n 1p "$LOG")" = "$REPO_URL/releases/latest" ] \
    || fail_case "latest release was not resolved via /releases/latest first"
grep -Fxq "$REPO_URL/releases/download/v1.2.3/dforge_Linux_x86_64.tar.gz" "$LOG" \
    || fail_case "resolved tag was not used for the asset URL"
[ "$(cat "$INSTALL/dforge")" = "latest binary" ] \
    || fail_case "latest-release install produced the wrong payload"
printf 'ok - latest release resolution\n'

LATEST_REDIRECT="$REPO_URL/releases/tag/badtag"
: >"$LOG"
expect_failure "malformed latest tag accepted" "$TMP/badtag.out"
grep -Fq "Unexpected release tag" "$TMP/badtag.out" \
    || fail_case "malformed tag failed for the wrong reason: $(tail -n 2 "$TMP/badtag.out")"
LATEST_REDIRECT="$REPO_URL/releases/tag/v1.2.3"
DFORGE_VERSION="v1.2.3"
printf 'ok - malformed latest tag rejected\n'

# --- Corrupted archive must fail before extraction ---------------------------

make_archive "dforge_Linux_x86_64.tar.gz" "new binary"
# Tamper with the archive after checksums.txt was written: verification must
# catch it and leave the installed binary untouched.
printf 'tamper' >>"$FIXTURE/dforge_Linux_x86_64.tar.gz"
printf 'old binary\n' >"$INSTALL/dforge"
expect_failure "corrupted archive accepted" "$TMP/corrupt.out"
grep -Fq "Checksum mismatch" "$TMP/corrupt.out" \
    || fail_case "corruption failed for the wrong reason: $(tail -n 2 "$TMP/corrupt.out")"
[ "$(cat "$INSTALL/dforge")" = "old binary" ] \
    || fail_case "corrupted download changed the installed binary"
printf 'ok - corrupted archive rejected, install untouched\n'

# --- Missing checksums.txt must fail ----------------------------------------

make_archive "dforge_Linux_x86_64.tar.gz" "new binary"
rm -f "$FIXTURE/checksums.txt"
printf 'old binary\n' >"$INSTALL/dforge"
expect_failure "missing checksums.txt accepted" "$TMP/nocheck.out"
grep -Fq "Download failed" "$TMP/nocheck.out" \
    || fail_case "missing checksums.txt failed for the wrong reason: $(tail -n 2 "$TMP/nocheck.out")"
[ "$(cat "$INSTALL/dforge")" = "old binary" ] \
    || fail_case "missing checksums.txt changed the installed binary"
printf 'ok - missing checksums.txt rejected\n'

# --- Unsupported OS and architecture must fail before download --------------

make_archive "dforge_Linux_x86_64.tar.gz" "new binary"
UNAME_S="FreeBSD"
expect_failure "unsupported OS accepted" "$TMP/os.out"
grep -Fq "Unsupported operating system" "$TMP/os.out" \
    || fail_case "unsupported OS failed for the wrong reason: $(tail -n 2 "$TMP/os.out")"
UNAME_S="Linux"
UNAME_M="riscv64"
expect_failure "unsupported architecture accepted" "$TMP/arch.out"
grep -Fq "Unsupported architecture" "$TMP/arch.out" \
    || fail_case "unsupported architecture failed for the wrong reason: $(tail -n 2 "$TMP/arch.out")"
UNAME_M="x86_64"
printf 'ok - unsupported OS and architecture rejected\n'

# --- Malformed pinned version must fail -------------------------------------

DFORGE_VERSION="../bad"
expect_failure "malformed version accepted" "$TMP/version.out"
grep -Fq "Invalid DFORGE_VERSION" "$TMP/version.out" \
    || fail_case "malformed version failed for the wrong reason: $(tail -n 2 "$TMP/version.out")"
DFORGE_VERSION="v1.2.3"
printf 'ok - malformed pinned version rejected\n'

# --- Static release-contract checks -----------------------------------------

# The asset names must keep matching GoReleaser's archive template, and the
# manifest name must stay `checksums.txt` (.goreleaser.yaml's checksum block).
grep -Fq 'if eq .Arch "amd64" }}x86_64' "$ROOT/.goreleaser.yaml" \
    || fail_case ".goreleaser.yaml no longer maps amd64 to x86_64"
grep -Fq 'name_template: checksums.txt' "$ROOT/.goreleaser.yaml" \
    || fail_case ".goreleaser.yaml no longer publishes checksums.txt"
grep -Fq 'scripts/install.sh' "$ROOT/.goreleaser.yaml" \
    || fail_case ".goreleaser.yaml does not publish scripts/install.sh"
grep -Fq 'scripts/install.ps1' "$ROOT/.goreleaser.yaml" \
    || fail_case ".goreleaser.yaml does not publish scripts/install.ps1"
# The Windows installer must consume the same manifest with Get-FileHash.
grep -Fq "checksumName = 'checksums.txt'" "$ROOT/scripts/install.ps1" \
    || fail_case "scripts/install.ps1 does not use checksums.txt"
grep -Fq 'Get-FileHash -LiteralPath $archivePath -Algorithm SHA256' "$ROOT/scripts/install.ps1" \
    || fail_case "scripts/install.ps1 does not verify SHA-256 with Get-FileHash"
printf 'ok - static release contract\n'

printf 'POSIX installer contract passed\n'
