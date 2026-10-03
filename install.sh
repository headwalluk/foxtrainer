#!/bin/sh
# Install the foxtrainer binary from GitHub Releases into ~/.local/bin (no root needed).
#
#   curl -fsSL https://raw.githubusercontent.com/headwalluk/foxtrainer/main/install.sh | sh
#
# Options (environment variables):
#   FOXTRAINER_VERSION       release tag to install, e.g. v1.0.0 (default: the latest release)
#   FOXTRAINER_INSTALL_DIR   where to put the binary (default: ${HOME}/.local/bin)

set -eu

REPOSITORY="headwalluk/foxtrainer"

# fail prints an error and exits.
fail() {
	printf 'foxtrainer install: %s\n' "$1" >&2
	exit 1
}

# download fetches a URL to a file with curl or wget.
download() {
	SOURCE_URL="$1"
	DESTINATION_FILE="$2"

	if command -v curl >/dev/null 2>&1; then
		curl -fsSL --proto '=https' --tlsv1.2 -o "${DESTINATION_FILE}" "${SOURCE_URL}" || fail "download failed: ${SOURCE_URL}"
	elif command -v wget >/dev/null 2>&1; then
		wget -q --https-only -O "${DESTINATION_FILE}" "${SOURCE_URL}" || fail "download failed: ${SOURCE_URL}"
	else
		fail "curl or wget is required"
	fi
}

# sha256_of prints a file's SHA-256 digest.
sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif command -v shasum >/dev/null 2>&1; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		fail "sha256sum or shasum is required to verify the download"
	fi
}

main() {
	[ -n "${HOME:-}" ] || fail "HOME is not set"

	REQUESTED_VERSION="${FOXTRAINER_VERSION:-latest}"
	INSTALL_DIR="${FOXTRAINER_INSTALL_DIR:-${HOME}/.local/bin}"

	OPERATING_SYSTEM="$(uname -s)"
	case "${OPERATING_SYSTEM}" in
		Linux) TARGET_OS="linux" ;;
		*) fail "pre-built binaries are Linux only; on ${OPERATING_SYSTEM}, build from source: go install github.com/${REPOSITORY}/cmd/foxtrainer@latest" ;;
	esac

	MACHINE="$(uname -m)"
	case "${MACHINE}" in
		x86_64 | amd64) TARGET_ARCH="amd64" ;;
		aarch64 | arm64) TARGET_ARCH="arm64" ;;
		*) fail "no pre-built binary for ${MACHINE}; build from source: go install github.com/${REPOSITORY}/cmd/foxtrainer@latest" ;;
	esac

	if [ "${REQUESTED_VERSION}" = "latest" ]; then
		RELEASE_URL="https://github.com/${REPOSITORY}/releases/latest/download"
	else
		RELEASE_URL="https://github.com/${REPOSITORY}/releases/download/${REQUESTED_VERSION}"
	fi

	ARCHIVE_NAME="foxtrainer_${TARGET_OS}_${TARGET_ARCH}"

	WORK_DIR="$(mktemp -d)"
	trap 'rm -rf "${WORK_DIR}"' EXIT INT TERM

	printf 'Downloading foxtrainer (%s, %s/%s)...\n' "${REQUESTED_VERSION}" "${TARGET_OS}" "${TARGET_ARCH}"
	download "${RELEASE_URL}/${ARCHIVE_NAME}.tar.gz" "${WORK_DIR}/${ARCHIVE_NAME}.tar.gz"
	download "${RELEASE_URL}/SHA256SUMS" "${WORK_DIR}/SHA256SUMS"

	EXPECTED_DIGEST="$(awk -v NAME="${ARCHIVE_NAME}.tar.gz" '$2 == NAME { print $1 }' "${WORK_DIR}/SHA256SUMS")"
	[ -n "${EXPECTED_DIGEST}" ] || fail "SHA256SUMS has no entry for ${ARCHIVE_NAME}.tar.gz"

	ACTUAL_DIGEST="$(sha256_of "${WORK_DIR}/${ARCHIVE_NAME}.tar.gz")"
	[ "${ACTUAL_DIGEST}" = "${EXPECTED_DIGEST}" ] || fail "checksum mismatch for ${ARCHIVE_NAME}.tar.gz (expected ${EXPECTED_DIGEST}, got ${ACTUAL_DIGEST})"

	tar -xzf "${WORK_DIR}/${ARCHIVE_NAME}.tar.gz" -C "${WORK_DIR}"

	mkdir -p "${INSTALL_DIR}"
	# Copy alongside, then rename: replacing a running binary in place would fail with "text file busy".
	cp "${WORK_DIR}/${ARCHIVE_NAME}/foxtrainer" "${INSTALL_DIR}/.foxtrainer.new"
	chmod 0755 "${INSTALL_DIR}/.foxtrainer.new"
	mv -f "${INSTALL_DIR}/.foxtrainer.new" "${INSTALL_DIR}/foxtrainer"

	printf 'Installed %s to %s/foxtrainer\n' "$("${INSTALL_DIR}/foxtrainer" version)" "${INSTALL_DIR}"

	# shellcheck disable=SC2016 # ${PATH} in the hint is printed literally for the user to paste.
	case ":${PATH}:" in
		*":${INSTALL_DIR}:"*) printf 'Next: run foxtrainer configure\n' ;;
		*) printf '\n%s is not on your PATH. Add this to your shell profile (e.g. ~/.profile):\n  export PATH="%s:${PATH}"\n' "${INSTALL_DIR}" "${INSTALL_DIR}" ;;
	esac
}

main "$@"
