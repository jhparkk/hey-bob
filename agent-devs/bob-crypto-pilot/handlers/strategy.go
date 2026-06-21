package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bob-crypto-pilot/db"
	"bob-crypto-pilot/models"

	"github.com/gin-gonic/gin"
)

// ── Models ─────────────────────────────────────────────────────────────────

type Strategy struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	Description   string  `json:"description"`
	Signal        string  `json:"signal"`
	RSIBuy        float64 `json:"rsi_buy"`
	RSISell       float64 `json:"rsi_sell"`
	ProfitTakePct float64 `json:"profit_take_pct"`
	StopLossPct   float64 `json:"stop_loss_pct"`
	Notes         string  `json:"notes"`
	Version       int     `json:"version"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type StrategyVersion struct {
	ID          int64  `json:"id"`
	StrategyID  int64  `json:"strategy_id"`
	Version     int    `json:"version"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
	ChangedAt   string `json:"changed_at"`
}

type PortfolioStrategyHistoryRow struct {
	ID           int64  `json:"id"`
	PortfolioID  int64  `json:"portfolio_id"`
	Coin         string `json:"coin"`
	StrategyID   int64  `json:"strategy_id"`
	StrategyName string `json:"strategy_name"`
	Action       string `json:"action"`
	ChangedBy    string `json:"changed_by"`
	ChangedAt    string `json:"changed_at"`
	Note         string `json:"note"`
}

type StrategyActiveRow struct {
	Coin            string `json:"coin"`
	StrategyID      int64  `json:"strategy_id"`
	SelectedBy      string `json:"selected_by"`
	SelectedAt      string `json:"selected_at"`
	SelectionReason string `json:"selection_reason"`
}

type StrategyHistoryRow struct {
	ID         int64  `json:"id"`
	Coin       string `json:"coin"`
	StrategyID int64  `json:"strategy_id"`
	Action     string `json:"action"`
	ChangedBy  string `json:"changed_by"`
	ChangedAt  string `json:"changed_at"`
	Snapshot   string `json:"snapshot"`
}

// ── Helpers ────────────────────────────────────────────────────────────────

func nowKST() string {
	kst := time.FixedZone("KST", 9*60*60)
	return time.Now().In(kst).Format("2006-01-02 15:04 KST")
}

func validateCoin(coin string) bool {
	return coin == "BTC" || coin == "ETH" || coin == "SOL"
}

