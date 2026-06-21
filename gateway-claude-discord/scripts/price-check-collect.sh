#!/usr/bin/env bash
# price-check-collect.sh
# 30분 주기 체크에 필요한 모든 API 데이터를 수집하여 압축된 텍스트로 출력한다.
# Claude는 이 출력을 입력으로 받아 매매 판단만 수행한다.

set -uo pipefail
BASE="http://localhost:8080/api/v1"
NOW=$(TZ='Asia/Seoul' date '+%H:%M KST')

# ── 1. 기본 데이터 수집 ──────────────────────────────────────────────────────
TICKER=$(curl -sf "$BASE/ticker" 2>/dev/null || echo "")
HOURLY=$(curl -sf "$BASE/ticker/hourly" 2>/dev/null || echo "")
UPBIT_TICKER=$(curl -sf "$BASE/upbit/ticker" 2>/dev/null || echo "")
UPBIT_HOURLY=$(curl -sf "$BASE/upbit/ticker/hourly" 2>/dev/null || echo "")
BITHUMB_TICKER=$(curl -sf "$BASE/bithumb/ticker" 2>/dev/null || echo "")
BITHUMB_HOURLY=$(curl -sf "$BASE/bithumb/ticker/hourly" 2>/dev/null || echo "")
PORTFOLIOS=$(curl -sf "$BASE/portfolios" 2>/dev/null || echo "")
STRATEGIES=$(curl -sf "$BASE/strategy" 2>/dev/null || echo "")

# ── 2. 외부 데이터 수집 ───────────────────────────────────────────────────────
# Fear & Greed Index
FNG_RAW=$(curl -sf "https://api.alternative.me/fng/?limit=1" 2>/dev/null || echo '{}')
FNG_VALUE=$(echo "$FNG_RAW" | jq -r '.data[0].value // "N/A"')
FNG_CLASS=$(echo "$FNG_RAW" | jq -r '.data[0].value_classification // "N/A"')

# USD/KRW 환율
USD_KRW=$(curl -sf "https://open.er-api.com/v6/latest/USD" 2>/dev/null | jq -r '.rates.KRW // 1480' || echo "1480")

# BTC 도미넌스 (CoinGecko)
GLOBAL_RAW=$(curl -sf "https://api.coingecko.com/api/v3/global" 2>/dev/null || echo '{}')
BTC_DOMINANCE=$(echo "$GLOBAL_RAW" | jq -r '.data.market_cap_percentage.btc // "N/A" | if . == "N/A" then . else (. * 10 | round | . / 10 | tostring) + "%" end' 2>/dev/null || echo "N/A")
ETH_DOMINANCE=$(echo "$GLOBAL_RAW" | jq -r '.data.market_cap_percentage.eth // "N/A" | if . == "N/A" then . else (. * 10 | round | . / 10 | tostring) + "%" end' 2>/dev/null || echo "N/A")

# 김치 프리미엄 계산 + 신규 지표
python3 << 'PYEOF' 2>/dev/null > /tmp/kimchi_calc.env || true
import json, subprocess, os

ticker_raw    = subprocess.run(["curl","-sf","http://localhost:8080/api/v1/ticker"],          capture_output=True, text=True).stdout
upbit_raw     = subprocess.run(["curl","-sf","http://localhost:8080/api/v1/upbit/ticker"],    capture_output=True, text=True).stdout
bithumb_raw   = subprocess.run(["curl","-sf","http://localhost:8080/api/v1/bithumb/ticker"],  capture_output=True, text=True).stdout
er_raw        = subprocess.run(["curl","-sf","https://open.er-api.com/v6/latest/USD"],        capture_output=True, text=True).stdout

try:
    binance = {t['coin']: t for t in json.loads(ticker_raw)}
    upbit   = {t['coin']: t for t in json.loads(upbit_raw)}
    bithumb = {t['coin']: t for t in json.loads(bithumb_raw)}
    r = json.loads(er_raw).get('rates', {}).get('KRW', 1480)
except:
    print("R=1480"); exit()

