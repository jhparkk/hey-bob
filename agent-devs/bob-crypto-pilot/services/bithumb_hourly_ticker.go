package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"bob-crypto-pilot/db"
)

func StartBithumbHourlyTicker() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		updateBithumbHourlyTicker()
		for range ticker.C {
			updateBithumbHourlyTicker()
		}
	}()
}

func updateBithumbHourlyTicker() {
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		if err := fetchAndUpsertBithumbHourly(coin); err != nil {
			log.Printf("[bithumb_hourly] %s: %v", coin, err)
		}
	}
}

func fetchBithumbHourlyCandles(coin string) ([]hourlyCandle, error) {
	symbol, ok := bithumbCoinMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}
	url := fmt.Sprintf("%s/candlestick/%s_KRW/1h", bithumbBaseURL, symbol)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Status string          `json:"status"`
		Data   [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil || raw.Status != "0000" {
		return nil, fmt.Errorf("bithumb hourly parse error (status=%s): %w", raw.Status, err)
	}

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
		ts     int64
		candle hourlyCandle
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
		// Bithumb format: [timestamp_ms, open, close, low, high, volume]
		entries = append(entries, entry{
			ts: int64(tsMs),
			candle: hourlyCandle{
				open:   pf(row[1]),
				close:  pf(row[2]),
				low:    pf(row[3]),
				high:   pf(row[4]),
				volume: pf(row[5]),
			},
		})
	}

	// Sort oldest-first
	sort.Slice(entries, func(i, j int) bool { return entries[i].ts < entries[j].ts })

	// Take last 100
	if len(entries) > 100 {
		entries = entries[len(entries)-100:]
	}

	candles := make([]hourlyCandle, len(entries))
	for i, e := range entries {
		candles[i] = e.candle
	}
	return candles, nil
}

func fetchAndUpsertBithumbHourly(coin string) error {
	candles, err := fetchBithumbHourlyCandles(coin)
	if err != nil {
		return err
	}
	if len(candles) < 26 {
		return fmt.Errorf("not enough candles: %d", len(candles))
	}

	n := len(candles)
	closes := make([]float64, n)
	highs := make([]float64, n)
	lows := make([]float64, n)
	volumes := make([]float64, n)
	for i, c := range candles {
		closes[i] = c.close
		highs[i] = c.high
		lows[i] = c.low
		volumes[i] = c.volume
	}

	ema9 := calcEMA(closes, 9)
	ema21 := calcEMA(closes, 21)

	rsi14 := 50.0
	if n >= 15 {
		rsi14 = calcRSI(closes[n-15:])
	}

	macdVal, macdSignal := 0.0, 0.0
	if n >= 35 {
		k12 := 2.0 / 13.0
		k26 := 2.0 / 27.0
		k9 := 2.0 / 10.0
		e12 := mean(closes[:12])
		e26 := mean(closes[:26])
		for _, v := range closes[12:26] {
			e12 = v*k12 + e12*(1-k12)
		}
		macdLine := make([]float64, 0, n-26)
		for _, v := range closes[26:] {
			e12 = v*k12 + e12*(1-k12)
			e26 = v*k26 + e26*(1-k26)
			macdLine = append(macdLine, e12-e26)
		}
		macdVal = macdLine[len(macdLine)-1]
		if len(macdLine) >= 9 {
			sig := mean(macdLine[:9])
			for _, v := range macdLine[9:] {
				sig = v*k9 + sig*(1-k9)
			}
			macdSignal = sig
		}
	}
	macdHist := macdVal - macdSignal

	bbUpper, bbMiddle, bbLower := 0.0, 0.0, 0.0
	if n >= 20 {
		bbMiddle = mean(closes[n-20:])
		std := stddev(closes[n-20:], bbMiddle)
		bbUpper = bbMiddle + 2*std
		bbLower = bbMiddle - 2*std
	}

	vwap := 0.0
	vLen := 24
	if n < vLen {
		vLen = n
	}
	sumTV, sumV := 0.0, 0.0
	for i := n - vLen; i < n; i++ {
		tp := (highs[i] + lows[i] + closes[i]) / 3
		sumTV += tp * volumes[i]
		sumV += volumes[i]
	}
	if sumV > 0 {
		vwap = sumTV / sumV
	}

	priceChange4h := 0.0
	if n >= 5 {
		priceChange4h = (closes[n-1] - closes[n-5]) / closes[n-5] * 100
	}
	priceChange24h := 0.0
	if n >= 25 {
		priceChange24h = (closes[n-1] - closes[n-25]) / closes[n-25] * 100
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(`
		INSERT OR REPLACE INTO bithumb_hourly_ticker
		(coin, checked_at, ema9_1h, ema21_1h, rsi14_1h,
		 macd_1h, macd_signal_1h, macd_hist_1h,
		 bb_upper_1h, bb_middle_1h, bb_lower_1h,
		 vwap_24h, price_change_4h, price_change_24h)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		coin, now,
		ema9, ema21, rsi14,
		macdVal, macdSignal, macdHist,
		bbUpper, bbMiddle, bbLower,
		vwap, priceChange4h, priceChange24h,
	)
	return err
}