// fetchAllStrategies returns all strategies (coin-agnostic)
func fetchAllStrategies() ([]Strategy, error) {
	rows, err := db.DB.Query(`
		SELECT id, name, description, signal,
		       rsi_buy, rsi_sell, profit_take_pct, stop_loss_pct,
		       notes, COALESCE(version,1), created_at, updated_at
		FROM strategies ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Strategy{}
	for rows.Next() {
		var s Strategy
		if err := rows.Scan(&s.ID, &s.Name, &s.Description, &s.Signal,
			&s.RSIBuy, &s.RSISell, &s.ProfitTakePct, &s.StopLossPct,
			&s.Notes, &s.Version, &s.CreatedAt, &s.UpdatedAt); err != nil {
			continue
		}
		list = append(list, s)
	}
	return list, nil
}

func fetchStrategyByID(id int64) (Strategy, error) {
	var s Strategy
	err := db.DB.QueryRow(`
		SELECT id, name, description, signal,
		       rsi_buy, rsi_sell, profit_take_pct, stop_loss_pct,
		       notes, COALESCE(version,1), created_at, updated_at
		FROM strategies WHERE id=?`, id).
		Scan(&s.ID, &s.Name, &s.Description, &s.Signal,
			&s.RSIBuy, &s.RSISell, &s.ProfitTakePct, &s.StopLossPct,
			&s.Notes, &s.Version, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

// fetchPortfolioActive fetches active strategy for a specific portfolio+coin
func fetchPortfolioActive(portfolioID int64, coin string) (models.PortfolioStrategy, error) {
	var ps models.PortfolioStrategy
	var selectedAt []byte
	err := db.DB.QueryRow(
		`SELECT id, portfolio_id, coin, strategy_id, selected_by, CAST(selected_at AS TEXT), selection_reason
		 FROM portfolio_strategies WHERE portfolio_id=? AND coin=?`, portfolioID, coin).
		Scan(&ps.ID, &ps.PortfolioID, &ps.Coin, &ps.StrategyID, &ps.SelectedBy, &selectedAt, &ps.SelectionReason)
	ps.SelectedAt = string(selectedAt)
	return ps, err
}

// fetchAllPortfolioActive returns all portfolio active strategies for a coin keyed by portfolio_id
func fetchAllPortfolioActive(coin string) (map[int64]models.PortfolioStrategy, error) {
	rows, err := db.DB.Query(
		`SELECT id, portfolio_id, coin, strategy_id, selected_by, CAST(selected_at AS TEXT), selection_reason
		 FROM portfolio_strategies WHERE coin=?`, coin)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[int64]models.PortfolioStrategy)
	for rows.Next() {
		var ps models.PortfolioStrategy
		var selectedAt []byte
		if err := rows.Scan(&ps.ID, &ps.PortfolioID, &ps.Coin, &ps.StrategyID, &ps.SelectedBy, &selectedAt, &ps.SelectionReason); err != nil {
			continue
		}
		ps.SelectedAt = string(selectedAt)
		result[ps.PortfolioID] = ps
	}
	return result, rows.Err()
}

func recordStrategyHistory(coin string, strategyID int64, action, changedBy string, snapshot interface{}) {
	snap := ""
	if snapshot != nil {
		b, _ := json.Marshal(snapshot)
		snap = string(b)
	}
	db.DB.Exec(
		`INSERT INTO strategy_history (coin, strategy_id, action, changed_by, changed_at, snapshot) VALUES (?, ?, ?, ?, ?, ?)`,
		coin, strategyID, action, changedBy, nowKST(), snap,
	)
}

// ── Strategy Library Handlers (coin-agnostic) ─────────────────────────────

// GetStrategies handles GET /api/v1/strategy
// Returns all strategies without coin distinction
func GetStrategies(c *gin.Context) {
	strategies, err := fetchAllStrategies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "strategies": strategies})
}

// CreateStrategy handles POST /api/v1/strategy
func CreateStrategy(c *gin.Context) {
	var body struct {
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		Signal        string  `json:"signal"`
		RSIBuy        float64 `json:"rsi_buy"`
		RSISell       float64 `json:"rsi_sell"`
		ProfitTakePct float64 `json:"profit_take_pct"`
		StopLossPct   float64 `json:"stop_loss_pct"`
		Notes         string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name은 필수입니다"})
		return
	}
	if body.Signal == "" {
		body.Signal = "HOLD"
	}
	now := nowKST()
	res, err := db.DB.Exec(`
		INSERT INTO strategies (coin, name, description, signal, rsi_buy, rsi_sell, profit_take_pct, stop_loss_pct, notes, created_at, updated_at)
		VALUES ('', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		body.Name, body.Description, body.Signal,
		body.RSIBuy, body.RSISell, body.ProfitTakePct, body.StopLossPct,
		body.Notes, now, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	newID, _ := res.LastInsertId()
	s, err := fetchStrategyByID(newID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	recordStrategyHistory("", newID, "CREATE", "system", s)
	c.JSON(http.StatusOK, gin.H{"success": true, "strategy": s})
}

// UpdateStrategy handles PUT /api/v1/strategy/:id
func UpdateStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	var body struct {
		Name          string  `json:"name"`
		Description   string  `json:"description"`
		Signal        string  `json:"signal"`
		RSIBuy        float64 `json:"rsi_buy"`
		RSISell       float64 `json:"rsi_sell"`
		ProfitTakePct float64 `json:"profit_take_pct"`
		StopLossPct   float64 `json:"stop_loss_pct"`
		Notes         string  `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}

	// Read existing strategy for version comparison
	old, err := fetchStrategyByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "전략을 찾을 수 없습니다"})
		return
	}

	// 빈 값은 기존 값 유지 (부분 업데이트)
	if body.Name == "" {
		body.Name = old.Name
	}
	if body.Description == "" {
		body.Description = old.Description
	}
	if body.Signal == "" {
		body.Signal = old.Signal
	}

	// notes가 변경된 경우 버전 증가 + 이전 스냅샷 저장
	notesChanged := body.Notes != old.Notes
	newVersion := old.Version
	now := nowKST()
	if notesChanged {
		newVersion = old.Version + 1
		db.DB.Exec(`INSERT INTO strategy_versions (strategy_id, version, name, description, notes, changed_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			id, old.Version, old.Name, old.Description, old.Notes, now)
	}

	res, err := db.DB.Exec(`
		UPDATE strategies SET name=?, description=?, signal=?,
		  rsi_buy=?, rsi_sell=?, profit_take_pct=?, stop_loss_pct=?, notes=?, version=?, updated_at=?
		WHERE id=?`,
		body.Name, body.Description, body.Signal,
		body.RSIBuy, body.RSISell, body.ProfitTakePct, body.StopLossPct,
		body.Notes, newVersion, now, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "전략을 찾을 수 없습니다"})
		return
	}
	s, err := fetchStrategyByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	recordStrategyHistory("", id, "UPDATE", "system", s)
	c.JSON(http.StatusOK, gin.H{"success": true, "strategy": s, "version": newVersion, "notes_changed": notesChanged})
}

