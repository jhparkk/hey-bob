// API wrapper for bob-crypto-pilot

export async function apiFetch<T = unknown>(
  url: string,
  options?: RequestInit
): Promise<T> {
  const res = await fetch(url, options);
  if (!res.ok) throw new Error(`HTTP ${res.status}`);
  return res.json();
}

// Price APIs
export const getPrices = (coin: string, period?: string, from?: string, to?: string) => {
  let url = `/api/v1/prices?coin=${coin}`;
  if (period) url += `&period=${period}`;
  if (from) url += `&from=${from}`;
  if (to)   url += `&to=${to}`;
  return apiFetch<{ success: boolean; data: PriceData[]; count: number }>(url);
};

export const getLivePrice = (coin: string) =>
  apiFetch<{ success: boolean; data: LivePriceData }>(`/api/v1/price/live?coin=${coin}`);

export const getTicker = () =>
  apiFetch<TickerData[]>(`/api/v1/ticker`);

// Strategy APIs
export const getStrategies = () =>
  apiFetch<{ success: boolean; strategies: Strategy[] }>('/api/v1/strategy');

export const getStrategyHistory = () =>
  apiFetch<{ success: boolean; history: StrategyHistory[] }>('/api/v1/strategy/history');

export const createStrategy = (body: Partial<Strategy>) =>
  apiFetch<{ success: boolean; strategy: Strategy }>('/api/v1/strategy', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

export const updateStrategy = (id: number, body: Partial<Strategy>) =>
  apiFetch<{ success: boolean }>(`/api/v1/strategy/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

export const deleteStrategy = (id: number) =>
  apiFetch<{ success: boolean }>(`/api/v1/strategy/${id}`, { method: 'DELETE' });

export const getStrategyVersions = (id: number) =>
  apiFetch<{ success: boolean; versions: StrategyVersion[] }>(`/api/v1/strategy/${id}/versions`);

// Portfolio APIs
export const getPortfolios = () =>
  apiFetch<{ success: boolean; portfolios: Portfolio[] }>('/api/v1/portfolios');

export const createPortfolio = (body: CreatePortfolioBody) =>
  apiFetch<{ success: boolean; portfolio: Portfolio }>('/api/v1/portfolios', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

export const updatePortfolio = (id: number, body: { name?: string; description?: string; notify_on_trade?: number; risk_limit_pct?: number }) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${id}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });


export const deletePortfolio = (id: number) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${id}`, { method: 'DELETE' });

export const resetPortfolio = (id: number) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${id}/reset`, { method: 'POST' });

export const addCoinToPortfolio = (portfolioId: number, coin: string, initialCapital: number) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${portfolioId}/coins`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ coin, initial_capital: initialCapital }),
  });

export const removeCoinFromPortfolio = (portfolioId: number, coin: string) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${portfolioId}/coins/${coin}`, { method: 'DELETE' });

export const getPortfolioStrategies = (id: number) =>
  apiFetch<Record<string, PortfolioCoinStrategy>>(`/api/v1/portfolios/${id}/strategies`);

export const patchPortfolioStrategy = (portfolioId: number, coin: string, strategyId: number) =>
  apiFetch<{ success: boolean }>(`/api/v1/portfolios/${portfolioId}/strategies/${coin}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ strategy_id: strategyId, selected_by: '수동' }),
  });

export const getPortfolioStrategyHistory = (portfolioId: number, coin: string) =>
  apiFetch<{ success: boolean; history: PortfolioStrategyHistory[] }>(
    `/api/v1/portfolios/${portfolioId}/strategy-history?coin=${coin}`
  );

// Simulation APIs
export const getSimPortfolios = () =>
  apiFetch<{ portfolios: SimPortfolioItem[] }>('/api/v1/simulation/portfolios');

export const getSimStatus = (coin: string, portfolioId: number) =>
  apiFetch<SimStatus>(`/api/v1/simulation/status?coin=${coin}&portfolio_id=${portfolioId}`);

export const getSimTrades = (coin: string, portfolioId: number) =>
  apiFetch<{ trades: SimTrade[] }>(`/api/v1/simulation/trades?coin=${coin}&portfolio_id=${portfolioId}`);

export const getSimPerformance = () =>
  apiFetch<PerformanceResponse>('/api/v1/simulation/performance');

// ── Upbit API ──────────────────────────────────────────────────────────────
export const getUpbitTicker = () =>
  apiFetch<TickerData[]>('/api/v1/upbit/ticker');

export const getUpbitSimPortfolios = () =>
  apiFetch<{ portfolios: SimPortfolioItem[] }>('/api/v1/upbit/simulation/portfolios');

