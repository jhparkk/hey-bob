package services

import (
	"database/sql"
	"fmt"
	"time"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/models"
)

// InitSimState initializes a new simulation state for a coin+portfolio with $100 capital
func InitSimState(coin string, portfolioID int64) error {
	if portfolioID <= 0 {
		portfolioID = 1
	}
	// Also derive account name for the account column
	account := portfolioIDToAccount(portfolioID)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.DB.Exec(`
		INSERT INTO sim_state (coin, account, portfolio_id, cash, units, initial_capital, position, avg_cost, updated_at)
		SELECT ?, ?, ?, 100.0, 0.0, 100.0, 'CASH', 0.0, ?
		WHERE NOT EXISTS (SELECT 1 FROM sim_state WHERE coin=? AND portfolio_id=?)`,
		coin, account, portfolioID, now, coin, portfolioID,
	)
	return err
}

// GetSimState retrieves the current simulation state for a coin+portfolio
func GetSimState(coin string, portfolioID int64) (*models.SimState, error) {
	if portfolioID <= 0 {
		portfolioID = 1
	}
	row := db.DB.QueryRow(`
		SELECT id, portfolio_id, coin, cash, units, initial_capital, position, avg_cost, updated_at
		FROM sim_state WHERE coin = ? AND portfolio_id = ?`, coin, portfolioID)

	var s models.SimState
	err := row.Scan(&s.ID, &s.PortfolioID, &s.Coin, &s.Cash, &s.Units, &s.InitialCapital, &s.Position, &s.AvgCost, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query sim_state: %w", err)
	}
	return &s, nil
}

// GetOrInitState gets or creates sim state for coin+portfolio
func GetOrInitState(coin string, portfolioID int64) (*models.SimState, error) {
	if portfolioID <= 0 {
		portfolioID = 1
	}
	state, err := GetSimState(coin, portfolioID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		if err := InitSimState(coin, portfolioID); err != nil {
			return nil, fmt.Errorf("init sim state: %w", err)
		}
		state, err = GetSimState(coin, portfolioID)
		if err != nil || state == nil {
			return nil, fmt.Errorf("failed to load sim state after init")
		}
	}
	return state, nil
}

