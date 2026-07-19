export const VOICE_SYNC_LIMITS = Object.freeze({
  shortSilenceMs: 800,
  longSilenceMs: 2000,
  energyThreshold: 0.012,
  minSpeechMs: 250,
  noSpeechAbortMs: 15000,
});

export function createVoiceSyncSession(now = Date.now()) {
  return {
    active: true,
    phase: 'listening',
    lastVoiceAt: now,
    spokeMs: 0,
    committing: false,
    startedAt: now,
  };
}

// Pure transition function for Voice Sync R1. Keeping timing decisions out of
// the Web Audio callback makes the 0.8s/2s/15s boundaries deterministic and
// independently testable without microphone hardware.
export function advanceVoiceSyncSession(session, sample, limits = VOICE_SYNC_LIMITS) {
  if (!session?.active || session.committing) {
    return {session, action: 'none', phaseChanged: false};
  }
  const now = Number.isFinite(sample?.now) ? sample.now : Date.now();
  const rms = Number.isFinite(sample?.rms) ? sample.rms : 0;
  const blockMs = Math.max(0, Number(sample?.blockMs) || 0);
  const next = {...session};

  if (rms >= limits.energyThreshold) {
    next.lastVoiceAt = now;
    next.spokeMs += blockMs;
    const phaseChanged = next.phase !== 'listening';
    next.phase = 'listening';
    return {session: next, action: 'none', phaseChanged};
  }

  const silentMs = now - next.lastVoiceAt;
  if (next.spokeMs >= limits.minSpeechMs && silentMs >= limits.longSilenceMs) {
    return {session: next, action: 'commit', phaseChanged: false};
  }
  if (next.spokeMs < limits.minSpeechMs && now - next.startedAt >= limits.noSpeechAbortMs) {
    return {session: next, action: 'abort', phaseChanged: false};
  }
  if (next.spokeMs >= limits.minSpeechMs && silentMs >= limits.shortSilenceMs && next.phase !== 'short_pause') {
    next.phase = 'short_pause';
    return {session: next, action: 'none', phaseChanged: true};
  }
  return {session: next, action: 'none', phaseChanged: false};
}
