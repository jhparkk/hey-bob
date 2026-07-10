# gateway-claude-discord × bob-crypto-pilot 연관 관계 분석

> 작성일: 2026-07-09  
> 작성자: Bob (Manager)

---

## 1. 시스템 개요

두 서비스는 독립된 프로세스로 실행되며, **HTTP REST API (localhost:8080)** 를 통해 느슨하게 결합된다.

```
┌─────────────────────────────────────────────────────────┐
│               jhparkk (Discord 사용자)                   │
└────────────────────────┬────────────────────────────────┘
                         │ Discord WebSocket
                         ▼
┌─────────────────────────────────────────────────────────┐
│           gateway-claude-discord  (:없음)                │
│                                                          │
│  ┌─────────────┐   ┌──────────────┐   ┌──────────────┐ │
│  │   discord/  │   │   session/   │   │    cron/     │ │
│  │  메시지수신  │──▶│  Claude CLI  │◀──│  스케줄러    │ │
│  │  명령어처리  │   │  세션관리    │   │  6개 잡      │ │
│  └─────────────┘   └──────────────┘   └──────┬───────┘ │
│                                               │          │
│  ┌─────────────────────────────────────────┐  │          │
│  │         scripts/                        │  │          │
│  │  price-check-collect.sh (30분)          │──┤          │
│  │  daily-report-collect.sh (09:00)        │  │          │
│  └─────────────────────────────────────────┘  │          │
└──────────────────────────────────────────────┼──────────┘
                                               │ curl
                                               │ localhost:8080
                                               ▼
┌─────────────────────────────────────────────────────────┐
│           bob-crypto-pilot  (:8080)                      │
│                                                          │
│  ┌──────────┐  ┌───────────┐  ┌────────────────────┐   │
│  │ handlers │  │ services  │  │  goroutines (7개)  │   │
│  │  REST API│  │  비즈니스  │  │  Binance 10s/10m   │   │
│  │  라우팅  │  │  로직     │  │  Upbit   10s/10m   │   │
│  └────┬─────┘  └─────┬─────┘  │  Bithumb 10s/10m   │   │
│       │              │         │  DailySync 01:00   │   │
│       ▼              ▼         └────────────────────┘   │
│  ┌────────────────────────┐                              │
│  │   SQLite (crypto.db)   │                              │
│  │  시세 / 포트폴리오     │                              │
│  │  전략 / 거래내역       │                              │
│  └────────────────────────┘                              │
└─────────────────────────────────────────────────────────┘
```

---

## 2. 두 서비스의 역할 구분

| 항목 | gateway-claude-discord | bob-crypto-pilot |
|------|----------------------|-----------------|
| 포트 | 없음 (WebSocket 클라이언트) | :8080 |
| 역할 | Claude AI 중계 + 크론 스케줄러 | 시세 수집 + 시뮬레이션 엔진 |
| 언어/프레임워크 | Go + discordgo | Go + Gin |
| DB | SQLite (gateway.db) — sessions 테이블 | SQLite (crypto.db) — 20개 테이블 |
| 외부 의존성 | Discord API, Claude CLI | Binance API, Upbit API, Bithumb API |
| 서비스 파일 | /etc/systemd/system/gateway-claude-discord.service | /etc/systemd/system/bob-crypto-pilot.service |

---

## 3. 연결 경로 (Connection Points)

gateway가 bob-crypto-pilot에 접근하는 경로는 **2가지**다. Go 코드에서 직접 HTTP 호출은 없으며, 모두 shell 또는 Claude를 통해 이루어진다.

### 경로 A — preprocessScript (Shell → bob-crypto-pilot API)

크론잡 실행 전 shell 스크립트가 시장 데이터를 수집해 Claude 컨텍스트로 주입한다.

```
cron.Handler.runJob()
    └─ runScript(ctx, preprocessScript)  # bash 실행
           └─ price-check-collect.sh / daily-report-collect.sh
                  └─ curl http://localhost:8080/api/v1/...
                         └─ bob-crypto-pilot이 응답
                                └─ stdout을 Claude 프롬프트 앞에 "\n\n## 수집된 시장 데이터\n"로 붙임
```

