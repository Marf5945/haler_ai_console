// EmbeddingPickerModal.jsx — Phase B M3：第一次拖檔時開啟的 embedding model picker。
//
// 行為：
//   - 區塊 A：本機已裝的 embed model 清單（GetInstalledEmbedModels）
//   - 區塊 B：「下載新 model」展開區，輸入框 + 常用建議 chip + 進度條
//   - 區塊 C：「跳過 → 用 TF-IDF」（DismissEmbeddingPicker）
//
// 設計原則：
//   - 單一元件、單一檔案；不再分子 component。
//   - 進度經 EventsOn 監聽 embedding:pull_progress；jobID match 才更新本 modal 顯示。
//   - 不需要 jest test（純 UI），整體靠手動驗證。
import React, {useEffect, useState} from 'react';
import {
  ListInstalledEmbedModels,
  PullEmbedModel,
  SetEmbeddingConfig,
  DismissEmbeddingPicker,
  DetectOllamaState,
  WakeOllamaDaemon,
} from '../../wailsjs/go/main/App';
import {EventsOn} from '../../wailsjs/runtime/runtime';

// 推薦下載清單——使用者可改成自己想要的。
const SUGGESTED_MODELS = [
  'nomic-embed-text',
  'mxbai-embed-large',
  'bge-m3',
];