out = {}
for coin in ['BTC', 'ETH', 'SOL']:
    b = binance.get(coin, {}); u = upbit.get(coin, {}); bh = bithumb.get(coin, {})
    bp = b.get('current_price', 0); up_p = u.get('current_price', 0); bh_p = bh.get('current_price', 0)
    bp_prev = b.get('prev_price', 0); up_prev = u.get('prev_price', 0)

    # 현재 김프 (업비트 기준)
    kimchi = round((up_p / (bp * r) - 1) * 100, 2) if bp > 0 and r > 0 else None
    # 어제 김프
    kimchi_prev = round((up_prev / (bp_prev * r) - 1) * 100, 2) if bp_prev > 0 and r > 0 else None
    # 김프 방향
    kimchi_dir = round(kimchi - kimchi_prev, 2) if kimchi is not None and kimchi_prev is not None else None
    # 빗썸 김프
    kimchi_bithumb = round((bh_p / (bp * r) - 1) * 100, 2) if bp > 0 and r > 0 else None

    # 한국 수요 비율 (업비트/바이낸스 거래량 비율)
    bvol = b.get('current_volume', 0); uvol = u.get('current_volume', 0)
    demand_ratio = round(uvol / bvol * 100, 2) if bvol > 0 else None
    # 업비트 거래량 강도
    uvol_ma = u.get('volume_ma20', 0)
    uvol_strength = round(uvol / uvol_ma, 2) if uvol_ma > 0 else None

    out[coin] = {
        'kimchi': kimchi, 'kimchi_prev': kimchi_prev, 'kimchi_dir': kimchi_dir,
        'kimchi_bithumb': kimchi_bithumb,
        'demand_ratio': demand_ratio, 'uvol_strength': uvol_strength,
        'upbit_vol': round(uvol, 2), 'binance_vol': round(bvol, 2)
    }

for coin, v in out.items():
    for k, val in v.items():
        print(f"KIMCHI_{coin}_{k.upper()}={val if val is not None else 'N/A'}")
PYEOF
source /tmp/kimchi_calc.env 2>/dev/null || true

BINANCE_BTC=$(echo "$TICKER" | jq -r '.[] | select(.coin=="BTC") | .current_price')
UPBIT_BTC=$(echo "$UPBIT_TICKER" | jq -r '.[] | select(.coin=="BTC") | .current_price')
BINANCE_ETH=$(echo "$TICKER" | jq -r '.[] | select(.coin=="ETH") | .current_price')
UPBIT_ETH=$(echo "$UPBIT_TICKER" | jq -r '.[] | select(.coin=="ETH") | .current_price')

# ── 3. 포트폴리오×코인 상태 수집 함수 ────────────────────────────────────────
collect_status() {
  local coin=$1 pf=$2
  curl -sf "$BASE/simulation/status?coin=${coin}&portfolio_id=${pf}" 2>/dev/null || echo "{}"
}

collect_strategy_history() {
  local coin=$1 pf=$2
  curl -sf "$BASE/portfolios/${pf}/strategy-history?coin=${coin}" 2>/dev/null || echo "{}"
}

collect_prices() {
  local coin=$1
  curl -sf "$BASE/prices?coin=${coin}&period=3d"
}

# ── 4. 출력 시작 ──────────────────────────────────────────────────────────────
echo "=== 수집시각: ${NOW} ==="
echo ""

# ── 5. 외부 시그널 ────────────────────────────────────────────────────────────
echo "=== [외부 시그널] ==="
echo "### Fear & Greed Index"
echo "  F&G: ${FNG_VALUE}/100 (${FNG_CLASS})"
echo "  해석: 0~25=Extreme Fear(역발상 매수) | 26~45=Fear | 46~54=Neutral | 55~74=Greed | 75~100=Extreme Greed(역발상 매도)"
echo ""
echo "### 김치 프리미엄 (USD/KRW: ${USD_KRW})"
echo "  [업비트 기준]"
echo "  BTC 김프: ${KIMCHI_BTC_KIMCHI}%  (어제: ${KIMCHI_BTC_KIMCHI_PREV}%  방향: ${KIMCHI_BTC_KIMCHI_DIR}%)"
echo "  ETH 김프: ${KIMCHI_ETH_KIMCHI}%  (어제: ${KIMCHI_ETH_KIMCHI_PREV}%  방향: ${KIMCHI_ETH_KIMCHI_DIR}%)"
echo "  SOL 김프: ${KIMCHI_SOL_KIMCHI}%  (어제: ${KIMCHI_SOL_KIMCHI_PREV}%  방향: ${KIMCHI_SOL_KIMCHI_DIR}%)"
echo "  [빗썸 기준]"
echo "  BTC 빗썸김프: ${KIMCHI_BTC_KIMCHI_BITHUMB}%"
echo "  ETH 빗썸김프: ${KIMCHI_ETH_KIMCHI_BITHUMB}%"
echo "  SOL 빗썸김프: ${KIMCHI_SOL_KIMCHI_BITHUMB}%"
echo "  [한국 수요 지표]"
echo "  BTC 한국수요비율: ${KIMCHI_BTC_DEMAND_RATIO}%  업비트거래량강도: ${KIMCHI_BTC_UVOL_STRENGTH}  (업비트:${KIMCHI_BTC_UPBIT_VOL}BTC / 바이낸스:${KIMCHI_BTC_BINANCE_VOL}BTC)"
echo "  ETH 한국수요비율: ${KIMCHI_ETH_DEMAND_RATIO}%  업비트거래량강도: ${KIMCHI_ETH_UVOL_STRENGTH}"
echo "  SOL 한국수요비율: ${KIMCHI_SOL_DEMAND_RATIO}%  업비트거래량강도: ${KIMCHI_SOL_UVOL_STRENGTH}"
echo "  해석: 김프≤-0.5%+수요비율<5%=강BUY | 김프≤-0.5%+RSI<50=BUY | 김프≥3%+수요비율>10%=강SELL | 김프≥5%=SELL"
echo "  수요비율: <5%=냉각(BUY신호) | 5~10%=정상 | >12%=과열(SELL신호)"
echo "  거래량강도: <0.8=위축 | 0.8~1.3=정상 | >1.3=급증(과열주의)"
echo ""
echo "### BTC 도미넌스"
echo "  BTC 도미넌스: ${BTC_DOMINANCE}  ETH 도미넌스: ${ETH_DOMINANCE}"
echo "  해석(전략 업데이트): BTC.D<58%=BUY(일반) | BTC.D<55%=BUY(강한) | BTC.D>60%=SELL | F&G<20+BTC.D<60%=공포장BUY"
echo ""

