package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}

	dbPath := filepath.Join(dataDir, "crypto.db")
	var err error
	DB, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	if err := DB.Ping(); err != nil {
		return err
	}

	if err := createTables(); err != nil {
		return err
	}

	log.Printf("Database initialized at %s", dbPath)
	return nil
}

func createTables() error {
	_, err := DB.Exec(`
	-- portfolios exchange 컬럼 추가 (기존 DB 호환)
	-- SQLite에서는 IF NOT EXISTS 미지원이므로 무시
	`); _ = err

	// portfolios에 exchange 컬럼이 없으면 추가
	if _, err := DB.Exec(`ALTER TABLE portfolios ADD COLUMN exchange TEXT NOT NULL DEFAULT 'binance'`); err != nil {
		// 이미 존재하는 경우 무시
	}

	_, err = DB.Exec(`
	CREATE TABLE IF NOT EXISTS daily_prices (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		coin        TEXT NOT NULL,
		date        TEXT NOT NULL,
		open        REAL NOT NULL,
		high        REAL NOT NULL,
		low         REAL NOT NULL,
		close       REAL NOT NULL,
		volume      REAL NOT NULL,
		ma7         REAL NOT NULL DEFAULT 0,
		ma20        REAL NOT NULL DEFAULT 0,
		ma50        REAL NOT NULL DEFAULT 0,
		ema9        REAL NOT NULL DEFAULT 0,
		ema21       REAL NOT NULL DEFAULT 0,
		rsi14       REAL NOT NULL DEFAULT 0,
		macd        REAL NOT NULL DEFAULT 0,
		macd_signal REAL NOT NULL DEFAULT 0,
		bb_upper    REAL NOT NULL DEFAULT 0,
		bb_middle   REAL NOT NULL DEFAULT 0,
		bb_lower    REAL NOT NULL DEFAULT 0,
		adx14       REAL NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(coin, date)
	);

	CREATE TABLE IF NOT EXISTS price_ticker (
		coin           TEXT PRIMARY KEY,
		checked_at     TEXT NOT NULL,
		current_price  REAL NOT NULL DEFAULT 0,
		prev_price     REAL NOT NULL DEFAULT 0,
		volatility     REAL NOT NULL DEFAULT 0,
		ma7            REAL NOT NULL DEFAULT 0,
		ma20           REAL NOT NULL DEFAULT 0,
		ma50           REAL NOT NULL DEFAULT 0,
		rsi14          REAL NOT NULL DEFAULT 0,
		macd           REAL NOT NULL DEFAULT 0,
		macd_signal    REAL NOT NULL DEFAULT 0,
		bb_upper       REAL NOT NULL DEFAULT 0,
		bb_middle      REAL NOT NULL DEFAULT 0,
		bb_lower       REAL NOT NULL DEFAULT 0,
		ema9           REAL NOT NULL DEFAULT 0,
		ema21          REAL NOT NULL DEFAULT 0,
		adx14          REAL NOT NULL DEFAULT 0,
		atr14          REAL NOT NULL DEFAULT 0,
		atr50          REAL NOT NULL DEFAULT 0,
		volume_ma20    REAL NOT NULL DEFAULT 0,
		highest_high20 REAL NOT NULL DEFAULT 0,
		current_volume REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS hourly_ticker (
		coin             TEXT PRIMARY KEY,
		checked_at       TEXT NOT NULL,
		ema9_1h          REAL NOT NULL DEFAULT 0,
		ema21_1h         REAL NOT NULL DEFAULT 0,
		rsi14_1h         REAL NOT NULL DEFAULT 0,
		macd_1h          REAL NOT NULL DEFAULT 0,
		macd_signal_1h   REAL NOT NULL DEFAULT 0,
		macd_hist_1h     REAL NOT NULL DEFAULT 0,
		bb_upper_1h      REAL NOT NULL DEFAULT 0,
		bb_middle_1h     REAL NOT NULL DEFAULT 0,
		bb_lower_1h      REAL NOT NULL DEFAULT 0,
		vwap_24h         REAL NOT NULL DEFAULT 0,
		price_change_4h  REAL NOT NULL DEFAULT 0,
		price_change_24h REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS portfolios (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		name            TEXT NOT NULL UNIQUE,
		description     TEXT NOT NULL DEFAULT '',
		notify_on_trade INTEGER NOT NULL DEFAULT 1,
		risk_limit_pct  REAL NOT NULL DEFAULT 15.0,
		created_at      TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS sim_state (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		coin            TEXT NOT NULL,
		account         TEXT NOT NULL DEFAULT 'default',
		portfolio_id    INTEGER NOT NULL DEFAULT 1,
		cash            REAL NOT NULL DEFAULT 100.0,
		units           REAL NOT NULL DEFAULT 0.0,
		initial_capital REAL NOT NULL DEFAULT 100.0,
		position        TEXT NOT NULL DEFAULT 'CASH',
		avg_cost        REAL NOT NULL DEFAULT 0,
		updated_at      TEXT NOT NULL,
		UNIQUE(portfolio_id, coin)
	);

	CREATE TABLE IF NOT EXISTS sim_trades (
		id           INTEGER PRIMARY KEY AUTOINCREMENT,
		coin         TEXT NOT NULL,
		account      TEXT NOT NULL DEFAULT 'default',
		portfolio_id INTEGER NOT NULL DEFAULT 1,
		action       TEXT NOT NULL,
		price        REAL NOT NULL,
		units        REAL NOT NULL DEFAULT 0.0,
		cash_before  REAL NOT NULL,
		cash_after   REAL NOT NULL,
		units_before REAL NOT NULL,
		units_after  REAL NOT NULL,
		reason       TEXT,
		executed_at  TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS portfolio_strategies (
		id               INTEGER PRIMARY KEY AUTOINCREMENT,
		portfolio_id     INTEGER NOT NULL REFERENCES portfolios(id),
		coin             TEXT NOT NULL,
		strategy_id      INTEGER NOT NULL DEFAULT 0,
		selected_by      TEXT NOT NULL DEFAULT 'system',
		selected_at      TEXT NOT NULL,
		selection_reason TEXT NOT NULL DEFAULT '',
		UNIQUE(portfolio_id, coin)
	);

	CREATE TABLE IF NOT EXISTS portfolio_strategy_history (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		portfolio_id  INTEGER NOT NULL,
		coin          TEXT NOT NULL,
		strategy_id   INTEGER,
		strategy_name TEXT NOT NULL DEFAULT '',
		action        TEXT NOT NULL,
		changed_by    TEXT NOT NULL DEFAULT 'system',
		changed_at    TEXT NOT NULL,
		note          TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS strategies (
		id              INTEGER PRIMARY KEY AUTOINCREMENT,
		coin            TEXT NOT NULL DEFAULT '',
		name            TEXT NOT NULL,
		description     TEXT NOT NULL DEFAULT '',
		signal          TEXT NOT NULL DEFAULT 'HOLD',
		rsi_buy         REAL NOT NULL DEFAULT 35,
		rsi_sell        REAL NOT NULL DEFAULT 65,
		profit_take_pct REAL NOT NULL DEFAULT 15,
		stop_loss_pct   REAL NOT NULL DEFAULT -20,
		notes           TEXT NOT NULL DEFAULT '',
		version         INTEGER NOT NULL DEFAULT 1,
		created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS strategy_versions (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		strategy_id INTEGER NOT NULL,
		version     INTEGER NOT NULL,
		name        TEXT NOT NULL,
		description TEXT NOT NULL DEFAULT '',
		notes       TEXT NOT NULL DEFAULT '',
		changed_at  TEXT NOT NULL,
		FOREIGN KEY (strategy_id) REFERENCES strategies(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS strategy_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		coin        TEXT NOT NULL,
		strategy_id INTEGER NOT NULL,
		action      TEXT NOT NULL,
		changed_by  TEXT NOT NULL DEFAULT 'system',
		changed_at  TEXT NOT NULL,
		snapshot    TEXT NOT NULL DEFAULT ''
	);

	CREATE TABLE IF NOT EXISTS upbit_daily_prices (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		coin        TEXT NOT NULL,
		date        TEXT NOT NULL,
		open        REAL NOT NULL,
		high        REAL NOT NULL,
		low         REAL NOT NULL,
		close       REAL NOT NULL,
		volume      REAL NOT NULL,
		ma7         REAL NOT NULL DEFAULT 0,
		ma20        REAL NOT NULL DEFAULT 0,
		ma50        REAL NOT NULL DEFAULT 0,
		ema9        REAL NOT NULL DEFAULT 0,
		ema21       REAL NOT NULL DEFAULT 0,
		rsi14       REAL NOT NULL DEFAULT 0,
		macd        REAL NOT NULL DEFAULT 0,
		macd_signal REAL NOT NULL DEFAULT 0,
		bb_upper    REAL NOT NULL DEFAULT 0,
		bb_middle   REAL NOT NULL DEFAULT 0,
		bb_lower    REAL NOT NULL DEFAULT 0,
		adx14       REAL NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(coin, date)
	);

	CREATE TABLE IF NOT EXISTS upbit_price_ticker (
		coin           TEXT PRIMARY KEY,
		checked_at     TEXT NOT NULL,
		current_price  REAL NOT NULL DEFAULT 0,
		prev_price     REAL NOT NULL DEFAULT 0,
		volatility     REAL NOT NULL DEFAULT 0,
		ma7            REAL NOT NULL DEFAULT 0,
		ma20           REAL NOT NULL DEFAULT 0,
		ma50           REAL NOT NULL DEFAULT 0,
		rsi14          REAL NOT NULL DEFAULT 0,
		macd           REAL NOT NULL DEFAULT 0,
		macd_signal    REAL NOT NULL DEFAULT 0,
		bb_upper       REAL NOT NULL DEFAULT 0,
		bb_middle      REAL NOT NULL DEFAULT 0,
		bb_lower       REAL NOT NULL DEFAULT 0,
		ema9           REAL NOT NULL DEFAULT 0,
		ema21          REAL NOT NULL DEFAULT 0,
		adx14          REAL NOT NULL DEFAULT 0,
		atr14          REAL NOT NULL DEFAULT 0,
		atr50          REAL NOT NULL DEFAULT 0,
		volume_ma20    REAL NOT NULL DEFAULT 0,
		highest_high20 REAL NOT NULL DEFAULT 0,
		current_volume REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS upbit_hourly_ticker (
		coin             TEXT PRIMARY KEY,
		checked_at       TEXT NOT NULL,
		ema9_1h          REAL NOT NULL DEFAULT 0,
		ema21_1h         REAL NOT NULL DEFAULT 0,
		rsi14_1h         REAL NOT NULL DEFAULT 0,
		macd_1h          REAL NOT NULL DEFAULT 0,
		macd_signal_1h   REAL NOT NULL DEFAULT 0,
		macd_hist_1h     REAL NOT NULL DEFAULT 0,
		bb_upper_1h      REAL NOT NULL DEFAULT 0,
		bb_middle_1h     REAL NOT NULL DEFAULT 0,
		bb_lower_1h      REAL NOT NULL DEFAULT 0,
		vwap_24h         REAL NOT NULL DEFAULT 0,
		price_change_4h  REAL NOT NULL DEFAULT 0,
		price_change_24h REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS bithumb_daily_prices (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		coin        TEXT NOT NULL,
		date        TEXT NOT NULL,
		open        REAL NOT NULL,
		high        REAL NOT NULL,
		low         REAL NOT NULL,
		close       REAL NOT NULL,
		volume      REAL NOT NULL,
		ma7         REAL NOT NULL DEFAULT 0,
		ma20        REAL NOT NULL DEFAULT 0,
		ma50        REAL NOT NULL DEFAULT 0,
		ema9        REAL NOT NULL DEFAULT 0,
		ema21       REAL NOT NULL DEFAULT 0,
		rsi14       REAL NOT NULL DEFAULT 0,
		macd        REAL NOT NULL DEFAULT 0,
		macd_signal REAL NOT NULL DEFAULT 0,
		bb_upper    REAL NOT NULL DEFAULT 0,
		bb_middle   REAL NOT NULL DEFAULT 0,
		bb_lower    REAL NOT NULL DEFAULT 0,
		adx14       REAL NOT NULL DEFAULT 0,
		created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(coin, date)
	);

	CREATE TABLE IF NOT EXISTS bithumb_price_ticker (
		coin           TEXT PRIMARY KEY,
		checked_at     TEXT NOT NULL,
		current_price  REAL NOT NULL DEFAULT 0,
		prev_price     REAL NOT NULL DEFAULT 0,
		volatility     REAL NOT NULL DEFAULT 0,
		ma7            REAL NOT NULL DEFAULT 0,
		ma20           REAL NOT NULL DEFAULT 0,
		ma50           REAL NOT NULL DEFAULT 0,
		rsi14          REAL NOT NULL DEFAULT 0,
		macd           REAL NOT NULL DEFAULT 0,
		macd_signal    REAL NOT NULL DEFAULT 0,
		bb_upper       REAL NOT NULL DEFAULT 0,
		bb_middle      REAL NOT NULL DEFAULT 0,
		bb_lower       REAL NOT NULL DEFAULT 0,
		ema9           REAL NOT NULL DEFAULT 0,
		ema21          REAL NOT NULL DEFAULT 0,
		adx14          REAL NOT NULL DEFAULT 0,
		atr14          REAL NOT NULL DEFAULT 0,
		atr50          REAL NOT NULL DEFAULT 0,
		volume_ma20    REAL NOT NULL DEFAULT 0,
		highest_high20 REAL NOT NULL DEFAULT 0,
		current_volume REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS bithumb_hourly_ticker (
		coin             TEXT PRIMARY KEY,
		checked_at       TEXT NOT NULL,
		ema9_1h          REAL NOT NULL DEFAULT 0,
		ema21_1h         REAL NOT NULL DEFAULT 0,
		rsi14_1h         REAL NOT NULL DEFAULT 0,
		macd_1h          REAL NOT NULL DEFAULT 0,
		macd_signal_1h   REAL NOT NULL DEFAULT 0,
		macd_hist_1h     REAL NOT NULL DEFAULT 0,
		bb_upper_1h      REAL NOT NULL DEFAULT 0,
		bb_middle_1h     REAL NOT NULL DEFAULT 0,
		bb_lower_1h      REAL NOT NULL DEFAULT 0,
		vwap_24h         REAL NOT NULL DEFAULT 0,
		price_change_4h  REAL NOT NULL DEFAULT 0,
		price_change_24h REAL NOT NULL DEFAULT 0
	);
	`)
	return err
}

