#!/bin/bash

# Скрипт для профилирования dive с использованием CPU, memory и goroutine профилей

set -e

# Создаем директорию для результатов профилирования
OUT=${OUT:-profiling}
mkdir -p "$OUT"

IMAGE=${1:-"alpine:latest"}

echo "Запуск dive с профилированием для образа: $IMAGE"

# Запускаем dive с флагом --ci и профилированием через переменные окружения
# -gcflags="all=-l" отключает инлайнинг для более точного профилирования
CPU_PROFILE="$OUT/cpu.pprof" \
MEM_PROFILE="$OUT/mem.pprof" \
GOROUTINE_PROFILE="$OUT/goroutine.pprof" \
go run -gcflags="all=-l" cmd/dive/main.go --ci "$IMAGE"

echo ""
echo "CPU profile..."
go tool pprof -top "$OUT/cpu.pprof" >"$OUT/cpu.txt"

echo "MEM profile..."
go tool pprof -top "$OUT/mem.pprof" >"$OUT/mem.txt"

echo "GOROUTINE profile..."
go tool pprof -top "$OUT/goroutine.pprof" >"$OUT/goroutine.txt"

echo ""
echo "Результаты профилирования сохранены в директории: $OUT"
echo "  - CPU:       $OUT/cpu.pprof (raw) и $OUT/cpu.txt (top)"
echo "  - Memory:    $OUT/mem.pprof (raw) и $OUT/mem.txt (top)"
echo "  - Goroutine: $OUT/goroutine.pprof (raw) и $OUT/goroutine.txt (top)"
