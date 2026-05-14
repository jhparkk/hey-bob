import React from 'react';

interface ModalProps {
  open: boolean;
  title: string;
  titleColor?: string;
  maxWidth?: number;
  onClose: () => void;
  children: React.ReactNode;
}

const Modal: React.FC<ModalProps> = ({
  open, title, titleColor = '#26a69a', maxWidth = 600, onClose, children
}) => {
  if (!open) return null;
  return (
    <div
      style={{
        position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.75)',
        zIndex: 9999, overflowY: 'auto'
      }}
    >
      <div style={{
        maxWidth, margin: '40px auto', background: '#1a1a2e',
        border: '1px solid #3a3a7a', borderRadius: 12, padding: 24
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 20 }}>
          <div style={{ fontSize: 17, fontWeight: 700, color: titleColor }}>{title}</div>
          <button
            onClick={onClose}
            style={{ background: 'none', border: 'none', color: '#aaa', fontSize: 22, cursor: 'pointer' }}
          >✕</button>
        </div>
        {children}
      </div>
    </div>
  );
};

export default Modal;

// Field wrapper
export const Field: React.FC<{ label: string; children: React.ReactNode; style?: React.CSSProperties }> = ({
  label, children, style
}) => (
  <div style={{ marginBottom: 12, ...style }}>
    <label style={{ display: 'block', fontSize: 11, color: '#888', textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: 4 }}>
      {label}
    </label>
    {children}
  </div>
);

export const inputStyle: React.CSSProperties = {
  width: '100%',
  background: '#16213e',
  border: '1px solid #2a2a4a',
  borderRadius: 5,
  color: '#e0e0e0',
  padding: '7px 10px',
  fontSize: 13,
  boxSizing: 'border-box',
  outline: 'none',
};

export const ModalFooter: React.FC<{
  onCancel: () => void;
  onSave: () => void;
  saveLabel?: string;
  statusMsg?: string;
  statusOk?: boolean;
}> = ({ onCancel, onSave, saveLabel = '💾 저장', statusMsg, statusOk = true }) => (
  <>
    {statusMsg && (
      <div style={{ color: statusOk ? '#26a69a' : '#ef5350', fontSize: 13, textAlign: 'center', minHeight: 18, marginTop: 8 }}>
        {statusMsg}
      </div>
    )}
    <div style={{ display: 'flex', gap: 10, justifyContent: 'flex-end', marginTop: 16 }}>
      <button
        onClick={onCancel}
        style={{ padding: '8px 16px', background: '#2a2a4a', border: '1px solid #5a5a9a', borderRadius: 6, color: '#ccc', fontSize: 14, cursor: 'pointer' }}
      >취소</button>
      <button
        onClick={onSave}
        style={{ padding: '8px 20px', background: '#26a69a', border: 'none', borderRadius: 6, color: '#fff', fontSize: 14, fontWeight: 700, cursor: 'pointer' }}
      >{saveLabel}</button>
    </div>
  </>
);
