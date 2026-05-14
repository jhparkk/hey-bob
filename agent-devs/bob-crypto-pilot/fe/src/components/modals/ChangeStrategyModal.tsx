import React, { useState, useEffect } from 'react';
import Modal, { Field, inputStyle, ModalFooter } from './Modal';
import type { Strategy } from '../../api';

interface Props {
  open: boolean;
  portfolioId: number | null;
  portfolioName: string;
  coin: string;
  strategies: Strategy[];
  currentStrategyId?: number;
  onClose: () => void;
  onSave: (portfolioId: number, coin: string, strategyId: number) => Promise<void>;
}

const ChangeStrategyModal: React.FC<Props> = ({
  open, portfolioId, portfolioName, coin, strategies, currentStrategyId, onClose, onSave
}) => {
  const [selectedId, setSelectedId] = useState<number>(0);
  const [status, setStatus] = useState('');
  const [isErr, setIsErr] = useState(false);

  useEffect(() => {
    if (open) {
      setSelectedId(currentStrategyId || 0);
      setStatus('');
    }
  }, [open, currentStrategyId]);

  const handleSave = async () => {
    if (!selectedId) { setIsErr(true); setStatus('❌ 전략을 선택하세요.'); return; }
    if (!portfolioId) return;
    try {
      await onSave(portfolioId, coin, selectedId);
      onClose();
    } catch (err: unknown) {
      setIsErr(true);
      setStatus('❌ ' + (err instanceof Error ? err.message : 'Error'));
    }
  };

  return (
    <Modal open={open} title="🔄 전략 변경" onClose={onClose} maxWidth={440}>
      <div style={{ color: '#9a9ada', fontSize: 13, marginBottom: 16 }}>
        포트폴리오: <strong style={{ color: '#e0e0e0' }}>{portfolioName}</strong>
        &nbsp;·&nbsp; 코인: <strong style={{ color: '#26a69a' }}>{coin}</strong>
      </div>
      <Field label="전략 선택">
        <select
          style={{ ...inputStyle }}
          value={selectedId}
          onChange={e => setSelectedId(parseInt(e.target.value, 10))}
        >
          <option value={0}>선택 안 함</option>
          {strategies.map(s => (
            <option key={s.id} value={s.id}>{s.name}</option>
          ))}
        </select>
      </Field>
      <ModalFooter onCancel={onClose} onSave={handleSave} saveLabel="✅ 변경" statusMsg={status} statusOk={!isErr} />
    </Modal>
  );
};

export default ChangeStrategyModal;
