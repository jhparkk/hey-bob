import React, { useState, useEffect } from 'react';
import Modal from './Modal';
import { getPortfolioStrategyHistory } from '../../api';
import type { PortfolioStrategyHistory } from '../../api';

interface Props {
  open: boolean;
  portfolioId: number | null;
  coin: string;
  onClose: () => void;
}

const ACTION_COLORS: Record<string, string> = {
  ASSIGN: '#26a69a',
  CHANGE: '#ffa726',
  REMOVE: '#ef5350',
};

const PortfolioStrategyHistoryModal: React.FC<Props> = ({ open, portfolioId, coin, onClose }) => {
  const [history, setHistory] = useState<PortfolioStrategyHistory[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open && portfolioId) {
      setLoading(true);
      setError('');
      getPortfolioStrategyHistory(portfolioId, coin)
        .then(data => setHistory(data.history || []))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false));
    }
  }, [open, portfolioId, coin]);

  return (
    <Modal open={open} title={`📜 전략 변경 이력 — ${coin}`} onClose={onClose} maxWidth={700}>
      <div style={{ maxHeight: 480, overflowY: 'auto', fontSize: 13 }}>
        {loading && <div style={{ color: '#aaa', textAlign: 'center', padding: 16 }}>로딩 중...</div>}
        {error && <div style={{ color: '#ef5350', padding: 12 }}>오류: {error}</div>}
        {!loading && !error && history.length === 0 && (
          <div style={{ color: '#aaa', textAlign: 'center', padding: 16 }}>변경 이력 없음</div>
        )}
        {history.length > 0 && (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #2a2a4a' }}>
                <th style={{ padding: '6px 10px', textAlign: 'left', color: '#666' }}>날짜</th>
                <th style={{ padding: '6px 10px', textAlign: 'left', color: '#666' }}>액션</th>
                <th style={{ padding: '6px 10px', textAlign: 'left', color: '#666' }}>변경 내용</th>
                <th style={{ padding: '6px 10px', textAlign: 'left', color: '#666' }}>변경자</th>
              </tr>
            </thead>
            <tbody>
              {history.map(h => {
                const color = ACTION_COLORS[h.action] || '#aaa';
                return (
                  <tr key={h.id} style={{ borderBottom: '1px solid #1e1e3a' }}>
                    <td style={{ padding: '8px 10px', color: '#666', whiteSpace: 'nowrap' }}>{h.changed_at}</td>
                    <td style={{ padding: '8px 10px' }}>
                      <span style={{ background: `${color}22`, color, padding: '2px 6px', borderRadius: 3, fontSize: 11, fontWeight: 700 }}>
                        {h.action}
                      </span>
                    </td>
                    <td style={{ padding: '8px 10px', color: '#ccc' }}>{h.note || h.strategy_name}</td>
                    <td style={{ padding: '8px 10px', color: '#aaa' }}>{h.changed_by}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>
    </Modal>
  );
};

export default PortfolioStrategyHistoryModal;
