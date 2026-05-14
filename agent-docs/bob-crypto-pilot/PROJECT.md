# PROJECT.md — bob-crypto-pilot

> 최종 업데이트: 2026-04-28

---

## 프로젝트 개요

| 항목 | 내용 |
|------|------|
| 프로젝트명 | bob-crypto-pilot |
| 서버 | `http://localhost:8080` |
| 개발 경로 | `/home/jhpark/hey-bob/agent-devs/bob-crypto-pilot/` |
| DB 경로 | `/home/jhpark/hey-bob/agent-devs/bob-crypto-pilot/data/crypto.db` |
| 빌드 | `cd /home/jhpark/hey-bob/agent-devs/bob-crypto-pilot && /usr/local/go/bin/go build -o bob-crypto-pilot .` |
| 실행 | `nohup ./bob-crypto-pilot > server.log 2>&1 &` |

---

## 기술 스택

| 영역 | 스택 |
|------|------|
| BE | Go + Gin 프레임워크 + SQLite (go-sqlite3) |
| FE | React + TypeScript + Vite + Zustand |
| 차트 | TradingView Lightweight Charts v3.8.0 (버전 변경 금지) |
| 가격 데이터 | Binance API (API 키 불필요) |
| 배포 | nohup 직접 실행 (WSL2) |

---

## 프로젝트 구조

```
bob-crypto-pilot/
├── main.go                  ← 서버 진입점, 라우터, //go:embed static
├── go.mod / go.sum
├── db/db.go                 ← DB 초기화, 스키마, 마이그레이션
├── handlers/                ← HTTP 핸들러 (price, simulation, strategy, ticker)
├── models/                  ← 데이터 구조체 (price, simulation)
├── services/                ← 비즈니스 로직 (binance, indicators, simulation, ticker, sync_scheduler)
├── static/                  ← FE 빌드 출력물 (Go embed 대상, fe/ 빌드 시 자동 생성)
├── data/crypto.db           ← SQLite DB
├── cmd/backfill/            ← 과거 가격 데이터 백필 CLI
├── fe/                      ← React+TypeScript 프론트엔드 소스
│   ├── src/
│   │   ├── api/index.ts     ← API 클라이언트 + 전체 타입 정의
│   │   ├── components/      ← Header, PortfolioROIChart, 각종 모달
│   │   ├── pages/           ← ChartPage, SimulationPage, StrategyPage
│   │   ├── hooks/           ← useSimulation, useStrategy
│   │   └── store/           ← Zustand 전역 상태 (simulationStore)
│   └── vite.config.ts       ← outDir: '../static'
└── bob-crypto-pilot         ← 컴파일된 바이너리 (실행 중)
```

---

## 메인 탭 구성

| 탭 | 설명 |
|----|------|
| 📊 시세 차트 | BTC/ETH/SOL 캔들 차트 + 기술 지표 |
| 📈 시뮬레이션 | 포트폴리오별 모의 투자 현황 + 기간별 수익률 + 수동 매매 |
| 📋 전략 | 전략 라이브러리 관리 + 포트폴리오 전략 매핑 |

---

## 기능 목록

### 📊 시세 차트 탭
| 기능 | API |
|------|-----|
| BTC/ETH/SOL 캔들스틱 차트 | `GET /api/v1/prices?coin=&period=` |
| 실시간 가격 (10초 폴링) | `GET /api/v1/ticker` |
| 기술 지표 토글 (MA7/MA20/MA50/BB/RSI/MACD) | `GET /api/v1/ticker` |

### 📈 시뮬레이션 탭
| 기능 | API |
|------|-----|
| 포트폴리오 요약 테이블 (총자산/ROI/포지션) | `GET /api/v1/simulation/portfolios` |
| 기간별 수익률 vs 코인 가격 상승률 (1일/7일/30일) | `GET /api/v1/simulation/performance` |
| 자산 현황 카드 (현금/코인/수익률) | `GET /api/v1/simulation/status?coin=&portfolio_id=` |
| 포트폴리오 ROI 차트 | `GET /api/v1/simulation/trades` |
| 거래 히스토리 | `GET /api/v1/simulation/trades?coin=&portfolio_id=` |
| 수동 BUY/SELL | `POST /api/v1/simulation/trade` |
| 포트폴리오 CRUD (id≤2 삭제 불가) | `POST/PUT/DELETE /api/v1/portfolios` |
| 알림 ON/OFF, 리스크 한도 설정 | `PUT /api/v1/portfolios/:id` |

### 📋 전략 탭
| 기능 | API |
|------|-----|
| 전략 라이브러리 (coin-agnostic) | `GET /api/v1/strategy` |
| 전략 CRUD | `POST/PUT/DELETE /api/v1/strategy/:id` |
| 버전 관리 (notes 변경 시 자동 증가) | `GET /api/v1/strategy/:id/versions` |
| 포트폴리오×코인 전략 매핑 | `PATCH /api/v1/portfolios/:id/strategies/:coin` |
| 전략 변경 이력 | `GET /api/v1/portfolios/:id/strategy-history` |

### ⚙️ 자동화 (Cron — gateway-claude-discord)
| 기능 | 스케줄 |
|------|--------|
| 매일 09:00 시장 리포트 (Discord DM) | `0 9 * * *` KST |
| 30분 자동 트레이드 (전 포트폴리오×코인) | `*/30 * * * *` |
| 일일 가격 동기화 Binance → daily_prices | 01:00 KST (내부 goroutine) |

---

## API 전체 목록

```
GET    /health

GET    /api/v1/prices?coin=&period=
GET    /api/v1/prices/latest?coin=
GET    /api/v1/price/live?coin=
POST   /api/v1/sync
GET    /api/v1/ticker

GET    /api/v1/strategy
POST   /api/v1/strategy
PUT    /api/v1/strategy/:id
DELETE /api/v1/strategy/:id
GET    /api/v1/strategy/:id/versions
GET    /api/v1/strategy/history

GET    /api/v1/portfolios
POST   /api/v1/portfolios
PUT    /api/v1/portfolios/:id
DELETE /api/v1/portfolios/:id
POST   /api/v1/portfolios/:id/reset
GET    /api/v1/portfolios/:id/strategies
PATCH  /api/v1/portfolios/:id/strategies/:coin
GET    /api/v1/portfolios/:id/strategy-history

GET    /api/v1/simulation/portfolios
GET    /api/v1/simulation/status?coin=&portfolio_id=
GET    /api/v1/simulation/trades?coin=&portfolio_id=&limit=
POST   /api/v1/simulation/trade
GET    /api/v1/simulation/performance
```

---

## DB 테이블

| 테이블 | 설명 |
|--------|------|
| `daily_prices` | Binance OHLCV 일봉 + 기술지표 (ma7/ma20/rsi14/adx14 등) |
| `price_ticker` | 실시간 가격 + 22개 기술지표 (10초 갱신) |
| `strategies` | coin-agnostic 전략 라이브러리 |
| `strategy_versions` | 전략 버전 스냅샷 (notes 변경 시) |
| `strategy_history` | 전략 생성/수정 이력 |
| `portfolios` | 포트폴리오 목록 (notify_on_trade, risk_limit_pct 포함) |
| `portfolio_strategies` | 포트폴리오×코인 활성 전략 매핑 |
| `portfolio_strategy_history` | 전략 변경 이력 |
| `sim_state` | 포트폴리오×코인 현재 포지션 (cash, units, avg_cost) |
| `sim_trades` | BUY/SELL 거래 내역 (before/after 스냅샷 포함) |