// ExecuteTrade executes a trade and records it in sim_trades
func ExecuteTrade(req models.TradeRequest) (*models.TradeResponse, error) {
	if req.PortfolioID <= 0 {
		req.PortfolioID = 1
	}

	// Ensure state exists
	state, err := GetOrInitState(req.Coin, req.PortfolioID)
	if err != nil {
		return nil, err
	}

	cashBefore := state.Cash
	unitsBefore := state.Units
	cashAfter := cashBefore
	unitsAfter := unitsBefore
	newAvgCost := state.AvgCost
	now := time.Now().UTC().Format(time.RFC3339)

	switch req.Action {
	case "BUY":
		if req.Price <= 0 {
			return nil, fmt.Errorf("invalid price for BUY: %f", req.Price)
		}
		if state.Position == "HOLDING" && state.Units > 0 && state.Cash == 0 {
			return nil, fmt.Errorf("already fully invested, cannot BUY")
		}
		spendAmount := cashBefore
		if req.Amount > 0 {
			if req.Amount > cashBefore {
				return nil, fmt.Errorf("amount %.2f exceeds available cash %.2f", req.Amount, cashBefore)
			}
			spendAmount = req.Amount
		}
		boughtUnits := spendAmount / req.Price
		unitsAfter = unitsBefore + boughtUnits
		cashAfter = cashBefore - spendAmount
		// Calculate new average cost: (prev_avg * prev_units + spend) / new_units
		if unitsAfter > 0 {
			newAvgCost = (state.AvgCost*unitsBefore + spendAmount) / unitsAfter
		}
	case "SELL":
		if req.Price <= 0 {
			return nil, fmt.Errorf("invalid price for SELL: %f", req.Price)
		}
		if state.Position == "CASH" {
			return nil, fmt.Errorf("already in CASH position, cannot SELL")
		}
		cashAfter = cashBefore + unitsBefore*req.Price
		unitsAfter = 0.0
		newAvgCost = 0.0 // reset avg cost on full sell
	case "HOLD":
		// HOLD는 상태 변경 없음, 히스토리 기록 생략
		currentValue := state.Cash + state.Units*req.Price
		returnPct := 0.0
		if state.InitialCapital > 0 {
			returnPct = (currentValue/state.InitialCapital - 1) * 100
		}
		state.CurrentPrice = req.Price
		state.CurrentValue = currentValue
		state.ReturnPct = returnPct
		return &models.TradeResponse{
			Success: true,
			Action:  "HOLD",
			Trade:   models.SimTrade{Coin: req.Coin, PortfolioID: req.PortfolioID, Action: "HOLD", Price: req.Price, Reason: req.Reason, ExecutedAt: now},
			State:   *state,
		}, nil
	default:
		return nil, fmt.Errorf("unknown action: %s", req.Action)
	}

	// 실제 거래 수량: BUY=매수한 수량, SELL=매도한 수량(unitsBefore)
	tradedUnits := unitsAfter - unitsBefore // BUY: 양수, SELL: 음수
	if tradedUnits < 0 {
		tradedUnits = -tradedUnits // 절댓값
	}

	account := portfolioIDToAccount(req.PortfolioID)

	// BUY/SELL만 히스토리 기록
	result, err := db.DB.Exec(`
		INSERT INTO sim_trades (coin, account, portfolio_id, action, price, units, cash_before, cash_after, units_before, units_after, reason, executed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		req.Coin, account, req.PortfolioID, req.Action, req.Price,
		tradedUnits,
		cashBefore, cashAfter,
		unitsBefore, unitsAfter,
		req.Reason, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert sim_trade: %w", err)
	}
	tradeID, _ := result.LastInsertId()

	// Update state (only for BUY/SELL)
	if req.Action != "HOLD" {
		newPosition := "CASH"
		if unitsAfter > 0 {
			newPosition = "HOLDING"
		}
		_, err = db.DB.Exec(`
			UPDATE sim_state SET cash=?, units=?, position=?, avg_cost=?, updated_at=? WHERE coin=? AND portfolio_id=?`,
			cashAfter, unitsAfter, newPosition, newAvgCost, now, req.Coin, req.PortfolioID,
		)
		if err != nil {
			return nil, fmt.Errorf("update sim_state: %w", err)
		}
		state.Cash = cashAfter
		state.Units = unitsAfter
		state.Position = newPosition
		state.AvgCost = newAvgCost
	}
	state.UpdatedAt = now

	trade := models.SimTrade{
		ID:          int(tradeID),
		Coin:        req.Coin,
		PortfolioID: req.PortfolioID,
		Action:      req.Action,
		Price:       req.Price,
		Units:       unitsAfter,
		CashBefore:  cashBefore,
		CashAfter:   cashAfter,
		UnitsBefore: unitsBefore,
		UnitsAfter:  unitsAfter,
		Reason:      req.Reason,
		ExecutedAt:  now,
	}

	// Calculate current value for response state
	currentValue := state.Cash + state.Units*req.Price
	returnPct := 0.0
	if state.InitialCapital > 0 {
		returnPct = (currentValue/state.InitialCapital - 1) * 100
	}
	state.CurrentPrice = req.Price
	state.CurrentValue = currentValue
	state.ReturnPct = returnPct

	return &models.TradeResponse{
		Success: true,
		Action:  req.Action,
		Trade:   trade,
		State:   *state,
	}, nil
}

// GetTradeHistory retrieves trade history for a coin+portfolio (newest first)
func GetTradeHistory(coin string, portfolioID int64, limit int) ([]models.SimTrade, error) {
	if portfolioID <= 0 {
		portfolioID = 1
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := db.DB.Query(`
		SELECT id, coin, COALESCE(portfolio_id,1), action, price, units,
		       cash_before, cash_after, units_before, units_after, COALESCE(reason,''), executed_at
		FROM sim_trades WHERE coin = ? AND portfolio_id = ?
		ORDER BY id DESC LIMIT ?`, coin, portfolioID, limit)
	if err != nil {
		return nil, fmt.Errorf("query sim_trades: %w", err)
	}
	defer rows.Close()

	var trades []models.SimTrade
	for rows.Next() {
		var t models.SimTrade
		if err := rows.Scan(&t.ID, &t.Coin, &t.PortfolioID, &t.Action, &t.Price, &t.Units,
			&t.CashBefore, &t.CashAfter, &t.UnitsBefore, &t.UnitsAfter,
			&t.Reason, &t.ExecutedAt); err != nil {
			return nil, err
		}
		trades = append(trades, t)
	}
	return trades, rows.Err()
}

// GetAllPortfolios retrieves all portfolios and their sim states
func GetAllPortfolios() ([]models.PortfolioSummary, error) {
	// Fetch all portfolios
	rows, err := db.DB.Query(`SELECT id, name, description, notify_on_trade, risk_limit_pct, created_at FROM portfolios ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query portfolios: %w", err)
	}
	defer rows.Close()

	var portfolios []models.Portfolio
	for rows.Next() {
		var p models.Portfolio
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.NotifyOnTrade, &p.RiskLimitPct, &p.CreatedAt); err != nil {
			return nil, err
		}
		portfolios = append(portfolios, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Get live prices
	priceMap := map[string]float64{}
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		if lp, err := FetchLivePrice(coin); err == nil {
			priceMap[coin] = lp.LastPrice
		}
	}

	var summaries []models.PortfolioSummary
	for _, p := range portfolios {
		stateRows, err := db.DB.Query(`
			SELECT id, portfolio_id, coin, cash, units, initial_capital, position, avg_cost, updated_at
			FROM sim_state WHERE portfolio_id = ? ORDER BY coin ASC`, p.ID)
		if err != nil {
			continue
		}

		var states []models.SimState
		totalValue := 0.0
		totalInitial := 0.0
		for stateRows.Next() {
			var s models.SimState
			if err := stateRows.Scan(&s.ID, &s.PortfolioID, &s.Coin, &s.Cash, &s.Units,
				&s.InitialCapital, &s.Position, &s.AvgCost, &s.UpdatedAt); err != nil {
				continue
			}
			price := priceMap[s.Coin]
			s.CurrentPrice = price
			s.CurrentValue = s.Cash + s.Units*price
			if s.InitialCapital > 0 {
				s.ReturnPct = (s.CurrentValue/s.InitialCapital - 1) * 100
			}
			totalValue += s.CurrentValue
			totalInitial += s.InitialCapital
			states = append(states, s)
		}
		stateRows.Close()

		totalReturn := 0.0
		if totalInitial > 0 {
			totalReturn = (totalValue/totalInitial - 1) * 100
		}

		summaries = append(summaries, models.PortfolioSummary{
			Portfolio:   p,
			States:      states,
			TotalValue:  totalValue,
			TotalReturn: totalReturn,
		})
	}

	return summaries, nil
}