export default function EmbeddingPickerModal({displayName, onClose}) {
  const [installed, setInstalled] = useState([]);
  const [ollamaState, setOllamaState] = useState(null); // {binaryFound, daemonRunning}
  const [waking, setWaking] = useState(false);
  const [showDownload, setShowDownload] = useState(false);
  const [downloadInput, setDownloadInput] = useState('nomic-embed-text');
  const [pullJob, setPullJob] = useState(null); // {jobId, modelId}
  const [progress, setProgress] = useState(null); // {percent, status}
  const [errorMsg, setErrorMsg] = useState('');

  // mount：載入已裝清單 + 偵測 ollama 狀態
  const refreshState = async () => {
    try {
      const [list, state] = await Promise.all([
        ListInstalledEmbedModels().catch(() => []),
        DetectOllamaState().catch(() => null),
      ]);
      setInstalled(Array.isArray(list) ? list : []);
      setOllamaState(state);
    } catch (_) { /* silent */ }
  };
  useEffect(() => {
    let cancelled = false;
    (async () => {
      await refreshState();
      if (cancelled) setInstalled([]);
    })();
    return () => { cancelled = true; };
  }, []);

  // 喚醒 Ollama daemon，然後重新抓 list/state
  const handleWakeOllama = async () => {
    setWaking(true);
    setErrorMsg("");
    try {
      await WakeOllamaDaemon();
      await refreshState();
    } catch (err) {
      setErrorMsg(String(err?.message || err));
    } finally {
      setWaking(false);
    }
  };

  // event subscribe：pull progress / done / failed
  useEffect(() => {
    if (!pullJob) return;
    const offProg = EventsOn('embedding:pull_progress', (payload) => {
      if (!payload || payload.jobId !== pullJob.jobId) return;
      setProgress({percent: payload.percent, status: payload.status || ''});
    });
    const offDone = EventsOn('embedding:pull_done', async (payload) => {
      if (!payload || payload.jobId !== pullJob.jobId) return;
      setProgress({percent: 100, status: '完成'});
      // 自動套用為當前 embedding model
      try { await SetEmbeddingConfig('ollama', pullJob.modelId); } catch (_) {}
      onClose?.({applied: true, modelID: pullJob.modelId});
    });
    const offFail = EventsOn('embedding:pull_failed', (payload) => {
      if (!payload || payload.jobId !== pullJob.jobId) return;
      setErrorMsg(payload.error || '下載失敗');
      setPullJob(null);
      setProgress(null);
    });
    return () => {
      offProg && offProg();
      offDone && offDone();
      offFail && offFail();
    };
  }, [pullJob, onClose]);

  const handlePickInstalled = async (modelID) => {
    try {
      await SetEmbeddingConfig('ollama', modelID);
      onClose?.({applied: true, modelID});
    } catch (err) {
      setErrorMsg(String(err?.message || err));
    }
  };

  const handleStartPull = async () => {
    const m = (downloadInput || '').trim();
    if (!m) return;
    setErrorMsg('');
    try {
      const job = await PullEmbedModel(m);
      setPullJob(job);
      setProgress({percent: 0, status: '啟動下載'});
    } catch (err) {
      setErrorMsg(String(err?.message || err));
    }
  };

  const handleDismiss = async () => {
    try { await DismissEmbeddingPicker(); } catch (_) {}
    onClose?.({applied: false, skipped: true});
  };

  const pulling = !!pullJob;

  return (
    <div className="embedding-picker-overlay" onMouseDown={(e) => {
      if (e.target.classList?.contains('embedding-picker-overlay')) handleDismiss();
    }}>
      <div className="embedding-picker">
        <div className="embedding-picker-title">為這份文件選擇 embedding model</div>
        <div className="embedding-picker-subtitle">
          {displayName ? `偵測到拖入：${displayName}` : '尚未設定 embedding'}
          ——挑一個 model 後，AI 才能語意搜尋你的引用文件。
        </div>

        {/* 區塊 A：已裝清單。依 Ollama 狀態顯示三種情境之一 */}
        <div className="embedding-picker-section">
          <div className="embedding-picker-section-title">本機已裝</div>
          {installed.length > 0 ? (
            <div className="embedding-picker-grid">
              {installed.map((m) => (
                <button
                  key={m.id}
                  type="button"
                  className="embedding-picker-item"
                  disabled={pulling}
                  onClick={() => handlePickInstalled(m.id)}
                >
                  {m.label || m.id}
                </button>
              ))}
            </div>
          ) : ollamaState && ollamaState.binaryFound && !ollamaState.daemonRunning ? (
            <div className="embedding-picker-empty">
              Ollama 已安裝但 daemon 沒在跑。喚醒後就能看到本機 embed model 清單。
              <div className="embedding-picker-empty-actions">
                <button
                  type="button"
                  className="embedding-picker-wake"
                  disabled={waking || pulling}
                  onClick={handleWakeOllama}
                >
                  {waking ? '喚醒中…' : '喚醒 Ollama'}
                </button>
              </div>
            </div>
          ) : ollamaState && !ollamaState.binaryFound ? (
            <div className="embedding-picker-empty">
              沒找到 Ollama。從 <span style={{textDecoration: 'underline'}}>https://ollama.com/download</span> 安裝後，再回來這個彈窗，或先用下面「下載新 model」（會需要 Ollama）。
            </div>
          ) : (
            <div className="embedding-picker-empty">
              Ollama 在跑但沒裝任何 embed model。點下面「下載新 model」自動處理。
            </div>
          )}
        </div>

        {/* 區塊 B：下載新 model */}
        <div className="embedding-picker-section">
          <button
            type="button"
            className="embedding-picker-toggle"
            disabled={pulling}
            onClick={() => setShowDownload((v) => !v)}
          >
            {showDownload ? '收起下載區' : '下載新 model'}
          </button>
          {showDownload && (
            <div className="embedding-picker-download">
              <div className="embedding-picker-chips">
                {SUGGESTED_MODELS.map((m) => (
                  <button
                    key={m}
                    type="button"
                    className={`embedding-picker-chip${downloadInput === m ? ' active' : ''}`}
                    disabled={pulling}
                    onClick={() => setDownloadInput(m)}
                  >
                    {m}
                  </button>
                ))}
              </div>
              <div className="embedding-picker-input-row">
                <input
                  type="text"
                  className="embedding-picker-input"
                  placeholder="e.g. nomic-embed-text"
                  value={downloadInput}
                  disabled={pulling}
                  onChange={(e) => setDownloadInput(e.target.value)}
                />
                <button
                  type="button"
                  className="embedding-picker-pull"
                  disabled={pulling || !downloadInput.trim()}
                  onClick={handleStartPull}
                >
                  {pulling ? '下載中…' : '開始下載'}
                </button>
              </div>
              {progress && (
                <div className="embedding-picker-progress">
                  <div className="embedding-picker-progress-bar">
                    <div
                      className="embedding-picker-progress-fill"
                      style={{width: `${Math.max(0, Math.min(100, progress.percent ?? 0))}%`}}
                    />
                  </div>
                  <div className="embedding-picker-progress-text">
                    {progress.percent != null ? `${progress.percent}%` : ''} {progress.status}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        {/* 區塊 C：跳過 */}
        <div className="embedding-picker-actions">
          <button type="button" className="embedding-picker-skip" disabled={pulling} onClick={handleDismiss}>
            跳過 — 用基本搜尋（TF-IDF）
          </button>
        </div>

        {errorMsg && <div className="embedding-picker-error">{errorMsg}</div>}
      </div>
    </div>
  );
}
