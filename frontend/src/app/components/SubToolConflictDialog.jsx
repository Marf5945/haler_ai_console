import useI18n from '../../locales/useI18n';

export default function SubToolConflictDialog({result, onResolve, onCancel}) {
  const {t} = useI18n();
  const conflicts = result?.tool_conflicts || [];
  return (
    <div className="export-dialog-overlay">
      <div className="export-dialog sub-import-dialog">
        <p>{t('package.conflictCount', { count: conflicts.length })}</p>
        <small>{t('package.conflictHint')}</small>
        <div className="sub-conflict-list">
          {conflicts.slice(0, 4).map((tool) => (
            <span key={`${tool.type}:${tool.original_id}`}>{tool.type} / {tool.original_id}</span>
          ))}
        </div>
        <div className="export-dialog-actions">
          <button type="button" onClick={() => onResolve('keep_all')}>{t('package.keepAll')}</button>
          <button type="button" onClick={() => onResolve('overwrite_all')}>{t('package.overwriteAll')}</button>
          <button type="button" onClick={onCancel}>{t('package.handleLater')}</button>
        </div>
      </div>
    </div>
  );
}
