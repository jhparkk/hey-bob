package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/handlers"
	"bob-crypto-pilot/services"

	"github.com/gin-gonic/gin"
)

//go:embed static
var staticFiles embed.FS

func main() {
	// Determine data directory relative to executable
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	projectRoot := filepath.Dir(ex)
	dataDir := filepath.Join(projectRoot, "data")

	// Initialize database
	if err := db.Init(dataDir); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Seed Box Hunter strategies (idempotent)
	db.SeedBoxHunterStrategies()

	// Start background price ticker (10s interval)
	services.StartPriceTicker()

	// Start hourly indicator ticker (10min interval, 1h candles)
	services.StartHourlyTicker()

	// Start daily sync scheduler (01:00 KST)
	services.StartDailySyncScheduler()

	// Start Upbit price tickers
	services.StartUpbitPriceTicker()
	services.StartUpbitHourlyTicker()

	// Start Bithumb price tickers
	services.StartBithumbPriceTicker()
	services.StartBithumbHourlyTicker()

	// Setup Gin router
	r := gin.Default()

	// Serve static files (embedded)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("Failed to create static sub-fs: %v", err)
	}
	r.StaticFS("/static", http.FS(staticFS))

	assetsFS, err := fs.Sub(staticFiles, "static/assets")
	if err != nil {
		log.Fatalf("Failed to create assets sub-fs: %v", err)
	}
	r.StaticFS("/assets", http.FS(assetsFS))

	// Serve index.html at root
	r.GET("/", func(c *gin.Context) {
		data, err := staticFiles.ReadFile("static/index.html")
		if err != nil {
			c.String(http.StatusInternalServerError, "index.html not found")
			return
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// Health check
	r.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		v1.POST("/sync", handlers.Sync)
		v1.GET("/prices", handlers.GetPrices)
		v1.GET("/prices/latest", handlers.GetLatestPrice)
		v1.GET("/price/live", handlers.GetLivePrice)
		v1.GET("/ticker", handlers.GetTicker)
		v1.GET("/ticker/hourly", handlers.GetHourlyTicker)

		// Strategy routes (coin-agnostic library)
		v1.GET("/strategy", handlers.GetStrategies)
		v1.GET("/strategy/history", handlers.GetStrategyHistory)
		v1.POST("/strategy", handlers.CreateStrategy)
		v1.PUT("/strategy/:id", handlers.UpdateStrategy)
		v1.DELETE("/strategy/:id", handlers.DeleteStrategy)
		v1.GET("/strategy/:id/versions", handlers.GetStrategyVersions)

		// Legacy strategy routes (kept for Bob AI compatibility)
		v1.PATCH("/strategy/:coin/active", handlers.PatchCoinActiveStrategy)

		// Portfolio routes
		v1.GET("/portfolios", handlers.GetPortfolios)
		v1.POST("/portfolios", handlers.CreatePortfolio)
		v1.PUT("/portfolios/:id", handlers.UpdatePortfolio)
		v1.DELETE("/portfolios/:id", handlers.DeletePortfolio)
		v1.POST("/portfolios/:id/reset", handlers.ResetPortfolio)
		v1.GET("/portfolios/:id/strategies", handlers.GetPortfolioStrategies)
		v1.PATCH("/portfolios/:id/strategies/:coin", handlers.PatchPortfolioStrategy)
		v1.GET("/portfolios/:id/strategy-history", handlers.GetPortfolioStrategyHistory)
		v1.POST("/portfolios/:id/coins", handlers.AddCoinToPortfolio)
		v1.DELETE("/portfolios/:id/coins/:coin", handlers.RemoveCoinFromPortfolio)

		// Simulation routes (Binance)
		sim := v1.Group("/simulation")
		{
			sim.GET("/status", handlers.GetSimStatus)
			sim.POST("/trade", handlers.ExecuteTrade)
			sim.GET("/portfolios", handlers.GetSimPortfolios)
			sim.GET("/trades", handlers.GetSimTrades)
			sim.GET("/performance", handlers.GetSimPerformance)
		}

		// Upbit routes
		upbit := v1.Group("/upbit")
		{
			upbit.GET("/ticker", handlers.GetUpbitTicker)
			upbit.GET("/ticker/hourly", handlers.GetUpbitHourlyTicker)
			upbit.GET("/simulation/portfolios", handlers.GetUpbitSimPortfolios)
			upbit.GET("/simulation/performance", handlers.GetUpbitSimPerformance)
			upbit.POST("/sync", handlers.SyncUpbit)
		}

		// Bithumb routes
		bithumb := v1.Group("/bithumb")
		{
			bithumb.GET("/ticker", handlers.GetBithumbTicker)
			bithumb.GET("/ticker/hourly", handlers.GetBithumbHourlyTicker)
			bithumb.GET("/simulation/portfolios", handlers.GetBithumbSimPortfolios)
			bithumb.GET("/simulation/performance", handlers.GetBithumbSimPerformance)
			bithumb.POST("/sync", handlers.SyncBithumb)
		}
	}

	log.Println("Starting crypto-tracker server on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
