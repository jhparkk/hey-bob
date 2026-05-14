package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"bob-crypto-pilot/db"
)

// StartHourlyTicker starts a background goroutine that refreshes 1h-candle indicators every 10 minutes.
func StartHourlyTicker() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		updateHourlyTicker()
		for range ticker.C {
			updateHourlyTicker()
		}
	}()
}

func updateHourlyTicker() {
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		if err := fetchAndUpsertHourly(coin); err != nil {
			log.Printf("[hourly_ticker] %s: %v", coin, err)
		}
	}
}

type hourlyCandle struct {
	open, high, low, close, volume float64
}

func fetchHourlyCandles(coin string, limit int) ([]hourlyCandle, error) {
	symbol, ok := coinSymbolMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}
	url := fmt.Sprintf("%s?symbol=%s&interval=1h&limit=%d", binanceBaseURL, symbol, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw [][]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	candles := make([]hourlyCandle, 0, len(raw))
	for _, k := range raw {
		if len(k) < 6 {
			continue
		}
		pf := func(v interface{}) float64 {
			s, _ := v.(string)
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}
		candles = append(candles, hourlyCandle{
			open: pf(k[1]), high: pf(k[2]), low: pf(k[3]),
			close: pf(k[4]), volume: pf(k[5]),
		})
	}
	return candles, nil
}

func fetchAndUpsertHourly(coin string) error {
	candles, err := fetchHourlyCandles(coin, 100)
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

	// EMA9, EMA21 (1h)
	ema9 := calcEMA(closes, 9)
	ema21 := calcEMA(closes, 21)

	// RSI14 (1h)
	rsi14 := 50.0
	if n >= 15 {
		rsi14 = calcRSI(closes[n-15:])
	}

	// MACD (12, 26, 9) on 1h
	macdVal, macdSignal := 0.0, 0.0
	if n >= 35 {
		k12 := 2.0 / (12.0 + 1)
		k26 := 2.0 / (26.0 + 1)
		k9 := 2.0 / (9.0 + 1)
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

	// Bollinger Bands (20, 1h)
	bbUpper, bbMiddle, bbLower := 0.0, 0.0, 0.0
	if n >= 20 {
		bbMiddle = mean(closes[n-20:])
		std := stddev(closes[n-20:], bbMiddle)
		bbUpper = bbMiddle + 2*std
		bbLower = bbMiddle - 2*std
	}

	// VWAP (24h) = Σ(typical_price × volume) / Σ(volume) for last 24 candles
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

	// % change from 4h ago
	priceChange4h := 0.0
	if n >= 5 {
		priceChange4h = (closes[n-1] - closes[n-5]) / closes[n-5] * 100
	}

	// % change from 24h ago
	priceChange24h := 0.0
	if n >= 25 {
		priceChange24h = (closes[n-1] - closes[n-25]) / closes[n-25] * 100
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(`
		INSERT OR REPLACE INTO hourly_ticker
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