### 경로 B — Claude Bash 도구 (Claude → bob-crypto-pilot API)

Claude가 Bash 도구를 통해 직접 API를 호출하여 매매를 실행한다.

```
session.RunIsolated() → claude CLI
    └─ Claude가 Bash 도구로 curl 실행
           └─ POST http://localhost:8080/api/v1/simulation/trade  ← 실제 매매 체결
           └─ PUT  http://localhost:8080/api/v1/strategy/6        ← Trend Rider 조건 업데이트
```

---

## 4. 크론잡별 API 호출 상세

### 4-1. daily-crypto-report (매일 09:00 KST)

**흐름:** daily-report-collect.sh → Claude 리포트 작성 → Discord DM

| 스크립트 호출 API | 용도 |
|-----------------|------|
| `GET /api/v1/ticker` | 바이낸스 BTC/ETH/SOL 일봉 지표 |
| `GET /api/v1/prices?coin=<coin>&period=3d` | 최근 3일 일봉 (양봉/음봉 방향) |
| `GET /api/v1/portfolios` | 전체 포트폴리오 목록 |
| `GET /api/v1/simulation/status?coin=<coin>&portfolio_id=<id>` | 포트폴리오별 현재 포지션/ROI |
| `GET /api/v1/simulation/trades?...&limit=20` | 어제 거래 내역 |

**Claude가 Bash로 호출하는 API:** 없음 (읽기 전용, 리포트만 생성)

---

### 4-2. trend-rider-optimizer (매일 08:30 KST)

**흐름:** Claude가 시장 데이터 수집 → Trend Rider 조건 계산 → 전략 업데이트

| Claude Bash 호출 API | 용도 |
|--------------------|------|
| `GET /api/v1/ticker` | 바이낸스 일봉 지표 |
| `GET /api/v1/ticker/hourly` | 바이낸스 1시간봉 지표 |
| `GET /api/v1/upbit/ticker` | 업비트 일봉 지표 |
| `GET /api/v1/upbit/ticker/hourly` | 업비트 1시간봉 지표 |
| `GET /api/v1/bithumb/ticker` | 빗썸 일봉 지표 |
| `GET /api/v1/bithumb/ticker/hourly` | 빗썸 1시간봉 지표 |
| `GET /api/v1/simulation/trades?coin=<coin>&portfolio_id=<4/8/15>&limit=10` | Trend Rider 포트폴리오 최근 거래 |
| **`PUT /api/v1/strategy/6`** | Trend Rider(id=6) 매수/매도 조건 업데이트 |

---

### 4-3. 30min-price-check (30분마다, stagger 최대 5분)

**흐름:** price-check-collect.sh → Claude 매매 판단 → 거래 실행

**preprocessScript (price-check-collect.sh) 호출:**

| API | 용도 |
|-----|------|
| `GET /api/v1/ticker` + `/ticker/hourly` | 바이낸스 일봉+1h 지표 |
| `GET /api/v1/upbit/ticker` + `/upbit/ticker/hourly` | 업비트 일봉+1h 지표 |
| `GET /api/v1/bithumb/ticker` + `/bithumb/ticker/hourly` | 빗썸 일봉+1h 지표 |
| `GET /api/v1/portfolios` | 포트폴리오 목록 |
| `GET /api/v1/strategy` | 전략 조건 참조 (중복 제거 후 1회) |
| `GET /api/v1/simulation/status?coin=<coin>&portfolio_id=<id>` | 포트폴리오×코인 현재 상태 |
| `GET /api/v1/portfolios/<id>/strategy-history?coin=<coin>` | 활성 전략 확인 |
| `GET /api/v1/prices?coin=<coin>&period=3d` | 최근 3일 일봉 방향 |

**대상 포트폴리오:** pf1,2,4(바낸) / pf5,6,8,10,19(업비트) / pf12,13,15,16,21(빗썸)

**Claude Bash 호출:** `POST /api/v1/simulation/trade` (매매 조건 충족 시)

---

### 4-4. news-sentiment-trade (4시간마다, stagger 최대 2분)

**흐름:** Claude가 WebSearch로 뉴스 수집 → 센티멘트 점수 → pf11 매매

