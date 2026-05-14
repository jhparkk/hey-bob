import React, { useState, useCallback } from 'react';
import Header from './components/Header';
import ChartPage from './pages/ChartPage';
import SimulationPage from './pages/SimulationPage';
import StrategyPage from './pages/StrategyPage';
import type { Page } from './types';

const SCROLL_KEY = (page: Page) => `scroll_pos_${page}`;

const App: React.FC = () => {
  const [activePage, setActivePage] = useState<Page>('chart');

  const handlePageChange = useCallback((next: Page) => {
    if (next === activePage) return;

    // 현재 탭 스크롤 저장
    sessionStorage.setItem(SCROLL_KEY(activePage), String(window.scrollY));

    setActivePage(next);

    // 다음 탭은 이미 마운트 상태 → 즉시 복원
    const saved = sessionStorage.getItem(SCROLL_KEY(next));
    window.scrollTo({ top: saved !== null ? Number(saved) : 0, behavior: 'instant' });
  }, [activePage]);

  return (
    <>
      <Header activePage={activePage} onPageChange={handlePageChange} />
      <main style={{ maxWidth: 1200, margin: '0 auto', padding: '16px 32px' }}>
        {/* 세 페이지 모두 항상 마운트 유지 — CSS display 로만 전환 */}
        <div style={{ display: activePage === 'chart' ? 'flex' : 'none', flexDirection: 'column', gap: 16 }}>
          <ChartPage isActive={activePage === 'chart'} />
        </div>
        <div style={{ display: activePage === 'sim' ? 'flex' : 'none', flexDirection: 'column', gap: 16 }}>
          <SimulationPage />
        </div>
        <div style={{ display: activePage === 'strategy' ? 'flex' : 'none', flexDirection: 'column', gap: 16 }}>
          <StrategyPage />
        </div>
      </main>
    </>
  );
};

export default App;