# ── 6. [바이낸스] 지표 ────────────────────────────────────────────────────────
echo "=== [바이낸스] 지표 (pf1~4, pf9, pf11 적용) ==="

echo "### 시세 및 지표 (일봉 기반)"
echo "$TICKER" | jq -r '.[] | "\(.coin)  $\(.current_price)  RSI:\(.rsi14|round)  ADX:\(.adx14|round)  MACD:\(if .macd>0 then "+" else "" end)\(.macd|round)  BBU:\(.bb_upper|round)  BBL:\(.bb_lower|round)  MA7:\(.ma7|round)  MA20:\(.ma20|round)  ATR14:\(.atr14|.*100|round/100)"'
echo ""

echo "### 1시간봉 지표 (Trend Rider용)"
echo "$HOURLY" | jq -r '.[] | "\(.coin)  EMA9:\(.ema9_1h|round)  EMA21:\(.ema21_1h|round)  RSI1h:\(.rsi14_1h|.*10|round/10)  MACDhist:\(if .macd_hist_1h>0 then "+" else "" end)\(.macd_hist_1h|.*100|round/100)  VWAP:\(.vwap_24h|round)  Δ4h:\(.price_change_4h|.*10|round/10)%  Δ24h:\(.price_change_24h|.*10|round/10)%"'
echo ""

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

# ── 7. [업비트] 지표 ──────────────────────────────────────────────────────────
echo "=== [업비트] 지표 (pf5~8, pf10 적용, KRW 기준) ==="

echo "### 시세 및 지표 (일봉 기반, KRW)"
echo "$UPBIT_TICKER" | jq -r '.[] | "\(.coin)  ₩\(.current_price|round)  RSI:\(.rsi14|round)  ADX:\(.adx14|round)  MACD:\(if .macd>0 then "+" else "" end)\(.macd|round)  BBU:\(.bb_upper|round)  BBL:\(.bb_lower|round)  MA7:\(.ma7|round)  MA20:\(.ma20|round)  ATR14:\(.atr14|.*10|round/10)"'
echo ""

echo "### 1시간봉 지표 (Trend Rider용)"
echo "$UPBIT_HOURLY" | jq -r '.[] | "\(.coin)  EMA9:\(.ema9_1h|round)  EMA21:\(.ema21_1h|round)  RSI1h:\(.rsi14_1h|.*10|round/10)  MACDhist:\(if .macd_hist_1h>0 then "+" else "" end)\(.macd_hist_1h|.*100|round/100)  VWAP:\(.vwap_24h|round)  Δ4h:\(.price_change_4h|.*10|round/10)%  Δ24h:\(.price_change_24h|.*10|round/10)%"'
echo ""

# ── 7-2. [빗썸] 지표 ──────────────────────────────────────────────────────────
echo "=== [빗썸] 지표 (pf12~15 적용, KRW 기준) ==="

echo "### 시세 및 지표 (일봉 기반, KRW)"
echo "$BITHUMB_TICKER" | jq -r '.[] | "\(.coin)  ₩\(.current_price|round)  RSI:\(.rsi14|round)  ADX:\(.adx14|round)  MACD:\(if .macd>0 then "+" else "" end)\(.macd|round)  BBU:\(.bb_upper|round)  BBL:\(.bb_lower|round)  MA7:\(.ma7|round)  MA20:\(.ma20|round)  ATR14:\(.atr14|.*10|round/10)"'
echo ""

echo "### 1시간봉 지표 (Trend Rider용)"
echo "$BITHUMB_HOURLY" | jq -r '.[] | "\(.coin)  EMA9:\(.ema9_1h|round)  EMA21:\(.ema21_1h|round)  RSI1h:\(.rsi14_1h|.*10|round/10)  MACDhist:\(if .macd_hist_1h>0 then "+" else "" end)\(.macd_hist_1h|.*100|round/100)  VWAP:\(.vwap_24h|round)  Δ4h:\(.price_change_4h|.*10|round/10)%  Δ24h:\(.price_change_24h|.*10|round/10)%"'
echo ""