| Claude Bash 호출 API | 용도 |
|--------------------|------|
| `GET /api/v1/simulation/status?coin=<coin>&portfolio_id=11` | pf11 현재 포지션 |
| `GET /api/v1/ticker` | 바이낸스 현재 USD 가격 |
| **`POST /api/v1/simulation/trade`** | pf11 매매 실행 (센티멘트 ≥+1.0 BUY / ≤-1.0 SELL) |

---

### 4-5. fear-greed-trade (09:05, 21:05 KST)

**흐름:** F&G API 조회 → pf9/23/24 매매

| Claude Bash 호출 API | 용도 |
|--------------------|------|
| `GET /api/v1/simulation/status?coin=<coin>&portfolio_id=<9/23/24>` | 3개 포트폴리오 현재 포지션 |
| `GET /api/v1/ticker` | 바이낸스 USD 가격 (pf9용) |
| `GET /api/v1/upbit/ticker` | 업비트 KRW 가격 (pf23용) |
| `GET /api/v1/bithumb/ticker` | 빗썸 KRW 가격 (pf24용) |
| **`POST /api/v1/simulation/trade`** | F&G≤25 → BUY / F&G≥75 → SELL |

---

### 4-6. btc-dominance-trade (2시간마다)

**흐름:** BTC 도미넌스 조회 → pf18/20/22 ETH/SOL 매매

| Claude Bash 호출 API | 용도 |
|--------------------|------|
| `GET /api/v1/simulation/status?coin=<ETH/SOL>&portfolio_id=<18/20/22>` | 포지션 확인 |
| `GET /api/v1/ticker` | 바이낸스 RSI + USD 가격 (pf18용) |
| `GET /api/v1/upbit/ticker` | 업비트 RSI + KRW 가격 (pf20용) |
| `GET /api/v1/bithumb/ticker` | 빗썸 RSI + KRW 가격 (pf22용) |
| **`POST /api/v1/simulation/trade`** | BTC.D<58% → BUY / BTC.D>60% → SELL |

---

## 5. 포트폴리오-크론잡 매핑 전체 현황

| 포트폴리오 | 이름 | 거래소 | 담당 크론잡 |
|-----------|------|--------|-----------|
| pf1 | [바낸] 분할매수 | Binance USD | 30min-price-check |
| pf2 | [바낸] Box Hunter | Binance USD | 30min-price-check |
| pf3 | [바낸] 수동매매 | Binance USD | 30min-price-check |
| pf4 | [바낸] Trend Rider | Binance USD | 30min-price-check |
| pf5 | [업비트] 분할매수 | Upbit KRW | 30min-price-check |
| pf6 | [업비트] Box Hunter | Upbit KRW | 30min-price-check |
| pf7 | [업비트] 수동매매 | Upbit KRW | 30min-price-check |
| pf8 | [업비트] Trend Rider | Upbit KRW | 30min-price-check |
| pf9 | [바낸] Fear&Greed | Binance USD | fear-greed-trade |
| pf10 | [업비트] 김치프리미엄 | Upbit KRW | 30min-price-check |
| pf11 | [바낸] 뉴스센티멘트 | Binance USD | news-sentiment-trade |
| pf12 | [빗썸] 분할매수 | Bithumb KRW | 30min-price-check |
| pf13 | [빗썸] Box Hunter | Bithumb KRW | 30min-price-check |
| pf14 | [빗썸] 수동매매 | Bithumb KRW | 30min-price-check |
| pf15 | [빗썸] Trend Rider | Bithumb KRW | 30min-price-check |
| pf16 | [빗썸] 김치프리미엄 | Bithumb KRW | 30min-price-check |
| pf17 | [바낸] 변동성 돌파 | Binance USD | 30min-price-check |
| pf18 | [바낸] BTC 도미넌스 | Binance USD | btc-dominance-trade |
| pf19 | [업비트] 변동성 돌파 | Upbit KRW | 30min-price-check |
| pf20 | [업비트] BTC 도미넌스 | Upbit KRW | btc-dominance-trade |
| pf21 | [빗썸] 변동성 돌파 | Bithumb KRW | 30min-price-check |
| pf22 | [빗썸] BTC 도미넌스 | Bithumb KRW | btc-dominance-trade |
| pf23 | [업비트] Fear&Greed 역발상 | Upbit KRW | fear-greed-trade |
| pf24 | [빗썸] Fear&Greed 역발상 | Bithumb KRW | fear-greed-trade |

