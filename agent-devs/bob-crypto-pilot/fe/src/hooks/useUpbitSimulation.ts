import { useState, useCallback } from 'react';
import {
  getUpbitSimPortfolios,
  getSimStatus,
  getSimTrades,
  executeTrade,
  getUpbitTicker,
} from '../api';
import type {
  SimPortfolioItem,
  SimStatus,
  SimTrade,
  TickerData,
  ExecuteTradeBody,
} from '../api';

export function useUpbitSimulation() {
  const [currentPortfolioId, setCurrentPortfolioId] = useState<number>(0);
  const [currentCoin, setCurrentCoin] = useState<string>('BTC');

  const [simPortfolios, setSimPortfolios] = useState<SimPortfolioItem[]>([]);
  const [simStatus, setSimStatus] = useState<SimStatus | null>(null);
  const [simTrades, setSimTrades] = useState<SimTrade[]>([]);
  const [tickers, setTickers] = useState<TickerData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadSimPortfolios = useCallback(async () => {
    const data = await getUpbitSimPortfolios();
    const portfolios = data.portfolios || [];
    setSimPortfolios(portfolios);
    return portfolios;
  }, []);

  const loadSimCoinDetail = useCallback(async (coin: string, portfolioId: number) => {
    setLoading(true);
    setError(null);
    try {
      const [statusData, tradesData] = await Promise.all([
        getSimStatus(coin, portfolioId),
        getSimTrades(coin, portfolioId),
      ]);
      if (!statusData.success) throw new Error(statusData.error || 'API error');
      setSimStatus(statusData);
      setSimTrades(tradesData.trades || []);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Unknown error';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  const loadTickers = useCallback(async () => {
    try {
      const data = await getUpbitTicker();
      setTickers(data);
    } catch (err) {
      console.error('loadUpbitTickers error:', err);
    }
  }, []);

  const doManualTrade = useCallback(async (
    coin: string,
    portfolioId: number,
    action: 'BUY' | 'SELL',
    amount?: number
  ) => {
    const tickerData = await getUpbitTicker();
    const ticker = tickerData.find(t => t.coin === coin);
    const price = ticker?.current_price || 0;
    if (!price) throw new Error('현재가 조회 실패');
    const tradePayload: ExecuteTradeBody & { amount?: number } = { coin, action, price, reason: '수동', portfolio_id: portfolioId };
    if (amount && amount > 0) tradePayload.amount = amount;
    const result = await executeTrade(tradePayload);
    if (!result.success) throw new Error(result.error || 'Trade failed');
    return price;
  }, []);

  return {
    simPortfolios,
    currentPortfolioId,
    setCurrentPortfolioId,
    currentCoin,
    setCurrentCoin,
    simStatus,
    simTrades,
    tickers,
    loading,
    error,
    loadSimPortfolios,
    loadSimCoinDetail,
    loadTickers,
    doManualTrade,
    setSimPortfolios,
  };
}
