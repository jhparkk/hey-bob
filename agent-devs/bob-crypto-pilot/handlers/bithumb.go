package handlers

import (
	"net/http"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/services"

	"github.com/gin-gonic/gin"
)

// GetBithumbTicker handles GET /api/v1/bithumb/ticker
func GetBithumbTicker(c *gin.Context) {
	coin := c.Query("coin")
	cols := `coin, checked_at, current_price, prev_price, volatility,
	         ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower,
	         ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume`

	var query string
	var args []interface{}
	if coin != "" {
		query = `SELECT ` + cols + ` FROM bithumb_price_ticker WHERE coin = ?`
		args = []interface{}{coin}
	} else {
		query = `SELECT ` + cols + ` FROM bithumb_price_ticker ORDER BY coin ASC`
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tickers := make([]TickerRow, 0)
	for rows.Next() {
		var t TickerRow
		if err := rows.Scan(
			&t.Coin, &t.CheckedAt, &t.CurrentPrice, &t.PrevPrice, &t.Volatility,
			&t.MA7, &t.MA20, &t.MA50, &t.RSI14,
			&t.MACD, &t.MACDSignal, &t.BBUpper, &t.BBMiddle, &t.BBLower,
			&t.EMA9, &t.EMA21, &t.ADX14, &t.ATR14, &t.ATR50,
			&t.VolumeMA20, &t.HighestHigh20, &t.CurrentVolume,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tickers = append(tickers, t)
	}
	c.JSON(http.StatusOK, tickers)
}

// GetBithumbHourlyTicker handles GET /api/v1/bithumb/ticker/hourly
func GetBithumbHourlyTicker(c *gin.Context) {
	coin := c.Query("coin")
	cols := `coin, checked_at, ema9_1h, ema21_1h, rsi14_1h,
	         macd_1h, macd_signal_1h, macd_hist_1h,
	         bb_upper_1h, bb_middle_1h, bb_lower_1h,
	         vwap_24h, price_change_4h, price_change_24h`

	var query string
	var args []interface{}
	if coin != "" {
		query = `SELECT ` + cols + ` FROM bithumb_hourly_ticker WHERE coin = ?`
		args = []interface{}{coin}
	} else {
		query = `SELECT ` + cols + ` FROM bithumb_hourly_ticker ORDER BY coin ASC`
	}

	rows, err := db.DB.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	tickers := make([]HourlyTickerRow, 0)
	for rows.Next() {
		var t HourlyTickerRow
		if err := rows.Scan(
			&t.Coin, &t.CheckedAt,
			&t.EMA9_1h, &t.EMA21_1h, &t.RSI14_1h,
			&t.MACD_1h, &t.MACDSignal_1h, &t.MACDHist_1h,
			&t.BBUpper_1h, &t.BBMiddle_1h, &t.BBLower_1h,
			&t.VWAP_24h, &t.PriceChange4h, &t.PriceChange24h,
		); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		tickers = append(tickers, t)
	}
	c.JSON(http.StatusOK, tickers)
}

// GetBithumbSimPortfolios handles GET /api/v1/bithumb/simulation/portfolios
func GetBithumbSimPortfolios(c *gin.Context) {
	summaries, err := services.GetAllPortfolios("bithumb")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "portfolios": summaries})
}

// GetBithumbSimPerformance handles GET /api/v1/bithumb/simulation/performance
func GetBithumbSimPerformance(c *gin.Context) {
	result, err := services.GetPerformance("bithumb")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "portfolios": result})
}

// SyncBithumb handles POST /api/v1/bithumb/sync
func SyncBithumb(c *gin.Context) {
	coins := []string{"BTC", "ETH", "SOL"}
	results := make(map[string]int)
	errors := make(map[string]string)
	for _, coin := range coins {
		count, err := services.FetchAndStoreBithumb(coin)
		if err != nil {
			errors[coin] = err.Error()
		} else {
			results[coin] = count
		}
	}
	if len(errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "synced": results, "errors": errors})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "synced": results, "message": "bithumb sync completed"})
}
