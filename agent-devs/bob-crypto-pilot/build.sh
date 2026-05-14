#!/usr/bin/env bash
# build.sh — bob-crypto-pilot 빌드 및 재시작
# 순서: 프론트 빌드 → Zone.Identifier 정리 → Go 빌드 → 서버 교체
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$SCRIPT_DIR/server.pid"
LOG_FILE="$SCRIPT_DIR/server.log"

cd "$SCRIPT_DIR"

echo "▶ [1/4] 프론트엔드 빌드..."
cd fe
node node_modules/typescript/bin/tsc -b --noEmit
node node_modules/vite/bin/vite.js build
cd "$SCRIPT_DIR"

echo "▶ [2/4] Zone.Identifier 정리..."
find static -name "*Zone.Identifier*" -delete 2>/dev/null || true

echo "▶ [3/4] Go 바이너리 빌드..."
/usr/local/go/bin/go build -o bob-crypto-pilot .

echo "▶ [4/4] 서버 교체..."
# 실제 포트 점유 프로세스를 직접 종료
OLD_PID=$(lsof -ti :8080 2>/dev/null || true)
if [[ -n "$OLD_PID" ]]; then
  echo "  기존 서버 종료 (PID $OLD_PID)..."
  kill "$OLD_PID"
  sleep 2
fi

nohup ./bob-crypto-pilot > "$LOG_FILE" 2>&1 &
NEW_PID=$!
echo "$NEW_PID" > "$PID_FILE"

sleep 2
if curl -sf http://localhost:8080/health > /dev/null; then
  echo "✅ 서버 시작 완료 (PID $NEW_PID)"
else
  echo "❌ 서버 헬스체크 실패 — 로그 확인: tail -f $LOG_FILE"
  exit 1
fi
