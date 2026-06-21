import React, { useState, useEffect } from 'react';
import Modal, { Field, inputStyle } from './Modal';
import { updatePortfolio, resetPortfolio, addCoinToPortfolio, removeCoinFromPortfolio } from '../../api';

const ALL_COINS = ['BTC', 'ETH', 'SOL'];

interface CoinWithCapital {
  coin: string;
  initial_capital: number;
}

interface Props {
  open: boolean;
  portfolioId: number | null;
  portfolioName: string;
  portfolioDesc: string;
  portfolioCoins: CoinWithCapital[];
  portfolioNotifyOnTrade?: number;
  portfolioRiskLimitPct?: number;
  onClose: () => void;
  onSaved: () => void;
  currency?: string;
  defaultCapital?: number;
}

const EditPortfolioModal: React.FC<Props> = ({
  open, portfolioId, portfolioName, portfolioDesc, portfolioCoins, portfolioNotifyOnTrade, portfolioRiskLimitPct, onClose, onSaved,
  currency = '$', defaultCapital = 100,
}) => {
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [notifyOnTrade, setNotifyOnTrade] = useState(1);
  const [riskLimitPct, setRiskLimitPct] = useState(15);
  const [status, setStatus] = useState('');
  const [isErr, setIsErr] = useState(false);

  // 현재 실제 코인 목록 (모달 열릴 때 portfolioCoins로 초기화)
  const [baseCoins, setBaseCoins] = useState<CoinWithCapital[]>([]);

  // 스테이징: 추가 예정 코인
  const [pendingAdds, setPendingAdds] = useState<{ coin: string; capital: number }[]>([]);

  // 스테이징: 삭제 예정 코인
  const [pendingRemoves, setPendingRemoves] = useState<string[]>([]);

  // 코인 추가 UI용
  const [addingCoin, setAddingCoin] = useState<string | null>(null);
  const [addCapital, setAddCapital] = useState('100');

  useEffect(() => {
    if (open) {
      setName(portfolioName);
      setDesc(portfolioDesc);
      setNotifyOnTrade(portfolioNotifyOnTrade ?? 1);
      setRiskLimitPct(portfolioRiskLimitPct ?? 15);
      setStatus('');
      setIsErr(false);
      setBaseCoins(portfolioCoins.map(c => ({ ...c })));
      setPendingAdds([]);
      setPendingRemoves([]);
      setAddingCoin(null);
      setAddCapital(String(defaultCapital));
    }
  }, [open, portfolioName, portfolioDesc, portfolioCoins, portfolioNotifyOnTrade, portfolioRiskLimitPct]);

  // 현재 모달에서 보이는 코인 목록 (base - removes + adds)
  const displayCoins: string[] = [
    ...baseCoins.filter(c => !pendingRemoves.includes(c.coin)).map(c => c.coin),
    ...pendingAdds.map(a => a.coin),
  ];

  // 추가 가능한 코인 (전체 - displayCoins에 없는 것)
  const addableCoins = ALL_COINS.filter(c => !displayCoins.includes(c));

  const handleStageAdd = (coin: string, capital: number) => {
    if (pendingRemoves.includes(coin)) {
      // 삭제 예정이던 기존 코인을 다시 추가 → 삭제 취소만
      setPendingRemoves(prev => prev.filter(r => r !== coin));
    } else {
      setPendingAdds(prev => [...prev, { coin, capital }]);
    }
    setAddingCoin(null);
    setAddCapital('100');
  };

  const handleStageRemove = (coin: string) => {
    const isPendingAdd = pendingAdds.find(a => a.coin === coin);

    if (isPendingAdd) {
      // 아직 저장 안 된 추가 코인 → 그냥 pendingAdds에서 제거
      setPendingAdds(prev => prev.filter(a => a.coin !== coin));
      return;
    }

    const isBase = baseCoins.some(c => c.coin === coin);
    if (isBase) {
      // 실제 저장된 코인 → 삭제 confirm 후 pendingRemoves에 추가
      if (!confirm(`'${coin}'을(를) 삭제하면 해당 코인의 모든 거래 내역이 삭제됩니다. 계속하시겠습니까?`)) return;
      setPendingRemoves(prev => [...prev, coin]);
    }
  };

  const handleSave = async () => {
    if (!name.trim()) { setIsErr(true); setStatus('❌ 이름은 필수입니다.'); return; }
    if (!portfolioId) return;
    try {
      // 1. 이름/설명/알림/리스크한도 저장
      const data = await updatePortfolio(portfolioId, { name: name.trim(), description: desc.trim(), notify_on_trade: notifyOnTrade, risk_limit_pct: riskLimitPct });
      if (!data.success) throw new Error('포트폴리오 저장 실패');

      // 2. 코인 삭제 (pendingRemoves)
      for (const coin of pendingRemoves) {
        await removeCoinFromPortfolio(portfolioId, coin);
      }

      // 3. 코인 추가 (pendingAdds)
      for (const { coin, capital } of pendingAdds) {
        await addCoinToPortfolio(portfolioId, coin, capital);
      }

      setIsErr(false);
      onClose();
      onSaved();
    } catch (err: unknown) {
      setIsErr(true);
      setStatus('❌ ' + (err instanceof Error ? err.message : 'Error'));
    }
  };

  const handleReset = async () => {
    if (!confirm(`포트폴리오 '${portfolioName}'의 모든 거래내역과 잔고를 초기화합니다. 계속하시겠습니까?`)) return;
    if (!portfolioId) return;
    try {
      const data = await resetPortfolio(portfolioId);
      if (!data.success) throw new Error('API error');
      setIsErr(false);
      setStatus('✅ 리셋 완료');
      onSaved();
    } catch (err: unknown) {
      setIsErr(true);
      setStatus('❌ ' + (err instanceof Error ? err.message : 'Error'));
    }
  };

  const changedCount = pendingAdds.length + pendingRemoves.length;

  return (
    <Modal open={open} title="✏️ 포트폴리오 편집" onClose={onClose} maxWidth={440}>
      {/* 1. 포트폴리오 이름 */}
      <Field label="포트폴리오 이름 *">
        <input style={inputStyle} value={name} onChange={e => setName(e.target.value)} />
      </Field>

      {/* 2. 설명 */}
      <Field label="설명">
        <input style={inputStyle} value={desc} onChange={e => setDesc(e.target.value)} />
      </Field>

      {/* 3. 거래 알림 토글 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginTop: 12, marginBottom: 4 }}>
        <span style={{ color: '#aaa', fontSize: 13 }}>거래 알림 (DM)</span>
        <label style={{ display: 'flex', alignItems: 'center', gap: 6, cursor: 'pointer' }}>
          <input
            type="checkbox"
            checked={notifyOnTrade === 1}
            onChange={(e) => setNotifyOnTrade(e.target.checked ? 1 : 0)}
            style={{ width: 16, height: 16, cursor: 'pointer' }}
          />
          <span style={{ color: notifyOnTrade ? '#26a69a' : '#888', fontSize: 13 }}>
            {notifyOnTrade ? '🔔 ON — 매수/매도 시 DM 수신' : '🔕 OFF — DM 없음'}
          </span>
        </label>
      </div>

      {/* 4. 리스크 한도 */}
      <div style={{ marginTop: 12 }}>
        <span style={{ color: '#aaa', fontSize: 13 }}>리스크 한도</span>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 6 }}>
          <span style={{ color: '#ef5350', fontSize: 13 }}>-</span>
          <input
            type="number" min="1" max="50" step="1"
            value={riskLimitPct}
            onChange={e => setRiskLimitPct(Number(e.target.value))}
            style={{ width: 60, padding: '4px 8px', background: '#1a1a3e', border: '1px solid #3a3a6a', borderRadius: 4, color: '#ef5350', fontSize: 14, textAlign: 'center' }}
          />
          <span style={{ color: '#aaa', fontSize: 13 }}>% 손실 시 자동 청산</span>
        </div>
        <div style={{ color: '#666', fontSize: 11, marginTop: 4 }}>
          포트폴리오 전체 손실이 이 수치 초과 시 모든 포지션 자동 청산
        </div>
      </div>

      {/* 5. 코인 관리 섹션 */}
      <div style={{ borderTop: '1px solid #2a2a4a', paddingTop: 12, marginTop: 12 }}>
        <div style={{ fontSize: 13, color: '#aaa', marginBottom: 8, fontWeight: 600 }}>코인 관리</div>

        {/* 현재 코인 목록 */}
        {displayCoins.length === 0 && (
          <div style={{ fontSize: 12, color: '#555', marginBottom: 8 }}>코인 없음</div>
        )}
        {displayCoins.map(c => {
          const isPendingRemove = pendingRemoves.includes(c);
          const pendingAdd = pendingAdds.find(a => a.coin === c);
          const isPendingAdd = !!pendingAdd;
          const baseInfo = baseCoins.find(b => b.coin === c);
          const capital = isPendingAdd ? pendingAdd!.capital : baseInfo?.initial_capital;
          return (
            <div key={c} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0' }}>
              <span style={{
                color: isPendingRemove ? '#888' : isPendingAdd ? '#26a69a' : '#e0e0e0',
                textDecoration: isPendingRemove ? 'line-through' : 'none',
                minWidth: 40, fontSize: 13,
              }}>
                {c}{' '}
                {capital != null && (
                  <span style={{ fontSize: 11, color: isPendingRemove ? '#666' : '#aaa' }}>{currency}{capital?.toLocaleString()}</span>
                )}
                {isPendingAdd && <span style={{ fontSize: 11, color: '#26a69a' }}> (추가 예정)</span>}
                {isPendingRemove && <span style={{ fontSize: 11, color: '#888' }}> (삭제 예정)</span>}
              </span>
              <button
                onClick={() => isPendingRemove
                  ? setPendingRemoves(prev => prev.filter(r => r !== c))
                  : handleStageRemove(c)
                }
                style={{
                  color: isPendingRemove ? '#888' : '#ef5350',
                  background: 'transparent',
                  border: `1px solid ${isPendingRemove ? '#555' : '#ef5350'}`,
                  borderRadius: 4, padding: '2px 8px', cursor: 'pointer', fontSize: 11,
                }}
              >
                {isPendingRemove ? '↩ 취소' : '🗑 삭제'}
              </button>
            </div>
          );
        })}

        {/* 추가 가능한 코인 버튼들 */}
        {addableCoins.length > 0 && (
          <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 8 }}>
            {addableCoins.map(c => (
              <button key={c}
                onClick={() => setAddingCoin(c)}
                style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid #26a69a', background: 'transparent', color: '#26a69a', cursor: 'pointer', fontSize: 12 }}
              >+ {c}</button>
            ))}
          </div>
        )}

        {/* 코인 추가 인라인 폼 */}
        {addingCoin && (
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginTop: 8, padding: '8px', background: '#0f0f2f', borderRadius: 6, border: '1px solid #26a69a' }}>
            <span style={{ color: '#26a69a', fontSize: 13, minWidth: 30 }}>{addingCoin}</span>
            <span style={{ color: '#aaa', fontSize: 12 }}>초기 자본 {currency}</span>
            <input
              type="number" min="1" value={addCapital}
              onChange={e => setAddCapital(e.target.value)}
              style={{ width: 90, padding: '4px 8px', background: '#1a1a3e', border: '1px solid #3a3a6a', borderRadius: 4, color: '#e0e0e0', fontSize: 13 }}
            />
            <button
              onClick={() => handleStageAdd(addingCoin, parseFloat(addCapital) || 100)}
              style={{ padding: '4px 12px', background: '#26a69a', border: 'none', borderRadius: 4, color: '#fff', fontSize: 12, cursor: 'pointer' }}
            >추가</button>

            <button
              onClick={() => setAddingCoin(null)}
              style={{ padding: '4px 12px', background: 'transparent', border: '1px solid #555', borderRadius: 4, color: '#aaa', fontSize: 12, cursor: 'pointer' }}
            >취소</button>
          </div>
        )}
      </div>

      {/* 구분선 */}
      <div style={{ borderTop: '1px solid #2a2a4a', marginTop: 16 }} />

      {/* 상태 메시지 */}
      {status && (
        <div style={{ color: isErr ? '#ef5350' : '#26a69a', fontSize: 13, textAlign: 'center', margin: '8px 0' }}>
          {status}
        </div>
      )}

      {/* 취소/저장 버튼 */}
      <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 16, marginBottom: 8 }}>
        <button
          onClick={onClose}
          style={{ padding: '8px 16px', background: '#2a2a4a', border: '1px solid #5a5a9a', borderRadius: 6, color: '#ccc', fontSize: 14, cursor: 'pointer' }}
        >취소</button>
        <button
          onClick={handleSave}
          style={{ padding: '8px 20px', background: '#26a69a', border: 'none', borderRadius: 6, color: '#fff', fontSize: 14, fontWeight: 700, cursor: 'pointer' }}
        >
          💾 저장{changedCount > 0 ? ` (변경 ${changedCount}건)` : ''}
        </button>
      </div>

      {/* Danger zone */}
      <div style={{ background: '#2a1010', border: '1px solid #7a3a3a', borderRadius: 8, padding: '14px 16px', marginTop: 8 }}>
        <div style={{ fontSize: 13, fontWeight: 700, color: '#e57373', marginBottom: 6 }}>
          ⚠️ 자산 리셋 — 모든 거래내역과 잔고를 초기화합니다
        </div>
        <div style={{ fontSize: 12, color: '#aaa', marginBottom: 10 }}>
          포트폴리오 <strong style={{ color: '#e0e0e0' }}>{portfolioName}</strong>의 sim_trades와 sim_state를 초기 상태로 되돌립니다.
        </div>
        <button
          onClick={handleReset}
          style={{ padding: '7px 18px', background: '#7a1a1a', border: '1px solid #c0392b', borderRadius: 6, color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}
        >🔄 리셋</button>
      </div>
    </Modal>
  );
};

export default EditPortfolioModal;
