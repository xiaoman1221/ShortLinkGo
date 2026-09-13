#!/usr/bin/env bash
# 一键构建脚本（前端 + 后端），等价于 make
set -euo pipefail
cd "$(dirname "$0")"

echo "==> 构建前端 web/dist"
(cd web && npm install && npm run build)

echo "==> 构建后端 shortlinkgo"
go build -trimpath -ldflags "-s -w" -o shortlinkgo ./server

echo "构建完成：./shortlinkgo"
