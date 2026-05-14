package handlers

import (
	"net/http"

	"bob-crypto-pilot/db"

	"github.com/gin-gonic/gin"
)

// TickerRow represents a row in the price_ticker table
type TickerRow struct {
	Coin          string  `json:"coin"`
	CheckedAt     string  `json:"checked_at"`
	CurrentPrice  float64 `json:"current_price"`
	PrevPrice     float64 `json:"prev_price"`
	Volatility    float64 `json:"volatility"`
	MA7           float64 `json:"ma7"`
	MA20          float64 `json:"ma20"`
	MA50          float64 `json:"ma50"`
	RSI14         float64 `json:"rsi14"`
	MACD          float64 `json:"macd"`
	MACDSignal    float64 `json:"macd_signal"`
	BBUpper       float64 `json:"bb_upper"`
	BBMiddle      float64 `json:"bb_middle"`
	BBLower       float64 `json:"bb_lower"`
	EMA9          float64 `json:"ema9"`
	EMA21         float64 `json:"ema21"`
	ADX14         float64 `json:"adx14"`
	ATR14         float64 `json:"atr14"`
	ATR50         float64 `json:"atr50"`
	VolumeMA20    float64 `json:"volume_ma20"`
	HighestHigh20 float64 `json:"highest_high20"`
	CurrentVolume float64 `json:"current_volume"`
}

// GetTicker handles GET /api/v1/ticker
// Optional query param: ?coin=BTC or ?coin=ETH
func GetTicker(c *gin.Context) {
	coin := c.Query("coin")

	var (
		query string
		args  []interface{}
	)

	if coin != "" {
		query = `SELECT coin, checked_at, current_price, prev_price, volatility,
		          ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower,
		          ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume
		          FROM price_ticker WHERE coin = ? ORDER BY coin ASC`
		args = []interface{}{coin}
	} else {
		query = `SELECT coin, checked_at, current_price, prev_price, volatility,
		          ma7, ma20, ma50, rsi14, macd, macd_signal, bb_upper, bb_middle, bb_lower,
		          ema9, ema21, adx14, atr14, atr50, volume_ma20, highest_high20, current_volume
		          FROM price_ticker ORDER BY coin ASC`
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
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tickers)
}

// HourlyTickerRow represents a row in the hourly_ticker table
type HourlyTickerRow struct {
	Coin            string  `json:"coin"`
	CheckedAt       string  `json:"checked_at"`
	EMA9_1h         float64 `json:"ema9_1h"`
	EMA21_1h        float64 `json:"ema21_1h"`
	RSI14_1h        float64 `json:"rsi14_1h"`
	MACD_1h         float64 `json:"macd_1h"`
	MACDSignal_1h   float64 `json:"macd_signal_1h"`
	MACDHist_1h     float64 `json:"macd_hist_1h"`
	BBUpper_1h      float64 `json:"bb_upper_1h"`
	BBMiddle_1h     float64 `json:"bb_middle_1h"`
	BBLower_1h      float64 `json:"bb_lower_1h"`
	VWAP_24h        float64 `json:"vwap_24h"`
	PriceChange4h   float64 `json:"price_change_4h"`
	PriceChange24h  float64 `json:"price_change_24h"`
}

// GetHourlyTicker handles GET /api/v1/ticker/hourly
func GetHourlyTicker(c *gin.Context) {
	coin := c.Query("coin")

	var query string
	var args []interface{}

	cols := `coin, checked_at, ema9_1h, ema21_1h, rsi14_1h,
	         macd_1h, macd_signal_1h, macd_hist_1h,
	         bb_upper_1h, bb_middle_1h, bb_lower_1h,
	         vwap_24h, price_change_4h, price_change_24h`

	if coin != "" {
		query = `SELECT ` + cols + ` FROM hourly_ticker WHERE coin = ?`
		args = []interface{}{coin}
	} else {
		query = `SELECT ` + cols + ` FROM hourly_ticker ORDER BY coin ASC`
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