# ── 8. 포트폴리오 현황 ────────────────────────────────────────────────────────
echo "### 포트폴리오 현황"

PF_META=$(echo "$PORTFOLIOS" | jq -r '.portfolios[] | "\(.id)|\(.name)|\(.notify_on_trade)|\(.risk_limit_pct)"')

while IFS='|' read -r PF_ID PF_NAME NOTIFY RISK_LIMIT; do
  case "$PF_ID" in
    1)  COINS="BTC ETH";        EXCHANGE="binance"; CURRENCY="$" ;;
    2)  COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    3)  COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    4)  COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    5)  COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    6)  COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    7)  COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    8)  COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    9)  COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    10) COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    11) COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    12) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    13) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    14) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    15) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    16) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    17) COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    18) COINS="BTC ETH SOL";    EXCHANGE="binance"; CURRENCY="$" ;;
    19) COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    20) COINS="BTC ETH SOL";    EXCHANGE="upbit";   CURRENCY="₩" ;;
    21) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    22) COINS="BTC ETH SOL";    EXCHANGE="bithumb"; CURRENCY="₩" ;;
    *)  COINS="" ;;
  esac
  [[ -z "$COINS" ]] && continue

  echo ""
  echo "#### [pf${PF_ID}] ${PF_NAME} | exchange:${EXCHANGE} | notify:${NOTIFY} | risk_limit:-${RISK_LIMIT}%"

  for COIN in $COINS; do
    STATUS=$(collect_status "$COIN" "$PF_ID")
    HIST=$(collect_strategy_history "$COIN" "$PF_ID")

    POSITION=$(echo "$STATUS" | jq -r '.position')
    CASH=$(echo "$STATUS" | jq -r '.cash | round')
    VALUE=$(echo "$STATUS" | jq -r '.current_value | round')
    ROI=$(echo "$STATUS" | jq -r '.return_pct | . * 10 | round | . / 10')
    AVG_COST=$(echo "$STATUS" | jq -r '.avg_cost // 0 | round')

    STRAT_NAME=$(echo "$HIST" | jq -r '.history[0].strategy_name // "없음"')
    STRAT_ID=$(echo "$HIST" | jq -r '.history[0].strategy_id // 0')

    NOTES=$(echo "$STRATEGIES" | jq -r --argjson sid "$STRAT_ID" \
      '.strategies[] | select(.id == $sid) | .notes' 2>/dev/null || echo "")

    echo "  ${COIN}: ${POSITION} | ${CURRENCY}${VALUE} (ROI:${ROI}%) | cash:${CURRENCY}${CASH} | avg_cost:${CURRENCY}${AVG_COST} | 전략:${STRAT_NAME}"
    if [[ -n "$NOTES" ]]; then
      CONDITIONS=$(echo "$NOTES" | grep -E '(RSI|ADX|MACD|MA|BB|EMA|ema|vwap|손절|익절|매수|매도|조건|핵심|보조|크로스|충족|hist|price_change|F&G|김프|센티멘트|Fear|Greed)' | head -15 | sed 's/^/    /')
      if [[ -n "$CONDITIONS" ]]; then
        echo "$CONDITIONS"
      fi
    fi
  done
done <<< "$PF_META"

echo ""
echo "### 거래 실행 API"
echo "POST $BASE/simulation/trade  body: {coin,action(BUY/SELL),price,amount(optional),reason,portfolio_id}"
echo "※ pf1-4,9,11,17,18: 바이낸스 지표 + USD 가격"
echo "※ pf5-8,10,19,20: 업비트 지표 + KRW 가격"
echo "※ pf12-16,21,22: 빗썸 지표 + KRW 가격"
echo "※ pf9(Fear&Greed): 외부시그널 F&G 값 기준"
echo "※ pf10(김치프리미엄): 외부시그널 김프 값 기준 + 업비트 KRW 가격"
echo "※ pf16(김치프리미엄): 외부시그널 김프 값 기준 + 빗썸 KRW 가격"
echo "※ pf11(뉴스센티멘트): 이 크론잡에서는 HOLD, 별도 뉴스센티멘트 크론잡이 처리"
echo "※ pf17,19,21(변동성 돌파): price > ma7+atr14×0.5 AND rsi14<70 AND adx14>15"
echo "※ pf18,20,22(BTC 도미넌스 로테이션): ETH/SOL만 대상, BTC.D<58% BUY / BTC.D>60% SELL / F&G<20+BTC.D<60% 공포장BUY"