---

## 6. 데이터 흐름 상세

### 6-1. 시세 데이터 갱신 주기

```
외부 거래소 API
    │
    ├─ Binance: /api/v3/klines, /ticker/24hr
    │     ├─ 일봉 ticker goroutine    → price_ticker 테이블        [10초]
    │     ├─ 1시간봉 ticker goroutine → hourly_ticker 테이블       [10분]
    │     └─ DailySyncScheduler      → daily_prices 테이블         [01:00 KST]
    │
    ├─ Upbit: api.upbit.com/v1/...
    │     ├─ 일봉 ticker goroutine    → upbit_price_ticker 테이블  [10초]
    │     └─ 1시간봉 ticker goroutine → upbit_hourly_ticker 테이블 [10분]
    │
    └─ Bithumb: api.bithumb.com/public/...
          ├─ 일봉 ticker goroutine    → bithumb_price_ticker 테이블  [10초]
          └─ 1시간봉 ticker goroutine → bithumb_hourly_ticker 테이블 [10분]
```

### 6-2. 매매 실행 전체 흐름 (30min-price-check 기준)

```
[T-5분] price-check-collect.sh 실행
    ├─ 3거래소 ticker/hourly API 수집
    ├─ 외부 API: F&G Index, USD/KRW 환율, BTC 도미넌스
    ├─ 김치프리미엄 계산 (Python3 인라인)
    ├─ 전략 조건 참조표 1회 출력
    └─ 포트폴리오×코인 상태 수집 (16개 포트폴리오 × 최대 3코인)

[T+0] Claude Haiku 실행 (claude --print --dangerously-skip-permissions)
    ├─ 시스템 프롬프트: "WSL Ubuntu 로컬 서버, localhost 접근 가능"
    ├─ 수집된 시장 데이터 컨텍스트 수신
    ├─ 판단 규칙 적용:
    │   ├─ 리스크 한도 체크 (ROI < -risk_limit_pct → 즉시 청산)
    │   ├─ ADX 기반 시장 국면 판단
    │   ├─ 일봉 바이어스 확인
    │   └─ 전략별 조건 적용 (Trend Rider는 1h 지표, 나머지는 일봉)
    └─ 조건 충족 시 Bash 도구로:
           POST http://localhost:8080/api/v1/simulation/trade

[T+최대 5분] 응답
    ├─ 거래 발생 시: "⚡ HH:MM KST\n[포트폴리오] [코인] BUY/SELL 가격\n..." 출력
    └─ delivery.notifyOnlyIf: "⚡" 포함 시에만 Discord DM 전송
           quietHours: 20:00~09:00 KST (야간 알림 억제)
```

---

## 7. 전략(Strategy) 시스템과 크론잡의 관계

### 전략 notes 필드가 AI 판단 기준 문서 역할

```
PUT /api/v1/strategy/6  ← trend-rider-optimizer가 매일 08:30에 업데이트
    ├─ notes 필드 변경 → strategy_versions에 이전 버전 스냅샷
    └─ version 번호 +1 (자동)

30min-price-check 실행 시
    └─ price-check-collect.sh가 /api/v1/strategy 호출
           └─ 전략 notes를 Claude 컨텍스트에 포함 (전략 id별 1회)
                  └─ Claude가 notes 기반으로 매매 조건 판단
```

### 전략-포트폴리오 매핑 구조

```
strategies (전략 라이브러리)
    │ id, name, notes (매매 규칙)
    │
    └─ portfolio_strategies (매핑 테이블)
           │ UNIQUE(portfolio_id, coin) → 포트폴리오×코인당 1개 전략
           │
           └─ portfolios (포트폴리오)
                  │ exchange: binance / upbit / bithumb
                  └─ sim_state (현재 포지션)
                         position: CASH / HOLDING
                         cash, units, avg_cost
```

