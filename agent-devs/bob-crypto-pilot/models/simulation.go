package models

// ── Portfolio Models ───────────────────────────────────────────────────────

type Portfolio struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	NotifyOnTrade int     `json:"notify_on_trade"`
	RiskLimitPct  float64 `json:"risk_limit_pct"`
	CreatedAt     string  `json:"created_at"`
}

type PortfolioStrategy struct {
	ID              int64  `json:"id"`
	PortfolioID     int64  `json:"portfolio_id"`
	Coin            string `json:"coin"`
	StrategyID      int64  `json:"strategy_id"`
	SelectedBy      string `json:"selected_by"`
	SelectedAt      string `json:"selected_at"`
	SelectionReason string `json:"selection_reason"`
}

// PortfolioSummary aggregates a portfolio with its sim states
type PortfolioSummary struct {
	Portfolio  Portfolio  `json:"portfolio"`
	States     []SimState `json:"states"`
	TotalValue float64    `json:"total_value"`
	TotalReturn float64   `json:"total_return_pct"`
}

// ── Simulation Models ──────────────────────────────────────────────────────

type SimState struct {
	ID             int64   `json:"id"`
	PortfolioID    int64   `json:"portfolio_id"`
	Coin           string  `json:"coin"`
	Cash           float64 `json:"cash"`
	Units          float64 `json:"units"`
	InitialCapital float64 `json:"initial_capital"`
	Position       string  `json:"position"` // CASH or HOLDING
	AvgCost        float64 `json:"avg_cost"`  // average purchase cost per unit
	CurrentPrice   float64 `json:"current_price"`
	CurrentValue   float64 `json:"current_value"` // cash + units*price
	ReturnPct      float64 `json:"return_pct"`
	UpdatedAt      string  `json:"updated_at"`
}

type SimTrade struct {
	ID          int     `json:"id"`
	Coin        string  `json:"coin"`
	PortfolioID int64   `json:"portfolio_id"`
	Action      string  `json:"action"`
	Price       float64 `json:"price"`
	Units       float64 `json:"units"`
	CashBefore  float64 `json:"cash_before"`
	CashAfter   float64 `json:"cash_after"`
	UnitsBefore float64 `json:"units_before"`
	UnitsAfter  float64 `json:"units_after"`
	Reason      string  `json:"reason"`
	ExecutedAt  string  `json:"executed_at"`
}

type SimStatusResponse struct {
	Success        bool       `json:"success"`
	Coin           string     `json:"coin"`
	PortfolioID    int64      `json:"portfolio_id"`
	InitialCapital float64    `json:"initial_capital"`
	CurrentValue   float64    `json:"current_value"`
	ReturnPct      float64    `json:"return_pct"`
	Position       string     `json:"position"`
	Units          float64    `json:"units"`
	Cash           float64    `json:"cash"`
	AvgCost        float64    `json:"avg_cost"`
	CurrentPrice   float64    `json:"current_price"`
	LastTrade      *SimTrade  `json:"last_trade"`
	Trades         []SimTrade `json:"trades"`
}

type TradeRequest struct {
	Coin        string  `json:"coin"`
	Action      string  `json:"action"`   // BUY / SELL / HOLD
	Price       float64 `json:"price"`
	Amount      float64 `json:"amount"`   // 매수 금액(USD). 0 또는 미입력 시 전액 올인
	Reason      string  `json:"reason"`
	PortfolioID int64   `json:"portfolio_id"` // 기본값 1
}

type TradeResponse struct {
	Success bool     `json:"success"`
	Action  string   `json:"action"`
	Trade   SimTrade `json:"trade"`
	State   SimState `json:"state"`
}

// ── Portfolio Response Models ──────────────────────────────────────────────

// SimPortfoliosResponse is the response for GET /simulation/portfolios
type SimPortfoliosResponse struct {
	Success    bool               `json:"success"`
	Portfolios []PortfolioSummary `json:"portfolios"`
}

// ── Performance Models ─────────────────────────────────────────────────────

type CoinPerformance struct {
	Coin            string   `json:"coin"`
	Return1D        *float64 `json:"return_1d"`
	Return7D        *float64 `json:"return_7d"`
	Return30D       *float64 `json:"return_30d"`
	ReturnLife      *float64 `json:"return_life"`
	PriceChange1D   *float64 `json:"price_change_1d"`
	PriceChange7D   *float64 `json:"price_change_7d"`
	PriceChange30D  *float64 `json:"price_change_30d"`
	PriceChangeLife *float64 `json:"price_change_life"`
}

type PortfolioPerformance struct {
	PortfolioID   int64             `json:"portfolio_id"`
	PortfolioName string            `json:"portfolio_name"`
	MaxPeriod     int               `json:"max_period"` // actual days since portfolio creation, capped at 30
	Coins         []CoinPerformance `json:"coins"`
}

type PerformanceResponse struct {
	Success    bool                   `json:"success"`
	Portfolios []PortfolioPerformance `json:"portfolios"`
}

// ── Legacy compatibility (kept for internal use) ───────────────────────────

// SimAccountInfo is kept for any legacy internal code
type SimAccountInfo struct {
	Account     string     `json:"account"`
	DisplayName string     `json:"display_name"`
	Coins       []SimState `json:"coins"`
	TotalValue  float64    `json:"total_value"`
	TotalReturn float64    `json:"total_return_pct"`
}