// GetStrategyVersions handles GET /api/v1/strategy/:id/versions
func GetStrategyVersions(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	rows, err := db.DB.Query(`
		SELECT id, strategy_id, version, name, description, notes, changed_at
		FROM strategy_versions WHERE strategy_id=? ORDER BY version DESC`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()
	versions := []StrategyVersion{}
	for rows.Next() {
		var v StrategyVersion
		rows.Scan(&v.ID, &v.StrategyID, &v.Version, &v.Name, &v.Description, &v.Notes, &v.ChangedAt)
		versions = append(versions, v)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "versions": versions})
}

// GetPortfolioStrategyHistory handles GET /api/v1/portfolios/:id/strategy-history
func GetPortfolioStrategyHistory(c *gin.Context) {
	portfolioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	coin := c.Query("coin") // optional filter

	var rows *sql.Rows
	if coin != "" {
		rows, err = db.DB.Query(`
			SELECT id, portfolio_id, coin, COALESCE(strategy_id,0), strategy_name, action, changed_by, changed_at, note
			FROM portfolio_strategy_history WHERE portfolio_id=? AND coin=? ORDER BY id DESC LIMIT 100`,
			portfolioID, strings.ToUpper(coin))
	} else {
		rows, err = db.DB.Query(`
			SELECT id, portfolio_id, coin, COALESCE(strategy_id,0), strategy_name, action, changed_by, changed_at, note
			FROM portfolio_strategy_history WHERE portfolio_id=? ORDER BY id DESC LIMIT 100`,
			portfolioID)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()
	history := []PortfolioStrategyHistoryRow{}
	for rows.Next() {
		var h PortfolioStrategyHistoryRow
		rows.Scan(&h.ID, &h.PortfolioID, &h.Coin, &h.StrategyID, &h.StrategyName, &h.Action, &h.ChangedBy, &h.ChangedAt, &h.Note)
		history = append(history, h)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "history": history})
}

