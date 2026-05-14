# rules/developer.md — Deb @ bob-crypto-pilot

## 이 프로젝트에서 Deb의 코딩 규칙

### 필수 환경 설정
```bash
export PATH=$PATH:/usr/local/go/bin
cd /home/jhpark/devel/workspace/bob-crypto-pilot/
```

### 코드 구조
```
bob-crypto-pilot/
├── main.go               # 서버 진입점, embed + 라우터
├── db/db.go              # SQLite 연결/초기화
├── models/price.go       # 시세 구조체
├── models/simulation.go  # 시뮬레이션 구조체
├── handlers/price.go     # 시세 HTTP 핸들러 (수정 금지)
├── handlers/simulation.go # 시뮬레이션 HTTP 핸들러
├── services/binance.go   # Binance API 연동 (수정 금지)
├── services/simulation.go # MA 계산 + 시뮬레이션 로직
├── static/               # 프론트엔드 (embed)
└── data/crypto.db        # DB (런타임)
```

### 트레이딩 시뮬레이터 규칙
- 전략: 7일/21일 MA Crossover (매수: 상향 돌파, 매도: 하향 돌파)
- 초기자본: BTC $100, ETH $100 (독립)
- 수수료 없음, 100% 포지션
- SQLite window function으로 MA 계산
- TradingView v3.8.0 `series.setMarkers()` 로 마커 표시

### 코딩 컨벤션
- 에러는 반드시 핸들링, 무시 금지
- 기존 파일(handlers/price.go, services/binance.go) 수정 최소화
- `//go:embed static` 사용 중 → static 파일 변경 후 반드시 재빌드

### 빌드 및 재시작 표준 절차
```bash
export PATH=$PATH:/usr/local/go/bin
cd /home/jhpark/devel/workspace/bob-crypto-pilot/
kill $(cat server.pid) 2>/dev/null || pkill -f "./bob-crypto-pilot" || true
sleep 1
go build -o bob-crypto-pilot .
nohup ./bob-crypto-pilot >> server.log 2>&1 &
echo $! > server.pid
sleep 2
curl -s http://localhost:8080/health
```

### 테스트 기준
- 빌드 성공 필수
- `/health` 200 응답 필수
- 변경된 기능 curl 테스트 필수
- 완료 보고 전 위 3가지 통과해야 함

### UI 규칙
- 다크 테마 유지 (배경: `#1a1a2e`)
- 차트: TradingView Lightweight Charts **v3.8.0** 고정 (버전 변경 금지)
- 상승: `#26a69a`, 하락: `#ef5350`
