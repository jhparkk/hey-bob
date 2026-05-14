import { useState, useCallback } from 'react';
import {
  getStrategies,
  getPortfolios,
  getPortfolioStrategies,
  createStrategy,
  updateStrategy,
  deleteStrategy,
  patchPortfolioStrategy,
} from '../api';
import type {
  Strategy,
  Portfolio,
  PortfolioCoinStrategy,
} from '../api';

export interface EnrichedPortfolio extends Portfolio {
  strategies: Record<string, PortfolioCoinStrategy>;
}

export function useStrategy() {
  const [strategies, setStrategies] = useState<Strategy[]>([]);
  const [portfolios, setPortfolios] = useState<EnrichedPortfolio[]>([]);
  const [loading, setLoading] = useState(false);

  const loadStrategies = useCallback(async () => {
    try {
      const data = await getStrategies();
      setStrategies(data.strategies || []);
    } catch (err) {
      console.error('loadStrategies error:', err);
    }
  }, []);

  const loadPortfolios = useCallback(async () => {
    try {
      const pfData = await getPortfolios();
      const portfolioList = pfData.portfolios || [];
      const enriched = await Promise.all(
        portfolioList.map(async (pf) => {
          try {
            const sData = await getPortfolioStrategies(pf.id);
            return { ...pf, strategies: sData };
          } catch {
            return { ...pf, strategies: {} };
          }
        })
      );
      setPortfolios(enriched);
    } catch (err) {
      console.error('loadPortfolios error:', err);
    }
  }, []);

  const saveStrategy = useCallback(async (
    id: number | null,
    body: { name: string; description: string; notes: string }
  ) => {
    if (id === null) {
      await createStrategy(body);
    } else {
      await updateStrategy(id, body);
    }
    await loadStrategies();
  }, [loadStrategies]);

  const removeStrategy = useCallback(async (id: number) => {
    await deleteStrategy(id);
    await loadStrategies();
  }, [loadStrategies]);

  const changePortfolioStrategy = useCallback(async (
    portfolioId: number,
    coin: string,
    strategyId: number
  ) => {
    await patchPortfolioStrategy(portfolioId, coin, strategyId);
    await loadPortfolios();
  }, [loadPortfolios]);

  const loadAll = useCallback(async () => {
    setLoading(true);
    try {
      await Promise.all([loadStrategies(), loadPortfolios()]);
    } finally {
      setLoading(false);
    }
  }, [loadStrategies, loadPortfolios]);

  return {
    strategies,
    portfolios,
    loading,
    loadStrategies,
    loadPortfolios,
    loadAll,
    saveStrategy,
    removeStrategy,
    changePortfolioStrategy,
  };
}
