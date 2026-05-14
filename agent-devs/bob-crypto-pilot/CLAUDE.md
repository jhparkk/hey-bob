# bob-crypto-pilot — CLAUDE.md

> 최종 업데이트: 2026-05-06

## 프로젝트 개요

BTC/ETH/SOL 모의 투자 시뮬레이션 서버. 포트폴리오별 자동 매매 전략을 운용하고 성과를 추적한다.

- **서버**: `http://localhost:8080`
- **DB**: `data/crypto.db` (SQLite, `runtime.Caller(0)` 기준 상대경로 — 실행 위치 무관)

---

## 배포 절차

### 코드 수정 후 배포 (유일한 방법)

```bash
cd /home/jhpark/hey-bob/agent-devs/bob-crypto-pilot
./build.sh
```

`build.sh`가 아래 순서를 자동으로 처리한다:

1. **프론트 빌드** — `fe/` TypeScript 타입 체크 + Vite 번들링 → `static/` 출력
2. **Zone.Identifier 정리** — WSL 환경에서 Go embed 빌드 실패 방지
3. **Go 바이너리 빌드** — `static/` 파일을 바이너리에 embed
4. **서버 교체** — `lsof -ti :8080`으로 실제 프로세스 찾아 종료 후 새 바이너리 시작
5. **헬스체크** — `/health` 응답 확인, 실패 시 즉시 에러 출력

> ⚠️ **주의**: 프론트/백 어느 쪽을 수정하든 반드시 `build.sh` 전체를 실행해야 한다.
> Go 바이너리가 static 파일을 embed하기 때문에, 프론트만 빌드하거나 Go만 빌드하면 변경사항이 반영되지 않는다.

### 서버 상태 확인

```bash
curl http://localhost:8080/health          # 헬스체크
lsof -ti :8080                             # 실행 중인 PID 확인
tail -f /home/jhpark/hey-bob/agent-devs/bob-crypto-pilot/server.log  # 로그
```

---

## 디렉토리 구조

```
bob-crypto-pilot/
├── main.go                  ← 서버 진입점, 라우터, //go:embed static
├── go.mod / go.sum
├── db/db.go                 ← DB 초기화, 스키마, 마이그레이션
├── handlers/
│   ├── price.go             ← 가격 조회, Binance 동기화
│   ├── simulation.go        ← 시뮬레이션 상태/거래/성과
│   ├── strategy.go          ← 전략 CRUD, 포트폴리오 CRUD
│   └── ticker.go            ← 일봉 ticker + 1시간봉 ticker
├── models/
│   ├── price.go             ← DailyPrice, LivePrice 구조체
│   └── simulation.go        ← Portfolio, SimState, SimTrade, 성과 구조체
├── services/
│   ├── binance.go           ← Binance API 호출, 일봉 OHLCV 수집
│   ├── indicators.go        ← 기술지표 계산 함수 (MA, EMA, RSI, MACD, BB, ADX, ATR)
│   ├── simulation.go        ← 거래 실행, 포트폴리오 집계, 기간별 성과 계산
│   ├── ticker.go            ← 10초 간격 일봉 지표 갱신
│   ├── hourly_ticker.go     ← 10분 간격 1시간봉 지표 갱신
│   └── sync_scheduler.go   ← 매일 01:00 KST Binance 동기화
├── static/                  ← FE 빌드 출력물 (Go embed 대상)
├── data/crypto.db           ← SQLite DB
├── cmd/backfill/            ← 과거 데이터 백필 CLI
└── fe/                      ← React+TypeScript 프론트엔드
    ├── src/
    │   ├── api/index.ts     ← API 클라이언트 + 전체 타입 정의
    │   ├── components/      ← Header, PortfolioROIChart, 각종 모달
    │   ├── pages/           ← ChartPage, SimulationPage, StrategyPage
    │   ├── hooks/           ← useSimulation, useStrategy
    │   └── store/           ← Zustand 전역 상태 (simulationStore)
    └── vite.config.ts       ← outDir: '../static'
```

---

## API 전체 목록

