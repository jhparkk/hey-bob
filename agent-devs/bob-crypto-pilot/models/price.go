package models

// DailyPrice represents a single day's OHLCV data for a coin
type DailyPrice struct {
	ID        int64   `json:"id"`
	Coin      string  `json:"coin"`
	Date      string  `json:"date"`
	Open      float64 `json:"open"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Close     float64 `json:"close"`
	Volume    float64 `json:"volume"`
	CreatedAt string  `json:"created_at,omitempty"`
	// 기술 지표
	MA7        float64 `json:"ma7"`
	MA20       float64 `json:"ma20"`
	MA50       float64 `json:"ma50"`
	EMA9       float64 `json:"ema9"`
	EMA21      float64 `json:"ema21"`
	RSI14      float64 `json:"rsi14"`
	MACD       float64 `json:"macd"`
	MACDSignal float64 `json:"macd_signal"`
	BBUpper    float64 `json:"bb_upper"`
	BBMiddle   float64 `json:"bb_middle"`
	BBLower    float64 `json:"bb_lower"`
	ADX14      float64 `json:"adx14"`
}

// APIResponse is the standard response wrapper
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
	Count   int         `json:"count"`
}

// ErrorResponse is used for error responses
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// LivePrice represents real-time ticker data for a coin
type LivePrice struct {
	Coin               string  `json:"coin"`
	LastPrice          float64 `json:"last_price"`
	PriceChange        float64 `json:"price_change"`
	PriceChangePercent float64 `json:"price_change_percent"`
	HighPrice          float64 `json:"high_price"`
	LowPrice           float64 `json:"low_price"`
	Volume             float64 `json:"volume"`
	QuoteVolume        float64 `json:"quote_volume"`
}
