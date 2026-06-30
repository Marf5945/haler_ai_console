// AlbumGallery.jsx — 「紀念照」相冊前端元件。
//
// 兩個對外元件：
//   <ConfirmKeepsakeCard scene contextDigest onSaved onDismiss />
//       聊天流程中收到 NeedsUser 的紀念照待確認卡時渲染；使用者按「拍下這一刻」
//       會呼叫 ConfirmCommemorativePhoto 產圖並落地相冊。
//   <AlbumGallery />
//       相冊主畫面：縮圖牆 + 說明編輯 + 刪除 + ComfyUI 設定。
//
// 後端綁定（由 wails 產生於 ../../wailsjs/go/main/App，build 後可用）：
//   ListAlbumPhotos / AlbumPhotoImage / ConfirmCommemorativePhoto /
//   SetAlbumPhotoCaption / DeleteAlbumPhoto / GetKeepsakeConfig / SaveKeepsakeConfig
import React, { useState, useEffect, useCallback } from 'react';
import { callWails } from '../../lib/callWails';
import {
  ListAlbumPhotos,
  AlbumPhotoImage,
  ConfirmCommemorativePhoto,
  SetAlbumPhotoCaption,
  DeleteAlbumPhoto,
  GetKeepsakeConfig,
  SaveKeepsakeConfig,
} from '../../wailsjs/go/main/App';

const s = {
  card: { border: '1px solid var(--border, #333)', borderRadius: 12, padding: 16, background: 'var(--panel, #1b1b1f)' },
  btn: { padding: '6px 14px', borderRadius: 8, border: '1px solid var(--border, #444)', cursor: 'pointer', background: 'transparent', color: 'inherit' },
  btnPrimary: { padding: '8px 18px', borderRadius: 8, border: 'none', cursor: 'pointer', background: 'var(--accent, #6b8afd)', color: '#fff', fontWeight: 600 },
  grid: { display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))', gap: 14 },
  thumb: { width: '100%', borderRadius: 10, display: 'block', aspectRatio: '3 / 4', objectFit: 'cover', background: '#0008' },
  meta: { fontSize: 12, opacity: 0.75, marginTop: 6 },
  input: { width: '100%', padding: '6px 8px', borderRadius: 6, border: '1px solid var(--border, #444)', background: 'transparent', color: 'inherit' },
};

// ── 聊天流程：待確認卡 ──
export function ConfirmKeepsakeCard({ scene = '', contextDigest = '', personaName = '', onSaved, onDismiss }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');

  const confirm = useCallback(async () => {
    setBusy(true);
    setError('');
    try {
      const photo = await callWails(() => ConfirmCommemorativePhoto(scene, contextDigest));
      onSaved && onSaved(photo);
    } catch (e) {
      setError(String(e && e.message ? e.message : e));
    } finally {
      setBusy(false);
    }
  }, [scene, contextDigest, onSaved]);

  return (
    <div style={{ ...s.card, maxWidth: 420 }}>
      <div style={{ fontWeight: 600, marginBottom: 6 }}>📸 紀念照</div>
      <div style={{ opacity: 0.85, marginBottom: 12 }}>
        要不要把這一刻拍下來當紀念？{personaName ? `${personaName}想和你留一張` : ''}
        {scene ? `：「${scene}」` : ''}。
      </div>
      {error && <div style={{ color: '#ff6b6b', marginBottom: 10, fontSize: 13 }}>產圖失敗：{error}</div>}
      <div style={{ display: 'flex', gap: 10 }}>
        <button style={s.btnPrimary} disabled={busy} onClick={confirm}>
          {busy ? '正在拍照…' : '拍下這一刻'}
        </button>
        <button style={s.btn} disabled={busy} onClick={() => onDismiss && onDismiss()}>下次吧</button>
      </div>
    </div>
  );
}

