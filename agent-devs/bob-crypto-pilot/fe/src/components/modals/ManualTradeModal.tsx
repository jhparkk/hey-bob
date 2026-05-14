import React, { useState, useEffect } from 'react';

interface Props {
  open: boolean;
  action: 'BUY' | 'SELL';
  coin: string;
  currentPrice: number;
  cash: number;
  units: number;
  onConfirm: (amount: number) => void;
  onClose: () => void;
}

const overlayStyle: React.CSSProperties = {
  position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.65)',
  display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000,
};

const cardStyle: React.CSSProperties = {
  background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 12,
  padding: 24, maxWidth: 380, width: '100%', boxShadow: '0 8px 32px rgba(0,0,0,0.5)',
};

const inputStyle: React.CSSProperties = {
  width: '100%', padding: '8px 12px', borderRadius: 6, border: '1px solid #3a3a5a',
  background: '#1a1a2e', color: '#ccc', fontSize: 14, outline: 'none', boxSizing: 'border-box',
};

const pctBtnStyle = (active: boolean): React.CSSProperties => ({
  flex: 1, padding: '6px 0', borderRadius: 6,
  border: `1px solid ${active ? '#26a69a' : '#3a3a5a'}`,
  background: active ? 'rgba(38,166,154,0.15)' : 'transparent',
  color: active ? '#26a69a' : '#888', cursor: 'pointer', fontSize: 12, fontWeight: active ? 700 : 400,
});

const ManualTradeModal: React.FC<Props> = ({
  open, action, coin, currentPrice, cash, units, onConfirm, onClose,
}) => {
  const [inputVal, setInputVal] = useState('');
  const [pctSel, setPctSel] = useState<number | null>(null);

  useEffect(() => {
    if (open) { setInputVal(''); setPctSel(null); }
  }, [open]);

  if (!open) return null;

  const totalCoinVal = units * currentPrice;
  const maxAmount = action === 'BUY' ? cash : totalCoinVal;

  const handlePct = (pct: number | null) => {
    setPctSel(pct);
    if (pct === null) { setInputVal(''); return; }
    const amt = maxAmount * pct;
    setInputVal(amt > 0 ? amt.toFixed(2) : '');
  };

  const handleInput = (v: string) => {
    setInputVal(v);
    setPctSel(null);
  };

  const amount = parseFloat(inputVal) || 0;

  // Preview calc
  let preview = '';
  if (action === 'BUY') {
    if (pctSel === null && amount === 0) {
      // 전량: use full cash
      const qty = cash / currentPrice;
      preview = `매수 수량: ~${qty.toFixed(8)} ${coin}`;
    } else {
      const qty = amount / currentPrice;
      preview = `매수 수량: ~${qty > 0 ? qty.toFixed(8) : '0'} ${coin}`;
    }
  } else {
    if (pctSel === null && amount === 0) {
      preview = `매도 금액: ~${fmtMoney(totalCoinVal)}`;
    } else {
      preview = `매도 금액: ~${fmtMoney(amount)}`;
    }
  }

  const isBuy = action === 'BUY';
  const titleColor = isBuy ? '#26a69a' : '#ef5350';
  const titlePrefix = isBuy ? '▲ BUY' : '▼ SELL';

  const handleConfirm = () => {
    // amount=0 means 전량
    onConfirm(amount);
  };

  return (
    <div style={overlayStyle}>
      <div style={cardStyle}>
        {/* Title */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 18 }}>
          <h3 style={{ margin: 0, fontSize: 18, fontWeight: 700, color: titleColor }}>
            {titlePrefix} {coin}
          </h3>
          <button onClick={onClose} style={{ background: 'none', border: 'none', color: '#888', cursor: 'pointer', fontSize: 18 }}>✕</button>
        </div>

        {/* Current price */}
        <div style={{ marginBottom: 14 }}>
          <span style={{ fontSize: 12, color: '#888' }}>현재가</span>
          <span style={{ fontSize: 16, fontWeight: 600, color: '#e0e0e0', marginLeft: 10 }}>
            {fmtMoney(currentPrice)}
          </span>
        </div>

        {/* Balance info */}
        <div style={{ background: '#1a1a2e', borderRadius: 8, padding: '10px 14px', marginBottom: 16, border: '1px solid #2a2a4a' }}>
          {isBuy ? (
            <div style={{ fontSize: 13, color: '#aaa' }}>
              현금 잔고: <span style={{ color: '#e0e0e0', fontWeight: 600 }}>{fmtMoney(cash)}</span>
            </div>
          ) : (
            <div style={{ fontSize: 13, color: '#aaa' }}>
              보유: <span style={{ color: '#e0e0e0', fontWeight: 600 }}>{units.toFixed(8)} {coin}</span>
              <span style={{ color: '#888' }}> ({fmtMoney(totalCoinVal)})</span>
            </div>
          )}
        </div>

        {/* Percent buttons */}
        <div style={{ display: 'flex', gap: 6, marginBottom: 10 }}>
          {[0.25, 0.5, 0.75].map(p => (
            <button key={p} style={pctBtnStyle(pctSel === p)} onClick={() => handlePct(pctSel === p ? null : p)}>
              {p * 100}%
            </button>
          ))}
          <button style={pctBtnStyle(pctSel === 1)} onClick={() => handlePct(pctSel === 1 ? null : 1)}>
            전량
          </button>
        </div>

        {/* Amount input */}
        <div style={{ marginBottom: 10 }}>
          <input
            type="number"
            min={0}
            step={0.01}
            value={inputVal}
            onChange={e => handleInput(e.target.value)}
            placeholder={isBuy ? '매수 금액 (비우면 전량)' : '매도 금액 (비우면 전량)'}
            style={inputStyle}
          />
        </div>

        {/* Preview */}
        <div style={{ fontSize: 12, color: '#7ab3ef', marginBottom: 18, minHeight: 18 }}>
          {preview}
        </div>

        {/* Buttons */}
        <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end' }}>
          <button
            onClick={onClose}
            style={{ padding: '8px 20px', borderRadius: 6, border: '1px solid #3a3a5a', background: 'transparent', color: '#aaa', cursor: 'pointer', fontSize: 14 }}
          >취소</button>
          <button
            onClick={handleConfirm}
            style={{ padding: '8px 24px', borderRadius: 6, border: 'none', background: isBuy ? '#26a69a' : '#ef5350', color: '#fff', cursor: 'pointer', fontSize: 14, fontWeight: 700 }}
          >확인</button>
        </div>
      </div>
    </div>
  );
};

function fmtMoney(n: number): string {
  if (isNaN(n)) return '—';
  return '$' + n.toLocaleString('en-US', { minimumFractionDigits: 2, maximumFractionDigits: 2 });
}

export default ManualTradeModal;
