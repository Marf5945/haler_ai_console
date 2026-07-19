import React, {useState} from 'react';
import {createPortal} from 'react-dom';
import useI18n from '../locales/useI18n';
import {
  formatSchedulerPayloadSummary,
  schedulerActiveNextLabel,
  schedulerDefaultPayload,
  schedulerJobNo,
} from './scheduler';

export default function SchedulerPanel({
  clock,
  jobs,
  draft,
  busy,
  error,
  onDraftChange,
  onRefresh,
  onCreate,
  onJobAction,
  onBootstrapSkill,
  onClose,
}) {
  const t = useI18n(s => s.t);
  const cronPresets = [
    {label: t('scheduler.presetHourly'), value: '@hourly'},
    {label: t('scheduler.presetDaily0900'), value: '0 9 * * *'},
    {label: t('scheduler.presetWeeklyMon0900'), value: '0 9 * * 1'},
  ];
  const hasDraft = Boolean(draft.name || draft.actionPayload);
  const [summaryJob, setSummaryJob] = useState(null);
  const summaryText = summaryJob ? formatSchedulerPayloadSummary(summaryJob.action_payload, summaryJob.name || t('scheduler.noSummary')) : '';
  const updateDraft = (patch) => onDraftChange((current) => ({...current, ...patch}));

  return createPortal(
    <div className="scheduler-overlay" role="dialog" aria-modal="true" aria-label={t('scheduler.dialogLabel')}>
      <section className="scheduler-panel">
        <header className="scheduler-header">
          <div>
            <span className="scheduler-kicker">Scheduler</span>
            <h3>{t('scheduler.title')}</h3>
          </div>
          <button className="scheduler-close" type="button" onClick={onClose} aria-label={t('common.close')}>×</button>
        </header>

        <div className="scheduler-clock-band">
          <div>
            <span>{t('scheduler.systemTime')}</span>
            <strong>{clock?.local_time || t('scheduler.syncing')}</strong>
          </div>
          <div>
            <span>{t('scheduler.timezone')}</span>
            <strong>{[clock?.timezone, clock?.utc_offset].filter(Boolean).join(' ') || t('scheduler.local')}</strong>
          </div>
          <button type="button" onClick={onRefresh} disabled={busy}>{t('scheduler.sync')}</button>
        </div>

        {hasDraft && (
          <div className="scheduler-draft-strip">
            <div className="scheduler-draft-fields">
              <label>
                <span>{t('scheduler.draftName')}</span>
                <input
                  type="text"
                  value={draft.name}
                  onChange={(event) => updateDraft({name: event.target.value})}
                  placeholder={t('scheduler.draftNamePlaceholder')}
                />
              </label>
              <label>
                <span>{t('scheduler.rule')}</span>
                <input
                  type="text"
                  value={draft.cronExpr}
                  onChange={(event) => updateDraft({cronExpr: event.target.value})}
                  placeholder="0 9 * * *"
                />
              </label>
              <label>
                <span>{t('scheduler.summary')}</span>
                <textarea
                  value={formatSchedulerPayloadSummary(draft.actionPayload, '')}
                  onChange={(event) => updateDraft({actionPayload: schedulerDefaultPayload(draft.name || t('scheduler.defaultReminder'), draft.name || t('scheduler.defaultReminder'), event.target.value)})}
                  placeholder={t('scheduler.summaryPlaceholder')}
                  rows={2}
                />
              </label>
            </div>
            <div className="scheduler-preset-row">
              {cronPresets.map((preset) => (
                <button type="button" key={preset.value} onClick={() => updateDraft({cronExpr: preset.value})}>
                  {preset.label}
                </button>
              ))}
              <button className="scheduler-create-btn" type="button" onClick={onCreate} disabled={busy}>
                {busy ? t('scheduler.busy') : t('scheduler.create')}
              </button>
            </div>
          </div>
        )}

        {error && <p className="scheduler-error">{error}</p>}

        <div className="scheduler-table">
          <div className="scheduler-table-scroll">
            <div className="scheduler-table-head">
              <span>#</span>
              <span>{t('scheduler.tableTitle')}</span>
              <span>{t('scheduler.tableTime')}</span>
              <span>{t('scheduler.tableStatus')}</span>
              <span></span>
            </div>
            {jobs.length === 0 ? (
              <div className="scheduler-empty">{t('scheduler.empty')}</div>
            ) : jobs.map((job, index) => (
              <article className="scheduler-job" key={job.id}>
                <span className="scheduler-job-index">{schedulerJobNo(job, index)}</span>
                <div className="scheduler-job-name">
                  <strong>{job.name}</strong>
                </div>
                <span>{schedulerActiveNextLabel(job)}</span>
                <small>{job.enabled ? t('scheduler.enabled') : t('scheduler.paused')}</small>
                <div className="scheduler-job-actions">
                  <button type="button" disabled={busy} onClick={() => setSummaryJob(job)}>{t('scheduler.summary')}</button>
                  {job.enabled ? (
                    <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'pause')}>{t('scheduler.pause')}</button>
                  ) : (
                    <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'resume')}>{t('scheduler.resume')}</button>
                  )}
                  <button type="button" disabled={busy} onClick={() => onJobAction(job.id, 'delete')}>{t('scheduler.delete')}</button>
                </div>
              </article>
            ))}
          </div>
        </div>

        {summaryJob && (
          <div className="scheduler-summary-modal" role="dialog" aria-label={t('scheduler.summary')}>
            <section className="scheduler-summary-card">
              <header>
                <div>
                  <span>{t('scheduler.summary')}</span>
                  <h4>{summaryJob.name || t('scheduler.unnamed')}</h4>
                </div>
                <button type="button" onClick={() => setSummaryJob(null)} aria-label={t('scheduler.closeSummary')}>×</button>
	              </header>
	              <p>{summaryText}</p>
	              <dl>
	                <div>
	                  <dt>{t('scheduler.tableTime')}</dt>
	                  <dd>{schedulerActiveNextLabel(summaryJob)}</dd>
	                </div>
	                <div>
	                  <dt>{t('scheduler.tableStatus')}</dt>
	                  <dd>{summaryJob.enabled ? t('scheduler.enabled') : t('scheduler.paused')}</dd>
	                </div>
	                <div>
	                  <dt>Skill</dt>
	                  <dd>{summaryJob.skill_id || t('scheduler.notCreated')}</dd>
	                </div>
	                <div>
	                  <dt>Sub</dt>
	                  <dd>{summaryJob.source_sub_id || ''}</dd>
	                </div>
	              </dl>
	              {!summaryJob.skill_id && (
	                <button
	                  className="scheduler-create-btn"
	                  type="button"
	                  disabled={busy}
	                  onClick={async () => {
	                    await onBootstrapSkill?.(summaryJob);
	                    setSummaryJob(null);
	                  }}
	                >
	                  {busy ? t('scheduler.busy') : t('scheduler.createAutoSkill')}
	                </button>
	              )}
	            </section>
	          </div>
	        )}
      </section>
    </div>,
    document.body,
  );
}

// v3.3.2 P0.3 — RightRail renders tool visibility state.
// Available=false → lightning-fracture overlay; never greyed-out / hidden.
/* i18n: right rail */