```
GET    /health

# 가격 데이터
GET    /api/v1/prices?coin=&period=              일봉 OHLCV (7d/30d/90d 또는 from~to)
GET    /api/v1/prices/latest?coin=               최신 일봉
GET    /api/v1/price/live?coin=                  Binance 실시간 가격
POST   /api/v1/sync                              Binance 90일치 수동 동기화

# Ticker
GET    /api/v1/ticker[?coin=]                   일봉 기반 지표 (10초 갱신, 22개)
GET    /api/v1/ticker/hourly[?coin=]            1시간봉 지표 (10분 갱신, 14개)

# 전략 라이브러리
GET    /api/v1/strategy                          전체 전략 목록
POST   /api/v1/strategy                          전략 생성 ← 크론잡 호출 금지
PUT    /api/v1/strategy/:id                      전략 수정 (notes 변경 시 버전 자동 증가)
DELETE /api/v1/strategy/:id                      전략 삭제
GET    /api/v1/strategy/:id/versions             버전 이력
GET    /api/v1/strategy/history                  전체 변경 이력

# 포트폴리오
GET    /api/v1/portfolios                        전체 포트폴리오 목록
POST   /api/v1/portfolios                        포트폴리오 생성
PUT    /api/v1/portfolios/:id                    수정 (이름/설명/알림/리스크한도)
DELETE /api/v1/portfolios/:id                    삭제 (id≤2 불가)
POST   /api/v1/portfolios/:id/reset              리셋 (거래내역/잔고 초기화)
POST   /api/v1/portfolios/:id/coins              코인 추가
DELETE /api/v1/portfolios/:id/coins/:coin        코인 삭제
GET    /api/v1/portfolios/:id/strategies         코인별 활성 전략 조회
PATCH  /api/v1/portfolios/:id/strategies/:coin   코인별 전략 변경
GET    /api/v1/portfolios/:id/strategy-history   전략 변경 이력

# 시뮬레이션
GET    /api/v1/simulation/portfolios             전체 포트폴리오 요약 (총자산/ROI)
GET    /api/v1/simulation/status?coin=&portfolio_id=   코인×포트폴리오 현재 포지션
GET    /api/v1/simulation/trades?coin=&portfolio_id=&limit=  거래 내역
POST   /api/v1/simulation/trade                  BUY/SELL/HOLD 실행
GET    /api/v1/simulation/performance            1일/7일/30일 기간별 수익률 vs 코인 상승률
```

---

## DB 테이블

| 테이블 | 갱신 주기 | 설명 |
|--------|----------|------|
| `daily_prices` | 01:00 KST / POST /sync | Binance 일봉 OHLCV + 기술지표 |
| `price_ticker` | 10초 | 실시간 가격 + 일봉 기반 22개 지표 |
| `hourly_ticker` | 10분 | 1시간봉 기반 14개 지표 |
| `strategies` | API | coin-agnostic 전략 라이브러리 |
| `strategy_versions` | 자동 | notes 변경 시 버전 스냅샷 |
| `strategy_history` | 자동 | 전략 생성/수정 이력 |
| `portfolios` | API | 포트폴리오 목록 (notify_on_trade, risk_limit_pct) |
| `portfolio_strategies` | API | 포트폴리오×코인 활성 전략 매핑 |
| `portfolio_strategy_history` | 자동 | 전략 변경 이력 |
| `sim_state` | POST /trade | 포트폴리오×코인 현재 포지션 |
| `sim_trades` | POST /trade | BUY/SELL 거래 내역 (before/after 스냅샷) |

### price_ticker 주요 컬럼 (22개)
`current_price, prev_price, volatility, ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower, ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume`

### hourly_ticker 컬럼 (14개)
`ema9_1h, ema21_1h, rsi14_1h, macd_1h, macd_signal_1h, macd_hist_1h, bb_upper_1h, bb_middle_1h, bb_lower_1h, vwap_24h, price_change_4h, price_change_24h`

---

## 등록된 전략 라이브러리

| id | 이름 | 핵심 진입 조건 | 적합 환경 |
|----|------|--------------|----------|
| 1 | 분할매수형 | RSI < 45, 가격 < MA20 | 횡보/하락 |
| 2 | 모멘텀형 | RSI > 55, MACD 양전환, MA7 > MA20 | 상승 추세 |
| 4 | 공격형 | RSI < 40, BB 하단 근접 | 과매도 반등 |
| 5 | Box Hunter v3 | ADX ≤ 25, RSI ≤ 42, 가격 ≤ BB하단×1.03 | 박스권 |
| 6 | Trend Rider | EMA9/21(1h) 크로스, MACD hist(1h) > 0, VWAP 확인 | 모든 국면 |

