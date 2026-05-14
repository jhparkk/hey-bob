import React, { useState, useEffect } from 'react';
import Modal, { Field, inputStyle, ModalFooter } from './Modal';
import { createPortfolio } from '../../api';

const AVAILABLE_COINS = ['BTC', 'ETH'];

interface Props {
  open: boolean;
  onClose: () => void;
  onCreated: () => void;
}

const AddPortfolioModal: React.FC<Props> = ({ open, onClose, onCreated }) => {
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [coins, setCoins] = useState<Record<string, { checked: boolean; capital: number }>>(
    Object.fromEntries(AVAILABLE_COINS.map(c => [c, { checked: true, capital: 100 }]))
  );
  const [status, setStatus] = useState('');
  const [isErr, setIsErr] = useState(false);

  useEffect(() => {
    if (open) {
      setName('');
      setDesc('');
      setCoins(Object.fromEntries(AVAILABLE_COINS.map(c => [c, { checked: true, capital: 100 }])));
      setStatus('');
    }
  }, [open]);

  const handleSave = async () => {
    if (!name.trim()) { setIsErr(true); setStatus('❌ 이름은 필수입니다.'); return; }
    const selectedCoins = AVAILABLE_COINS
      .filter(c => coins[c].checked)
      .map(c => ({ coin: c, initial_capital: coins[c].capital }));
    if (selectedCoins.length === 0) { setIsErr(true); setStatus('❌ 코인을 최소 1개 선택하세요.'); return; }
    try {
      const data = await createPortfolio({ name: name.trim(), description: desc.trim(), coins: selectedCoins });
      if (!data.success) throw new Error('API error');
      onClose();
      onCreated();
    } catch (err: unknown) {
      setIsErr(true);
      setStatus('❌ ' + (err instanceof Error ? err.message : 'Error'));
    }
  };

  return (
    <Modal open={open} title="📁 새 포트폴리오" onClose={onClose} maxWidth={440}>
      <Field label="포트폴리오 이름 *">
        <input style={inputStyle} value={name} onChange={e => setName(e.target.value)} placeholder="예: 모멘텀 전략 포트폴리오" />
      </Field>
      <Field label="설명">
        <input style={inputStyle} value={desc} onChange={e => setDesc(e.target.value)} placeholder="예: 강세장 대응 포트폴리오" />
      </Field>
      <Field label="운영할 코인 * (최소 1개)">
        {AVAILABLE_COINS.map(coin => (
          <div key={coin} style={{
            display: 'flex', alignItems: 'center', gap: 10, padding: '8px 12px',
            border: '1px solid #444', borderRadius: 6, marginBottom: 6
          }}>
            <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer', fontSize: 14, userSelect: 'none', minWidth: 60 }}>
              <input
                type="checkbox"
                checked={coins[coin].checked}
                onChange={e => setCoins(prev => ({ ...prev, [coin]: { ...prev[coin], checked: e.target.checked } }))}
                style={{ width: 16, height: 16, cursor: 'pointer' }}
              />
              <strong>{coin}</strong>
            </label>
            <label style={{ display: 'flex', alignItems: 'center', gap: 6, fontSize: 13, color: '#aaa' }}>
              초기 자본: <span style={{ color: '#888' }}>$</span>
              <input
                type="number"
                value={coins[coin].capital}
                disabled={!coins[coin].checked}
                onChange={e => setCoins(prev => ({ ...prev, [coin]: { ...prev[coin], capital: parseFloat(e.target.value) || 100 } }))}
                style={{ width: 90, ...inputStyle, padding: '4px 8px', fontSize: 13 }}
              />
            </label>
          </div>
        ))}
      </Field>
      <ModalFooter onCancel={onClose} onSave={handleSave} saveLabel="💾 생성" statusMsg={status} statusOk={!isErr} />
    </Modal>
  );
};

export default AddPortfolioModal;
