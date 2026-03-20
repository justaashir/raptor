#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SERVER="${RAPTOR_SERVER:-https://raptor.raptorthree.com}"

# Bump version: major, minor (default), or patch
BUMP="${1:-minor}"
CURRENT=$(cat "$ROOT_DIR/VERSION" | tr -d '[:space:]')
MAJOR=$(echo "$CURRENT" | cut -d. -f1)
MINOR=$(echo "$CURRENT" | cut -d. -f2)
PATCH=$(echo "$CURRENT" | cut -d. -f3)

case "$BUMP" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
    *) echo "Usage: release.sh [major|minor|patch]" >&2; exit 1 ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"
echo "$VERSION" > "$ROOT_DIR/VERSION"

echo "Releasing v${VERSION} → ${SERVER}"

# Set Railway env var
railway variables set VERSION="$VERSION" 2>/dev/null || echo "Warning: could not set Railway VERSION (set manually)"

# Cross-compile
PLATFORMS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64"
LDFLAGS="-X raptor/cmd.Version=${VERSION} -X raptor/cmd.DefaultServer=${SERVER}"
DIST_DIR="$ROOT_DIR/dist"
mkdir -p "$DIST_DIR"

for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "$DIST_DIR/raptor-${GOOS}-${GOARCH}" "$ROOT_DIR"
done

# Upload to Railway server
for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    echo "Uploading $GOOS/$GOARCH..."
    curl -fsSL -X PUT --data-binary "@$DIST_DIR/raptor-${GOOS}-${GOARCH}" "${SERVER}/admin/releases/${GOOS}/${GOARCH}"
done

# Commit, tag, GitHub release
cd "$ROOT_DIR"
git add VERSION
git commit -m "Release v${VERSION}" || true
git push || true

TAG="v${VERSION}"
git tag "$TAG"
git push origin "$TAG"
gh release create "$TAG" \
    --title "Raptor $TAG" \
    --generate-notes \
    "$DIST_DIR"/raptor-*

# Cleanup
rm -rf "$DIST_DIR"

echo "✅ Released v${VERSION}"
