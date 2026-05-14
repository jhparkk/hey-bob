import React, { useEffect, useState } from 'react';
import { useStrategy } from '../hooks/useStrategy';
import { getPortfolioStrategies, getSimPortfolios } from '../api';
import type { Strategy } from '../api';
import EditStrategyModal from '../components/modals/EditStrategyModal';
import AddPortfolioModal from '../components/modals/AddPortfolioModal';
import ChangeStrategyModal from '../components/modals/ChangeStrategyModal';
import VersionHistoryModal from '../components/modals/VersionHistoryModal';
import StrategyHistoryModal from '../components/modals/StrategyHistoryModal';
import PortfolioStrategyHistoryModal from '../components/modals/PortfolioStrategyHistoryModal';

function escHtml(s: string): string {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

// ── Strategy Card ─────────────────────────────────────────────────────────

interface StrategyCardProps {
  s: Strategy;
  onEdit: () => void;
  onDelete: () => void;
  onVersionHistory: () => void;
}

const StrategyCardItem: React.FC<StrategyCardProps> = ({ s, onEdit, onDelete, onVersionHistory }) => (
  <div style={{ background: '#16213e', border: '2px solid #2a2a4a', borderRadius: 8, padding: '10px 14px' }}>
    <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', gap: 8 }}>
      <div style={{ flex: 1, minWidth: 0 }}>
        <div style={{ fontSize: 15, fontWeight: 700, color: '#e0e0e0', display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
          {escHtml(s.name)}
          <span style={{ background: '#2d4a7a', color: '#7ab3ef', padding: '2px 6px', borderRadius: 4, fontSize: 11, fontWeight: 600 }}>
            v{s.version || 1}
          </span>
        </div>
        <div style={{ fontSize: 12, color: '#666', marginTop: 3 }}>{s.description}</div>
      </div>
      <div style={{ display: 'flex', gap: 4, flexShrink: 0 }}>
        <button
          onClick={onVersionHistory} onMouseDown={(e) => e.preventDefault()}
          style={{ padding: '4px 10px', background: '#3a3a6a', border: '1px solid #5a5a9a', borderRadius: 5, color: '#ccc', fontSize: 12, cursor: 'pointer' }}
        >📋 버전이력</button>
        <button
          onClick={onEdit} onMouseDown={(e) => e.preventDefault()}
          style={{ padding: '4px 10px', background: '#3a3a6a', border: '1px solid #5a5a9a', borderRadius: 5, color: '#ccc', fontSize: 12, cursor: 'pointer' }}
        >✏️ 편집</button>
        <button
          onClick={onDelete} onMouseDown={(e) => e.preventDefault()}
          style={{ padding: '4px 10px', background: '#3a1a1a', border: '1px solid #7a3a3a', borderRadius: 5, color: '#e57373', fontSize: 12, cursor: 'pointer' }}
        >🗑 삭제</button>
      </div>
    </div>
  </div>
);

const StrategyPage: React.FC = () => {
  const {
    strategies, portfolios, loadAll, loadStrategies, loadPortfolios,
    saveStrategy, removeStrategy, changePortfolioStrategy,
  } = useStrategy();

  // Modal state
  const [editStrat, setEditStrat] = useState<Strategy | null | 'new'>('new'); // null = closed, 'new' = create
  const [editStratOpen, setEditStratOpen] = useState(false);

  const [addPfOpen, setAddPfOpen] = useState(false);
  const [histAllOpen, setHistAllOpen] = useState(false);

  const [versionModal, setVersionModal] = useState<{ open: boolean; id: number; name: string; version?: number }>({ open: false, id: 0, name: '' });

  const [changeStratModal, setChangeStratModal] = useState<{
    open: boolean; portfolioId: number; portfolioName: string; coin: string; currentStratId?: number
  }>({ open: false, portfolioId: 0, portfolioName: '', coin: '' });

  const [pshModal, setPshModal] = useState<{ open: boolean; portfolioId: number | null; coin: string }>({ open: false, portfolioId: null, coin: '' });

  // Portfolio mapping section
  const [selectedPfId, setSelectedPfId] = useState<number>(0);
  const [pfMapping, setPfMapping] = useState<{
    coin: string; stratName: string; version?: number; stratId?: number
  }[]>([]);
  const [pfMappingLoading, setPfMappingLoading] = useState(false);

  useEffect(() => { loadAll(); }, [loadAll]);

  // Load portfolio mapping when dropdown changes
  useEffect(() => {
    if (!selectedPfId) { setPfMapping([]); return; }
    setPfMappingLoading(true);
    Promise.all([
      getSimPortfolios(),
      getPortfolioStrategies(selectedPfId),
    ]).then(([simData, stratData]) => {
      const pfItem = (simData.portfolios || []).find(i => i.portfolio?.id === selectedPfId);
      const pfCoins = pfItem ? pfItem.states.map(s => s.coin) : ['BTC', 'ETH'];
      const rows = pfCoins.map(coin => {
        const coinData = stratData[coin] || {};
        const ps = coinData.portfolio_strategy;
        const stratName = coinData.strategy_name || (ps?.strategy_id ? `#${ps.strategy_id}` : '없음');
        const strat = ps?.strategy_id ? strategies.find(s => s.id === ps.strategy_id) : null;
        return { coin, stratName, version: strat?.version, stratId: ps?.strategy_id };
      });
      setPfMapping(rows);
    }).catch(console.error).finally(() => setPfMappingLoading(false));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selectedPfId, strategies]);

  // Auto-select first portfolio
  useEffect(() => {
    if (!selectedPfId && portfolios.length > 0) setSelectedPfId(portfolios[0].id);
  }, [portfolios, selectedPfId]);

  const openEditStrategy = (s: Strategy) => {
    setEditStrat(s);
    setEditStratOpen(true);
  };
  const openNewStrategy = () => {
    setEditStrat(null);
    setEditStratOpen(true);
  };

  const handleSaveStrategy = async (
    id: number | null,
    body: { name: string; description: string; notes: string }
  ) => {
    await saveStrategy(id, body);
  };

  const handleDeleteStrategy = async (id: number, name: string) => {
    if (!confirm(`"${name}" 전략을 삭제하시겠습니까?`)) return;
    try { await removeStrategy(id); } catch (err: unknown) { alert('삭제 실패: ' + (err instanceof Error ? err.message : 'Error')); }
  };

  const handleChangeStrategy = async (portfolioId: number, coin: string, strategyId: number) => {
    await changePortfolioStrategy(portfolioId, coin, strategyId);
    // Refresh mapping
    setSelectedPfId(prev => { setTimeout(() => setSelectedPfId(prev), 10); return 0; });
  };

  return (
    <div style={{ padding: 16 }}>

      {/* ── Strategy Library ── */}
      <div style={{ marginBottom: 32 }}>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14, padding: '12px 16px', background: '#16213e', borderRadius: 10, border: '1px solid #2a2a4a' }}>
          <h3 style={{ fontSize: 17, fontWeight: 700, color: '#e0e0e0', margin: 0 }}>📚 전략 라이브러리</h3>
          <div style={{ display: 'flex', gap: 8 }}>
            <button
              onClick={openNewStrategy}
              style={{ padding: '6px 14px', background: '#26a69a', border: 'none', borderRadius: 6, color: '#fff', fontSize: 13, fontWeight: 600, cursor: 'pointer' }}
            >＋ 전략 추가</button>
            <button
              onClick={() => setHistAllOpen(true)}
              style={{ padding: '6px 14px', background: '#2a2a4a', border: '1px solid #5a5a9a', borderRadius: 6, color: '#ccc', fontSize: 13, cursor: 'pointer' }}
            >📜 변경이력</button>
          </div>
        </div>

        {strategies.length === 0 ? (
          <div style={{ color: '#555', fontSize: 13, padding: 12, textAlign: 'center' }}>전략 없음</div>
        ) : (
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            {strategies.map(s => (
              <StrategyCardItem
                key={s.id}
                s={s}
                onEdit={() => openEditStrategy(s)}
                onDelete={() => handleDeleteStrategy(s.id, s.name)}
                onVersionHistory={() => setVersionModal({ open: true, id: s.id, name: s.name, version: s.version })}
              />
            ))}
          </div>
        )}
      </div>

      {/* ── Portfolio Mapping ── */}
      <div>
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 14, padding: '12px 16px', background: '#16213e', borderRadius: 10, border: '1px solid #2a2a4a' }}>
          <h3 style={{ fontSize: 17, fontWeight: 700, color: '#e0e0e0', margin: 0 }}>🗂 포트폴리오 전략 매핑</h3>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <select
              value={selectedPfId}
              onChange={e => setSelectedPfId(parseInt(e.target.value, 10))}
              style={{ background: '#16213e', border: '1px solid #2a2a4a', borderRadius: 6, color: '#e0e0e0', padding: '5px 10px', fontSize: 13 }}
            >
              <option value={0}>포트폴리오 선택...</option>
              {portfolios.map(pf => <option key={pf.id} value={pf.id}>{pf.name}</option>)}
            </select>
            <button
              onClick={() => setAddPfOpen(true)}
              style={{ padding: '6px 14px', background: '#2a2a4a', border: '1px solid #5a5a9a', borderRadius: 6, color: '#ccc', fontSize: 13, cursor: 'pointer' }}
            >＋ 포트폴리오</button>
          </div>
        </div>

        {!selectedPfId ? (
          <div style={{ color: '#555', fontSize: 13, padding: 12, textAlign: 'center' }}>포트폴리오를 선택하세요</div>
        ) : pfMappingLoading ? (
          <div style={{ color: '#555', fontSize: 13, padding: 12, textAlign: 'center' }}>로딩 중...</div>
        ) : (
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 13 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #2a2a4a' }}>
                <th style={{ padding: '8px 12px', textAlign: 'left', color: '#666', fontWeight: 500 }}>코인</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', color: '#666', fontWeight: 500 }}>현재 전략</th>
                <th style={{ padding: '8px 12px', textAlign: 'left', color: '#666', fontWeight: 500 }}>액션</th>
              </tr>
            </thead>
            <tbody>
              {pfMapping.map(row => {
                const currentPf = portfolios.find(p => p.id === selectedPfId);
                return (
                  <tr key={row.coin}>
                    <td style={{ padding: '10px 12px', color: '#7ab3ef', fontWeight: 600 }}>{row.coin}</td>
                    <td style={{ padding: '10px 12px', color: '#e0e0e0' }}>
                      {row.stratName}
                      {row.version !== undefined && (
                        <span style={{ background: '#2d4a7a', color: '#7ab3ef', padding: '2px 6px', borderRadius: 4, fontSize: 11, marginLeft: 6 }}>
                          v{row.version}
                        </span>
                      )}
                    </td>
                    <td style={{ padding: '10px 12px' }}>
                      <button
                        onClick={() => setChangeStratModal({
                          open: true, portfolioId: selectedPfId,
                          portfolioName: currentPf?.name || '', coin: row.coin,
                          currentStratId: row.stratId
                        })}
                        style={{ padding: '4px 10px', borderRadius: 5, border: '1px solid #444', background: '#1e2a3a', color: '#7ab3ef', cursor: 'pointer', fontSize: 12, marginRight: 6 }}
                      >변경</button>
                      <button
                        onClick={() => setPshModal({ open: true, portfolioId: selectedPfId, coin: row.coin })}
                        style={{ padding: '4px 10px', borderRadius: 5, border: '1px solid #444', background: '#1e2a3a', color: '#9a9ada', cursor: 'pointer', fontSize: 12 }}
                      >이력</button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        )}
      </div>

      {/* Modals */}
      <EditStrategyModal
        open={editStratOpen}
        strategy={editStrat === 'new' ? null : editStrat}
        onClose={() => { setEditStratOpen(false); setEditStrat('new'); }}
        onSave={async (id, body) => { await handleSaveStrategy(id, body); loadStrategies(); }}
      />
      <AddPortfolioModal
        open={addPfOpen}
        onClose={() => setAddPfOpen(false)}
        onCreated={() => loadPortfolios()}
      />
      <StrategyHistoryModal
        open={histAllOpen}
        onClose={() => setHistAllOpen(false)}
      />
      <VersionHistoryModal
        open={versionModal.open}
        strategyId={versionModal.id}
        strategyName={versionModal.name}
        currentVersion={versionModal.version}
        onClose={() => setVersionModal(prev => ({ ...prev, open: false }))}
      />
      <ChangeStrategyModal
        open={changeStratModal.open}
        portfolioId={changeStratModal.portfolioId}
        portfolioName={changeStratModal.portfolioName}
        coin={changeStratModal.coin}
        strategies={strategies}
        currentStrategyId={changeStratModal.currentStratId}
        onClose={() => setChangeStratModal(prev => ({ ...prev, open: false }))}
        onSave={handleChangeStrategy}
      />
      <PortfolioStrategyHistoryModal
        open={pshModal.open}
        portfolioId={pshModal.portfolioId}
        coin={pshModal.coin}
        onClose={() => setPshModal(prev => ({ ...prev, open: false }))}
      />
    </div>
  );
};

export default StrategyPage;
