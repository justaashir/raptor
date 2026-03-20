#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION=$(cat "$ROOT_DIR/VERSION" | tr -d '[:space:]')
SERVER="${RAPTOR_SERVER:-https://raptor.raptorthree.com}"

if [ -z "$VERSION" ]; then
    echo "ERROR: VERSION file is empty" >&2
    exit 1
fi

echo "Releasing v${VERSION} → ${SERVER}"

PLATFORMS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64"
LDFLAGS="-X raptor/cmd.Version=${VERSION} -X raptor/cmd.DefaultServer=${SERVER}"
DIST_DIR="$ROOT_DIR/dist"
mkdir -p "$DIST_DIR"

# Cross-compile
for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    OUT="$DIST_DIR/raptor-${GOOS}-${GOARCH}"
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "$OUT" "$ROOT_DIR"
done

# Upload to Railway server
for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    echo "Uploading $GOOS/$GOARCH..."
    curl -fsSL -X PUT --data-binary "@$DIST_DIR/raptor-${GOOS}-${GOARCH}" "${SERVER}/admin/releases/${GOOS}/${GOARCH}"
done

# GitHub Release
TAG="v${VERSION}"
if git rev-parse "$TAG" >/dev/null 2>&1; then
    echo "Tag $TAG already exists, skipping GitHub release"
else
    echo "Creating GitHub release $TAG..."
    git tag "$TAG"
    git push origin "$TAG"
    gh release create "$TAG" \
        --title "Raptor $TAG" \
        --generate-notes \
        "$DIST_DIR"/raptor-*
fi

# Cleanup
rm -rf "$DIST_DIR"

echo "Released v${VERSION}"
