package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/models"
)

const bithumbBaseURL = "https://api.bithumb.com/public"

var bithumbCoinMap = map[string]string{
	"BTC": "BTC",
	"ETH": "ETH",
	"SOL": "SOL",
}

// FetchBithumbLivePrice fetches real-time ticker from Bithumb (KRW price).
func FetchBithumbLivePrice(coin string) (*models.LivePrice, error) {
	symbol, ok := bithumbCoinMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s/ticker/%s_KRW", bithumbBaseURL, symbol)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("bithumb ticker fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Status string `json:"status"`
		Data   struct {
			ClosingPrice  string `json:"closing_price"`
			MinPrice      string `json:"min_price"`
			MaxPrice      string `json:"max_price"`
			FluctateAmt   string `json:"fluctate_24H"`
			FluctateRate  string `json:"fluctate_rate_24H"`
			UnitsTraded   string `json:"units_traded_24H"`
			AccTradeValue string `json:"acc_trade_value_24H"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Status != "0000" {
		return nil, fmt.Errorf("bithumb ticker parse error (status=%s): %w", raw.Status, err)
	}
	d := raw.Data

	pf := func(s string) float64 { v, _ := strconv.ParseFloat(s, 64); return v }

	return &models.LivePrice{
		Coin:               coin,
		LastPrice:          pf(d.ClosingPrice),
		PriceChange:        pf(d.FluctateAmt),
		PriceChangePercent: pf(d.FluctateRate),
		HighPrice:          pf(d.MaxPrice),
		LowPrice:           pf(d.MinPrice),
		Volume:             pf(d.UnitsTraded),
		QuoteVolume:        pf(d.AccTradeValue),
	}, nil
}

// FetchAndStoreBithumb fetches daily OHLCV from Bithumb and stores in bithumb_daily_prices.
func FetchAndStoreBithumb(coin string) (int, error) {
	symbol, ok := bithumbCoinMap[coin]
	if !ok {
		return 0, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s/candlestick/%s_KRW/24h", bithumbBaseURL, symbol)
	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("bithumb candles fetch: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var raw struct {
		Status string          `json:"status"`
		Data   [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Status != "0000" {
		return 0, fmt.Errorf("bithumb candles parse error (status=%s): %w", raw.Status, err)
	}

	// Bithumb candlestick format: [timestamp_ms, open, close, low, high, volume]
	pf := func(v interface{}) float64 {
		switch x := v.(type) {
		case float64:
			return x
		case string:
			f, _ := strconv.ParseFloat(x, 64)
			return f
		}
		return 0
	}

	type entry struct {
		date   string
		open   float64
		high   float64
		low    float64
		close  float64
		volume float64
	}
	entries := make([]entry, 0, len(raw.Data))
	for _, row := range raw.Data {
		if len(row) < 6 {
			continue
		}
		tsMs, ok := row[0].(float64)
		if !ok {
			continue
		}
		t := time.UnixMilli(int64(tsMs)).UTC()
		entries = append(entries, entry{
			date:   t.Format("2006-01-02"),
			open:   pf(row[1]),
			close:  pf(row[2]),
			low:    pf(row[3]),
			high:   pf(row[4]),
			volume: pf(row[5]),
		})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].date < entries[j].date })

	prices := make([]models.DailyPrice, 0, len(entries))
	for _, e := range entries {
		prices = append(prices, models.DailyPrice{
			Coin:   coin,
			Date:   e.date,
			Open:   e.open,
			High:   e.high,
			Low:    e.low,
			Close:  e.close,
			Volume: e.volume,
		})
	}

	return upsertBithumbPrices(prices)
}

func upsertBithumbPrices(prices []models.DailyPrice) (int, error) {
	stmt, err := db.DB.Prepare(`
		INSERT INTO bithumb_daily_prices (coin, date, open, high, low, close, volume)
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

// GetBithumbPrices retrieves Bithumb daily prices for a coin with optional date range.
func GetBithumbPrices(coin, from, to string) ([]models.DailyPrice, error) {
	query := `SELECT id, coin, date, open, high, low, close, volume, created_at,
	          ma7, ma20, ma50, ema9, ema21, rsi14, macd, macd_signal,
	          bb_upper, bb_middle, bb_lower, adx14
	          FROM bithumb_daily_prices WHERE coin = ?`
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
	return queryBithumbPrices(query, args...)
}

func queryBithumbPrices(query string, args ...interface{}) ([]models.DailyPrice, error) {
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