// ── 相冊縮圖（lazy 載圖）──
function PhotoTile({ photo, onChanged }) {
  const [src, setSrc] = useState('');
  const [editing, setEditing] = useState(false);
  const [caption, setCaption] = useState(photo.caption || '');

  useEffect(() => {
    let alive = true;
    callWails(() => AlbumPhotoImage(photo.id))
      .then((dataURL) => { if (alive) setSrc(dataURL); })
      .catch(() => {});
    return () => { alive = false; };
  }, [photo.id]);

  const saveCaption = async () => {
    await callWails(() => SetAlbumPhotoCaption(photo.id, caption)).catch(() => {});
    setEditing(false);
    onChanged && onChanged();
  };
  const remove = async () => {
    if (!window.confirm('刪除這張紀念照？此動作無法復原。')) return;
    await callWails(() => DeleteAlbumPhoto(photo.id)).catch(() => {});
    onChanged && onChanged();
  };

  return (
    <div style={s.card}>
      {src
        ? <img src={src} alt={photo.scene} style={s.thumb} />
        : <div style={{ ...s.thumb, display: 'flex', alignItems: 'center', justifyContent: 'center', opacity: 0.4 }}>載入中…</div>}
      <div style={{ marginTop: 8 }}>
        {editing ? (
          <div style={{ display: 'flex', gap: 6 }}>
            <input style={s.input} value={caption} onChange={(e) => setCaption(e.target.value)}
              placeholder="寫句話留念…" autoFocus />
            <button style={s.btn} onClick={saveCaption}>存</button>
          </div>
        ) : (
          <div onClick={() => setEditing(true)} style={{ cursor: 'text', minHeight: 20 }}>
            {photo.caption || <span style={{ opacity: 0.45 }}>＋ 加說明</span>}
          </div>
        )}
      </div>
      <div style={s.meta}>{photo.scene}</div>
      <div style={s.meta}>
        {photo.personaName} · {(photo.createdAt || '').slice(0, 10)}
        <button style={{ ...s.btn, float: 'right', padding: '2px 8px', fontSize: 11 }} onClick={remove}>刪除</button>
      </div>
    </div>
  );
}

// ── ComfyUI 設定 ──
function KeepsakeSettings({ onClose }) {
  const [cfg, setCfg] = useState(null);
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    callWails(() => GetKeepsakeConfig()).then(setCfg).catch(() => setCfg({}));
  }, []);
  if (!cfg) return null;

  const field = (key, label, placeholder) => (
    <label style={{ display: 'block', marginBottom: 10 }}>
      <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>{label}</div>
      <input style={s.input} value={cfg[key] ?? ''} placeholder={placeholder}
        onChange={(e) => { setCfg({ ...cfg, [key]: e.target.value }); setSaved(false); }} />
    </label>
  );
  const numField = (key, label) => (
    <label style={{ display: 'block', marginBottom: 10 }}>
      <div style={{ fontSize: 12, opacity: 0.7, marginBottom: 4 }}>{label}</div>
      <input style={s.input} type="number" value={cfg[key] ?? 0}
        onChange={(e) => { setCfg({ ...cfg, [key]: Number(e.target.value) }); setSaved(false); }} />
    </label>
  );

  const save = async () => {
    await callWails(() => SaveKeepsakeConfig(cfg)).catch(() => {});
    setSaved(true);
  };

  return (
    <div style={{ ...s.card, marginBottom: 16 }}>
      <div style={{ fontWeight: 600, marginBottom: 12 }}>紀念照設定（本機 ComfyUI）</div>
      {field('comfyui_url', 'ComfyUI 位址', 'http://127.0.0.1:8188')}
      {field('checkpoint', '模型 checkpoint（必填）', 'anything-v5.safetensors')}
      {field('style_preset', '畫風前綴（可空）', 'masterpiece, anime style…')}
      <div style={{ display: 'flex', gap: 10 }}>
        {numField('width', '寬')}
        {numField('height', '高')}
        {numField('steps', '步數')}
      </div>
      <div style={{ display: 'flex', gap: 10, marginTop: 6 }}>
        <button style={s.btnPrimary} onClick={save}>{saved ? '已儲存 ✓' : '儲存設定'}</button>
        <button style={s.btn} onClick={onClose}>關閉</button>
      </div>
    </div>
  );
}

// ── 相冊主畫面 ──
export default function AlbumGallery() {
  const [photos, setPhotos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showSettings, setShowSettings] = useState(false);

  const reload = useCallback(() => {
    setLoading(true);
    callWails(() => ListAlbumPhotos())
      .then((list) => setPhotos(list || []))
      .catch(() => setPhotos([]))
      .finally(() => setLoading(false));
  }, []);
  useEffect(() => { reload(); }, [reload]);

  return (
    <div style={{ padding: 16 }}>
      <div style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
        <h2 style={{ margin: 0, fontSize: 18 }}>📸 紀念相冊</h2>
        <div style={{ flex: 1 }} />
        <button style={s.btn} onClick={() => setShowSettings((v) => !v)}>設定</button>
        <button style={{ ...s.btn, marginLeft: 8 }} onClick={reload}>重新整理</button>
      </div>

      {showSettings && <KeepsakeSettings onClose={() => setShowSettings(false)} />}

      {loading ? (
        <div style={{ opacity: 0.6 }}>載入中…</div>
      ) : photos.length === 0 ? (
        <div style={{ opacity: 0.6, padding: '40px 0', textAlign: 'center' }}>
          還沒有紀念照。聊一段時間，{`{角色}`}會主動提議拍一張。
        </div>
      ) : (
        <div style={s.grid}>
          {photos.map((p) => <PhotoTile key={p.id} photo={p} onChanged={reload} />)}
        </div>
      )}
    </div>
  );
}
