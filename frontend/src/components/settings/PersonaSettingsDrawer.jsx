import {useEffect, useRef, useState} from 'react';
import {ListPixelAvatarPacks, RenderPixelAvatarPreview, SetPersonaPixelAvatarPack} from '../../../wailsjs/go/main/App';
import useI18n from '../../locales/useI18n';
import {avatarStateLabel, avatarStateOptions, bytesToDataUrl, fallbackSettings, findActivePersona, getReplyStrategyPresets, limitChineseText, lockedPersonaId, lockedPersonaName, maxPersonas, parseToneChance, pixelAvatarRenderSize, pixelPackForPersona, replyStrategyPresetFor, resolveAvatarProvider, resolvePersonaAvatarSrc} from '../../lib/appShared';
import DragActionModal from '../tools/DragActionModal';

export default function PersonaSettingsDrawer({
  settingsState, onPersonaAdd, onPersonaDrop, onPersonaChange, onPersonaNativeDrag, onPersonaNativeExportAction, onPersonaReorder,
  avatarConfigs = {}, avatarExpression, avatarModeNotice, renderedPixelAvatars = {},
  staticAvatarPreviews = {}, onAvatarProviderSelect, onAvatarStateSelect, onAvatarLoad,
}) {
  const t = useI18n(s => s.t);
  const personas = settingsState.personas;
  const activePersona = findActivePersona(settingsState) || fallbackSettings.personas[0];
  const activeAvatarConfig = avatarConfigs[activePersona.id] || null;
  const activeAvatarProvider = resolveAvatarProvider(activeAvatarConfig);
  const activeRenderedPixelAvatar = renderedPixelAvatars[activePersona.id] || '';
  const activeAvatarSrc = resolvePersonaAvatarSrc(activePersona, activeAvatarConfig, staticAvatarPreviews, activeRenderedPixelAvatar);
  const activeReplyStrategy = activePersona.replyStrategy || '';
  const activeReplyStrategyIsPreset = !activeReplyStrategy || Boolean(replyStrategyPresetFor(activeReplyStrategy));
  const activeAvatarLocked = activePersona.id === lockedPersonaId;
  const canAddPersona = personas.length < maxPersonas;
  const personaSlotCount = personas.length + (canAddPersona ? 1 : 0);
  const personaRowRef = useRef(null);
  const personaFormRef = useRef(null);
  const draggedPersonaIdRef = useRef(null);
  const personaPointerDragRef = useRef(null);
  const suppressPersonaClickRef = useRef(false);
  const [draggedPersonaId, setDraggedPersonaId] = useState(null);
  const [personaExportDialog, setPersonaExportDialog] = useState(null);
  const [strengthPickerOpen, setStrengthPickerOpen] = useState(false);
  const [avatarPickerOpen, setAvatarPickerOpen] = useState(false);
  const [pixelPackPopup2, setPixelPackPopup2] = useState(false);
  const [pixelPacks2, setPixelPacks2] = useState([]);
  const [currentPack2, setCurrentPack2] = useState('wolf');
  const [packPreviews2, setPackPreviews2] = useState({});
  const [strengthDraft, setStrengthDraft] = useState(parseToneChance(activePersona.roleStrength));

  useEffect(() => {
    setStrengthDraft(parseToneChance(activePersona.roleStrength));
    setStrengthPickerOpen(false);
    setAvatarPickerOpen(false);
    onAvatarLoad?.(activePersona.id);
  }, [activePersona.id]);

  // ── Pixel Pack Switcher（設定面板版）──
  useEffect(() => {
    if (!pixelPackPopup2) return;
    let cancelled = false;
    (async () => {
      try {
        const [packs] = await Promise.all([ListPixelAvatarPacks()]);
        if (cancelled) return;
        setPixelPacks2(packs);
        setCurrentPack2(pixelPackForPersona(activePersona, activeAvatarConfig));
        const previews = {};
        for (const p of packs) {
          try {
            const bytes = await RenderPixelAvatarPreview(p.id, 'idle', pixelAvatarRenderSize);
            if (bytes && bytes.length) {
              previews[p.id] = bytesToDataUrl(bytes);
            }
          } catch { /* ignore */ }
        }
        if (!cancelled) setPackPreviews2(previews);
      } catch { /* ignore */ }
    })();
    return () => { cancelled = true; };
  }, [pixelPackPopup2, activePersona.id, activeAvatarConfig]);

  async function switchPixelPack2(packId) {
    if (activeAvatarLocked) return;
    try {
      await SetPersonaPixelAvatarPack(activePersona.id, packId);
      setCurrentPack2(packId);
      setPixelPackPopup2(false);
      await onAvatarLoad?.(activePersona.id);
    } catch { /* ignore */ }
  }

  function collectPersonaFormPatch() {
    const form = personaFormRef.current;
    if (!form) return {};
    return {
      name: limitChineseText(form.elements.personaName?.value || '', 100),
      identity: form.elements.identity?.value || '',
      replyStrategy: form.elements.replyStrategy?.value || '',
      roleStrength: `${strengthDraft}%`,
      personality: form.elements.personality?.value || '',
      scenario: form.elements.scenario?.value || '',
      description: form.elements.description?.value || '',
    };
  }

  function commitActivePersonaForm() {
    if (!activePersona?.id) return activePersona;
    const patch = collectPersonaFormPatch();
    if (Object.keys(patch).length === 0) return activePersona;
    onPersonaChange(activePersona.id, patch);
    return {...activePersona, ...patch};
  }

  function scrollPersonas(direction) {
    const row = personaRowRef.current;
    if (!row) return;
    row.scrollBy({left: direction * row.clientWidth, behavior: 'smooth'});
  }

  async function handlePersonaExportAction(action) {
    if (!personaExportDialog) return;
    const target = personaExportDialog;
    setPersonaExportDialog(null);
    await onPersonaNativeExportAction?.(action, target);
  }

  function movePersonaBefore(sourcePersonaId, targetPersonaId) {
    if (!sourcePersonaId || sourcePersonaId === targetPersonaId) return;
    const orderIds = personas.map((persona) => persona.id);
    const fromIndex = orderIds.indexOf(sourcePersonaId);
    const toIndex = orderIds.indexOf(targetPersonaId);
    if (fromIndex < 0 || toIndex < 0) return;

    const nextOrderIds = [...orderIds];
    nextOrderIds.splice(fromIndex, 1);
    nextOrderIds.splice(toIndex, 0, sourcePersonaId);
    onPersonaReorder(nextOrderIds);
  }

  function personaIdAtPoint(clientX, clientY) {
    const target = document.elementFromPoint(clientX, clientY);
    return target?.closest?.('[data-persona-id]')?.dataset?.personaId || '';
  }

  function pointInsidePersonaRow(clientX, clientY) {
    const rect = personaRowRef.current?.getBoundingClientRect();
    return Boolean(rect && clientX >= rect.left && clientX <= rect.right && clientY >= rect.top && clientY <= rect.bottom);
  }

  function finishPersonaPointerDrag(event, sourcePersonaId, moved) {
    personaPointerDragRef.current = null;
    draggedPersonaIdRef.current = null;
    setDraggedPersonaId(null);
    if (!moved) return;

    const targetPersonaId = personaIdAtPoint(event.clientX, event.clientY);
    if (targetPersonaId && targetPersonaId !== sourcePersonaId) {
      movePersonaBefore(sourcePersonaId, targetPersonaId);
      return;
    }

    const isZeroCoords = event.clientX === 0 && event.clientY === 0;
    const inWindow = !isZeroCoords && event.clientX > 0 && event.clientY > 0
      && event.clientX < window.innerWidth && event.clientY < window.innerHeight;
    if (!inWindow || !pointInsidePersonaRow(event.clientX, event.clientY)) {
      return;
    }
  }

  function startPersonaPointerDrag(event, persona) {
    if (event.button != null && event.button !== 0) return;
    personaPointerDragRef.current = {
      id: persona.id,
      startX: event.clientX,
      startY: event.clientY,
      moved: false,
      nativeStarted: false,
    };

    const cleanupPointerDrag = () => {
      window.removeEventListener('pointermove', handlePointerMove);
      window.removeEventListener('pointerup', handlePointerUp);
      window.removeEventListener('pointercancel', handlePointerCancel);
    };

    const beginNativePersonaDrag = () => {
      const drag = personaPointerDragRef.current;
      if (!drag || drag.id !== persona.id || drag.nativeStarted) return;
      drag.nativeStarted = true;
      cleanupPointerDrag();
      personaPointerDragRef.current = null;
      draggedPersonaIdRef.current = null;
      setDraggedPersonaId(null);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 120);
      const dragPromise = onPersonaNativeDrag?.(persona.id);
      if (!dragPromise) return;
      dragPromise.then((result) => {
        if (result?.status !== 'success' || !result?.landed_path) return;
        setPersonaExportDialog({
          ...result,
          id: result.persona_id || persona.id,
          name: result.display_name || persona.name,
        });
      });
    };

    const handlePointerMove = (moveEvent) => {
      const drag = personaPointerDragRef.current;
      if (!drag || drag.id !== persona.id) return;
      const moved = Math.hypot(moveEvent.clientX - drag.startX, moveEvent.clientY - drag.startY) > 8;
      if (!moved) return;
      drag.moved = true;
      suppressPersonaClickRef.current = true;
      draggedPersonaIdRef.current = persona.id;
      setDraggedPersonaId(persona.id);
      moveEvent.preventDefault();
      if (!pointInsidePersonaRow(moveEvent.clientX, moveEvent.clientY)) {
        beginNativePersonaDrag();
      }
    };

    const handlePointerUp = (upEvent) => {
      cleanupPointerDrag();
      const moved = Boolean(personaPointerDragRef.current?.moved);
      finishPersonaPointerDrag(upEvent, persona.id, moved);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 0);
    };

    const handlePointerCancel = () => {
      cleanupPointerDrag();
      personaPointerDragRef.current = null;
      draggedPersonaIdRef.current = null;
      setDraggedPersonaId(null);
      window.setTimeout(() => {
        suppressPersonaClickRef.current = false;
      }, 0);
    };

    window.addEventListener('pointermove', handlePointerMove, {passive: false});
    window.addEventListener('pointerup', handlePointerUp);
    window.addEventListener('pointercancel', handlePointerCancel);
  }

  function commitRoleStrength() {
    onPersonaChange(activePersona.id, {roleStrength: `${strengthDraft}%`});
    setStrengthPickerOpen(false);
  }

  function updateRoleStrength(value) {
    const next = parseToneChance(`${value}%`);
    setStrengthDraft(next);
    onPersonaChange(activePersona.id, {roleStrength: `${next}%`});
  }

  // v3.3.2 P0.1 — file drop routes through quarantine (onPersonaDrop), never directly.
  function installPersonaPackage(event) {
    event.preventDefault();
    if (!canAddPersona) return;
    const file = event.dataTransfer.files?.[0];
    if (!file) return;
    const reader = new FileReader();
    reader.onload = () => {
      onPersonaDrop(file.name, String(reader.result || ''));
    };
    reader.readAsText(file);
  }

  return (
    <main className="settings-persona-drawer" aria-label={t('persona.settingsAriaLabel')}>
      <div className="persona-card-row" ref={personaRowRef}>
        {personas.map((persona) => (
          <button
            className={`settings-persona-card ${persona.id === activePersona.id ? 'settings-persona-card-active' : ''} ${persona.id === draggedPersonaId ? 'settings-persona-card-dragging' : ''} ${persona.id === lockedPersonaId ? 'settings-persona-card-locked' : ''}`}
            type="button"
            key={persona.id}
            data-persona-id={persona.id}
            draggable={false}
            onDragStart={(event) => event.preventDefault()}
            onClick={(event) => {
              if (suppressPersonaClickRef.current) {
                event.preventDefault();
                event.stopPropagation();
                return;
              }
              commitActivePersonaForm();
              onPersonaChange(persona.id, {});
            }}
            onPointerDown={(event) => startPersonaPointerDrag(event, persona)}
          >
            <img
              className="settings-persona-avatar"
              draggable={false}
              onDragStart={(event) => event.preventDefault()}
              src={resolvePersonaAvatarSrc(persona, avatarConfigs[persona.id], staticAvatarPreviews, renderedPixelAvatars[persona.id] || '')}
              alt={t('persona.avatarAlt', { name: persona.name })}
            />
            <strong>{persona.name}</strong>
            {persona.id === lockedPersonaId && <small className="settings-persona-lock">{t('persona.lockedName')}</small>}
          </button>
        ))}
        {personaExportDialog && (
          <DragActionModal
            ariaLabel={t('persona.dragAriaLabel')}
            icon="◎"
            title={personaExportDialog.name}
            detail={personaExportDialog.landed_path || (personaExportDialog.id === lockedPersonaId ? t('persona.lockedPersonaExportNote') : t('persona.fileCopied'))}
            actions={[
              {
                label: t('persona.removeLocalLabel'),
                disabled: personaExportDialog.id === lockedPersonaId,
                onClick: () => handlePersonaExportAction('remove'),
              },
              {label: t('persona.keepCopyLabel'), onClick: () => handlePersonaExportAction('copy')},
              {label: t('persona.cancelDeleteLabel'), onClick: () => handlePersonaExportAction('cancel')},
            ]}
          />
        )}
        {canAddPersona && (
          <button
            className="settings-persona-card settings-persona-add-card"
            type="button"
            onClick={() => onPersonaAdd()}
            onDragOver={(event) => event.preventDefault()}
            onDrop={installPersonaPackage}
          >
            <span className="settings-persona-add-mark">＋</span>
            <strong>{t('persona.addPersona')}</strong>
            <small>{t('persona.dropRolepack')}</small>
          </button>
        )}
      </div>
      <div className="settings-card-track">
        <button type="button" onClick={() => scrollPersonas(-1)}>‹</button>
        <span>1-{Math.min(4, personaSlotCount)} / {personaSlotCount}</span>
        <button type="button" onClick={() => scrollPersonas(1)}>›</button>
      </div>
      <form
        className="persona-form"
        ref={personaFormRef}
        onFocusCapture={(event) => {
          if (strengthPickerOpen && !event.target.closest('.role-strength-field')) {
            commitRoleStrength();
          }
        }}
      >
        <section className="settings-avatar-editor" aria-label={t('persona.avatarEditorLabel', { name: activePersona.name })}>
          <button
            className={`settings-avatar-preview ${activeAvatarLocked ? 'settings-avatar-preview-locked' : ''}`}
            type="button"
            onClick={() => {
              if (!activeAvatarLocked) setAvatarPickerOpen((open) => !open);
            }}
            title={activeAvatarLocked ? t('avatar.locked') : t('persona.changeAvatarTitle')}
            aria-disabled={activeAvatarLocked}
          >
            <img src={activeAvatarSrc} alt={t('persona.avatarAlt', { name: activePersona.name })}/>
            <span>{avatarStateLabel(avatarExpression)}</span>
          </button>
          <div className="settings-avatar-copy">
            <strong>{t('persona.avatarTitle')}</strong>
            <small>{activeAvatarProvider === 'static_image' ? t('persona.staticAvatarLabel') : activeAvatarProvider === 'user_image_api' ? t('persona.genApiLabel') : t('persona.builtinAvatarLabel')}</small>
          </div>
          <button
            className="settings-avatar-action"
            type="button"
            disabled={activeAvatarLocked}
            onClick={() => setAvatarPickerOpen((open) => !open)}
          >
            {activeAvatarLocked ? t('persona.avatarLocked') : t('persona.changeAvatar')}
          </button>
          {avatarPickerOpen && !activeAvatarLocked && (
            <div className="avatar-provider-popover settings-avatar-popover">
              {/* Settings edits target the active persona card, while the main UI
                  avatar follows settingsState.personas[0]. */}
              <button type="button" onClick={() => onAvatarProviderSelect?.('user_image_api', activePersona.id)}>
                <strong>{t('persona.genApiLabel')}</strong>
                <small>{t('persona.genApiHint')}</small>
              </button>
              <button type="button" onClick={() => onAvatarProviderSelect?.('static_image', activePersona.id)}>
                <strong>{t('persona.staticAvatarLabel')}</strong>
                <small>{t('persona.staticAvatarHint')}</small>
              </button>
              <button type="button" onClick={() => onAvatarProviderSelect?.('built_in_pixel', activePersona.id)}>
                <strong>{t('persona.builtinAvatarLabel')}</strong>
                <small>{t('persona.builtinAvatarHint')}</small>
              </button>
              <div className="avatar-state-grid" aria-label={t('persona.avatarStatePreview')}>
                {avatarStateOptions.map((state) => (
                  <button
                    className={state === avatarExpression ? 'avatar-state-active' : ''}
                    type="button"
                    key={state}
                    onClick={() => onAvatarStateSelect?.(state)}
                  >
                    {avatarStateLabel(state)}
                  </button>
                ))}
                <button type="button" onClick={() => onAvatarStateSelect?.('')}>{t('persona.autoState')}</button>
                <button
                  type="button"
                  className="pixel-pack-switch-btn"
                  onClick={() => setPixelPackPopup2((v) => !v)}
                  title={t('persona.togglePixelPack')}
                >
                  {t('persona.pixelPack')}
                </button>
              </div>
              {pixelPackPopup2 && (
                <div className="pixel-pack-popup">
                  <div className="pixel-pack-popup-title">{t('persona.selectPixelPack')}</div>
                  <div className="pixel-pack-list">
                    {pixelPacks2.map((p) => (
                      <button
                        key={p.id}
                        type="button"
                        className={`pixel-pack-card${p.id === currentPack2 ? ' pixel-pack-active' : ''}`}
                        onClick={() => switchPixelPack2(p.id)}
                      >
                        {packPreviews2[p.id] && (
                          <img
                            src={packPreviews2[p.id]}
                            alt={p.name}
                            className="pixel-pack-preview"
                          />
                        )}
                        <span className="pixel-pack-name">{p.name}</span>
                        <span className="pixel-pack-desc">{p.desc}</span>
                      </button>
                    ))}
                  </div>
                  <button
                    type="button"
                    className="pixel-pack-close"
                    onClick={() => setPixelPackPopup2(false)}
                  >
                    {t('persona.closePopup')}
                  </button>
                </div>
              )}
              <p>{avatarModeNotice || t('persona.currentProvider', { provider: activeAvatarProvider })}</p>
            </div>
          )}
        </section>
        <input
          aria-label={t('persona.nameAriaLabel')}
          defaultValue={activePersona.id === lockedPersonaId ? lockedPersonaName : activePersona.name}
          disabled={activePersona.id === lockedPersonaId}
          key={`${activePersona.id}-name`}
          name="personaName"
          placeholder={t('persona.namePlaceholder')}
          maxLength={100}
          onBlur={(event) => {
            const value = limitChineseText(event.target.value, 100);
            event.target.value = value;
            if (activePersona.id === lockedPersonaId) return;
            onPersonaChange(activePersona.id, {name: value});
          }}
        />
        <input
          aria-label={t('persona.identityAriaLabel')}
          defaultValue={activePersona.identity}
          key={`${activePersona.id}-identity`}
          name="identity"
          placeholder={t('persona.identityPlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {identity: event.target.value})}
        />
        <div className="persona-form-grid">
          <label className="persona-strategy-select" aria-label={t('persona.replyStrategyAriaLabel')}>
            <select
              defaultValue={activeReplyStrategy}
              key={`${activePersona.id}-reply`}
              name="replyStrategy"
              onChange={(event) => onPersonaChange(activePersona.id, {replyStrategy: event.target.value})}
            >
              <option value="">{t('persona.replyStrategyDefault')}</option>
              {getReplyStrategyPresets().map((preset) => (
                <option key={preset.id} value={preset.id}>{preset.label}</option>
              ))}
              {!activeReplyStrategyIsPreset && (
                <option value={activeReplyStrategy}>{t('persona.customStrategy', { value: activeReplyStrategy })}</option>
              )}
            </select>
          </label>
          <div
            className="role-strength-field"
            onBlurCapture={(event) => {
              if (!event.currentTarget.contains(event.relatedTarget)) {
                commitRoleStrength();
              }
            }}
          >
            <input
              name="roleStrength"
              type="hidden"
              value={`${strengthDraft}%`}
              readOnly
            />
            <button
              aria-expanded={strengthPickerOpen}
              aria-label={t('persona.roleStrengthAriaLabel')}
              className="role-strength-trigger"
              type="button"
              onClick={() => setStrengthPickerOpen((open) => !open)}
            >
              <span>{t('persona.roleStrengthLabel')}</span>
              <strong>{strengthDraft}%</strong>
            </button>
            {strengthPickerOpen && (
              <div className="role-strength-popover">
                <input
                  aria-label={t('persona.toneProbAriaLabel')}
                  max="100"
                  min="10"
                  step="10"
                  type="range"
                  value={strengthDraft}
                  onChange={(event) => updateRoleStrength(event.target.value)}
                />
                <div className="role-strength-ticks">
                  <span>10%</span>
                  <span>50%</span>
                  <span>100%</span>
                </div>
                <small>{t('persona.tonePromptHint')}</small>
              </div>
            )}
          </div>
          <input
            aria-label={t('persona.personalityAriaLabel')}
            defaultValue={activePersona.personality}
            key={`${activePersona.id}-personality`}
            name="personality"
            placeholder={t('persona.personalityPlaceholder')}
            onBlur={(event) => onPersonaChange(activePersona.id, {personality: event.target.value})}
          />
          <input
            aria-label={t('persona.scenarioAriaLabel')}
            defaultValue={activePersona.scenario}
            key={`${activePersona.id}-scenario`}
            name="scenario"
            placeholder={t('persona.scenePlaceholder')}
            onBlur={(event) => onPersonaChange(activePersona.id, {scenario: event.target.value})}
          />
        </div>
        <textarea
          aria-label={t('persona.descriptionAriaLabel')}
          defaultValue={activePersona.description}
          key={`${activePersona.id}-description`}
          name="description"
          placeholder={t('persona.extraPlaceholder')}
          onBlur={(event) => onPersonaChange(activePersona.id, {description: event.target.value})}
        />
      </form>
      <button className="settings-bottom-action" type="button">↓</button>
    </main>
  );
}
