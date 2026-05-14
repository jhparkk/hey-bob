import { create } from 'zustand';

interface SimulationStore {
  currentPortfolioId: number;
  currentCoin: string;
  setCurrentPortfolioId: (id: number) => void;
  setCurrentCoin: (coin: string) => void;
}

export const useSimulationStore = create<SimulationStore>((set) => ({
  currentPortfolioId: 1,
  currentCoin: 'BTC',
  setCurrentPortfolioId: (id) => set({ currentPortfolioId: id }),
  setCurrentCoin: (coin) => set({ currentCoin: coin }),
}));