---

## 8. bob-crypto-pilot이 제공하는 전체 API 목록

### 시스템
- `GET /health`

### 가격 (Binance)
- `POST /api/v1/sync`
- `GET /api/v1/prices`
- `GET /api/v1/prices/latest`
- `GET /api/v1/price/live`

### Ticker
- `GET /api/v1/ticker`
- `GET /api/v1/ticker/hourly`
- `GET /api/v1/upbit/ticker`
- `GET /api/v1/upbit/ticker/hourly`
- `GET /api/v1/bithumb/ticker`
- `GET /api/v1/bithumb/ticker/hourly`

### 전략
- `GET/POST /api/v1/strategy`
- `PUT/DELETE /api/v1/strategy/:id`
- `GET /api/v1/strategy/:id/versions`
- `GET /api/v1/strategy/history`
- `PATCH /api/v1/strategy/:coin/active`

### 포트폴리오
- `GET/POST /api/v1/portfolios`
- `PUT/DELETE /api/v1/portfolios/:id`
- `POST /api/v1/portfolios/:id/reset`
- `GET /api/v1/portfolios/:id/strategies`
- `PATCH /api/v1/portfolios/:id/strategies/:coin`
- `GET /api/v1/portfolios/:id/strategy-history`
- `POST /api/v1/portfolios/:id/coins`
- `DELETE /api/v1/portfolios/:id/coins/:coin`

### 시뮬레이션 (Binance)
- `GET /api/v1/simulation/status`
- `POST /api/v1/simulation/trade`  ← 크론잡이 가장 많이 호출하는 엔드포인트
- `GET /api/v1/simulation/portfolios`
- `GET /api/v1/simulation/trades`
- `GET /api/v1/simulation/performance`

### Upbit/Bithumb 시뮬레이션
- `GET /api/v1/upbit/simulation/portfolios`
- `GET /api/v1/upbit/simulation/performance`
- `POST /api/v1/upbit/sync`
- `GET /api/v1/bithumb/simulation/portfolios`
- `GET /api/v1/bithumb/simulation/performance`
- `POST /api/v1/bithumb/sync`

---

## 9. 주요 설계 결정 및 주의사항

### 의존 방향: 단방향
gateway → bob-crypto-pilot (항상 gateway가 클라이언트)  
bob-crypto-pilot은 gateway의 존재를 모른다.

### bob-crypto-pilot이 내려가면
모든 크론잡이 API 호출 실패 → Claude가 에러 응답 → Discord 알림 없음 (⚡ 미포함)  
`/health` 엔드포인트로 사전 확인 가능.

### 거래소별 가격 혼용 방지
bob-crypto-pilot 내부에 KRW 최솟값 검증 존재:
- BTC < ₩10,000,000 이면 USD 가격으로 간주하고 400 에러 반환
- ETH < ₩100,000, SOL < ₩1,000 동일 적용

### POST /simulation/trade 직접 호출 금지
`POST /api/v1/strategy` (전략 생성)는 크론잡 프롬프트에서 명시적으로 금지.  
`PUT /api/v1/strategy/:id` (수정)만 허용 (id=6 Trend Rider 전용).

### KRW/USD 가격 구분
- pf1~4, pf9, pf11, pf17: Binance USD 가격 사용
- pf5~8, pf10, pf19, pf23: Upbit KRW 가격 사용
- pf12~16, pf21, pf24: Bithumb KRW 가격 사용

---

## 10. Discord 명령어 → gateway → bob-crypto-pilot 체인

```
Discord: "!cron run 30min-price-check"
    └─ discord.go: handleCronCommand("run", "30min-price-check")
           └─ cron.Trigger("30min-price-check")
                  └─ runJob(job) goroutine
                         ├─ bash price-check-collect.sh → localhost:8080 (GET)
                         └─ claude CLI → Bash 도구 → localhost:8080 (POST /simulation/trade)

Discord: "@봇 pf3 BTC 수동 매수해줘"
    └─ session.SendMessage(channelID, message)
           └─ claude --print --resume <session_id>
                  └─ Claude가 Bash 도구로:
                         POST http://localhost:8080/api/v1/simulation/trade
```
