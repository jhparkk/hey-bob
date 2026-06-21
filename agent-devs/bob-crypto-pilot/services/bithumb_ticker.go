package services

import (
	"log"
	"time"

	"bob-crypto-pilot/db"
)

func StartBithumbPriceTicker() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		updateBithumbTicker()
		for range ticker.C {
			updateBithumbTicker()
		}
	}()
}

func updateBithumbTicker() {
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		lp, err := FetchBithumbLivePrice(coin)
		if err != nil {
			log.Printf("[bithumb_ticker] %s price fetch: %v", coin, err)
			continue
		}
		if err := upsertBithumbTicker(coin, lp.LastPrice, lp.Volume); err != nil {
			log.Printf("[bithumb_ticker] %s upsert: %v", coin, err)
		}
	}
}

func upsertBithumbTicker(coin string, newPrice float64, currentVolume float64) error {
	var prevPrice float64
	_ = db.DB.QueryRow("SELECT current_price FROM bithumb_price_ticker WHERE coin = ?", coin).Scan(&prevPrice)

	var volatility float64
	if prevPrice > 0 {
		volatility = (newPrice - prevPrice) / prevPrice * 100
	}

	ind, err := CalcIndicatorsBithumb(coin)
	if err != nil || ind == nil {
		ind = &Indicators{}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(`
		INSERT OR REPLACE INTO bithumb_price_ticker
		(coin, checked_at, current_price, prev_price, volatility,
		 ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower,
		 ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		coin, now, newPrice, prevPrice, volatility,
		ind.MA7, ind.MA20, ind.MA50, ind.RSI14,
		ind.MACD, ind.MACDSignal, ind.BBUpper, ind.BBMiddle, ind.BBLower,
		ind.EMA9, ind.EMA21, ind.ADX14, ind.ATR14, ind.ATR50,
		ind.VolumeMA20, ind.HighestHigh20, currentVolume,
	)
	return err
}
