#!/usr/bin/env bash
# daily-report-collect.sh
# 9시 일일 리포트용 데이터를 수집하여 압축 출력한다.
# Claude는 이 출력 + 웹 뉴스 검색으로 시장 분석 리포트를 생성한다.

set -euo pipefail
BASE="http://localhost:8080/api/v1"
NOW=$(TZ='Asia/Seoul' date '+%Y-%m-%d %H:%M KST')
YESTERDAY=$(TZ='Asia/Seoul' date -d 'yesterday' '+%Y-%m-%d')

echo "=== 수집시각: ${NOW} ==="
echo ""

# ── 1. 시세 및 기술 지표 ─────────────────────────────────────────────────────
echo "### 시세 및 지표"
curl -sf "$BASE/ticker" | jq -r '.[] |
  "\(.coin)  $\(.current_price|round)  RSI:\(.rsi14|round)  ADX:\(.adx14|round)  MACD:\(if .macd>0 then "+" else "" end)\(.macd|round)  BBU:\(.bb_upper|round)  BBL:\(.bb_lower|round)  MA7:\(.ma7|round)  MA20:\(.ma20|round)  EMA9:\(.ema9|round)  EMA21:\(.ema21|round)"'
echo ""

# ── 2. 일봉 방향 ─────────────────────────────────────────────────────────────
echo "### 일봉 방향 (최근 3일, ▲양봉 ▼음봉)"
for coin in BTC ETH SOL; do
  DIR=$(curl -sf "$BASE/prices?coin=${coin}&period=3d" | jq -r '
    .data[-3:] | map(if .close >= .open then "▲" else "▼" end) | join(" ")
  ' 2>/dev/null || echo "N/A")
  echo "  ${coin}: ${DIR}"
done
echo ""

# ── 3. 포트폴리오 현황 ───────────────────────────────────────────────────────
echo "### 포트폴리오 현황"
PORTFOLIOS=$(curl -sf "$BASE/portfolios")

echo "$PORTFOLIOS" | jq -r '.portfolios[] | "\(.id)|\(.name)|\(.notify_on_trade)|\(.risk_limit_pct)"' | while IFS='|' read -r PF_ID PF_NAME NOTIFY RISK; do
  case "$PF_ID" in
    1) COINS="BTC ETH SOL" ;;
    2) COINS="BTC ETH SOL" ;;
    3) COINS="BTC ETH SOL" ;;
    4) COINS="BTC ETH SOL" ;;
    *) continue ;;
  esac

  TOTAL=0
  INITIAL=0
  COIN_LINES=""
  for COIN in $COINS; do
    S=$(curl -sf "$BASE/simulation/status?coin=${COIN}&portfolio_id=${PF_ID}" 2>/dev/null) || continue
    VAL=$(echo "$S" | jq -r '.current_value // 0')
    ROI=$(echo "$S" | jq -r '.return_pct // 0')
    POS=$(echo "$S" | jq -r '.position')
    INIT=$(echo "$S" | jq -r '.initial_capital // 100')
    COIN_LINES="${COIN_LINES}  ${COIN}: ${POS} \$${VAL%.*} (ROI:$(printf '%.1f' $ROI)%)\n"
    TOTAL=$(echo "$TOTAL + $VAL" | bc)
    INITIAL=$(echo "$INITIAL + $INIT" | bc)
  done
  PF_ROI=$(echo "scale=2; ($TOTAL - $INITIAL) / $INITIAL * 100" | bc 2>/dev/null || echo "0")
  echo ""
  echo "[pf${PF_ID}] ${PF_NAME} | 총 \$${TOTAL%.*} | ROI ${PF_ROI}% | risk:-${RISK}%"
  echo -e "$COIN_LINES"
done

# ── 4. 어제 거래 내역 ────────────────────────────────────────────────────────
echo "### 어제(${YESTERDAY}) 거래 내역"
HAS_TRADE=0

echo "$PORTFOLIOS" | jq -r '.portfolios[].id' | while read -r PF_ID; do
  PF_NAME=$(echo "$PORTFOLIOS" | jq -r --argjson id "$PF_ID" '.portfolios[] | select(.id==$id) | .name')
  case "$PF_ID" in
    1) COINS="BTC ETH SOL" ;;
    2) COINS="BTC ETH SOL" ;;
    3) COINS="BTC ETH SOL" ;;
    4) COINS="BTC ETH SOL" ;;
    *) continue ;;
  esac
  for COIN in $COINS; do
    TRADES=$(curl -sf "$BASE/simulation/trades?coin=${COIN}&portfolio_id=${PF_ID}&limit=20" 2>/dev/null) || continue
    echo "$TRADES" | jq -r --arg d "$YESTERDAY" --arg pf "$PF_NAME" --arg c "$COIN" '
      .trades // [] |
      map(select(.executed_at | startswith($d))) |
      .[] |
      "  [\($pf)] \($c) \(.action) $\(.price|round) @\(.executed_at[11:16])"
    ' 2>/dev/null && HAS_TRADE=1
  done
done

if [ "$HAS_TRADE" -eq 0 ]; then
  echo "  없음"
fi
echo ""
echo "---"