// DeleteStrategy handles DELETE /api/v1/strategy/:id
func DeleteStrategy(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	// Block delete if active in any portfolio
	var activeCount int
	db.DB.QueryRow(`SELECT COUNT(*) FROM portfolio_strategies WHERE strategy_id=?`, id).Scan(&activeCount)
	if activeCount > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "포트폴리오에서 활성 중인 전략은 삭제할 수 없습니다"})
		return
	}
	// Snapshot before delete
	s, _ := fetchStrategyByID(id)
	res, err := db.DB.Exec(`DELETE FROM strategies WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "전략을 찾을 수 없습니다"})
		return
	}
	recordStrategyHistory("", id, "DELETE", "system", s)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// ── Strategy History Handlers ──────────────────────────────────────────────

// GetStrategyHistory handles GET /api/v1/strategy/history
func GetStrategyHistory(c *gin.Context) {
	rows, err := db.DB.Query(`
		SELECT id, coin, strategy_id, action, changed_by, changed_at, snapshot
		FROM strategy_history ORDER BY id DESC LIMIT 100`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()
	history := []StrategyHistoryRow{}
	for rows.Next() {
		var h StrategyHistoryRow
		rows.Scan(&h.ID, &h.Coin, &h.StrategyID, &h.Action, &h.ChangedBy, &h.ChangedAt, &h.Snapshot)
		history = append(history, h)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "history": history})
}

// ── Portfolio Handlers ─────────────────────────────────────────────────────

// GetPortfolios handles GET /api/v1/portfolios
func GetPortfolios(c *gin.Context) {
	rows, err := db.DB.Query(`SELECT id, name, description, COALESCE(notify_on_trade,1), COALESCE(risk_limit_pct,15.0), created_at FROM portfolios ORDER BY id ASC`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()
	portfolios := []models.Portfolio{}
	for rows.Next() {
		var p models.Portfolio
		rows.Scan(&p.ID, &p.Name, &p.Description, &p.NotifyOnTrade, &p.RiskLimitPct, &p.CreatedAt)
		portfolios = append(portfolios, p)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "portfolios": portfolios})
}

// CreatePortfolio handles POST /api/v1/portfolios
// Body: { "name": "...", "description": "...", "coins": [{"coin":"BTC","initial_capital":200},{"coin":"ETH","initial_capital":150}] }
func CreatePortfolio(c *gin.Context) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Exchange    string `json:"exchange"`
		Coins       []struct {
			Coin           string  `json:"coin"`
			InitialCapital float64 `json:"initial_capital"`
		} `json:"coins"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}
	if body.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name은 필수입니다"})
		return
	}
	if len(body.Coins) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "코인을 최소 1개 선택하세요"})
		return
	}

	exchange := body.Exchange
	if exchange == "" {
		exchange = "binance"
	}
	now := nowKST()
	res, err := db.DB.Exec(`INSERT INTO portfolios (name, description, exchange, created_at) VALUES (?, ?, ?, ?)`,
		body.Name, body.Description, exchange, now)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	portfolioID, _ := res.LastInsertId()

	// Initialize sim_state for selected coins with individual initial_capital
	for _, coinEntry := range body.Coins {
		initCap := coinEntry.InitialCapital
		if initCap <= 0 {
			initCap = 100.0
		}
		db.DB.Exec(`INSERT OR IGNORE INTO sim_state (coin, portfolio_id, account, cash, units, initial_capital, position, avg_cost, updated_at)
			VALUES (?, ?, ?, ?, 0.0, ?, 'CASH', 0.0, ?)`,
			coinEntry.Coin, portfolioID, body.Name, initCap, initCap, now)
	}

	portfolio := models.Portfolio{
		ID:            portfolioID,
		Name:          body.Name,
		Description:   body.Description,
		NotifyOnTrade: 1,
		CreatedAt:     now,
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "portfolio": portfolio})
}

