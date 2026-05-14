package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"bob-crypto-pilot/models"
	"bob-crypto-pilot/services"

	"github.com/gin-gonic/gin"
)

// GetSimStatus godoc
// GET /api/v1/simulation/status?coin=BTC&portfolio_id=1
func GetSimStatus(c *gin.Context) {
	coin := strings.ToUpper(c.Query("coin"))
	if coin == "" {
		coin = "BTC"
	}

	portfolioID := parsePortfolioID(c.DefaultQuery("portfolio_id", "1"))

	// Ensure state exists
	state, err := services.GetOrInitState(coin, portfolioID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	// Fetch live price from Binance
	live, err := services.FetchLivePrice(coin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to fetch live price: " + err.Error()})
		return
	}
	currentPrice := live.LastPrice

	// Compute current value and return pct
	currentValue := state.Cash + state.Units*currentPrice
	returnPct := 0.0
	if state.InitialCapital > 0 {
		returnPct = (currentValue/state.InitialCapital - 1) * 100
	}

	// Trade history (latest 50)
	trades, err := services.GetTradeHistory(coin, portfolioID, 50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to load trade history: " + err.Error()})
		return
	}

	var lastTrade *models.SimTrade
	if len(trades) > 0 {
		lastTrade = &trades[0]
	}

	c.JSON(http.StatusOK, models.SimStatusResponse{
		Success:        true,
		Coin:           coin,
		PortfolioID:    portfolioID,
		InitialCapital: state.InitialCapital,
		CurrentValue:   currentValue,
		ReturnPct:      returnPct,
		Position:       state.Position,
		Units:          state.Units,
		Cash:           state.Cash,
		AvgCost:        state.AvgCost,
		CurrentPrice:   currentPrice,
		LastTrade:      lastTrade,
		Trades:         trades,
	})
}

// ExecuteTrade godoc
// POST /api/v1/simulation/trade
// Body: {"coin":"BTC","action":"BUY","price":85000,"reason":"...","portfolio_id":1}
func ExecuteTrade(c *gin.Context) {
	var req models.TradeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	req.Coin = strings.ToUpper(req.Coin)
	req.Action = strings.ToUpper(req.Action)
	if req.PortfolioID <= 0 {
		req.PortfolioID = 1
	}

	if req.Coin == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "coin is required"})
		return
	}
	if req.Action != "BUY" && req.Action != "SELL" && req.Action != "HOLD" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "action must be BUY, SELL, or HOLD"})
		return
	}

	resp, err := services.ExecuteTrade(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetSimTrades godoc
// GET /api/v1/simulation/trades?coin=BTC&portfolio_id=1&limit=50
func GetSimTrades(c *gin.Context) {
	coin := strings.ToUpper(c.Query("coin"))
	if coin == "" {
		coin = "BTC"
	}
	portfolioID := parsePortfolioID(c.DefaultQuery("portfolio_id", "1"))
	limit := 50
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}
	trades, err := services.GetTradeHistory(coin, portfolioID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if trades == nil {
		trades = []models.SimTrade{}
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"coin":         coin,
		"portfolio_id": portfolioID,
		"trades":       trades,
	})
}

// GetSimPortfolios godoc
// GET /api/v1/simulation/portfolios
func GetSimPortfolios(c *gin.Context) {
	portfolios, err := services.GetAllPortfolios()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if portfolios == nil {
		portfolios = []models.PortfolioSummary{}
	}
	c.JSON(http.StatusOK, models.SimPortfoliosResponse{
		Success:    true,
		Portfolios: portfolios,
	})
}

// GetSimPerformance godoc
// GET /api/v1/simulation/performance
func GetSimPerformance(c *gin.Context) {
	portfolios, err := services.GetPerformance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if portfolios == nil {
		portfolios = []models.PortfolioPerformance{}
	}
	c.JSON(http.StatusOK, models.PerformanceResponse{Success: true, Portfolios: portfolios})
}

// parsePortfolioID parses portfolio_id from string, defaulting to 1
func parsePortfolioID(s string) int64 {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil || id <= 0 {
		return 1
	}
	return id
}
