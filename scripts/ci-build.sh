#!/bin/sh
set -eu

GOARCH=${1:-amd64}
GOOS=$(go env GOHOSTOS)

case "${GITHUB_REF:-}" in
  refs/tags/v*) VERSION=${GITHUB_REF#refs/tags/v} ;;
  *) VERSION=0.2.0 ;;
esac

case $GOOS in
  linux)
    EXT=so
    ;;
  darwin)
    EXT=dylib
    case $GOARCH in
      amd64) export CC="clang -arch x86_64" ;;
      *) export CC="clang -arch arm64" ;;
    esac
    ;;
  windows)
    EXT=dll
    ;;
  *)
    echo "Unsupported OS: $GOOS" >&2
    exit 1
    ;;
esac

mkdir -p dist
export CGO_ENABLED=1 GOARCH

go build -buildvcs=false -trimpath -buildmode=c-shared \
  -ldflags "-s -w" \
  -o "dist/model-quota.$EXT" .

rm -f dist/*.h
echo "Built dist/model-quota.$EXT for $GOOS/$GOARCH"