// UpdatePortfolio handles PUT /api/v1/portfolios/:id
// Body: { "name": "...", "description": "...", "notify_on_trade": 0|1, "risk_limit_pct": 15.0 }
func UpdatePortfolio(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	var body struct {
		Name          string   `json:"name"`
		Description   string   `json:"description"`
		NotifyOnTrade *int     `json:"notify_on_trade"` // pointer: nil이면 변경 안함
		RiskLimitPct  *float64 `json:"risk_limit_pct"`  // pointer: nil이면 변경 안함
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}

	// 최소한 하나의 필드는 있어야 함
	if body.Name == "" && body.NotifyOnTrade == nil && body.RiskLimitPct == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "name, notify_on_trade, risk_limit_pct 중 하나는 필수입니다"})
		return
	}

	// 동적 SET 구성
	setClauses := []string{}
	args := []interface{}{}

	if body.Name != "" {
		setClauses = append(setClauses, "name=?")
		args = append(args, body.Name)
		setClauses = append(setClauses, "description=?")
		args = append(args, body.Description)
	}
	if body.NotifyOnTrade != nil {
		setClauses = append(setClauses, "notify_on_trade=?")
		args = append(args, *body.NotifyOnTrade)
	}
	if body.RiskLimitPct != nil {
		setClauses = append(setClauses, "risk_limit_pct=?")
		args = append(args, *body.RiskLimitPct)
	}

	query := "UPDATE portfolios SET " + strings.Join(setClauses, ", ") + " WHERE id=?"
	args = append(args, id)

	res, err := db.DB.Exec(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "포트폴리오를 찾을 수 없습니다"})
		return
	}
	// 업데이트된 포트폴리오 반환
	var p models.Portfolio
	if err := db.DB.QueryRow(`SELECT id, name, description, COALESCE(notify_on_trade,1), COALESCE(risk_limit_pct,15.0), created_at FROM portfolios WHERE id=?`, id).
		Scan(&p.ID, &p.Name, &p.Description, &p.NotifyOnTrade, &p.RiskLimitPct, &p.CreatedAt); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "portfolio": p})
}

