#!/usr/bin/env bash
# price-check-collect.sh
# 30분 주기 체크에 필요한 모든 API 데이터를 수집하여 압축된 텍스트로 출력한다.
# Claude는 이 출력을 입력으로 받아 매매 판단만 수행한다.

set -euo pipefail
BASE="http://localhost:8080/api/v1"
NOW=$(TZ='Asia/Seoul' date '+%H:%M KST')
TODAY=$(TZ='Asia/Seoul' date '+%Y-%m-%d')

# ── 1. 기본 데이터 수집 ──────────────────────────────────────────────────────
TICKER=$(curl -sf "$BASE/ticker")
HOURLY=$(curl -sf "$BASE/ticker/hourly")
PORTFOLIOS=$(curl -sf "$BASE/portfolios")
STRATEGIES=$(curl -sf "$BASE/strategy")

# ── 2. 포트폴리오×코인 상태 수집 ─────────────────────────────────────────────
collect_status() {
  local coin=$1 pf=$2
  curl -sf "$BASE/simulation/status?coin=${coin}&portfolio_id=${pf}"
}

collect_strategy_history() {
  local coin=$1 pf=$2
  curl -sf "$BASE/portfolios/${pf}/strategy-history?coin=${coin}"
}

collect_prices() {
  local coin=$1
  curl -sf "$BASE/prices?coin=${coin}&period=3d"
}

# ── 3. 출력: 시각 ─────────────────────────────────────────────────────────────
echo "=== 수집시각: ${NOW} ==="
echo ""

# ── 4. 시세 및 지표 ────────────────────────────────────────────────────────────
echo "### 시세 및 지표 (일봉 기반)"
echo "$TICKER" | jq -r '.[] | "\(.coin)  $\(.current_price)  RSI:\(.rsi14|round)  ADX:\(.adx14|round)  MACD:\(if .macd>0 then "+" else "" end)\(.macd|round)  BBU:\(.bb_upper|round)  BBL:\(.bb_lower|round)  MA7:\(.ma7|round)  MA20:\(.ma20|round)"'
echo ""

echo "### 1시간봉 지표 (Trend Rider용)"
echo "$HOURLY" | jq -r '.[] | "\(.coin)  EMA9:\(.ema9_1h|round)  EMA21:\(.ema21_1h|round)  RSI1h:\(.rsi14_1h|.*10|round/10)  MACDhist:\(if .macd_hist_1h>0 then "+" else "" end)\(.macd_hist_1h|.*100|round/100)  VWAP:\(.vwap_24h|round)  Δ4h:\(.price_change_4h|.*10|round/10)%  Δ24h:\(.price_change_24h|.*10|round/10)%"'
echo ""

# ── 5. 일봉 방향 (3일) ─────────────────────────────────────────────────────────
echo "### 일봉 방향 (최근 3일, 양봉=▲ 음봉=▼)"
for coin in BTC ETH SOL; do
  DIR=$(collect_prices "$coin" | jq -r '
    .data[-3:] |
    map(if .close >= .open then "▲" else "▼" end) |
    join(" ")
  ' 2>/dev/null || echo "N/A")
  echo "  ${coin}: ${DIR}"
done
echo ""

# ── 6. 포트폴리오 현황 및 전략 ───────────────────────────────────────────────
echo "### 포트폴리오 현황"

# notify_on_trade, risk_limit_pct 추출
PF_META=$(echo "$PORTFOLIOS" | jq -r '.portfolios[] | "\(.id)|\(.name)|\(.notify_on_trade)|\(.risk_limit_pct)"')

while IFS='|' read -r PF_ID PF_NAME NOTIFY RISK_LIMIT; do
  # 포트폴리오별 코인 목록
  case "$PF_ID" in
    1) COINS="BTC ETH" ;;
    2) COINS="BTC ETH SOL" ;;
    3) COINS="BTC ETH SOL" ;;
    4) COINS="BTC ETH SOL" ;;
    *) COINS="" ;;
  esac
  [[ -z "$COINS" ]] && continue

  echo ""
  echo "#### [pf${PF_ID}] ${PF_NAME} | notify:${NOTIFY} | risk_limit:-${RISK_LIMIT}%"

  for COIN in $COINS; do
    STATUS=$(collect_status "$COIN" "$PF_ID")
    HIST=$(collect_strategy_history "$COIN" "$PF_ID")

    POSITION=$(echo "$STATUS" | jq -r '.position')
    CASH=$(echo "$STATUS" | jq -r '.cash | round')
    VALUE=$(echo "$STATUS" | jq -r '.current_value | round')
    ROI=$(echo "$STATUS" | jq -r '.return_pct | . * 10 | round | . / 10')
    AVG_COST=$(echo "$STATUS" | jq -r '.avg_cost // 0 | round')
    CUR_PRICE=$(echo "$STATUS" | jq -r '.current_price | round')

    STRAT_NAME=$(echo "$HIST" | jq -r '.history[0].strategy_name // "없음"')
    STRAT_ID=$(echo "$HIST" | jq -r '.history[0].strategy_id // 0')

    # 전략 notes (핵심 조건만 추출)
    NOTES=$(echo "$STRATEGIES" | jq -r --argjson sid "$STRAT_ID" \
      '.strategies[] | select(.id == $sid) | .notes' 2>/dev/null || echo "")

    echo "  ${COIN}: ${POSITION} | \$${VALUE} (ROI:${ROI}%) | cash:\$${CASH} | avg_cost:\$${AVG_COST} | 전략:${STRAT_NAME}"
    if [[ -n "$NOTES" ]]; then
      # 조건 섹션 추출 (매수/매도/손절/익절/지표 조건 라인, 15줄)
      CONDITIONS=$(echo "$NOTES" | grep -E '(RSI|ADX|MACD|MA|BB|EMA|ema|vwap|손절|익절|매수|매도|조건|핵심|보조|크로스|충족|hist|price_change|Breakout|momentum|횡보|추세)' | head -15 | sed 's/^/    /')
      if [[ -n "$CONDITIONS" ]]; then
        echo "$CONDITIONS"
      fi
    fi
  done
done <<< "$PF_META"

echo ""
echo "### 거래 실행 API"
echo "POST $BASE/simulation/trade  body: {coin,action(BUY/SELL),price,amount(optional),reason,portfolio_id}"
