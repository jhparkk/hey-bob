package handlers

import (
	"database/sql"
	"net/http"
	"time"

	"bob-crypto-pilot/models"
	"bob-crypto-pilot/services"

	"github.com/gin-gonic/gin"
)

// HealthCheck handles GET /health
func HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"status":  "ok",
	})
}

// Sync handles POST /api/v1/sync
// Fetches latest 90-day data for BTC and ETH from Binance and stores in SQLite
func Sync(c *gin.Context) {
	coins := []string{"BTC", "ETH", "SOL"}
	results := make(map[string]int)
	errors := make(map[string]string)

	for _, coin := range coins {
		count, err := services.FetchAndStore(coin)
		if err != nil {
			errors[coin] = err.Error()
		} else {
			results[coin] = count
		}
	}

	if len(errors) > 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"synced":  results,
			"errors":  errors,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"synced":  results,
		"message": "sync completed",
	})
}

// GetPrices handles GET /api/v1/prices?coin=BTC[&from=YYYY-MM-DD&to=YYYY-MM-DD&period=2w]
func GetPrices(c *gin.Context) {
	coin := c.Query("coin")
	if coin == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "coin parameter is required (e.g. ?coin=BTC)",
		})
		return
	}

	from := c.Query("from")
	to := c.Query("to")

	// period 파라미터 처리: from/to가 없으면 period로 계산 (기본 2주)
	if from == "" && to == "" {
		period := c.Query("period")
		days := 14
		switch period {
		case "1w":
			days = 7
		case "3w":
			days = 21
		case "1m":
			days = 30
		case "1y":
			days = 365
		}
		now := time.Now()
		to = now.Format("2006-01-02")
		from = now.AddDate(0, 0, -days).Format("2006-01-02")
	}

	prices, err := services.GetPrices(coin, from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	if prices == nil {
		prices = []models.DailyPrice{}
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    prices,
		Count:   len(prices),
	})
}

// GetLivePrice handles GET /api/v1/price/live?coin=BTC
func GetLivePrice(c *gin.Context) {
	coin := c.Query("coin")
	if coin == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "coin parameter is required (e.g. ?coin=BTC)",
		})
		return
	}

	live, err := services.FetchLivePrice(coin)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    live,
		Count:   1,
	})
}

// GetLatestPrice handles GET /api/v1/prices/latest?coin=BTC
func GetLatestPrice(c *gin.Context) {
	coin := c.Query("coin")
	if coin == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Success: false,
			Error:   "coin parameter is required (e.g. ?coin=BTC)",
		})
		return
	}

	price, err := services.GetLatestPrice(coin)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{
			Success: false,
			Error:   "no data found for coin: " + coin,
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    price,
		Count:   1,
	})
}
