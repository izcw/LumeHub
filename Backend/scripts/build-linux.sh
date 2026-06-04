#!/usr/bin/env bash
# 在 Git Bash / WSL / macOS / Linux 上交叉编译 Linux 可执行文件
set -euo pipefail

ARCH="${1:-amd64}"
BACKEND_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT_DIR="${BACKEND_ROOT}/dist/linux-${ARCH}"
OUT_FILE="${OUT_DIR}/lumehub"

mkdir -p "$OUT_DIR"

echo ">> 交叉编译 linux/${ARCH} -> ${OUT_FILE}"

export GOOS=linux
export GOARCH="$ARCH"
export CGO_ENABLED=0

cd "$BACKEND_ROOT"
go build -trimpath -ldflags '-s -w' -o "$OUT_FILE" ./cmd/lumehub

# ELF magic check
if ! head -c 4 "$OUT_FILE" | od -An -tx1 | grep -q '7f 45 4c 46'; then
  echo "错误: 输出不是 ELF 格式" >&2
  exit 1
fi

echo ">> 完成: ${OUT_FILE} ($(du -h "$OUT_FILE" | cut -f1))"
