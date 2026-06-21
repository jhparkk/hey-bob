import React from 'react';
import type { Page } from '../types';

interface HeaderProps {
  activePage: Page;
  onPageChange: (page: Page) => void;
}

const Header: React.FC<HeaderProps> = ({ activePage, onPageChange }) => {
  const tabBtn = (page: Page, label: string) => (
    <button
      key={page}
      onClick={() => onPageChange(page)}
      onMouseDown={(e) => e.preventDefault()}
      style={{
        background: 'transparent',
        border: 'none',
        borderBottom: `3px solid ${activePage === page ? '#26a69a' : 'transparent'}`,
        color: activePage === page ? '#e0e0e0' : '#888',
        fontSize: 14,
        fontWeight: 600,
        padding: '6px 16px 8px',
        cursor: 'pointer',
        transition: 'color 0.2s, border-color 0.2s',
        marginBottom: -2,
        whiteSpace: 'nowrap',
      }}
    >
      {label}
    </button>
  );

  return (
    <header style={{
      position: 'sticky',
      top: 0,
      zIndex: 100,
      background: '#16213e',
      borderBottom: '2px solid #2a2a4a',
    }}>
      {/* 타이틀 */}
      <div style={{ padding: '6px 20px 4px' }}>
        <span style={{ fontSize: 15, fontWeight: 700, letterSpacing: 0.5, color: '#e0e0e0' }}>
          📈 Bob Crypto Pilot
        </span>
      </div>
      {/* 네비게이션 탭 */}
      <nav style={{ display: 'flex', gap: 4, padding: '0 16px' }}>
        {tabBtn('chart',       '📊 시세 차트')}
        {tabBtn('sim',         '📈 바이낸스 시뮬')}
        {tabBtn('sim-upbit',   '🇰🇷 업비트 시뮬')}
        {tabBtn('sim-bithumb', '🟡 빗썸 시뮬')}
        {tabBtn('strategy',    '📋 전략')}
      </nav>
    </header>
  );
};

export default Header;
