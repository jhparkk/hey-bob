import React, { useEffect, useState, useCallback } from 'react';
import { useUpbitSimulation } from '../hooks/useUpbitSimulation';
import { executeTrade, updatePortfolio, getUpbitSimPerformance, getUpbitTicker } from '../api';
import type { SimStatus, SimTrade, PortfolioPerformance } from '../api';
import AddPortfolioModal from '../components/modals/AddPortfolioModal';
import EditPortfolioModal from '../components/modals/EditPortfolioModal';
import ManualTradeModal from '../components/modals/ManualTradeModal';
import PortfolioROIChart from '../components/PortfolioROIChart';

function fmtMoney(n: number): string {
  if (isNaN(n)) return '—';
  return '₩' + n.toLocaleString('ko-KR', { maximumFractionDigits: 0 });
}
function fmtUnits(n: number, coin: string): string {
  if (isNaN(n) || n === 0) return `0 ${coin}`;
  return n.toFixed(8) + ' ' + coin;
}
function fmtDatetime(s: string): string {
  if (!s) return '—';
  try {
    const d = new Date(s);
    return d.toLocaleString('ko-KR', { timeZone: 'Asia/Seoul', hour12: false });
  } catch { return s; }
}

const UpbitSimulationPage: React.FC<{ isActive?: boolean }> = ({ isActive }) => {
  const {
    simPortfolios, currentPortfolioId, setCurrentPortfolioId,
    currentCoin, setCurrentCoin, simStatus, simTrades,
    tickers, loading, error, loadSimPortfolios,
    loadSimCoinDetail, loadTickers,
  } = useUpbitSimulation();

  const [showAddModal, setShowAddModal] = useState(false);
  const [editPf, setEditPf] = useState<{ id: number; name: string; desc: string } | null>(null);
  const [showTradeModal, setShowTradeModal] = useState<'BUY' | 'SELL' | null>(null);
  const [performance, setPerformance] = useState<PortfolioPerformance[]>([]);

  useEffect(() => {
    getUpbitSimPerformance().then(r => setPerformance(r.portfolios)).catch(() => {});
  }, []);

  const currentPortfolio = simPortfolios.find(i => i.portfolio?.id === currentPortfolioId)
    ?? simPortfolios[0];

  const pfObj = currentPortfolio?.portfolio;
  const states = currentPortfolio?.states || [];
  const coins = states.map(s => s.coin);

  const loadAll = useCallback(async () => {
    const portfolios = await loadSimPortfolios();
    if (portfolios.length > 0) {
      const firstId = portfolios[0].portfolio?.id ?? 0;
      const hasCurrent = portfolios.find(i => i.portfolio?.id === currentPortfolioId);
      const pfId = hasCurrent ? currentPortfolioId : firstId;
      setCurrentPortfolioId(pfId);
      const pf = portfolios.find(i => i.portfolio?.id === pfId);
      const pfCoins = pf?.states?.map(s => s.coin) || [];
      const activeCoin = pfCoins.includes(currentCoin) ? currentCoin : (pfCoins[0] || 'BTC');
      setCurrentCoin(activeCoin);
      await loadSimCoinDetail(activeCoin, pfId);
    }
    await loadTickers();
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => { loadAll(); }, [loadAll]);
  useEffect(() => { if (isActive) loadAll(); }, [isActive, loadAll]);

  const switchPortfolio = async (pfId: number) => {
    const scrollY = window.scrollY;
    setCurrentPortfolioId(pfId);
    const pf = simPortfolios.find(i => i.portfolio?.id === pfId);
    const pfCoins = pf?.states?.map(s => s.coin) || [];
    const activeCoin = pfCoins.includes(currentCoin) ? currentCoin : (pfCoins[0] || 'BTC');
    setCurrentCoin(activeCoin);
    await loadSimCoinDetail(activeCoin, pfId);
    await loadTickers();
    requestAnimationFrame(() => { window.scrollTo(0, scrollY); });
  };

  const switchCoin = async (coin: string, e?: React.MouseEvent) => {
    e?.preventDefault();
    const scrollY = window.scrollY;
    setCurrentCoin(coin);
    await loadSimCoinDetail(coin, currentPortfolioId);
    requestAnimationFrame(() => { window.scrollTo(0, scrollY); });
  };

  const toggleNotify = async (pfId: number, current: number) => {
    const newVal = current ? 0 : 1;
    await updatePortfolio(pfId, { notify_on_trade: newVal });
    await loadAll();
  };

  const handleTradeConfirm = async (amount: number) => {
    if (!simStatus || !showTradeModal) return;
    try {
      const tickerData = await getUpbitTicker();
      const ticker = tickerData.find(t => t.coin === currentCoin);
      const price = ticker?.current_price || simStatus.current_price || 0;
      const payload: { coin: string; action: string; price: number; reason: string; portfolio_id: number; amount?: number } = {
        coin: currentCoin, action: showTradeModal, price, reason: '수동', portfolio_id: currentPortfolioId,
      };
      if (amount > 0) payload.amount = amount;
      const result = await executeTrade(payload);
      if (!result.success) throw new Error(result.error || 'Trade failed');
      setShowTradeModal(null);
      await loadSimCoinDetail(currentCoin, currentPortfolioId);
      await loadTickers();
    } catch (err) {
      alert('오류: ' + (err instanceof Error ? err.message : 'Unknown'));
    }
  };

  const handleDeletePortfolio = async () => {
    if (!pfObj) return;
    if (!confirm(`"${pfObj.name}" 포트폴리오를 삭제하시겠습니까?`)) return;
    try {
      await fetch(`/api/v1/portfolios/${currentPortfolioId}`, { method: 'DELETE' });
      setCurrentPortfolioId(0);
      setCurrentCoin('BTC');
      await loadAll();
    } catch (e) { console.error(e); }
  };

  const priceMap: Record<string, number> = {};
  tickers.forEach(t => { priceMap[t.coin] = t.current_price; });

  return (
    <div style={{ padding: 16 }}>
      {/* Portfolio Summary Table */}
      <div style={{ marginBottom: 16 }}>
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13, userSelect: 'none' }}>
          <thead>
            <tr style={{ borderBottom: '1px solid #333', color: '#888' }}>
              <th style={{ padding: 8, textAlign: 'left' }}>포트폴리오</th>
              <th style={{ padding: 8, textAlign: 'right' }}>총 자산</th>
              <th style={{ padding: 8, textAlign: 'right' }}>수익률</th>
              <th style={{ padding: 8, textAlign: 'center' }}>포지션</th>
              <th style={{ padding: 8, textAlign: 'right' }}>현금/코인</th>
            </tr>
          </thead>
          <tbody>
            {simPortfolios.length === 0 ? (
              <tr><td colSpan={5} style={{ textAlign: 'center', color: '#555', padding: 16 }}>로딩 중...</td></tr>
            ) : simPortfolios.map(item => {
              const pf = item.portfolio;
              const itemStates = item.states || [];
              let totalAsset = 0, totalInitial = 0, totalCash = 0;
              let hasHolding = false;
              itemStates.forEach(s => {
                const price = priceMap[s.coin] || 0;
                totalAsset  += s.cash + s.units * price;
                totalInitial += s.initial_capital;
                totalCash   += s.cash;
                if (s.position === 'HOLDING') hasHolding = true;
              });
              const roi = totalInitial > 0 ? (totalAsset - totalInitial) / totalInitial * 100 : 0;
              const cashRatio = totalAsset > 0 ? (totalCash / totalAsset * 100).toFixed(0) : 0;
              const coinRatio = 100 - Number(cashRatio);
              const roiColor = roi > 0 ? '#2ecc71' : roi < 0 ? '#e74c3c' : '#aaa';
              const isActive = currentPortfolioId === pf?.id;
              return (
                <tr
                  key={pf?.id}
                  style={{ borderBottom: '1px solid #222', cursor: 'pointer', background: isActive ? '#1e2a3a' : 'transparent', userSelect: 'none' }}
                  onClick={() => switchPortfolio(pf!.id)}
                  onMouseDown={(e) => e.preventDefault()}
                >
                  <td style={{ padding: 8, fontWeight: isActive ? 'bold' : 'normal', color: isActive ? '#7ab3ef' : '#ccc' }}>{pf?.name || ''}</td>
                  <td style={{ padding: 8, textAlign: 'right' }}>{fmtMoney(totalAsset)}</td>
                  <td style={{ padding: 8, textAlign: 'right', color: roiColor }}>{roi > 0 ? '+' : ''}{roi.toFixed(2)}%</td>
                  <td style={{ padding: 8, textAlign: 'center' }}>
                    <span style={{ background: hasHolding ? '#b8860b' : '#555', color: '#fff', padding: '2px 6px', borderRadius: 4, fontSize: 11 }}>
                      {hasHolding ? 'HOLDING' : 'CASH'}
                    </span>
                  </td>
                  <td style={{ padding: 8, textAlign: 'right', color: '#aaa' }}>{cashRatio}% / {coinRatio}%</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>

      {/* Portfolio tabs */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 0, marginBottom: 0, flexWrap: 'wrap' }}>
        {simPortfolios.map(item => {
          const pf = item.portfolio;
          const isActive = currentPortfolioId === pf?.id;
          return (
            <button
              key={pf?.id}
              onClick={() => switchPortfolio(pf!.id)}
              onMouseDown={(e) => e.preventDefault()}
              style={{
                padding: '7px 18px', borderRadius: '6px 6px 0 0',
                border: `1px solid ${isActive ? '#2a2a4a' : '#222'}`,
                borderBottom: isActive ? '1px solid #12121f' : '1px solid #2a2a4a',
                background: isActive ? '#12121f' : 'transparent',
                color: isActive ? '#7ab3ef' : '#888',
                cursor: 'pointer', fontSize: 13, fontWeight: isActive ? 700 : 400,
                marginRight: 2, display: 'flex', alignItems: 'center', gap: 4,
              }}
            >
              {pf?.name || ''}
              <span
                onClick={(e) => {
                  e.stopPropagation();
                  toggleNotify(pf!.id, pf!.notify_on_trade ?? 1);
                }}
                title={pf?.notify_on_trade ? '알림 ON (클릭 시 OFF)' : '알림 OFF (클릭 시 ON)'}
                style={{ marginLeft: 4, fontSize: 11, cursor: 'pointer', opacity: 0.75 }}
              >
                {pf?.notify_on_trade ? '🔔' : '🔕'}
              </span>
            </button>
          );
        })}
        <button
          onClick={() => setShowAddModal(true)}
          style={{
            padding: '7px 14px', borderRadius: '6px 6px 0 0',
            border: '1px dashed #444', borderBottom: '1px solid #2a2a4a',
            background: 'transparent', color: '#666', cursor: 'pointer', fontSize: 13, marginLeft: 4,
          }}
          title="새 포트폴리오 추가"
        >＋</button>
      </div>

      {/* Portfolio content */}
      <div style={{ border: '1px solid #2a2a4a', borderTop: 'none', borderRadius: '0 0 8px 8px', padding: 16, background: '#12121f' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 14 }}>
          <h3 style={{ margin: 0, fontSize: 15, fontWeight: 700, color: '#e0e0e0' }}>{pfObj?.name || ''}</h3>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              onClick={() => pfObj && setEditPf({ id: pfObj.id, name: pfObj.name, desc: pfObj.description || '' })}
              style={{ padding: '5px 12px', borderRadius: 6, border: '1px solid #444', background: 'transparent', color: '#aaa', cursor: 'pointer', fontSize: 12 }}
            >✏️ 편집</button>
            <button
              onClick={handleDeletePortfolio}
              style={{ padding: '5px 12px', borderRadius: 6, border: '1px solid #5a2a2a', background: 'transparent', color: '#e57373', cursor: 'pointer', fontSize: 12 }}
            >🗑 삭제</button>
          </div>
        </div>

        <PortfolioROIChart portfolioId={currentPortfolioId} exchange="upbit" />

        {/* Coin tabs */}
        <div style={{ display: 'flex', gap: 0, marginBottom: 0, flexWrap: 'wrap' }}>
          {coins.map(c => {
            const isActive = currentCoin === c;
            return (
              <button
                key={c}
                onMouseDown={(e) => e.preventDefault()}
                onClick={(e) => switchCoin(c, e)}
                style={{
                  padding: '6px 18px', borderRadius: '6px 6px 0 0',
                  border: `1px solid ${isActive ? '#2a2a4a' : '#222'}`,
                  borderBottom: isActive ? '1px solid #0f0f1f' : '1px solid #2a2a4a',
                  background: isActive ? '#0f0f1f' : 'transparent',
                  color: isActive ? '#7ab3ef' : '#aaa',
                  cursor: 'pointer', fontSize: 13, fontWeight: isActive ? 700 : 400, marginRight: 2,
                }}
              >{c}</button>
            );
          })}
        </div>

        {/* Coin detail */}
        <div style={{ border: '1px solid #2a2a4a', borderTop: 'none', borderRadius: '0 0 6px 6px', padding: 14, background: '#0f0f1f', minHeight: 320 }}>
          {loading && <div style={{ textAlign: 'center', color: '#888', padding: '24px 0', fontSize: 14 }}>⏳ 로딩 중...</div>}
          {error && <div style={{ textAlign: 'center', color: '#ef5350', padding: '24px 0', fontSize: 14 }}>⚠️ 에러: {error}</div>}
          {!loading && !error && simStatus && (
            <>
              <UpbitAssetCard status={simStatus} coin={currentCoin} />
              <CoinPerformanceRow performance={performance} portfolioId={currentPortfolioId} coin={currentCoin} />
              <div style={{ display: 'flex', gap: 8, alignItems: 'center', flexWrap: 'wrap', marginBottom: 16 }}>
                <span style={{ fontSize: 12, color: '#888', minWidth: 60 }}>수동 매매</span>
                <button
                  onClick={() => setShowTradeModal('BUY')}
                  style={{ padding: '6px 20px', borderRadius: 6, border: '1px solid #26a69a', background: 'transparent', color: '#26a69a', cursor: 'pointer', fontSize: 13, fontWeight: 600 }}
                >▲ BUY</button>
                <button
                  onClick={() => setShowTradeModal('SELL')}
                  style={{ padding: '6px 20px', borderRadius: 6, border: '1px solid #ef5350', background: 'transparent', color: '#ef5350', cursor: 'pointer', fontSize: 13, fontWeight: 600 }}
                >▼ SELL</button>
              </div>
              <TradeHistory trades={simTrades} coin={currentCoin} />
            </>
          )}
          {!loading && !error && !simStatus && simPortfolios.length === 0 && (
            <div style={{ textAlign: 'center', color: '#555', padding: '40px 0', fontSize: 14 }}>
              업비트 포트폴리오가 없습니다. ＋ 버튼으로 추가하세요.
            </div>
          )}
        </div>
      </div>

      {/* Modals */}
      <AddPortfolioModal
        open={showAddModal}
        onClose={() => setShowAddModal(false)}
        onCreated={() => loadAll()}
        exchange="upbit"
        currency="₩"
        defaultCapital={100000}
      />
      {showTradeModal && simStatus && (
        <ManualTradeModal
          open={!!showTradeModal}
          action={showTradeModal}
          coin={currentCoin}
          currentPrice={simStatus.current_price}
          cash={simStatus.cash}
          units={simStatus.units}
          onConfirm={handleTradeConfirm}
          onClose={() => setShowTradeModal(null)}
        />
      )}
      <EditPortfolioModal
        open={!!editPf}
        portfolioId={editPf?.id ?? null}
        portfolioName={editPf?.name ?? ''}
        portfolioDesc={editPf?.desc ?? ''}
        portfolioNotifyOnTrade={simPortfolios.find(i => i.portfolio?.id === editPf?.id)?.portfolio?.notify_on_trade ?? 1}
        portfolioRiskLimitPct={simPortfolios.find(i => i.portfolio?.id === editPf?.id)?.portfolio?.risk_limit_pct ?? 15}
        portfolioCoins={simPortfolios.find(i => i.portfolio?.id === editPf?.id)?.states?.map(s => ({ coin: s.coin, initial_capital: s.initial_capital })) ?? []}
        onClose={() => setEditPf(null)}
        onSaved={async () => {
          setEditPf(null);
          await loadAll();
        }}
        currency="₩"
        defaultCapital={100000}
      />
    </div>
  );
};

// ── Sub-components ──────────────────────────────────────────────────────────

const fmtPct = (v: number | null, showSign = true): string => {
  if (v === null || v === undefined) return '—';
  const sign = showSign && v > 0 ? '+' : '';
  return `${sign}${v.toFixed(2)}%`;
};

const pctColor = (v: number | null): string => {
  if (v === null || v === undefined) return '#888';
  return v > 0 ? '#26a69a' : v < 0 ? '#ef5350' : '#888';
};

const CoinPerformanceRow: React.FC<{ performance: PortfolioPerformance[]; portfolioId: number; coin: string }> = ({ performance, portfolioId, coin }) => {
  const pfPerf = performance.find(p => p.portfolio_id === portfolioId);
  const coinPerf = pfPerf?.coins.find(c => c.coin === coin);
  if (!coinPerf) return null;

  const lifeDays = pfPerf?.max_period ?? 30;
  const periods: { label: string; ret: number | null; price: number | null }[] = [
    { label: '1일', ret: coinPerf.return_1d, price: coinPerf.price_change_1d },
    { label: '7일', ret: coinPerf.return_7d, price: coinPerf.price_change_7d },
    { label: '30일', ret: coinPerf.return_30d, price: coinPerf.price_change_30d },
    { label: `${lifeDays}일`, ret: coinPerf.return_life, price: coinPerf.price_change_life },
  ];

  return (
    <div style={{ display: 'flex', gap: 10, marginBottom: 16, flexWrap: 'wrap' }}>
      {periods.map(({ label, ret, price }) => (
        <div key={label} style={{ flex: 1, minWidth: 90, background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 8, padding: '10px 14px' }}>
          <div style={{ fontSize: 11, color: '#666', marginBottom: 6, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{label}</div>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', gap: 8 }}>
            <div>
              <div style={{ fontSize: 10, color: '#7ab3ef', marginBottom: 2 }}>수익률</div>
              <div style={{ fontSize: 15, fontWeight: 700, color: pctColor(ret) }}>{fmtPct(ret)}</div>
            </div>
            <div style={{ width: 1, height: 32, background: '#2a2a4a' }} />
            <div style={{ textAlign: 'right' }}>
              <div style={{ fontSize: 10, color: '#888', marginBottom: 2 }}>코인가격</div>
              <div style={{ fontSize: 15, fontWeight: 700, color: pctColor(price) }}>{fmtPct(price)}</div>
            </div>
          </div>
        </div>
      ))}
    </div>
  );
};

const UpbitAssetCard: React.FC<{ status: SimStatus; coin: string }> = ({ status, coin }) => {
  const { cash, units, current_price, initial_capital, avg_cost } = status;
  const coinVal  = units * current_price;
  const total    = cash + coinVal;
  const retPct   = initial_capital > 0 ? (total / initial_capital - 1) * 100 : 0;
  const retColor = retPct >= 0 ? '#26a69a' : '#ef5350';
  const retSign  = retPct >= 0 ? '+' : '';

  const cards = [
    {
      label: '💰 자산',
      value: fmtMoney(total),
      sub: `현금 ${fmtMoney(cash)} + 코인 ${fmtMoney(coinVal)}`,
      valueColor: '#e0e0e0',
    },
    {
      label: '💵 현금',
      value: fmtMoney(cash),
      sub: null,
      valueColor: '#e0e0e0',
    },
    {
      label: `🪙 ${coin}`,
      value: fmtUnits(units, coin),
      sub: `${fmtMoney(coinVal)}${avg_cost > 0 ? ` · 평단 ${fmtMoney(avg_cost)}` : ''}`,
      valueColor: '#e0e0e0',
    },
    {
      label: '📈 수익률',
      value: `${retSign}${retPct.toFixed(2)}%`,
      sub: `${fmtMoney(total)} / ${fmtMoney(initial_capital)}`,
      valueColor: retColor,
    },
  ];

  return (
    <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 16 }}>
      {cards.map(card => (
        <div key={card.label} style={{ flex: 1, minWidth: 130, background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 8, padding: '14px 18px' }}>
          <div style={{ fontSize: 12, color: '#888', marginBottom: 6, textTransform: 'uppercase', letterSpacing: '0.05em' }}>{card.label}</div>
          <div style={{ fontSize: 18, fontWeight: 700, color: card.valueColor }}>{card.value}</div>
          {card.sub && <div style={{ fontSize: 11, color: '#666', marginTop: 4 }}>{card.sub}</div>}
        </div>
      ))}
    </div>
  );
};

const TradeHistory: React.FC<{ trades: SimTrade[]; coin: string }> = ({ trades, coin }) => (
  <div>
    <div style={{ fontSize: 15, fontWeight: 600, color: '#ccc', marginBottom: 10 }}>거래 히스토리</div>
    <div style={{ overflowX: 'auto' }}>
      <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
        <thead>
          <tr>
            {['날짜', '액션', '가격', '수량', '수수료', '잔고 (거래 후)', '사유'].map(h => (
              <th key={h} style={{ background: '#16213e', color: '#888', textAlign: 'left', padding: '8px 12px', borderBottom: '1px solid #2a2a4a', textTransform: 'uppercase', fontSize: 11, letterSpacing: '0.05em' }}>{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {trades.length === 0 ? (
            <tr><td colSpan={7} style={{ textAlign: 'center', color: '#555', padding: '24px 0', fontSize: 14 }}>거래 내역 없음</td></tr>
          ) : trades.map(t => {
            const coinValAfter = t.units_after * t.price;
            const actionColor = t.action === 'BUY' ? '#26a69a' : t.action === 'SELL' ? '#ef5350' : '#888';
            return (
              <tr key={t.id}>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#ccc' }}>{fmtDatetime(t.executed_at)}</td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: actionColor, fontWeight: 700 }}>{t.action}</td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#ccc' }}>
                  {'₩' + t.price.toLocaleString('ko-KR', { maximumFractionDigits: 0 })}
                </td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#ccc' }}>{fmtUnits(t.units, coin)}</td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#e57373' }}>
                  {t.fee > 0 ? '₩' + t.fee.toLocaleString('ko-KR', { maximumFractionDigits: 0 }) : '—'}
                </td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#ccc' }}>
                  {t.units_after > 0 ? (
                    <>{fmtUnits(t.units_after, coin)}<br /><small style={{ color: '#aaa' }}>{'₩' + coinValAfter.toLocaleString('ko-KR', { maximumFractionDigits: 0 })}</small></>
                  ) : <span style={{ color: '#aaa' }}>0 {coin}</span>}
                </td>
                <td style={{ padding: '8px 12px', borderBottom: '1px solid #1e1e38', color: '#ccc' }}>{t.reason || '—'}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  </div>
);

export default UpbitSimulationPage;
