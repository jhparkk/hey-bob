package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/models"
)

const upbitBaseURL = "https://api.upbit.com/v1"

var upbitMarketMap = map[string]string{
	"BTC": "KRW-BTC",
	"ETH": "KRW-ETH",
	"SOL": "KRW-SOL",
}

// FetchUpbitLivePrice fetches real-time ticker from Upbit (KRW price).
func FetchUpbitLivePrice(coin string) (*models.LivePrice, error) {
	market, ok := upbitMarketMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s/ticker?markets=%s", upbitBaseURL, market)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("upbit ticker fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		TradePrice        float64 `json:"trade_price"`
		PrevClosingPrice  float64 `json:"prev_closing_price"`
		SignedChangePrice float64 `json:"signed_change_price"`
		SignedChangeRate  float64 `json:"signed_change_rate"`
		HighPrice         float64 `json:"high_price"`
		LowPrice          float64 `json:"low_price"`
		AccTradeVolume24h float64 `json:"acc_trade_volume_24h"`
		AccTradePrice24h  float64 `json:"acc_trade_price_24h"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || len(raw) == 0 {
		return nil, fmt.Errorf("upbit ticker parse: %w", err)
	}
	r := raw[0]
	return &models.LivePrice{
		Coin:               coin,
		LastPrice:          r.TradePrice,
		PriceChange:        r.SignedChangePrice,
		PriceChangePercent: r.SignedChangeRate * 100,
		HighPrice:          r.HighPrice,
		LowPrice:           r.LowPrice,
		Volume:             r.AccTradeVolume24h,
		QuoteVolume:        r.AccTradePrice24h,
	}, nil
}

// FetchAndStoreUpbit fetches 200 days of daily OHLCV from Upbit and stores in upbit_daily_prices.
func FetchAndStoreUpbit(coin string) (int, error) {
	market, ok := upbitMarketMap[coin]
	if !ok {
		return 0, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s/candles/days?market=%s&count=200", upbitBaseURL, market)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("upbit candles fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var raw []struct {
		CandleDateTimeUTC string  `json:"candle_date_time_utc"`
		OpeningPrice      float64 `json:"opening_price"`
		HighPrice         float64 `json:"high_price"`
		LowPrice          float64 `json:"low_price"`
		TradePrice        float64 `json:"trade_price"`
		CandleAccVolume   float64 `json:"candle_acc_trade_volume"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return 0, fmt.Errorf("upbit candles parse: %w", err)
	}

	prices := make([]models.DailyPrice, 0, len(raw))
	for _, r := range raw {
		t, err := time.Parse("2006-01-02T15:04:05", r.CandleDateTimeUTC)
		if err != nil {
			continue
		}
		prices = append(prices, models.DailyPrice{
			Coin:   coin,
			Date:   t.Format("2006-01-02"),
			Open:   r.OpeningPrice,
			High:   r.HighPrice,
			Low:    r.LowPrice,
			Close:  r.TradePrice,
			Volume: r.CandleAccVolume,
		})
	}

	return upsertUpbitPrices(prices)
}

func upsertUpbitPrices(prices []models.DailyPrice) (int, error) {
	stmt, err := db.DB.Prepare(`
		INSERT INTO upbit_daily_prices (coin, date, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(coin, date) DO UPDATE SET
			open   = excluded.open,
			high   = excluded.high,
			low    = excluded.low,
			close  = excluded.close,
			volume = excluded.volume
	`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	tx, err := db.DB.Begin()
	if err != nil {
		return 0, err
	}
	txStmt := tx.Stmt(stmt)

	count := 0
	for _, p := range prices {
		result, err := txStmt.Exec(p.Coin, p.Date, p.Open, p.High, p.Low, p.Close, p.Volume)
		if err != nil {
			tx.Rollback()
			return 0, err
		}
		rows, _ := result.RowsAffected()
		if rows > 0 {
			count++
		}
	}
	return count, tx.Commit()
}

// GetUpbitPrices retrieves Upbit daily prices for a coin with optional date range.
func GetUpbitPrices(coin, from, to string) ([]models.DailyPrice, error) {
	query := `SELECT id, coin, date, open, high, low, close, volume, created_at,
	          ma7, ma20, ma50, ema9, ema21, rsi14, macd, macd_signal,
	          bb_upper, bb_middle, bb_lower, adx14
	          FROM upbit_daily_prices WHERE coin = ?`
	args := []interface{}{coin}
	if from != "" {
		query += " AND date >= ?"
		args = append(args, from)
	}
	if to != "" {
		query += " AND date <= ?"
		args = append(args, to)
	}
	query += " ORDER BY date ASC"
	return queryUpbitPrices(query, args...)
}

func queryUpbitPrices(query string, args ...interface{}) ([]models.DailyPrice, error) {
	rows, err := db.DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []models.DailyPrice
	for rows.Next() {
		var p models.DailyPrice
		if err := rows.Scan(
			&p.ID, &p.Coin, &p.Date, &p.Open, &p.High, &p.Low, &p.Close, &p.Volume, &p.CreatedAt,
			&p.MA7, &p.MA20, &p.MA50, &p.EMA9, &p.EMA21, &p.RSI14, &p.MACD, &p.MACDSignal,
			&p.BBUpper, &p.BBMiddle, &p.BBLower, &p.ADX14,
		); err != nil {
			return nil, err
		}
		prices = append(prices, p)
	}
	return prices, rows.Err()
}
