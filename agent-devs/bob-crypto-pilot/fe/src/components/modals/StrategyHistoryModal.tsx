import React, { useState, useEffect } from 'react';
import Modal from './Modal';
import { getStrategyHistory } from '../../api';
import type { StrategyHistory } from '../../api';

interface Props {
  open: boolean;
  onClose: () => void;
}

const ACTION_COLORS: Record<string, string> = {
  CREATE: '#26a69a',
  UPDATE: '#ffa726',
  DELETE: '#ef5350',
  ACTIVATE: '#7e57c2',
};

const StrategyHistoryModal: React.FC<Props> = ({ open, onClose }) => {
  const [history, setHistory] = useState<StrategyHistory[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open) {
      setLoading(true);
      setError('');
      getStrategyHistory()
        .then(data => setHistory(data.history || []))
        .catch(err => setError(err.message))
        .finally(() => setLoading(false));
    }
  }, [open]);

  return (
    <Modal open={open} title="📜 전략 변경이력" onClose={onClose} maxWidth={700}>
      <div style={{ fontSize: 13 }}>
        {loading && <div style={{ color: '#aaa', textAlign: 'center', padding: 16 }}>로딩 중...</div>}
        {error && <div style={{ color: '#ef5350', padding: 12 }}>오류: {error}</div>}
        {!loading && !error && history.length === 0 && (
          <p style={{ color: '#aaa', textAlign: 'center' }}>변경이력이 없습니다.</p>
        )}
        {history.map(h => {
          const color = ACTION_COLORS[h.action] || '#aaa';
          return (
            <div key={h.id} style={{ borderBottom: '1px solid #2a2a5a', padding: '10px 0' }}>
              <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap' }}>
                <span style={{ background: `${color}22`, color, padding: '2px 8px', borderRadius: 4, fontWeight: 700, fontSize: 12 }}>
                  {h.action}
                </span>
                <span style={{ color: '#e0e0e0', fontWeight: 600 }}>#{h.strategy_id}</span>
                <span style={{ color: '#aaa', fontSize: 12 }}>by {h.changed_by}</span>
                <span style={{ color: '#666', fontSize: 11, marginLeft: 'auto' }}>{h.changed_at}</span>
              </div>
            </div>
          );
        })}
      </div>
    </Modal>
  );
};

export default StrategyHistoryModal;