// ResetPortfolio handles POST /api/v1/portfolios/:id/reset
// Deletes all sim_trades and resets sim_state to initial_capital
func ResetPortfolio(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	now := nowKST()
	// 1. sim_trades 삭제
	db.DB.Exec("DELETE FROM sim_trades WHERE portfolio_id = ?", id)
	// 2. sim_state 리셋 (cash = initial_capital)
	db.DB.Exec("UPDATE sim_state SET cash=initial_capital, units=0.0, position='CASH', avg_cost=0.0, updated_at=? WHERE portfolio_id=?", now, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// DeletePortfolio handles DELETE /api/v1/portfolios/:id
func DeletePortfolio(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}
	// Protect built-in portfolios (id=1,2)
	if id <= 2 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "기본 포트폴리오는 삭제할 수 없습니다"})
		return
	}
	res, err := db.DB.Exec(`DELETE FROM portfolios WHERE id=?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "포트폴리오를 찾을 수 없습니다"})
		return
	}
	// Clean up related data
	db.DB.Exec(`DELETE FROM portfolio_strategies WHERE portfolio_id=?`, id)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// GetPortfolioStrategies handles GET /api/v1/portfolios/:id/strategies
func GetPortfolioStrategies(c *gin.Context) {
	portfolioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 ID"})
		return
	}

	// Get portfolio info
	var p models.Portfolio
	if err := db.DB.QueryRow(`SELECT id, name, description, COALESCE(notify_on_trade,1), COALESCE(risk_limit_pct,15.0), created_at FROM portfolios WHERE id=?`, portfolioID).
		Scan(&p.ID, &p.Name, &p.Description, &p.NotifyOnTrade, &p.RiskLimitPct, &p.CreatedAt); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "포트폴리오를 찾을 수 없습니다"})
		return
	}

	result := gin.H{"success": true, "portfolio_id": portfolioID, "portfolio": p}
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		ps, err := fetchPortfolioActive(portfolioID, coin)
		if err != nil {
			ps = models.PortfolioStrategy{
				PortfolioID: portfolioID,
				Coin:        coin,
				StrategyID:  0,
			}
		}
		// Enrich with strategy name
		var stratName string
		if ps.StrategyID > 0 {
			db.DB.QueryRow(`SELECT name FROM strategies WHERE id=?`, ps.StrategyID).Scan(&stratName)
		}
		result[coin] = gin.H{
			"portfolio_strategy": ps,
			"strategy_name":      stratName,
		}
	}
	c.JSON(http.StatusOK, result)
}

// PatchPortfolioStrategy handles PATCH /api/v1/portfolios/:id/strategies/:coin
// Body: { "strategy_id": 3, "selected_by": "Bob", "selection_reason": "..." }
func PatchPortfolioStrategy(c *gin.Context) {
	portfolioID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "유효하지 않은 포트폴리오 ID"})
		return
	}
	coin := strings.ToUpper(c.Param("coin"))
	if !validateCoin(coin) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "coin은 BTC, ETH, 또는 SOL이어야 합니다"})
		return
	}
	var req struct {
		StrategyID      int64  `json:"strategy_id"`
		SelectedBy      string `json:"selected_by"`
		SelectionReason string `json:"selection_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}
	// Validate strategy exists (coin-agnostic)
	var cnt int
	db.DB.QueryRow(`SELECT COUNT(*) FROM strategies WHERE id=?`, req.StrategyID).Scan(&cnt)
	if cnt == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "전략을 찾을 수 없습니다"})
		return
	}
	if req.SelectedBy == "" {
		req.SelectedBy = "수동"
	}
	now := nowKST()

	// Fetch old strategy for history tracking
	var oldStrategyID int64
	var oldStrategyName string
	db.DB.QueryRow(`SELECT ps.strategy_id, COALESCE(s.name,'')
		FROM portfolio_strategies ps
		LEFT JOIN strategies s ON s.id=ps.strategy_id
		WHERE ps.portfolio_id=? AND ps.coin=?`, portfolioID, coin).Scan(&oldStrategyID, &oldStrategyName)

	// Fetch new strategy name
	var newStrategyName string
	db.DB.QueryRow(`SELECT name FROM strategies WHERE id=?`, req.StrategyID).Scan(&newStrategyName)

	db.DB.Exec(`INSERT INTO portfolio_strategies (portfolio_id, coin, strategy_id, selected_by, selected_at, selection_reason)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(portfolio_id, coin) DO UPDATE SET
		  strategy_id=excluded.strategy_id,
		  selected_by=excluded.selected_by,
		  selected_at=excluded.selected_at,
		  selection_reason=excluded.selection_reason`,
		portfolioID, coin, req.StrategyID, req.SelectedBy, now, req.SelectionReason)

	// Record portfolio_strategy_history
	action := "ASSIGN"
	if oldStrategyID > 0 {
		action = "CHANGE"
	}
	note := fmt.Sprintf("%s → %s", oldStrategyName, newStrategyName)
	db.DB.Exec(`INSERT INTO portfolio_strategy_history
		(portfolio_id, coin, strategy_id, strategy_name, action, changed_by, changed_at, note)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		portfolioID, coin, req.StrategyID, newStrategyName, action, req.SelectedBy, now, note)

	ps := models.PortfolioStrategy{
		PortfolioID:     portfolioID,
		Coin:            coin,
		StrategyID:      req.StrategyID,
		SelectedBy:      req.SelectedBy,
		SelectedAt:      now,
		SelectionReason: req.SelectionReason,
	}
	recordStrategyHistory(coin, req.StrategyID, "ACTIVATE", req.SelectedBy, ps)
	c.JSON(http.StatusOK, gin.H{"success": true, "active": ps})
}

// ── Legacy / Compatibility Handlers ───────────────────────────────────────

// GetAllStrategies handles GET /api/v1/strategy (legacy - same as GetStrategies but with per-coin structure)
// Kept for backward compat with older UI code; new UI uses GetStrategies.
func GetAllStrategies(c *gin.Context) {
	strategies, err := fetchAllStrategies()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	// Also return per-portfolio active info for both coins
	portfolioActive := map[string]map[int64]models.PortfolioStrategy{}
	for _, coin := range []string{"BTC", "ETH", "SOL"} {
		pa, _ := fetchAllPortfolioActive(coin)
		portfolioActive[coin] = pa
	}
	c.JSON(http.StatusOK, gin.H{
		"success":          true,
		"strategies":       strategies,
		"portfolio_active": portfolioActive,
	})
}

// PatchCoinActiveStrategy handles PATCH /api/v1/strategy/:coin/active (legacy)
// Body: { "strategy_id": 3, "portfolio_id": 1, "selected_by": "Bob", "selection_reason": "..." }
func PatchCoinActiveStrategy(c *gin.Context) {
	coin := strings.ToUpper(c.Param("coin"))
	if !validateCoin(coin) {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "coin은 BTC, ETH, 또는 SOL이어야 합니다"})
		return
	}
	var req struct {
		StrategyID      int64  `json:"strategy_id"`
		PortfolioID     int64  `json:"portfolio_id"`
		SelectedBy      string `json:"selected_by"`
		SelectionReason string `json:"selection_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid JSON: " + err.Error()})
		return
	}
	if req.PortfolioID <= 0 {
		req.PortfolioID = 1
	}
	// Validate strategy exists (coin-agnostic now)
	var cnt int
	db.DB.QueryRow(`SELECT COUNT(*) FROM strategies WHERE id=?`, req.StrategyID).Scan(&cnt)
	if cnt == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "전략을 찾을 수 없습니다"})
		return
	}
	if req.SelectedBy == "" {
		req.SelectedBy = "수동"
	}
	now := nowKST()

	db.DB.Exec(`INSERT INTO portfolio_strategies (portfolio_id, coin, strategy_id, selected_by, selected_at, selection_reason)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(portfolio_id, coin) DO UPDATE SET
		  strategy_id=excluded.strategy_id,
		  selected_by=excluded.selected_by,
		  selected_at=excluded.selected_at,
		  selection_reason=excluded.selection_reason`,
		req.PortfolioID, coin, req.StrategyID, req.SelectedBy, now, req.SelectionReason)

	active := models.PortfolioStrategy{
		PortfolioID:     req.PortfolioID,
		Coin:            coin,
		StrategyID:      req.StrategyID,
		SelectedBy:      req.SelectedBy,
		SelectedAt:      now,
		SelectionReason: req.SelectionReason,
	}
	recordStrategyHistory(coin, req.StrategyID, "ACTIVATE", req.SelectedBy, active)
	c.JSON(http.StatusOK, gin.H{"success": true, "active": active})
}