// GetPerformance calculates daily/weekly/monthly returns per portfolio×coin
// and coin price changes for the same periods.
func GetPerformance() ([]models.PortfolioPerformance, error) {
	now := time.Now().UTC()
	periods := []int{1, 7, 30}

	// Fetch all portfolios
	pfRows, err := db.DB.Query(`SELECT id, name FROM portfolios ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("query portfolios: %w", err)
	}
	defer pfRows.Close()

	type pfMeta struct {
		id   int64
		name string
	}
	var portfolios []pfMeta
	for pfRows.Next() {
		var p pfMeta
		if err := pfRows.Scan(&p.id, &p.name); err == nil {
			portfolios = append(portfolios, p)
		}
	}

	var result []models.PortfolioPerformance

	for _, pf := range portfolios {
		// Get current states for this portfolio
		stateRows, err := db.DB.Query(`
			SELECT coin, cash, units, initial_capital, position
			FROM sim_state WHERE portfolio_id = ? ORDER BY coin ASC`, pf.id)
		if err != nil {
			continue
		}

		type stateRow struct {
			coin           string
			cash, units    float64
			initialCapital float64
			position       string
		}
		var states []stateRow
		for stateRows.Next() {
			var s stateRow
			if err := stateRows.Scan(&s.coin, &s.cash, &s.units, &s.initialCapital, &s.position); err == nil {
				states = append(states, s)
			}
		}
		stateRows.Close()

		if len(states) == 0 {
			continue
		}

		// Fetch current prices from price_ticker
		priceMap := map[string]float64{}
		tickerRows, err := db.DB.Query(`SELECT coin, current_price FROM price_ticker`)
		if err == nil {
			for tickerRows.Next() {
				var coin string
				var price float64
				if tickerRows.Scan(&coin, &price) == nil {
					priceMap[coin] = price
				}
			}
			tickerRows.Close()
		}

		var coinPerfs []models.CoinPerformance
		for _, s := range states {
			currentPrice := priceMap[s.coin]
			currentValue := s.cash + s.units*currentPrice

			perf := models.CoinPerformance{Coin: s.coin}

			for _, days := range periods {
				cutoff := now.AddDate(0, 0, -days)
				cutoffStr := cutoff.Format("2006-01-02T15:04:05Z")

				// Find last trade before cutoff → gives state at that point
				var cashAtT, unitsAtT float64
				err := db.DB.QueryRow(`
					SELECT cash_after, units_after FROM sim_trades
					WHERE portfolio_id = ? AND coin = ? AND executed_at < ?
					ORDER BY executed_at DESC LIMIT 1`,
					pf.id, s.coin, cutoffStr,
				).Scan(&cashAtT, &unitsAtT)
				if err == sql.ErrNoRows {
					// No trades before cutoff: all cash at initial_capital
					cashAtT = s.initialCapital
					unitsAtT = 0
				} else if err != nil {
					continue
				}

				// Get close price at cutoff date from daily_prices
				dateStr := cutoff.Format("2006-01-02")
				var priceAtT float64
				err = db.DB.QueryRow(`
					SELECT close FROM daily_prices
					WHERE coin = ? AND date <= ?
					ORDER BY date DESC LIMIT 1`,
					s.coin, dateStr,
				).Scan(&priceAtT)
				if err != nil || priceAtT == 0 {
					continue
				}

				valueAtT := cashAtT + unitsAtT*priceAtT
				if valueAtT <= 0 {
					continue
				}
				ret := (currentValue - valueAtT) / valueAtT * 100

				// Coin price change %
				var priceChange *float64
				if currentPrice > 0 {
					change := (currentPrice - priceAtT) / priceAtT * 100
					priceChange = &change
				}

				retPtr := ret
				switch days {
				case 1:
					perf.Return1D = &retPtr
					perf.PriceChange1D = priceChange
				case 7:
					perf.Return7D = &retPtr
					perf.PriceChange7D = priceChange
				case 30:
					perf.Return30D = &retPtr
					perf.PriceChange30D = priceChange
				}
			}

			coinPerfs = append(coinPerfs, perf)
		}

		result = append(result, models.PortfolioPerformance{
			PortfolioID:   pf.id,
			PortfolioName: pf.name,
			Coins:         coinPerfs,
		})
	}

	return result, nil
}

// portfolioIDToAccount maps portfolio_id back to account name for legacy columns
func portfolioIDToAccount(portfolioID int64) string {
	switch portfolioID {
	case 1:
		return "default"
	case 2:
		return "box-hunter"
	default:
		return fmt.Sprintf("portfolio-%d", portfolioID)
	}
}
