import {useState, useEffect} from 'react';
import useI18n from '../../locales/useI18n';
import {EventsOn} from '../../lib/wailsRuntime';
import {
  ListGoProgramAuthoringCatalog,
  GetGoProgramAuthoringDetail,
  NativeDragExportGoProgramAuthoring,
  FinalizeNativeGoProgramAuthoringExport,
} from '../../../wailsjs/go/main/App';
import {callWails} from '../../lib/callWails';
import ToolFlowSectionLabel from './ToolFlowSectionLabel';
import DragActionModal from './DragActionModal';
import {openGoProgramSourcePreview} from './GoCodePreviewDock';

const goProgramDagStepCatalog = [
  {
    id: 'import',
    title: '接收程式草稿',
    detail: '模型只提交 JSON draft；系統建立受控工作區與 Go 原始碼檔案。',
  },
  {
    id: 'validate',
    title: '契約與權限檢查',
    detail: '檢查 package main、schema、stdlib/vendor allowlist，攔下未授權網路或子程序。',
  },
  {
    id: 'build',
    title: '受控編譯',
    detail: '使用內建 Go toolchain 編譯，禁止下載 module，套用 timeout 與輸出限制。',
  },
  {
    id: 'plan',
    title: '準備資料輸入',
    detail: '把天氣 JSON、衣服表格或 DB 查詢結果整理成 stdin JSON，寫入只走 scratch。',
  },
  {
    id: 'execute',
    title: '執行與結果審核',
    detail: '限時執行，驗證 stdout JSON contract；成功後只保存為 pending skill。',
  },
];

function goProgramStepStatus(run, index) {
  const status = String(run?.status || '').toLowerCase();
  if (status === 'completed') return 'done';
  if (status === 'needs_user_decision') return index === 0 ? 'blocked' : 'pending';
  if ((run?.attempt_count || 0) > 0) return index === 0 ? 'done' : (index === 1 ? 'active' : 'pending');
  return index === 0 ? 'active' : 'pending';
}

function goProgramStepStatusLabel(status) {
  return {done: '完成', active: '進行中', blocked: '待確認', pending: '等待'}[status] || '等待';
}

function summarizeGoProgramPermissions(permissions = {}) {
  const labels = [];
  if (permissions.read_app_data) labels.push('讀取 app data');
  if (permissions.read_outputs) labels.push('讀取 outputs');
  if (permissions.write_outputs_scratch) labels.push('寫入 scratch outputs');
  if (permissions.read_db_as_json) labels.push('讀取系統提供 DB JSON');
  if (Array.isArray(permissions.read_mounted_paths) && permissions.read_mounted_paths.length) labels.push('讀取受限掛載路徑');
  if (permissions.network) labels.push('網路需審核');
  if (permissions.shell_subprocess) labels.push('子程序需審核');
  return labels.length ? labels.join('、') : '無額外權限';
}

function summarizeGoProgramDataSources(sources = []) {
  const labels = sources.map((source) => source?.name || source?.kind).filter(Boolean);
  return labels.length ? labels.join('、') : '由系統在執行時注入 JSON；尚未綁定外部資料來源';
}

function goProgramLifecycleLabel(run = {}) {
  if (run?.pending_skill_id) return 'Pending skill';
  return {completed: '完成', ready: '待製作', authoring: '製作中', needs_user_decision: '待確認'}[run?.status] || run?.status || '待命';
}