// AddCoinToPortfolio godoc
// POST /api/v1/portfolios/:id/coins
// Body: {"coin": "SOL", "initial_capital": 100}
func AddCoinToPortfolio(c *gin.Context) {
	idStr := c.Param("id")
	portfolioID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "invalid portfolio id"})
		return
	}
	var body struct {
		Coin           string  `json:"coin"`
		InitialCapital float64 `json:"initial_capital"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(400, gin.H{"success": false, "error": err.Error()})
		return
	}
	body.Coin = strings.ToUpper(body.Coin)
	if !validateCoin(body.Coin) {
		c.JSON(400, gin.H{"success": false, "error": "지원하지 않는 코인: " + body.Coin})
		return
	}
	if body.InitialCapital <= 0 {
		body.InitialCapital = 100.0
	}
	account := fmt.Sprintf("portfolio-%d", portfolioID)
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = db.DB.Exec(`
		INSERT OR IGNORE INTO sim_state (coin, portfolio_id, account, cash, units, initial_capital, position, avg_cost, updated_at)
		VALUES (?, ?, ?, ?, 0.0, ?, 'CASH', 0.0, ?)`,
		body.Coin, portfolioID, account, body.InitialCapital, body.InitialCapital, now)
	if err != nil {
		c.JSON(500, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"success": true, "coin": body.Coin, "portfolio_id": portfolioID, "initial_capital": body.InitialCapital})
}

// RemoveCoinFromPortfolio handles DELETE /api/v1/portfolios/:id/coins/:coin
func RemoveCoinFromPortfolio(c *gin.Context) {
	idStr := c.Param("id")
	portfolioID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(400, gin.H{"success": false, "error": "invalid portfolio id"})
		return
	}
	coin := strings.ToUpper(c.Param("coin"))
	if !validateCoin(coin) {
		c.JSON(400, gin.H{"success": false, "error": "invalid coin"})
		return
	}
	db.DB.Exec("DELETE FROM sim_state WHERE portfolio_id=? AND coin=?", portfolioID, coin)
	db.DB.Exec("DELETE FROM sim_trades WHERE portfolio_id=? AND coin=?", portfolioID, coin)
	db.DB.Exec("DELETE FROM portfolio_strategies WHERE portfolio_id=? AND coin=?", portfolioID, coin)
	c.JSON(200, gin.H{"success": true, "removed": coin, "portfolio_id": portfolioID})
}
