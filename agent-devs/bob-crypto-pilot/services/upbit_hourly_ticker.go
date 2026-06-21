package services

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"bob-crypto-pilot/db"
)

func StartUpbitHourlyTicker() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		updateUpbitHourlyTicker()
		for range ticker.C {
			updateUpbitHourlyTicker()
		}
	}()
}

func updateUpbitHourlyTicker() {
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		if err := fetchAndUpsertUpbitHourly(coin); err != nil {
			log.Printf("[upbit_hourly] %s: %v", coin, err)
		}
	}
}

func fetchUpbitHourlyCandles(coin string, limit int) ([]hourlyCandle, error) {
	market, ok := upbitMarketMap[coin]
	if !ok {
		return nil, fmt.Errorf("unsupported coin: %s", coin)
	}
	url := fmt.Sprintf("%s/candles/minutes/60?market=%s&count=%d", upbitBaseURL, market, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var raw []struct {
		OpeningPrice    float64 `json:"opening_price"`
		HighPrice       float64 `json:"high_price"`
		LowPrice        float64 `json:"low_price"`
		TradePrice      float64 `json:"trade_price"`
		CandleAccVolume float64 `json:"candle_acc_trade_volume"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	// Upbit returns newest-first, reverse to oldest-first
	candles := make([]hourlyCandle, len(raw))
	for i, r := range raw {
		candles[len(raw)-1-i] = hourlyCandle{
			open: r.OpeningPrice, high: r.HighPrice, low: r.LowPrice,
			close: r.TradePrice, volume: r.CandleAccVolume,
		}
	}
	return candles, nil
}

func fetchAndUpsertUpbitHourly(coin string) error {
	candles, err := fetchUpbitHourlyCandles(coin, 100)
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
		INSERT OR REPLACE INTO upbit_hourly_ticker
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