---

## 포트폴리오 목록

| id | 이름 | 코인 | 활성 전략 | 삭제 |
|----|------|------|----------|------|
| 1 | 기본 | BTC, ETH | 분할매수형 | ❌ |
| 2 | Box Hunter | BTC, ETH, SOL | Box Hunter v3 | ❌ |
| 3 | 수동매매 | BTC, ETH, SOL | (수동) | ✅ |
| 4 | Trend Rider | BTC, ETH, SOL | Trend Rider | ✅ |

---

## 서비스 레이어 설명

### `services/ticker.go`
- `StartPriceTicker()`: 10초 goroutine
- Binance 실시간 가격 + `CalcIndicators()` → `price_ticker` upsert

### `services/hourly_ticker.go`
- `StartHourlyTicker()`: 10분 goroutine
- Binance 1h 캔들 100개 수집 → EMA9/21, RSI14, MACD, BB(20), VWAP(24h), Δ4h/Δ24h 계산 → `hourly_ticker` upsert

### `services/binance.go`
- `FetchOHLCV()`: 일봉 데이터 수집 + 지표 계산 → `daily_prices`
- `FetchLivePrice()`: Binance 24hr ticker 실시간 조회
- `fetchHourlyCandles()`: 1시간봉 캔들 수집 (hourly_ticker.go 내부용)

### `services/indicators.go`
- `CalcIndicators()`: 일봉 기반 지표 계산 (price_ticker 갱신용)
- 순수 함수: `calcEMA()`, `calcRSI()`, `calcADX()`, `calcTR()`, `mean()`, `stddev()`

### `services/simulation.go`
- `GetOrInitState()`: sim_state 없으면 초기자본으로 초기화
- `ExecuteTrade()`: BUY/SELL 로직, avg_cost 계산, sim_trades 기록
- `GetAllPortfolios()`: 전체 포트폴리오 + 코인별 현재 가치 집계
- `GetPerformance()`: sim_trades 히스토리 역산 → 1일/7일/30일 수익률

---

## 자동화 데이터 흐름

```
Binance API
    ├─ ticker.go (10초)        →  price_ticker  (일봉 22개 지표)
    ├─ hourly_ticker.go (10분) →  hourly_ticker (1h봉 14개 지표)
    └─ sync_scheduler.go (01:00 KST)  →  daily_prices

gateway-claude-discord
    └─ 30분 크론 (price-check-collect.sh)
           ├─ GET /api/v1/ticker          (일봉 지표)
           ├─ GET /api/v1/ticker/hourly   (1h 지표)
           ├─ GET /api/v1/portfolios
           ├─ GET /api/v1/simulation/status × 포트폴리오 × 코인
           └─ Claude 매매 판단 → POST /api/v1/simulation/trade

브라우저
    ├─ GET /simulation/portfolios    포트폴리오 요약
    ├─ GET /simulation/performance   기간별 수익률
    ├─ GET /simulation/status        코인별 포지션
    └─ GET /simulation/trades        거래 내역
```

---

## 프론트엔드 탭 구성

| 탭 | 주요 컴포넌트 | 설명 |
|----|-------------|------|
| 📊 시세 차트 | ChartPage.tsx | 캔들 차트, MA/BB/RSI/MACD 토글 (TradingView v3.8.0) |
| 📈 시뮬레이션 | SimulationPage.tsx | 포트폴리오 요약, 기간별 수익률 비교, 자산 카드, 거래 히스토리, ROI 차트 |
| 📋 전략 | StrategyPage.tsx | 전략 라이브러리 CRUD, 포트폴리오×코인 전략 매핑 |

---

## 주의사항

- `POST /api/v1/strategy` 크론잡에서 **절대 호출 금지** (새 전략이 생성됨)
- TradingView Lightweight Charts **v3.8.0 고정** (버전 변경 시 API 호환 깨짐)
- 포트폴리오 id=1, id=2 **삭제 불가** (기본/Box Hunter)
- DB 경로: `runtime.Caller(0)` 기준 상대경로 → 실행 위치 변경 시 경로 확인 필요
- Zone.Identifier 파일이 static/ 에 있으면 Go embed 빌드 실패
  → `find static -name "*Zone.Identifier*" -delete` 후 빌드
