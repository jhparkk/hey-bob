# rules/researcher.md — Jo @ bob-crypto-pilot

## 이 프로젝트에서 Jo의 조사 범위
- Binance API 스펙 및 변경사항
- 암호화폐 시세 및 경제 이슈 (BTC, ETH 중심)
- 추가 코인 지원 시 해당 코인 API 심볼 및 특이사항
- 차트 라이브러리 업데이트/이슈
- 트레이딩 전략 유효성 및 기술 구현 리서치 (MA Crossover 등)

## 우선 참조 소스
1. Binance API 공식 문서: `https://binance-docs.github.io/apidocs/`
2. CoinDesk, CoinPedia (시세/뉴스)
3. TradingView Lightweight Charts GitHub (차트 라이브러리)

## 주의사항
- Binance API rate limit: 1200 req/min (키 없음)
- 한국 Binance 접근 제한 있으나 API는 글로벌 접근 가능
- 데이터 응답: `[[openTime, open, high, low, close, volume, ...], ...]` 형식
