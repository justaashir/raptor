#!/bin/sh
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"

SERVER="${RAPTOR_SERVER:-https://raptor.raptorthree.com}"

CURRENT=$(cat "$ROOT_DIR/VERSION" | tr -d '[:space:]')
CUR_MAJOR=$(echo "$CURRENT" | cut -d. -f1)
CUR_MINOR=$(echo "$CURRENT" | cut -d. -f2)
CUR_PATCH=$(echo "$CURRENT" | cut -d. -f3)

# Prompt for bump type if not passed as argument
if [ -n "$1" ]; then
    BUMP="$1"
else
    printf "Current version: %s\n\n" "$CURRENT"
    printf "  [p]atch → %d.%d.%d\n" "$CUR_MAJOR" "$CUR_MINOR" "$((CUR_PATCH + 1))"
    printf "  [m]inor → %d.%d.0\n" "$CUR_MAJOR" "$((CUR_MINOR + 1))"
    printf "  [M]ajor → %d.0.0\n\n" "$((CUR_MAJOR + 1))"
    printf "Bump type? "
    read -r choice
    case "$choice" in
        p|patch) BUMP="patch" ;;
        M|major) BUMP="major" ;;
        *) BUMP="minor" ;;
    esac
fi
MAJOR=$CUR_MAJOR
MINOR=$CUR_MINOR
PATCH=$CUR_PATCH

case "$BUMP" in
    major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
    minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
    patch) PATCH=$((PATCH + 1)) ;;
    *) echo "Usage: release.sh [major|minor|patch]" >&2; exit 1 ;;
esac

VERSION="${MAJOR}.${MINOR}.${PATCH}"

echo "Releasing v${VERSION}"

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

# Bump version and deploy server
echo "$VERSION" > "$ROOT_DIR/VERSION"
railway variables set VERSION="$VERSION" 2>/dev/null || echo "Warning: could not set Railway VERSION (set manually)"

echo "Deploying server..."
cd "$ROOT_DIR"
railway up --detach

# Commit, tag, and create GitHub release with binaries
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
