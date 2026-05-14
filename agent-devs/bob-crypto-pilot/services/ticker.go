package services

import (
	"log"
	"time"

	"bob-crypto-pilot/db"
)

// StartPriceTicker starts a background goroutine polling prices every 10s
func StartPriceTicker() {
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		// 시작 즉시 1회 실행
		updateTicker()
		for range ticker.C {
			updateTicker()
		}
	}()
}

func updateTicker() {
	coins := []string{"BTC", "ETH", "SOL"}
	for _, coin := range coins {
		lp, err := FetchLivePrice(coin)
		if err != nil {
			log.Printf("[ticker] failed to get %s price: %v", coin, err)
			continue
		}
		if err := upsertTicker(coin, lp.LastPrice, lp.Volume); err != nil {
			log.Printf("[ticker] failed to upsert %s ticker: %v", coin, err)
		}
	}
}

func upsertTicker(coin string, newPrice float64, currentVolume float64) error {
	// 기존 가격 조회
	var prevPrice float64
	row := db.DB.QueryRow("SELECT current_price FROM price_ticker WHERE coin = ?", coin)
	_ = row.Scan(&prevPrice) // 행 없으면 0

	// 변동성 계산 (직전 대비 %)
	var volatility float64
	if prevPrice > 0 {
		volatility = (newPrice - prevPrice) / prevPrice * 100
	}

	// 기술 지표 계산
	ind, err := CalcIndicators(coin)
	if err != nil || ind == nil {
		ind = &DailyIndicators{}
	}

	now := time.Now().Format("2006-01-02 15:04:05")
	_, err = db.DB.Exec(`
		INSERT OR REPLACE INTO price_ticker
		(coin, checked_at, current_price, prev_price, volatility,
		 ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower,
		 ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		coin, now, newPrice, prevPrice, volatility,
		ind.MA7, ind.MA20, ind.MA50, ind.RSI14,
		ind.MACD, ind.MACDSignal, ind.BBUpper, ind.BBMiddle, ind.BBLower,
		ind.EMA9, ind.EMA21, ind.ADX14, ind.ATR14, ind.ATR50,
		ind.VolumeMA20, ind.HighestHigh20,
		currentVolume,
	)
	return err
}
