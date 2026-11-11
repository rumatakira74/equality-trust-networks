#!/bin/sh
# Проба WASM-сборки верификатора (предпроектная работа, решение 11.11):
# игрушечный модуль, импортирующий crypto/wire-пакеты ядра через
# go-модульную зависимость (replace => ../../../), собирается под
# GOOS=js GOARCH=wasm. Продуктовый код не создаётся.
#
# Запуск: sh build.sh   (из каталога reports/verifier-wasm-probe/)
# Ядра: GOMAXPROCS=6 (бюджет прогона ≤6 ядер из 8).

set -e
cd "$(dirname "$0")/toy"

echo "== go mod tidy (GOPROXY=off, только кэш модулей) =="
GOPROXY=off GOMAXPROCS=6 go mod tidy

echo "== нативный sanity-прогон (linux/amd64) =="
GOPROXY=off GOMAXPROCS=6 go run .

echo "== GOOS=js GOARCH=wasm go build =="
GOPROXY=off GOMAXPROCS=6 GOOS=js GOARCH=wasm go build -o toy.wasm .
ls -la toy.wasm

echo "== go vet (linux) =="
GOPROXY=off GOMAXPROCS=6 go vet .

echo "probe build ok"
