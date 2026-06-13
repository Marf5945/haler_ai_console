import {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import useI18n from '../../locales/useI18n';

export default function AvatarUploadModal({persona, onApply, onClose}) {
  const t = useI18n(s => s.t);
  const fileInputRef = useRef(null);
  const [selectedFile, setSelectedFile] = useState(null);
  const [previewSrc, setPreviewSrc] = useState('');
  const [dragActive, setDragActive] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    // The modal owns temporary preview URLs so pasted or dropped images can be
    // inspected before any data is sent to the backend.
    if (!selectedFile) {
      setPreviewSrc('');
      return undefined;
    }
    const url = URL.createObjectURL(selectedFile);
    setPreviewSrc(url);
    return () => URL.revokeObjectURL(url);
  }, [selectedFile]);

  useEffect(() => {
    // Clipboard paste is scoped to this modal's lifetime; closing the dialog
    // removes the listener and avoids surprising paste behavior elsewhere.
    function handlePaste(event) {
      const file = Array.from(event.clipboardData?.files || []).find((item) => item.type?.startsWith('image/'));
      if (file) pickFile(file);
    }
    window.addEventListener('paste', handlePaste);
    return () => window.removeEventListener('paste', handlePaste);
  }, []);

  function pickFile(file) {
    if (!file?.type?.startsWith('image/')) {
      setError(t('persona.uploadImageError'));
      return;
    }
    setError('');
    setSelectedFile(file);
  }

  function handleDrop(event) {
    event.preventDefault();
    setDragActive(false);
    const file = event.dataTransfer.files?.[0];
    if (file) pickFile(file);
  }

  return createPortal(
    <div className="avatar-upload-backdrop" role="dialog" aria-modal="true" aria-label={t('persona.uploadTitle')}>
      <section className="avatar-upload-modal">
        <header>
          <span>{t('persona.uploadPersonaLabel', { name: persona?.name || t('persona.defaultName') })}</span>
          <button type="button" onClick={onClose}>×</button>
        </header>
        <button
          className={`avatar-drop-zone ${dragActive ? 'avatar-drop-zone-active' : ''}`}
          type="button"
          onClick={() => fileInputRef.current?.click()}
          onDragEnter={(event) => { event.preventDefault(); setDragActive(true); }}
          onDragOver={(event) => event.preventDefault()}
          onDragLeave={() => setDragActive(false)}
          onDrop={handleDrop}
        >
          {previewSrc ? (
            <img src={previewSrc} alt={t('persona.previewAlt')}/>
          ) : (
            <>
              <strong>{t('persona.dropImage')}</strong>
              <span>{t('persona.dropHint')}</span>
            </>
          )}
        </button>
        <input
          ref={fileInputRef}
          type="file"
          accept="image/png,image/jpeg,image/webp"
          hidden
          onChange={(event) => {
            const file = event.target.files?.[0];
            if (file) pickFile(file);
          }}
        />
        {error && <p className="avatar-upload-error">{error}</p>}
        <footer>
          <button type="button" onClick={() => fileInputRef.current?.click()}>{t('persona.selectFile')}</button>
          <button type="button" onClick={() => { setSelectedFile(null); setError(''); }}>{t('persona.rePaste')}</button>
          <button type="button" onClick={onClose}>{t('common.cancel')}</button>
          <button type="button" disabled={!selectedFile} onClick={() => selectedFile && onApply(selectedFile)}>{t('persona.apply')}</button>
        </footer>
      </section>
    </div>,
    document.body
  );
}