export const getUpbitSimPerformance = () =>
  apiFetch<PerformanceResponse>('/api/v1/upbit/simulation/performance');

// ── Bithumb API ────────────────────────────────────────────────────────────
export const getBithumbTicker = () =>
  apiFetch<TickerData[]>('/api/v1/bithumb/ticker');

export const getBithumbSimPortfolios = () =>
  apiFetch<{ portfolios: SimPortfolioItem[] }>('/api/v1/bithumb/simulation/portfolios');

export const getBithumbSimPerformance = () =>
  apiFetch<PerformanceResponse>('/api/v1/bithumb/simulation/performance');

export const executeTrade = (body: ExecuteTradeBody) =>
  apiFetch<{ success: boolean; error?: string }>('/api/v1/simulation/trade', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });

// ── Types ──────────────────────────────────────────────────────────────────

export interface PriceData {
  id: number;
  coin: string;
  date: string;
  open: number;
  high: number;
  low: number;
  close: number;
  volume: number;
  created_at?: string;
  // 기술 지표
  ma7: number;
  ma20: number;
  ma50: number;
  ema9: number;
  ema21: number;
  rsi14: number;
  macd: number;
  macd_signal: number;
  bb_upper: number;
  bb_middle: number;
  bb_lower: number;
  adx14: number;
}

export interface LivePriceData {
  coin: string;
  last_price: number;
  price_change: number;
  price_change_percent: number;
  high_price: number;
  low_price: number;
  volume: number;
  quote_volume: number;
  open_price?: number;
}

export interface TickerData {
  coin: string;
  current_price: number;
  checked_at: string;
}

export interface Strategy {
  id: number;
  coin: string;
  name: string;
  description: string;
  signal: string;
  rsi_buy: number;
  rsi_sell: number;
  profit_take_pct: number;
  stop_loss_pct: number;
  notes: string;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface StrategyVersion {
  id: number;
  strategy_id: number;
  version: number;
  name: string;
  description: string;
  notes: string;
  changed_at: string;
}

export interface StrategyHistory {
  id: number;
  coin: string;
  strategy_id: number;
  action: string;
  changed_by: string;
  changed_at: string;
  snapshot: string;
}

export interface Portfolio {
  id: number;
  name: string;
  description: string;
  notify_on_trade: number;
  risk_limit_pct?: number;
  created_at: string;
}


export interface PortfolioCoinStrategy {
  portfolio_strategy?: {
    strategy_id: number;
    selected_by: string;
    selected_at: string;
    selection_reason: string;
  };
  strategy_name?: string;
}

export interface PortfolioStrategyHistory {
  id: number;
  portfolio_id: number;
  coin: string;
  strategy_id: number;
  strategy_name: string;
  action: string;
  changed_by: string;
  changed_at: string;
  note: string;
}

export interface CreatePortfolioBody {
  name: string;
  description: string;
  coins: { coin: string; initial_capital: number }[];
  exchange?: string;
}

export interface SimPortfolioItem {
  portfolio: Portfolio;
  states: SimState[];
  total_value: number;
  total_return_pct: number;
}

export interface SimState {
  coin: string;
  cash: number;
  units: number;
  initial_capital: number;
  position: string;
  avg_cost: number;
  portfolio_id: number;
  current_value: number;
  return_pct: number;
  current_price: number;
}

export interface SimStatus {
  success: boolean;
  coin: string;
  cash: number;
  units: number;
  initial_capital: number;
  position: string;
  avg_cost: number;
  current_price: number;
  error?: string;
}

export interface SimTrade {
  id: number;
  coin: string;
  action: string;
  price: number;
  units: number;
  fee: number;
  cash_before: number;
  cash_after: number;
  units_before: number;
  units_after: number;
  reason: string;
  executed_at: string;
  portfolio_id: number;
}

export interface CoinPerformance {
  coin: string;
  return_1d: number | null;
  return_7d: number | null;
  return_30d: number | null;
  return_life: number | null;
  price_change_1d: number | null;
  price_change_7d: number | null;
  price_change_30d: number | null;
  price_change_life: number | null;
}

export interface PortfolioPerformance {
  portfolio_id: number;
  portfolio_name: string;
  max_period: number;
  coins: CoinPerformance[];
}

export interface PerformanceResponse {
  success: boolean;
  portfolios: PortfolioPerformance[];
}

export interface ExecuteTradeBody {
  coin: string;
  action: string;
  price: number;
  reason: string;
  portfolio_id: number;
}
