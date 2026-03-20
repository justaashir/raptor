#!/bin/sh
set -e

VERSION="${1:?Usage: release.sh <version> <server-url>}"
SERVER="${2:?Usage: release.sh <version> <server-url>}"

PLATFORMS="darwin/arm64 darwin/amd64 linux/amd64 linux/arm64"
LDFLAGS="-X raptor/cmd.Version=${VERSION} -X raptor/cmd.DefaultServer=${SERVER}"

for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    echo "Building $GOOS/$GOARCH..."
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 go build -ldflags "$LDFLAGS" -o "raptor-${GOOS}-${GOARCH}" .
done

for platform in $PLATFORMS; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    echo "Uploading $GOOS/$GOARCH..."
    curl -fsSL -X PUT --data-binary "@raptor-${GOOS}-${GOARCH}" "${SERVER}/admin/releases/${GOOS}/${GOARCH}"
    rm "raptor-${GOOS}-${GOARCH}"
done

echo "Released v${VERSION}"
