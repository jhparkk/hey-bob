package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/models"
)

const binanceBaseURL = "https://api.binance.com/api/v3/klines"
const binanceTickerURL = "https://api.binance.com/api/v3/ticker/24hr"

// coinSymbolMap maps coin names to Binance symbols
var coinSymbolMap = map[string]string{
	"BTC": "BTCUSDT",
	"ETH": "ETHUSDT",
	"SOL": "SOLUSDT",
}

// FetchLivePrice fetches real-time 24hr ticker data from Binance for a given coin
func FetchLivePrice(coin string) (*models.LivePrice, error) {
	symbol, ok := coinSymbolMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s?symbol=%s", binanceTickerURL, symbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch ticker from Binance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Binance API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read ticker response: %w", err)
	}

	var raw struct {
		LastPrice          string `json:"lastPrice"`
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse ticker: %w", err)
	}

	parseF := func(s string) float64 {
		v, _ := strconv.ParseFloat(s, 64)
		return v
	}

	return &models.LivePrice{
		Coin:               coin,
		LastPrice:          parseF(raw.LastPrice),
		PriceChange:        parseF(raw.PriceChange),
		PriceChangePercent: parseF(raw.PriceChangePercent),
		HighPrice:          parseF(raw.HighPrice),
		LowPrice:           parseF(raw.LowPrice),
		Volume:             parseF(raw.Volume),
		QuoteVolume:        parseF(raw.QuoteVolume),
	}, nil
}

// FetchAndStore fetches 90 days of kline data from Binance and stores in SQLite
func FetchAndStore(coin string) (int, error) {
	symbol, ok := coinSymbolMap[coin]
	if !ok {
		return 0, fmt.Errorf("unsupported coin: %s", coin)
	}

	url := fmt.Sprintf("%s?symbol=%s&interval=1d&limit=90", binanceBaseURL, symbol)

	resp, err := http.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch from Binance: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("Binance API error %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse the raw kline array: [[openTime, open, high, low, close, volume, ...], ...]
	var rawKlines [][]interface{}
	if err := json.Unmarshal(body, &rawKlines); err != nil {
		return 0, fmt.Errorf("failed to parse klines: %w", err)
	}

	prices, err := parseKlines(coin, rawKlines)
	if err != nil {
		return 0, fmt.Errorf("failed to parse kline data: %w", err)
	}

	count, err := upsertPrices(prices)
	if err != nil {
		return 0, fmt.Errorf("failed to store prices: %w", err)
	}

	return count, nil
}

func parseKlines(coin string, rawKlines [][]interface{}) ([]models.DailyPrice, error) {
	prices := make([]models.DailyPrice, 0, len(rawKlines))

	for _, kline := range rawKlines {
		if len(kline) < 6 {
			continue
		}

		// [0] openTime in milliseconds
		openTimeMs, ok := kline[0].(float64)
		if !ok {
			continue
		}
		t := time.Unix(int64(openTimeMs)/1000, 0).UTC()
		date := t.Format("2006-01-02")

		open, err := strconv.ParseFloat(kline[1].(string), 64)
		if err != nil {
			continue
		}
		high, err := strconv.ParseFloat(kline[2].(string), 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(kline[3].(string), 64)
		if err != nil {
			continue
		}
		close, err := strconv.ParseFloat(kline[4].(string), 64)
		if err != nil {
			continue
		}
		volume, err := strconv.ParseFloat(kline[5].(string), 64)
		if err != nil {
			continue
		}

		prices = append(prices, models.DailyPrice{
			Coin:   coin,
			Date:   date,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		})
	}

	return prices, nil
}

func upsertPrices(prices []models.DailyPrice) (int, error) {
	stmt, err := db.DB.Prepare(`
		INSERT INTO daily_prices (coin, date, open, high, low, close, volume)
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

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return count, nil
}

// GetPrices retrieves prices for a coin with optional date range filter
func GetPrices(coin, from, to string) ([]models.DailyPrice, error) {
	query := `SELECT id, coin, date, open, high, low, close, volume, created_at,
	          ma7, ma20, ma50, ema9, ema21, rsi14, macd, macd_signal,
	          bb_upper, bb_middle, bb_lower, adx14
	          FROM daily_prices WHERE coin = ?`
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

	return queryPrices(query, args...)
}

// GetLatestPrice retrieves the most recent price for a coin
func GetLatestPrice(coin string) (*models.DailyPrice, error) {
	query := `SELECT id, coin, date, open, high, low, close, volume, created_at,
	          ma7, ma20, ma50, ema9, ema21, rsi14, macd, macd_signal,
	          bb_upper, bb_middle, bb_lower, adx14
	          FROM daily_prices WHERE coin = ? ORDER BY date DESC LIMIT 1`

	rows, err := queryPrices(query, coin)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, sql.ErrNoRows
	}
	return &rows[0], nil
}

func queryPrices(query string, args ...interface{}) ([]models.DailyPrice, error) {
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
