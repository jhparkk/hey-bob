import React, { useState, useEffect } from 'react';
import Modal from './Modal';
import { getStrategyVersions } from '../../api';
import type { StrategyVersion } from '../../api';

interface Props {
  open: boolean;
  strategyId: number | null;
  strategyName: string;
  currentVersion?: number;
  onClose: () => void;
}

function escHtml(s: string): string {
  return String(s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

const VersionHistoryModal: React.FC<Props> = ({ open, strategyId, strategyName, currentVersion, onClose }) => {
  const [versions, setVersions] = useState<StrategyVersion[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (open && strategyId) {
      setLoading(true);
      setError('');
      getStrategyVersions(strategyId)
        .then(data => { setVersions(data.versions || []); })
        .catch(err => setError(err.message))
        .finally(() => setLoading(false));
    }
  }, [open, strategyId]);

  return (
    <Modal open={open} title={`📋 버전 히스토리 — ${strategyName}`} onClose={onClose} maxWidth={700}>
      <div style={{ maxHeight: 480, overflowY: 'auto', fontSize: 13 }}>
        {loading && <div style={{ color: '#aaa', textAlign: 'center', padding: 16 }}>로딩 중...</div>}
        {error && <div style={{ color: '#ef5350', padding: 12 }}>오류: {error}</div>}
        {!loading && !error && versions.length === 0 && (
          <div style={{ color: '#aaa', textAlign: 'center', padding: 16 }}>
            버전 이력 없음 — 현재 v{currentVersion || 1}
          </div>
        )}
        {versions.map(v => (
          <div key={v.id} style={{ borderBottom: '1px solid #2a2a4a', padding: '12px 0' }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <span style={{ background: '#2d4a7a', color: '#7ab3ef', padding: '2px 8px', borderRadius: 4, fontSize: 12, fontWeight: 700 }}>
                v{v.version}
              </span>
              <span style={{ color: '#e0e0e0', fontWeight: 600 }}>{v.name}</span>
<span style={{ color: '#666', fontSize: 11, marginLeft: 'auto' }}>{v.changed_at}</span>
            </div>
            {v.description && (
              <div style={{ color: '#888', fontSize: 12, marginTop: 4 }}>{v.description}</div>
            )}
            {v.notes && (
              <details style={{ marginTop: 8 }}>
                <summary style={{ cursor: 'pointer', color: '#7ab3ef', fontSize: 12 }}>📝 노트 보기</summary>
                <div
                  style={{ marginTop: 8, background: '#0d1117', borderRadius: 6, padding: 10, fontSize: 12, color: '#ccc', whiteSpace: 'pre-wrap' }}
                  dangerouslySetInnerHTML={{ __html: escHtml(v.notes) }}
                />
              </details>
            )}
          </div>
        ))}
      </div>
    </Modal>
  );
};

export default VersionHistoryModal;
