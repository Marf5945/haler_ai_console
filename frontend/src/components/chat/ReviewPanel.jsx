import {useEffect, useRef, useState} from 'react';
import {createPortal} from 'react-dom';
import {GetDAGRunDebug, SubmitTaskLoopInput} from '../../../wailsjs/go/main/App';
import DocumentReviewCard from '../DocumentReviewCard';
import useI18n from '../../locales/useI18n';
import {callWails, categorizePendingDigest, dagStatusLabel, formatDagTime, formatNodeRiskLabel, reviewConsequenceText, reviewStatusLabel, taskProgressDebugEnabled} from '../../lib/appShared';

export default function ReviewPanel({
  activePopup, dagRun, onPopupChange, reviewState, onSkillSelect, onSnooze,
  onSnoozeHoursChange, snoozeHours, onAcknowledgeDigestItem, onConfirmSkillBuild,
  reviewArchive, w3aImportPopup, w3aDetail, w3aPollutionResult, w3aTransferGuidance, w3aTrustList = [],
  w3aActionBusy = '', w3aActionError = '', w3aStatusConfig = {}, w3aToastMsg,
  onLoadW3AInfo = () => {}, onDetectW3APollution = () => {}, onShowW3AGuidance = () => {},
  onTrustW3ADeveloper = () => {}, onExportW3ACopy = () => {},
  onDismissW3AImportPopup = () => {}, onShowW3AToast = () => {}, onDismissW3AToast = () => {},
}) {
  const t = useI18n(s => s.t);
  const {highRisk, lowRiskAmbiguity, hookCandidates, pendingDigest, pendingPackages} = reviewState;
  const ambiguityHidden = Boolean(lowRiskAmbiguity.dismissedUntil);
  const panelRef = useRef(null);
  const [popupFrame, setPopupFrame] = useState(null);
  const [digestExpanded, setDigestExpanded] = useState({urgent: true, later: false, archive: false});
  const [taskDebugDump, setTaskDebugDump] = useState(null);
  const activeW3AInfo = w3aDetail || w3aImportPopup?.info || {};
  const activeW3ATraining = activeW3AInfo.training || {};
  const activeW3APollution = w3aPollutionResult || activeW3AInfo.pollution;
  const activeW3APath = w3aImportPopup?.source_path || activeW3AInfo.file_path || '';
  const activeW3ASidecar = w3aImportPopup?.sidecar_path || '';

  // Count total hook candidates for badge
  const hookCount = (hookCandidates?.tagPatches?.length || 0)
    + (hookCandidates?.subagentCandidates?.length || 0)
    + (hookCandidates?.registryProposals?.length || 0);
  const packageCount = pendingPackages?.length || 0;

  useEffect(() => {
    if (!activePopup) return undefined;

    const updatePopupFrame = () => {
      const rect = panelRef.current?.getBoundingClientRect();
      if (!rect) return;

      const viewportGap = 24;
      const preferredWidth = activePopup === 'skills' ? 460 : activePopup === 'dag' ? 720 : 640;
      const width = Math.min(preferredWidth, Math.max(280, rect.width - 8), window.innerWidth - viewportGap * 2);
      const left = activePopup === 'skills'
        ? Math.min(Math.max(viewportGap, rect.right - width), window.innerWidth - width - viewportGap)
        : Math.min(Math.max(viewportGap, rect.left), window.innerWidth - width - viewportGap);

      setPopupFrame({
        left,
        top: Math.min(rect.bottom + 8, window.innerHeight - 120),
        width,
      });
    };

    updatePopupFrame();
    window.addEventListener('resize', updatePopupFrame);
    window.addEventListener('scroll', updatePopupFrame, true);

    return () => {
      window.removeEventListener('resize', updatePopupFrame);
      window.removeEventListener('scroll', updatePopupFrame, true);
    };
  }, [activePopup]);

  useEffect(() => {
    setTaskDebugDump(null);
  }, [dagRun?.id]);

  // I-1: Categorize pending digest items into three UI blocks
  const digestBlocks = categorizePendingDigest(pendingDigest);
  const dagSummaryCount = dagRun?.summaries?.length || 0;
  const pendingDigestCount = (pendingDigest?.items || []).length;
  const pendingTotalCount = hookCount + packageCount + dagSummaryCount + pendingDigestCount;

  return (
    <section className="review-panel" aria-label="Review Panel" ref={panelRef}>
      <div className="review-summary-row">
        <button
          className={`review-summary-btn review-summary-dag ${['blocked', 'waiting_review'].includes(dagRun?.status) ? 'review-summary-hot' : ''}`}
          type="button"
          onClick={() => onPopupChange(activePopup === 'dag' ? null : 'dag')}
        >
          <span>任務進度</span>
          <strong>{dagRun ? dagStatusLabel(dagRun.status) : t('dag.standby')}</strong>
        </button>
        <button
          className={`review-summary-btn ${highRisk.status === 'pending' ? 'review-summary-hot' : ''}`}
          type="button"
          onClick={() => onPopupChange(activePopup === 'risk' ? null : 'risk')}
        >
          <span>{t('review.highRiskTitle')}</span>
          <strong>{reviewStatusLabel(highRisk.status)}</strong>
        </button>
        <button
          className="review-summary-btn"
          type="button"
          onClick={() => onPopupChange(activePopup === 'skills' ? null : 'skills')}
        >
          <span>{t('dag.switchableOptions')}</span>
          <strong>{ambiguityHidden ? t('dag.dismissLabel', { time: lowRiskAmbiguity.dismissedUntil }) : 'low risk'}</strong>
        </button>
        {pendingTotalCount > 0 && (
          <button
            className="review-summary-btn"
            type="button"
            onClick={() => onPopupChange(activePopup === 'digest' ? null : 'digest')}
          >
            <span>Pending</span>
            <strong>{t('dag.pendingItems', { count: pendingTotalCount })}</strong>
          </button>
        )}
      </div>

      {/* 任務進度內容：一般使用者看步驟，開發用 ID 藏在後端 debug API。 */}
      {activePopup === 'dag' && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-dag"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{dagRun?.title || t('dag.noDagRun')}</span>
            <strong>{dagRun ? dagStatusLabel(dagRun.status) : 'idle'}</strong>
          </header>
          {dagRun ? (
            <>
              <div className="dag-node-list">
                {dagRun.nodes.map((node) => (
                  <div className={`dag-node-item dag-node-${node.status}`} key={node.id}>
                    <i className={`dag-node-dot dag-risk-${node.risk}`} />
                    <div>
                      <strong>{node.title}</strong>
                      <span>{node.action}</span>
                      <small>
                        {dagStatusLabel(node.status)}
                        {' · '}
                        {formatDagTime(node.startedAt)}
                        {' → '}
                        {formatDagTime(node.endedAt)}
                        {node.durationMs != null ? ` · ${(node.durationMs / 1000).toFixed(1)}s` : ''}
                      </small>
                      {node.resultSummary && <small>{node.resultSummary}</small>}
                      {node.status === 'running' && taskLoopRounds[node.id] && (
                        <small className="dag-loop-round">
                          {`第 ${taskLoopRounds[node.id].iteration} 輪：${taskLoopRounds[node.id].action} ${taskLoopRounds[node.id].target}`}
                        </small>
                      )}
                      {node.status === 'waiting_user' && (
                        <span className="dag-loop-reply">
                          <input
                            type="text"
                            value={taskLoopReply[node.id] || ''}
                            placeholder="輸入補充資訊後送出"
                            onChange={(e) => {
                              const value = e.target.value;
                              setTaskLoopReply((prev) => ({...prev, [node.id]: value}));
                            }}
                            onKeyDown={(e) => {
                              if (e.key !== 'Enter') return;
                              const reply = (taskLoopReply[node.id] || '').trim();
                              if (!reply) return;
                              callWails(() => SubmitTaskLoopInput(dagRun.id, node.id, reply))
                                .then(() => setTaskLoopReply((prev) => ({...prev, [node.id]: ''})))
                                .catch((error) => setTaskLoopReply((prev) => ({...prev, [node.id]: reply, [`${node.id}:error`]: error?.message || String(error)})));
                            }}
                          />
                          {taskLoopReply[`${node.id}:error`] && <small className="dag-loop-error">{taskLoopReply[`${node.id}:error`]}</small>}
                        </span>
                      )}
                    </div>
                    <em>{formatNodeRiskLabel(node)}</em>
                  </div>
                ))}
              </div>
              {taskProgressDebugEnabled && (
                <div className="task-debug-box">
                  <button
                    type="button"
                    onClick={() => callWails(() => GetDAGRunDebug(dagRun.id)).then(setTaskDebugDump).catch((error) => setTaskDebugDump({error: error?.message || String(error)}))}
                  >
                    開發工具，穩定後移除
                  </button>
                  {taskDebugDump && <pre>{JSON.stringify(taskDebugDump, null, 2)}</pre>}
                </div>
              )}
            </>
          ) : (
            <p>{t('dag.dagRunHint')}</p>
          )}
        </article>,
        document.body,
      )}

      {/* High-risk popup */}
      {activePopup === 'risk' && popupFrame && createPortal(
        <article
          className={`review-card review-card-popup review-card-risk review-card-${highRisk.status}`}
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{highRisk.title}</span>
            <strong>{reviewStatusLabel(highRisk.status)}</strong>
          </header>
          <div className="review-detail-row">
            <div>
              <span>這一步要做什麼</span>
              <p>{highRisk.action}</p>
            </div>
            <div>
              <span>會影響什麼</span>
              <p>{highRisk.permissionSummary}</p>
            </div>
            <div>
              <span>使用的工具/模型</span>
              <p>{highRisk.skillId || highRisk.summaryHash || '目前模型'}</p>
            </div>
            <div>
              <span>為什麼需要確認</span>
              <p>{highRisk.diff?.[1] || highRisk.diff?.[0] || '這一步風險較高，執行前需要確認。'}</p>
            </div>
          </div>
          <small>{reviewConsequenceText(highRisk.status, highRisk.note)}</small>
          {highRisk.status === 'pending' && highRisk.id && onConfirmSkillBuild && (
            <div className="review-action-row">
              <button type="button" className="review-confirm-btn" onClick={() => onConfirmSkillBuild(highRisk.id)}>{t('dag.confirmExecute')}</button>
            </div>
          )}
        </article>,
        document.body,
      )}

      {/* Low-risk ambiguity popup */}
      {activePopup === 'skills' && !ambiguityHidden && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-skills skill-ambiguity-card"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>{t('dag.switchable')}</span>
            <strong>low risk</strong>
          </header>
          <div className="skill-choice-list">
            {lowRiskAmbiguity.candidates.map((candidate) => (
              <button
                className={candidate.id === lowRiskAmbiguity.selectedSkillId ? 'skill-choice-active' : ''}
                type="button"
                key={candidate.id}
                onClick={() => onSkillSelect(candidate.id)}
              >
                <span>{candidate.id}</span>
                <small>{candidate.reason} · score {candidate.score} · {candidate.risk}</small>
              </button>
            ))}
          </div>
          <div className="skill-snooze-row">
            <label>
              {t('review.snoozeLabel')}
              <input
                max="72"
                min="1"
                type="number"
                value={snoozeHours}
                onChange={(event) => onSnoozeHoursChange(Math.max(1, Number(event.target.value) || 1))}
              />
              {t('review.hours')}
            </label>
            <button type="button" onClick={onSnooze}>{t('review.applySnooze')}</button>
          </div>
        </article>,
        document.body,
      )}

      {/* I-1: Pending Digest + Hook Candidates + Package Queue popup */}
      {activePopup === 'digest' && popupFrame && createPortal(
        <article
          className="review-card review-card-popup review-card-digest"
          style={{left: `${popupFrame.left}px`, top: `${popupFrame.top}px`, width: `${popupFrame.width}px`}}
        >
          <header>
            <span>Pending Digest</span>
            <strong>{t('review.pendingCount', { count: pendingTotalCount })}</strong>
          </header>

          {/* I-7: Node summaries surface here as Pending Digest entries. */}
          {dagSummaryCount > 0 && (
            <div className="digest-section">
              <h4>DAG Node Summary ({dagSummaryCount})</h4>
              {dagRun.summaries.map((summary) => (
                <div className="digest-item" key={summary.nodeId}>
                  <span>{summary.title}</span>
                  <small>{summary.text} · {formatDagTime(summary.generatedAt)}</small>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Tag Patches */}
          {hookCandidates?.tagPatches?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.tagPatchTitle', { count: hookCandidates.tagPatches.length })}</h4>
              {hookCandidates.tagPatches.map((patch, i) => (
                <div className="digest-item" key={patch.id || i}>
                  <span>{patch.tag || patch.id}</span>
                  <small>{patch.reason || 'hook evidence'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(patch.id, 'keep')}>{t('review.keep')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(patch.id, 'archive')}>{t('review.archive')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Subagent Candidates */}
          {hookCandidates?.subagentCandidates?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.subagentTitle', { count: hookCandidates.subagentCandidates.length })}</h4>
              {hookCandidates.subagentCandidates.map((candidate, i) => (
                <div className="digest-item" key={candidate.id || i}>
                  <span>{candidate.name || candidate.id}</span>
                  <small>{candidate.source || 'hook run'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(candidate.id, 'review_now')}>{t('review.reviewAction')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(candidate.id, 'archive')}>{t('review.archive')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Hook Candidates: Registry Proposals */}
          {hookCandidates?.registryProposals?.length > 0 && (
            <div className="digest-section">
              <h4>Tool Registry Patch ({hookCandidates.registryProposals.length})</h4>
              {hookCandidates.registryProposals.map((proposal, i) => (
                <div className="digest-item" key={proposal.id || i}>
                  <span>{proposal.toolId || proposal.id}</span>
                  <small>{proposal.action || 'patch proposal'}</small>
                  <div className="digest-item-actions">
                    <button type="button" onClick={() => onAcknowledgeDigestItem(proposal.id, 'keep')}>{t('review.apply')}</button>
                    <button type="button" onClick={() => onAcknowledgeDigestItem(proposal.id, 'archive')}>{t('review.ignore')}</button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Pending Digest (weekly summary) — three blocks */}
          {pendingDigest && (
            <>
              {digestBlocks.urgent.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, urgent: !prev.urgent}))}>
                    <h4>{t('review.urgentTitle', { count: digestBlocks.urgent.length })}</h4>
                    <span>{digestExpanded.urgent ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.urgent && digestBlocks.urgent.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category} · {item.risk || 'unknown'}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'review_now')}>{t('review.reviewAction')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'delete')}>{t('review.delete')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {digestBlocks.later.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, later: !prev.later}))}>
                    <h4>{t('review.laterTitle', { count: digestBlocks.later.length })}</h4>
                    <span>{digestExpanded.later ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.later && digestBlocks.later.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'archive')}>{t('review.archive')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
              {digestBlocks.archive.length > 0 && (
                <div className="digest-section">
                  <button className="digest-section-toggle" type="button" onClick={() => setDigestExpanded((prev) => ({...prev, archive: !prev.archive}))}>
                    <h4>{t('review.archiveTitle', { count: digestBlocks.archive.length })}</h4>
                    <span>{digestExpanded.archive ? '▾' : '▸'}</span>
                  </button>
                  {digestExpanded.archive && digestBlocks.archive.map((item, i) => (
                    <div className="digest-item" key={item.id || i}>
                      <span>{item.title || item.id}</span>
                      <small>{item.category}</small>
                      <div className="digest-item-actions">
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'batch_archive_low_value')}>{t('review.batchArchive')}</button>
                        <button type="button" onClick={() => onAcknowledgeDigestItem(item.id, 'keep')}>{t('review.keep')}</button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}

          {/* Pending Packages */}
          {pendingPackages?.length > 0 && (
            <div className="digest-section">
              <h4>{t('review.pendingPackagesTitle', { count: pendingPackages.length })}</h4>
              {pendingPackages.map((pkg, i) => (
                <div className="digest-item" key={pkg.id || i}>
                  <span>{pkg.manifest?.name || pkg.sourcePath || pkg.id}</span>
                  <small>risk: {pkg.manifest?.riskTag || 'unknown'}</small>
                </div>
              ))}
            </div>
          )}

          {/* #I-1002: Rejected Review Card 歷史 — 顯示安裝失敗 / 被拒絕的套件軌跡 */}
          {reviewArchive?.filter((c) => c.status === 'rejected').length > 0 && (
            <div className="digest-section">
              <h4>{t('review.rejectedTitle', { count: reviewArchive.filter((c) => c.status === 'rejected').length })}</h4>
              {reviewArchive.filter((c) => c.status === 'rejected').slice(-10).map((card, i) => (
                <div className="digest-item digest-item-rejected" key={card.id || i}>
                  <span>{card.plain_reason}</span>
                  <small className="rejected-reason">{t('review.rejectedReason', { reason: card.reject_reason || card.engineer_reason })}</small>
                  <small>{card.archived_at}</small>
                </div>
              ))}
            </div>
          )}
        </article>,
        document.body,
      )}

      {/* ── v3.6.2 W3A Media Provenance（§9A）UI ── */}

      {/* W3A 匯入選單：偵測到 sidecar 後顯示功能說明 */}
      {w3aImportPopup && (
        <div className="w3a-import-overlay" onClick={onDismissW3AImportPopup}>
          <div className="w3a-import-popup" onClick={(e) => e.stopPropagation()}>
            <div className="w3a-import-header">
              <span className="w3a-import-icon">
                {w3aStatusConfig[w3aImportPopup.info?.status]?.icon || '❓'}
              </span>
              <span className="w3a-import-title">{t("w3a.title")}</span>
            </div>
            <div className="w3a-import-status" style={{color: w3aStatusConfig[activeW3AInfo.status]?.color || '#95a5a6'}}>
              {w3aStatusConfig[activeW3AInfo.status]?.label || t('w3a.unknownStatus')}
            </div>
            <div className="w3a-import-recommendation">{w3aImportPopup.recommendation}</div>
            {w3aImportPopup.has_sidecar && (
              <div className="w3a-import-sidecar-badge">{t("w3a.sidecarDetected")}</div>
            )}
            <div className="w3a-detail-grid">
              {activeW3APath && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.sourcePath')}</span>
                  <code>{activeW3APath}</code>
                </div>
              )}
              {activeW3ASidecar && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.sidecarPath')}</span>
                  <code>{activeW3ASidecar}</code>
                </div>
              )}
              {activeW3AInfo.media_scope && (
                <div className="w3a-detail-row">
                  <span>{t('w3a.mediaScope')}</span>
                  <strong>{activeW3AInfo.media_scope}</strong>
                </div>
              )}
              {activeW3AInfo.training && (
                <>
                  <div className="w3a-detail-row">
                    <span>{t('w3a.trainingSafe')}</span>
                    <strong>{activeW3ATraining.training_safe ? t('w3a.safeYes') : t('w3a.safeNo')}</strong>
                  </div>
                  <div className="w3a-detail-row">
                    <span>{t('w3a.filterRequired')}</span>
                    <strong>{activeW3ATraining.filter_required ? t('common.yes') : t('common.no')}</strong>
                  </div>
                </>
              )}
              <div className="w3a-detail-row">
                <span>{t('w3a.trustCount')}</span>
                <strong>{w3aTrustList.length}</strong>
              </div>
            </div>
            {activeW3APollution && (
              <div className={`w3a-pollution-card ${activeW3APollution.is_pollution_risk ? 'w3a-pollution-risk' : ''}`}>
                <span>{t('w3a.weightedTotal')}</span>
                <strong>{typeof activeW3APollution.weighted_total === 'number' ? activeW3APollution.weighted_total.toFixed(2) : activeW3APollution.weighted_total}</strong>
                {activeW3APollution.details && <small>{activeW3APollution.details}</small>}
              </div>
            )}
            {w3aTransferGuidance && (
              <div className="w3a-guidance-card">
                <strong>{w3aTransferGuidance.ui_message}</strong>
                <div className="w3a-guidance-list">
                  <span>{t('w3a.recommended')}</span>
                  {(w3aTransferGuidance.recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
                <div className="w3a-guidance-list">
                  <span>{t('w3a.notRecommended')}</span>
                  {(w3aTransferGuidance.not_recommended || []).map((item) => <small key={item}>{item}</small>)}
                </div>
              </div>
            )}
            <div className="w3a-import-capabilities">
              <span className="w3a-import-cap-title">{t("w3a.capTitle")}</span>
              {(w3aImportPopup.capabilities || []).map((cap, i) => (
                <div className="w3a-import-cap-item" key={i}>▸ {cap}</div>
              ))}
            </div>
            {w3aActionError && <div className="w3a-import-error">{w3aActionError}</div>}
            <div className="w3a-import-actions">
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onLoadW3AInfo}>{w3aActionBusy === 'info' ? t('w3a.actionBusy') : t('w3a.infoAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onDetectW3APollution}>{w3aActionBusy === 'pollution' ? t('w3a.actionBusy') : t('w3a.pollutionAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onShowW3AGuidance}>{w3aActionBusy === 'guidance' ? t('w3a.actionBusy') : t('w3a.guidanceAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-secondary" type="button" disabled={!!w3aActionBusy} onClick={onTrustW3ADeveloper}>{w3aActionBusy === 'trust' ? t('w3a.actionBusy') : t('w3a.trustAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-warning" type="button" disabled={!!w3aActionBusy} onClick={onExportW3ACopy}>{w3aActionBusy === 'export' ? t('w3a.actionBusy') : t('w3a.exportCopyAction')}</button>
              <button className="w3a-import-btn w3a-import-btn-primary" type="button" onClick={onDismissW3AImportPopup}>{t('w3a.confirm')}</button>
            </div>
          </div>
        </div>
      )}

      {/* §24: 文件寫入確認卡片 */}
      <DocumentReviewCard onToast={onShowW3AToast} />

      {/* W3A 傳輸引導 toast（軟性提示） */}
      {w3aToastMsg && (
        <div className="w3a-toast" onClick={onDismissW3AToast}>
          <span className="w3a-toast-icon">🔏</span>
          <span className="w3a-toast-text">{w3aToastMsg}</span>
        </div>
      )}
    </section>
  );
}
