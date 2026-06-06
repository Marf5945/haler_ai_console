import React from 'react';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {cleanup, render, waitFor} from '@testing-library/react';
import App from '../App.jsx';

function defaultBindingValue(name) {
  if (name === 'ListReferenceFiles') {
    return [{name: '測試用教學文件-001.txt', path: '/tmp/測試用教學文件-001.txt', source: 'library', status: 'ready', detail: '已從引用庫載入'}];
  }
  if (/^(List|Scan|GetRecent|GetPending|GetNew|GetToolRegistry|GetTalkMessages)/.test(name)) return [];
  if (/^(Has|Is|Can)/.test(name)) return false;
  if (name === 'GetConsoleState') return {greeting: 'ready', adapters: [], haoras: ['main'], messages: []};
  if (name === 'GetSettingsState') return null;
  if (name === 'GetToolVisibility') return {};
  if (name === 'GetAppSessionID') return 'test-session';
  if (name === 'GetDegradedState') return {active: false, blocked_ops: []};
  if (name === 'GetReadinessGateState') return {phase: 'normal'};
  if (name === 'GetVoiceSettings') return null;
  if (name === 'GetMemoryHealth') return null;
  if (name === 'GetConfigPublic') return null;
  if (name === 'GetCredentialMigrationStatus') return null;
  if (name === 'GetSummaryModelSettings') return null;
  if (name === 'GetMemoryPipelineState') return null;
  if (name === 'GetBrowserPreference') return null;
  if (name === 'GetUISettings') return {};
  if (name === 'GetOnboardingState') return null;
  if (name === 'PollStatusRail') return null;
  return undefined;
}

describe('App smoke render', () => {
  beforeEach(() => {
    const bindings = new Map();
    window.go = {
      main: {
        App: new Proxy({}, {
          get(_target, prop) {
            if (typeof prop !== 'string') return undefined;
            if (!bindings.has(prop)) {
              bindings.set(prop, vi.fn(() => Promise.resolve(defaultBindingValue(prop))));
            }
            return bindings.get(prop);
          },
        }),
      },
    };
    window.runtime = {
      BrowserOpenURL: vi.fn(),
      ClipboardGetText: vi.fn(() => Promise.resolve('')),
      ClipboardSetText: vi.fn(() => Promise.resolve()),
      EventsOn: vi.fn(() => () => {}),
      EventsOnMultiple: vi.fn(() => () => {}),
      EventsOff: vi.fn(),
      EventsOffAll: vi.fn(),
      OnFileDrop: vi.fn(),
      OnFileDropOff: vi.fn(),
    };
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('mounts the main shell without a first-render crash', async () => {
    render(<App />);
    await waitFor(() => {
      expect(document.querySelector('.console-shell')).toBeInTheDocument();
    });
  });

  it('renders reference cards without vectorizer badges', async () => {
    render(<App />);
    await waitFor(() => {
      expect(document.querySelector('.reference-file-name')).toBeInTheDocument();
      expect(document.querySelector('.reference-file-vectorizer-badge')).not.toBeInTheDocument();
    });
  });

  it('keeps loaded library reference cards compact', async () => {
    render(<App />);
    await waitFor(() => {
      expect(document.querySelector('.reference-file-status')).toHaveTextContent('已載入');
      expect(document.querySelector('.reference-file-detail')).not.toBeInTheDocument();
    });
  });
});
