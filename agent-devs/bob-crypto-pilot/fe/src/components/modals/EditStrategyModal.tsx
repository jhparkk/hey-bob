import React, { useState, useEffect } from 'react';
import Modal, { Field, inputStyle, ModalFooter } from './Modal';
import type { Strategy } from '../../api';

interface Props {
  open: boolean;
  strategy: Strategy | null; // null = create
  onClose: () => void;
  onSave: (id: number | null, body: { name: string; description: string; notes: string }) => Promise<void>;
}

type MdTab = 'edit' | 'preview';

function renderMarkdown(text: string): string {
  if (!text) return '<span style="color:#555;">내용 없음</span>';
  const escaped = text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
  const lines = escaped.split('\n');
  const out: string[] = [];
  let inUl = false;
  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    line = line.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
    if (/^### /.test(line)) {
      if (inUl) { out.push('</ul>'); inUl = false; }
      out.push('<h3>' + line.slice(4) + '</h3>');
    } else if (/^## /.test(line)) {
      if (inUl) { out.push('</ul>'); inUl = false; }
      out.push('<h2>' + line.slice(3) + '</h2>');
    } else if (/^- /.test(line)) {
      if (!inUl) { out.push('<ul>'); inUl = true; }
      out.push('<li>' + line.slice(2) + '</li>');
    } else {
      if (inUl) { out.push('</ul>'); inUl = false; }
      out.push(line === '' ? '<br>' : line + '<br>');
    }
  }
  if (inUl) out.push('</ul>');
  return out.join('');
}

const EditStrategyModal: React.FC<Props> = ({ open, strategy, onClose, onSave }) => {
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [notes, setNotes] = useState('');
  const [mdTab, setMdTab] = useState<MdTab>('edit');
  const [status, setStatus] = useState('');
  const [isErr, setIsErr] = useState(false);

  useEffect(() => {
    if (open) {
      setName(strategy?.name ?? '');
      setDesc(strategy?.description ?? '');
      setNotes(strategy?.notes ?? '');
      setMdTab('edit');
      setStatus('');
    }
  }, [open, strategy]);

  const handleSave = async () => {
    if (!name.trim()) { setIsErr(true); setStatus('❌ 이름은 필수입니다.'); return; }
    try {
      await onSave(strategy?.id ?? null, { name: name.trim(), description: desc, notes });
      onClose();
    } catch (err: unknown) {
      setIsErr(true);
      setStatus('❌ ' + (err instanceof Error ? err.message : 'Error'));
    }
  };

  const tabBtn = (tab: MdTab, label: string) => (
    <button
      type="button"
      onClick={() => setMdTab(tab)}
      style={{
        background: 'transparent', border: 'none',
        borderBottom: `2px solid ${mdTab === tab ? '#26a69a' : 'transparent'}`,
        color: mdTab === tab ? '#26a69a' : '#888',
        fontSize: 12, fontWeight: 600, padding: '6px 16px', cursor: 'pointer',
        marginBottom: -1
      }}
    >{label}</button>
  );

  return (
    <Modal
      open={open}
      title={strategy ? '✏️ 전략 편집' : '➕ 새 전략 추가'}
      onClose={onClose}
    >
      <Field label="전략 이름">
        <input style={inputStyle} value={name} onChange={e => setName(e.target.value)} placeholder="예: 분할매수형" />
      </Field>
      <Field label="설명">
        <input style={inputStyle} value={desc} onChange={e => setDesc(e.target.value)} placeholder="예: 보수적 분할매수 전략" />
      </Field>
      <Field label="📝 분석 노트 (마크다운)">
        <div style={{ border: '1px solid #2a2a4a', borderRadius: 5, overflow: 'hidden' }}>
          <div style={{ display: 'flex', background: '#111827', borderBottom: '1px solid #2a2a4a' }}>
            {tabBtn('edit', '✏️ 편집')}
            {tabBtn('preview', '👁 미리보기')}
          </div>
          {mdTab === 'edit' ? (
            <textarea
              value={notes}
              onChange={e => setNotes(e.target.value)}
              rows={10}
              placeholder={"## 전략 설명\n\n### 매수 조건\n- RSI < 45\n\n### 매도 조건\n- RSI > 70"}
              style={{
                ...inputStyle, border: 'none', borderRadius: 0, resize: 'vertical',
                fontFamily: "'Courier New', monospace", fontSize: 12, padding: '10px 12px'
              }}
            />
          ) : (
            <div
              style={{
                background: '#1e2a3a', color: '#d0d8e8', padding: '12px 16px',
                minHeight: 180, fontSize: 13, lineHeight: 1.7, overflowY: 'auto'
              }}
              dangerouslySetInnerHTML={{ __html: renderMarkdown(notes) }}
            />
          )}
        </div>
      </Field>
      <ModalFooter
        onCancel={onClose}
        onSave={handleSave}
        statusMsg={status}
        statusOk={!isErr}
      />
    </Modal>
  );
};

export default EditStrategyModal;
