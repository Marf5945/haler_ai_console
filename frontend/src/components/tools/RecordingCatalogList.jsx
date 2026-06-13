import {useEffect, useState} from 'react';
import {FinalizeNativeLearningRunExport, ListLearningReplayCatalog, NativeDragExportLearningRun} from '../../../wailsjs/go/main/App';
import {EventsOn} from '../../../wailsjs/runtime/runtime';
import useI18n from '../../locales/useI18n';
import {callWails} from '../../lib/appShared';
import DragActionModal from './DragActionModal';
import ToolFlowSectionLabel from './ToolFlowSectionLabel';

export default function RecordingCatalogList() {
  const t = useI18n(s => s.t);
  const [recordings, setRecordings] = useState([]);
  const [loading, setLoading] = useState(false);
  const [exportDialog, setExportDialog] = useState(null);
  const [note, setNote] = useState('');

  function loadRecordings() {
    setLoading(true);
    callWails(() => ListLearningReplayCatalog(20))
      .then((data) => setRecordings(Array.isArray(data) ? data : []))
      .catch(() => setRecordings([]))
      .finally(() => setLoading(false));
  }

  // 拖出成功後跳出 移除／複製／取消（與 skill / go-program 一致）。
  function showExportDialog(result) {
    if (!result || result.status !== 'success' || !result.landed_path) {
      if (result?.message) setNote(result.message);
      return;
    }
    setExportDialog({
      runID: result.run_id,
      name: result.title || result.run_id || t('recordingCatalog.untitled'),
      tempExportDir: result.export_dir,
      landedPath: result.landed_path,
      detail: result.drop_target_kind && result.drop_target_dir
        ? `${result.landed_path}\n${result.drop_target_kind}: ${result.drop_target_dir}`
        : result.landed_path,
    });
  }

  useEffect(() => {
    loadRecordings();
    const offStopped = EventsOn('visual_learning:recording_stopped', loadRecordings);
    const offCatalog = EventsOn('learningrun:catalog_updated', loadRecordings);
    const offNative = EventsOn('learningrun:native_completed', showExportDialog);
    return () => {
      offStopped && offStopped();
      offCatalog && offCatalog();
      offNative && offNative();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function startRecordingExport(event, recording) {
    event?.preventDefault?.();
    event?.stopPropagation?.();
    try {
      event?.dataTransfer?.setData?.('application/x-ai-console-learning-run-id', recording?.run_id || '');
      if (event?.dataTransfer) event.dataTransfer.effectAllowed = 'copy';
    } catch (_) {}
    if (!recording?.run_id) return;
    setNote('');
    try {
      const result = await callWails(() => NativeDragExportLearningRun(recording.run_id));
      showExportDialog(result);
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  async function finalizeRecordingExport(action) {
    if (!exportDialog) return;
    const target = exportDialog;
    setExportDialog(null);
    try {
      await callWails(() => FinalizeNativeLearningRunExport(
        action,
        target.runID || '',
        target.tempExportDir || '',
        target.landedPath || '',
      ));
      if (action === 'remove') {
        setRecordings((current) => current.filter((r) => r.run_id !== target.runID));
      }
    } catch (error) {
      setNote(error?.message || String(error));
    }
  }

  if (loading || !recordings || recordings.length === 0) return null;

  return (
    <>
        <ToolFlowSectionLabel>{t('tool.flowCategoryRecording')}</ToolFlowSectionLabel>
        {recordings.map((recording) => {
          const tag = recording.tag || recording.operation_tag || recording.run_id || t('recordingCatalog.untitled');
          const title = recording.title || recording.summary || tag;
          const stoppedAt = recording.stopped_at ? new Date(recording.stopped_at).toLocaleString('zh-TW', {hour12: false}) : '';
          return (
            <button
              key={recording.run_id || tag}
              type="button"
              className="tool-menu-item flow-asset-card flow-recording-card"
              draggable
              data-full-title={title}
              title="拖出資料夾可打包／移除"
              aria-label={`${tag} ${title}`}
              onDragStart={(event) => startRecordingExport(event, recording)}
            >
              {/* Keep recordings framed like tool cards so the flow popup stays visually unified. */}
              <span className="tool-menu-icon">◇</span>
              <strong>{tag}</strong>
              <small>
                {recording.step_count || 0} {t('recordingCatalog.steps')}
                {stoppedAt ? ` · ${stoppedAt}` : ''}
              </small>
            </button>
          );
        })}
        {note && <p className="go-program-flow-note">{note}</p>}
        {exportDialog && (
          <DragActionModal
            ariaLabel="Learning run export"
            icon="↗"
            title={exportDialog.name}
            detail={exportDialog.detail}
            actions={[
              {label: t('adapter.remove'), onClick: () => finalizeRecordingExport('remove')},
              {label: t('adapter.copyAction'), onClick: () => finalizeRecordingExport('copy')},
              {label: t('common.cancel'), onClick: () => finalizeRecordingExport('cancel')},
            ]}
          />
        )}
    </>
  );
}
