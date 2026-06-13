import {useState} from 'react';
import useI18n from '../../locales/useI18n';

export default function SkillActivityCard({injections}) {
  const t = useI18n(s => s.t);
  const [expanded, setExpanded] = useState(null);
  if (!injections || injections.length === 0) return null;

  return (
    <div className="skill-activity-card" aria-label="Skill Activity">
      <div className="skill-activity-header">
        <span>Skill Activity</span>
        <small>{t('skill.injectionCount', { count: injections.length })}</small>
      </div>
      {injections.map((inj, i) => (
        <div className="skill-activity-item" key={inj.skillID || inj.skill_id || i}>
          <button
            className="skill-activity-summary"
            type="button"
            onClick={() => setExpanded(expanded === i ? null : i)}
          >
            <span>{inj.skillID || inj.skill_id || 'unknown'}</span>
            <small>{inj.reason || inj.matchReason || inj.match_reason || ''}</small>
            <code>{inj.summaryHash || inj.summary_hash || ''}</code>
          </button>
          {expanded === i && (
            <div className="skill-activity-detail">
              <div className="review-meta-grid">
                <span>score</span><strong>{inj.score ?? '—'}</strong>
                <span>risk</span><strong>{inj.risk || '—'}</strong>
                <span>session</span><strong>{inj.sessionID || inj.session_id || '—'}</strong>
                <span>cleared</span><strong>{inj.clearedAt || inj.cleared_at ? t('skill.clearedYes') : t('skill.clearedNo')}</strong>
                {inj.clearReason || inj.clear_reason ? (<><span>clear reason</span><strong>{inj.clearReason || inj.clear_reason}</strong></>) : null}
              </div>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