export default function GoProgramAuthoringCatalogList({showLabel = true}) {
  const t = useI18n(s => s.t);
  const [runs, setRuns] = useState([]);
  const [detail, setDetail] = useState(null);
  const [loading, setLoading] = useState(false);
  const [note, setNote] = useState('');
  const [exportDialog, setExportDialog] = useState(null);

  const loadRuns = () => {
    setLoading(true);
    callWails(() => ListGoProgramAuthoringCatalog(20))
      .then((data) => setRuns(Array.isArray(data) ? data : []))
      .catch(() => setRuns([]))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadRuns();
    const offUpdated = EventsOn('goprogram:authoring_updated', loadRuns);
    const offNative = EventsOn('goprogram:native_completed', showExportDialog);
    const timer = setInterval(loadRuns, 12000);
    return () => {
      offUpdated && offUpdated();
      offNative && offNative();
      clearInterval(timer);
    };
  }, []);

  async function openDetail(runID) {
    try {
      const data = await callWails(() => GetGoProgramAuthoringDetail(runID));
      setDetail(data);
      setNote('');
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  function showExportDialog(result) {
    if (!result || result.status !== 'success' || !result.landed_path) {
      if (result?.message) setNote(result.message);
      return;
    }
    setExportDialog({
      runID: result.run_id,
      name: result.program_name || result.program_id || 'Go program',
      tempExportDir: result.export_dir,
      landedPath: result.landed_path,
      detail: result.drop_target_kind && result.drop_target_dir
        ? `${result.landed_path}\n${result.drop_target_kind}: ${result.drop_target_dir}`
        : result.landed_path,
    });
  }

  async function startNativeExport(event, run) {
    event?.preventDefault?.();
    event?.stopPropagation?.();
    try {
      event?.dataTransfer?.setData?.('application/x-ai-console-go-program-run-id', run?.run_id || '');
      if (event?.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
    } catch (_) {}
    if (!run?.run_id) return;
    setNote('');
    try {
      const result = await callWails(() => NativeDragExportGoProgramAuthoring(run.run_id));
      showExportDialog(result);
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  async function finalizeExport(action) {
    if (!exportDialog) return;
    const target = exportDialog;
    setExportDialog(null);
    try {
      await callWails(() => FinalizeNativeGoProgramAuthoringExport(
        action,
        target.runID || '',
        target.tempExportDir || '',
        target.landedPath || '',
      ));
      if (action === 'remove') {
        setRuns((current) => current.filter((run) => run.run_id !== target.runID));
      }
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  if ((loading && runs.length === 0) || !runs || runs.length === 0) return null;

  const selected = detail || null;
  const selectedPermissions = selected?.manifest?.permissions || {};
  const selectedSources = selected?.manifest?.data_sources || [];

  return (
    <>
        {showLabel && <ToolFlowSectionLabel>{t('tool.flowCategorySkill')}</ToolFlowSectionLabel>}
        {runs.map((run) => {
          const updated = run.updated_at ? new Date(run.updated_at).toLocaleString('zh-TW', {hour12: false}) : '';
          const title = run.program_name || run.program_id || run.run_id;
          return (
            <button
              key={run.run_id || title}
              type="button"
              className="tool-menu-item flow-asset-card go-program-flow-card"
              draggable
              data-full-title={run.message || run.purpose || title}
              title="點按檢視，拖出資料夾"
              onClick={() => openDetail(run.run_id)}
              onDragStart={(event) => startNativeExport(event, run)}
            >
              <span className="tool-menu-icon go-program-flow-icon">⌘</span>
              <strong>{title}</strong>
              <small>{goProgramLifecycleLabel(run)} · {run.attempt_count || 0}{updated ? ` · ${updated}` : ''}</small>
            </button>
          );
        })}
      {selected && (
        <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label="Go program authoring detail">
          <section className="tool-drag-modal go-program-flow-detail-modal">
            <header>
              <span>⌘</span>
              <div className="drag-action-title">
                <strong>{selected.program_name || selected.program_id}</strong>
                <small>{goProgramLifecycleLabel(selected)} · {selected.attempt_count || 0} attempts</small>
              </div>
            </header>
            <div className="go-program-flow-detail">
              <div className="go-program-flow-meta">
                <span>Run</span><strong>{selected.run_id}</strong>
                <span>Skill</span><strong>{selected.pending_skill_id || '尚未保存'}</strong>
                <span>Attempt</span><strong>{selected.attempt_count || 0} 次；{selected.latest_attempt_hash ? `版本 ${String(selected.latest_attempt_hash).slice(0, 12)}` : '尚無版本'}</strong>
              </div>
              <div className="go-program-dag" aria-label="Go program authoring DAG">
                {goProgramDagStepCatalog.map((step, index) => {
                  const status = goProgramStepStatus(selected, index);
                  return (
                    <div className={`go-program-dag-node go-program-dag-node-${status}`} key={step.id}>
                      <i>{index + 1}</i>
                      <div>
                        <strong>{step.title}</strong>
                        <small>{step.detail}</small>
                      </div>
                      <em>{goProgramStepStatusLabel(status)}</em>
                    </div>
                  );
                })}
              </div>
              <div className="go-program-flow-meta">
                <span>權限</span><strong>{summarizeGoProgramPermissions(selectedPermissions)}</strong>
                <span>資料來源</span><strong>{summarizeGoProgramDataSources(selectedSources)}</strong>
              </div>
            </div>
            <div className="tool-drag-actions" style={{gridTemplateColumns: 'repeat(3, minmax(0, 1fr))'}}>
              <button type="button" onClick={(event) => startNativeExport(event, selected)}>拖出</button>
              <button
                type="button"
                disabled={!selected.attempt_count}
                onClick={() => openGoProgramSourcePreview(selected.run_id)}
              >查看程式碼</button>
              <button type="button" onClick={() => setDetail(null)}>關閉</button>
            </div>
          </section>
        </div>
      )}
      {note && <p className="go-program-flow-note">{note}</p>}
      {exportDialog && (
        <DragActionModal
          ariaLabel="Go program export"
          icon="↗"
          title={exportDialog.name}
          detail={exportDialog.detail}
          actions={[
            {label: t('adapter.remove'), onClick: () => finalizeExport('remove')},
            {label: t('adapter.copyAction'), onClick: () => finalizeExport('copy')},
            {label: t('common.cancel'), onClick: () => finalizeExport('cancel')},
          ]}
        />
      )}
    </>
  );
}
