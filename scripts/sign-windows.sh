#!/usr/bin/env bash
# Sign the Windows executables of a draft GitHub release with the Certum
# "Open Source Code Signing" certificate (SimplySign cloud), then publish it.
#
#   scripts/sign-windows.sh v0.2.2
#   scripts/sign-windows.sh --check v0.2.1   # sign copies only, change nothing
#
# The release workflow creates every release as a DRAFT. This script
#   1. downloads the draft's Windows .exe files,
#   2. signs them with ssign, which talks to Certum's SimplySign cloud over
#      HTTPS (https://github.com/Le-Syl21/ssign): you type the current 6-digit
#      code from the SimplySign app, so the TOTP seed never leaves your phone,
#   3. verifies the signatures with osslsigncode,
#   4. replaces the unsigned files in the draft, and
#   5. publishes the release after you confirm.
#
# Needs: gh (logged in), osslsigncode, and ssign:
#   brew install rust osslsigncode
#   cargo install ssign --version 0.1.8 --locked
#
# Your Certum account e-mail comes from CERTUM_EMAIL, or is asked for.
set -euo pipefail

REPO=tim-riep/go-xslt
SSIGN_VERSION=0.1.8

CHECK=0
if [ "${1:-}" = "--check" ]; then
  CHECK=1
  shift
fi
TAG=${1:-}
if [ -z "$TAG" ]; then
  echo "usage: $0 [--check] <tag>   (e.g. $0 v0.2.2)" >&2
  exit 2
fi

# cargo installs into ~/.cargo/bin, which is often not on PATH.
PATH="$PATH:$HOME/.cargo/bin"

for tool in gh osslsigncode ssign; do
  if ! command -v "$tool" >/dev/null 2>&1; then
    echo "missing: $tool (see the header of $0)" >&2
    exit 1
  fi
done
if ! ssign --version 2>/dev/null | grep -q "$SSIGN_VERSION"; then
  echo "warning: expected ssign $SSIGN_VERSION, found: $(ssign --version 2>/dev/null)" >&2
fi

draft=$(gh release view "$TAG" -R "$REPO" --json isDraft --jq .isDraft)
if [ "$CHECK" = 0 ] && [ "$draft" != "true" ]; then
  echo "$TAG is not a draft release; refusing to change a published release." >&2
  exit 1
fi

WORK=$(mktemp -d)
# ssign caches its ~30-minute signing token; nothing of it should outlive the run.
cleanup() {
  rm -rf "$WORK"
  rm -f "${XDG_RUNTIME_DIR:-$HOME/.cache}/ssign/session.json"
}
trap cleanup EXIT

echo "== downloading the Windows executables of $TAG"
gh release download "$TAG" -R "$REPO" -p '*windows*.exe' -D "$WORK"
shopt -s nullglob
files=("$WORK"/*.exe)
if [ ${#files[@]} -eq 0 ]; then
  echo "no Windows executables in $TAG" >&2
  exit 1
fi
printf '   %s\n' "${files[@]##*/}"

if [ -z "${CERTUM_EMAIL:-}" ]; then
  read -r -p "Certum account e-mail: " CERTUM_EMAIL
fi
# The code goes to ssign through the environment, not the command line, so it
# shows up neither in the shell history nor in the process list.
read -r -s -p "Current 6-digit code from the SimplySign app: " CERTUM_TOKEN
echo
export CERTUM_EMAIL CERTUM_TOKEN

echo "== signing"
ssign -n "go-xslt" -u "https://github.com/$REPO" "${files[@]}"
unset CERTUM_TOKEN

# osslsigncode defaults to /etc/ssl/cert.pem, which on macOS lacks Certum's root
# (Certum Trusted Network CA 2); the Mozilla bundle from Homebrew has it, as
# does Windows' own root store.
CAFILE=""
for c in "$(brew --prefix 2>/dev/null)/etc/ca-certificates/cert.pem" /etc/ssl/certs/ca-certificates.crt /etc/ssl/cert.pem; do
  if [ -f "$c" ]; then
    CAFILE=$c
    break
  fi
done

echo "== verifying"
for f in "${files[@]}"; do
  if ! out=$(osslsigncode verify -CAfile "$CAFILE" -TSA-CAfile "$CAFILE" -in "$f" 2>&1); then
    echo "$out" >&2
    echo "verification failed for ${f##*/}; the draft was not changed." >&2
    exit 1
  fi
  echo "   ${f##*/}: $(echo "$out" | grep -m1 'Subject:.*Open Source\|Subject:' | sed 's/^[[:space:]]*//')"
done

if [ "$CHECK" = 1 ]; then
  echo "check passed: signing and verification work; nothing was uploaded."
  exit 0
fi

echo "== replacing the unsigned files in the draft"
gh release upload "$TAG" -R "$REPO" --clobber "${files[@]}"

read -r -p "Publish $TAG now? [y/N] " answer
if [ "$answer" = "y" ] || [ "$answer" = "Y" ]; then
  gh release edit "$TAG" -R "$REPO" --draft=false --latest
  echo "published: https://github.com/$REPO/releases/tag/$TAG"
else
  echo "left as a draft; publish later with: gh release edit $TAG -R $REPO --draft=false --latest"
fi