// SeedBoxHunterStrategies inserts Box Hunter strategy if it doesn't already exist.
func SeedBoxHunterStrategies() {
	var count int
	if err := DB.QueryRow("SELECT COUNT(*) FROM strategies WHERE name='Box Hunter'").Scan(&count); err != nil || count > 0 {
		return
	}

	notes := `## Box Hunter (박스권 최적화 전략)

### 핵심 철학
ADX 낮을 때(횡보) BB 하단 + RSI 과매도에서 분할 매수, 거래량 없는 상단 돌파 시 과감히 익절. 3~8% 수익을 복리로 누적.

### 1단계: 환경 판별 (진입 전 필수)
- ADX < 20 → 박스권 확인, 전략 활성화
- ADX 20~25 → 경계, 포지션 축소
- ADX > 25 상승 중 → 즉시 박스권 매매 중단
- ATR14 급등 (전일比 20%↑) → 박스 이탈 신호, 즉시 청산

### 2단계: 매수 조건
필수 조건 (모두 충족):
- rsi14 ≤ 35 (과매도)
- current_price ≤ bb_lower × 1.01 (BB 하단 1% 이내)
- atr14 ≤ atr50 (변동성 낮음 = 진짜 지지)

강화 조건 (1개 이상):
- MACD Histogram 상승 (상승 다이버전스)
- current_volume > volume_ma20 × 1.2 + 양봉

분할 진입:
- 1차 30% → BB 하단 터치 시
- 2차 20% → 하단 재확인 시

### 3단계: 익절 (분할)
- 1차: ema21 도달 → 보유량 40% 매도
- 2차: bb_middle 도달 → 보유량 40% 추가 매도
- 3차: bb_upper × 0.99 도달 → 잔여 전량 매도
- 거짓 돌파: highest_high20 돌파 시도 + 거래량 < volume_ma20 → 매도

추세 전환 시:
- highest_high20 강한 거래량 돌파 → Box Hunter 종료, Momentum Breakout 전환

### 4단계: 손절
- 기본: bb_lower - ATR14 × 1.5 이탈 시 전량 청산
- 긴급: ADX > 25 급등 or ATR14 전일比 20%↑ → 즉시 청산

### 지표 활용 요약
- ADX14: 박스권 vs 추세장 판별 필터
- ATR14/50: 손절 기준 + 변동성 감지
- EMA21: 분할 익절 1차 기준선
- VolumeMA20: 돌파 진위 판별 (가짜 돌파 필터)
- BB: 매수/매도 범위 설정

### 전략 특성
- 적합 환경: ADX < 20 박스권
- 예상 승률: 55~65%
- 목표 수익: 3~8% per trade
- 손절: ATR 기반 기계적 손절`

	DB.Exec(`INSERT INTO strategies (coin, name, description, signal, rsi_buy, rsi_sell, profit_take_pct, stop_loss_pct, notes)
		VALUES ('', 'Box Hunter', '박스권 최적화 - BB하단+RSI과매도 매수, 분할 익절', 'HOLD', 35, 68, 8, -7, ?)`,
		notes)
	log.Println("Box Hunter strategy seeded")
}
