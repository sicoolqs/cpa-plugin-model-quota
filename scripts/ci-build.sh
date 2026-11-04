#!/bin/sh
set -eu

GOARCH=${1:-amd64}
GOOS=$(go env GOHOSTOS)

case "${GITHUB_REF:-}" in
  refs/tags/v*) VERSION=${GITHUB_REF#refs/tags/v} ;;
  *) VERSION=0.4.2 ;;
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

# 钉住 CPU 基线指令集：防止 CI runner（如 ubuntu-latest）编译产物携带 AVX 等新指令，
# 在老 CPU 或虚拟化宿主上触发 SIGILL（CLIProxyAPI 官方插件构建指引，examples/plugin/README.md）
case $GOARCH in
  amd64)
    export GOAMD64=v1
    export CGO_CFLAGS="-march=x86-64"
    ;;
  arm64)
    export GOARM64=v8.0
    export CGO_CFLAGS="-march=armv8-a"
    ;;
esac

go build -buildvcs=false -trimpath -buildmode=c-shared \
  -ldflags "-s -w" \
  -o "dist/model-quota.$EXT" .

rm -f dist/*.h
echo "Built dist/model-quota.$EXT for $GOOS/$GOARCH"
