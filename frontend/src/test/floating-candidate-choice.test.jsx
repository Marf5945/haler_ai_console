import React from 'react';
import {readFileSync} from 'node:fs';
import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {cleanup, fireEvent, render, screen, waitFor} from '@testing-library/react';
import App from '../App.jsx';

const searchCandidates = [
  {id: 'search-local', label: 'Local search', draft: 'local search'},
  {id: 'search-web', label: 'Web search', draft: 'web search'},
  {id: 'search-vcs', label: 'Git search', draft: 'git search'},
];

const yesNoCandidates = [
  {id: 'answer-yes', label: 'Yes', draft: 'yes'},
  {id: 'answer-no', label: 'No', draft: 'no'},
];

const confirmCancelCandidates = [
  {id: 'answer-confirm', label: 'Confirm', draft: 'confirm'},
  {id: 'answer-cancel', label: 'Cancel', draft: 'cancel'},
];

function defaultBindingValue(name, fixtures) {
  if (name === 'GetReadinessGateState') {
    return fixtures.readinessGate;
  }
  if (name === 'GetConsoleState') {
    return {
      greeting: 'ready',
      adapters: ['main'],
      active_adapter_id: 'main',
      haoras: ['main'],
      messages: ['Ai: ready'],
    };
  }
  if (name === 'GetSettingsState') {
    return {
      panel: {
        panelLanguage: 'Traditional Chinese',
        roleLanguage: 'Auto',
        fontPreset: 'Default',
        fontScale: '100%',
        panelStyle: fixtures.panelStyle,
      },
      activePersonaId: 'persona-a',
      personas: [{id: 'persona-a', name: 'Tester', icon: '', avatarUrl: '', identity: '', replyStrategy: '', roleStrength: '20%', personality: '', scenario: '', description: ''}],
    };
  }
  if (name === 'GetAppSessionID') return 'test-session';
  if (name === 'GetDegradedState') return {active: false, blocked_ops: []};
  if (name === 'GetToolVisibility') return {};
  if (name === 'GetVoiceSettings') return {status: 'ready', settings: {}};
  if (name === 'GetUISettings') return {};
  if (name === 'GetBrowserPreference') return null;
  if (name === 'GetMemoryHealth') return null;
  if (name === 'GetConfigPublic') return null;
  if (name === 'GetCredentialMigrationStatus') return null;
  if (name === 'GetSummaryModelSettings') return null;
  if (name === 'GetMemoryPipelineState') return null;
  if (name === 'GetOnboardingState') return null;
  if (/^(List|Scan|GetRecent|GetPending|GetNew|GetToolRegistry|GetTalkMessages)/.test(name)) return [];
  if (/^(Has|Is|Can)/.test(name)) return false;
  if (name === 'EscapeExternalTokens') return fixtures.lastEscapeText || '';
  if (name === 'StageSessionImages') return null;
  if (name === 'ExecuteSkillMessage') return {text: 'ok'};
  if (name === 'DismissFloatingCandidates') return null;
  if (name === 'PersistTalkMessage') return null;
  return undefined;
}

function installWailsMocks(fixtures) {
  const bindings = new Map();
  const calls = {};
  window.go = {
    main: {
      App: new Proxy({}, {
        get(_target, prop) {
          if (typeof prop !== 'string') return undefined;
          if (!bindings.has(prop)) {
            const fn = vi.fn((...args) => {
              if (!calls[prop]) calls[prop] = [];
              calls[prop].push(args);
              if (prop === 'EscapeExternalTokens') fixtures.lastEscapeText = args[0] || '';
              return Promise.resolve(defaultBindingValue(prop, fixtures));
            });
            bindings.set(prop, fn);
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
  return {calls};
}

async function renderChoiceApp({candidates, panelStyle = 'default'}) {
  const fixtures = {
    panelStyle,
    readinessGate: {
      risk_tier: 'none',
      missing_slots: [],
      floating_candidates: candidates,
      retrieval_scanning: false,
      retrieval_sources: [],
    },
  };
  const {calls} = installWailsMocks(fixtures);
  const view = render(<App />);
  await waitFor(() => expect(screen.getByText(candidates[0].label)).toBeInTheDocument());
  return {calls, ...view};
}

function sendButton() {
  const button = document.querySelector('.send-btn');
  expect(button).toBeInTheDocument();
  return button;
}

function shell() {
  const node = document.querySelector('.console-shell');
  expect(node).toBeInTheDocument();
  return node;
}

async function submitSelectedText(expectedText, calls) {
  fireEvent.click(sendButton());
  await waitFor(() => expect(calls.ExecuteSkillMessage?.[0]?.[2]).toBe(expectedText));
  expect(calls.DismissFloatingCandidates?.length || 0).toBeGreaterThan(0);
}

describe('floating candidate choice flow', () => {
  beforeEach(() => {
    localStorage.clear();
    Element.prototype.scrollTo = vi.fn();
  });

  afterEach(() => {
    cleanup();
    vi.restoreAllMocks();
  });

  it('multi-selects search choices, highlights them, enables send, and submits selected text', async () => {
    const {calls} = await renderChoiceApp({candidates: searchCandidates, panelStyle: 'default'});
    const local = screen.getByRole('button', {name: 'Local search'});
    const web = screen.getByRole('button', {name: 'Web search'});
    const send = sendButton();

    expect(local).toHaveClass('floating-candidate-btn');
    expect(send).not.toHaveClass('send-btn-ready');

    fireEvent.click(local);
    expect(local).toHaveClass('is-selected');
    expect(local).toHaveAttribute('aria-pressed', 'true');
    expect(send).toHaveClass('send-btn-ready');

    fireEvent.click(web);
    expect(local).toHaveClass('is-selected');
    expect(web).toHaveClass('is-selected');

    fireEvent.click(send);
    await waitFor(() => expect(calls.ExecuteSkillMessage?.[0]?.[2]).toBe('local search\nweb search'));
    expect(calls.DismissFloatingCandidates?.length || 0).toBeGreaterThan(0);
  });

  it('runs the 8-round choice dialogue across default and switched themes', async () => {
    const roundLog = [];

    let view = await renderChoiceApp({candidates: searchCandidates, panelStyle: 'default'});
    roundLog.push('1 default search choices appeared');
    expect(screen.getByRole('button', {name: 'Local search'})).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Web search'})).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Git search'})).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', {name: 'Local search'}));
    fireEvent.click(screen.getByRole('button', {name: 'Web search'}));
    roundLog.push('2 default search multi-select highlighted and sent');
    expect(screen.getByRole('button', {name: 'Local search'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'Web search'})).toHaveClass('is-selected');
    expect(sendButton()).toHaveClass('send-btn-ready');
    await submitSelectedText('local search\nweb search', view.calls);
    cleanup();

    view = await renderChoiceApp({candidates: yesNoCandidates, panelStyle: 'default'});
    fireEvent.click(screen.getByRole('button', {name: 'Yes'}));
    fireEvent.click(screen.getByRole('button', {name: 'No'}));
    roundLog.push('3 default yes/no stayed exclusive and sent no');
    expect(screen.getByRole('button', {name: 'No'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'Yes'})).not.toHaveClass('is-selected');
    await submitSelectedText('no', view.calls);
    cleanup();

    view = await renderChoiceApp({candidates: confirmCancelCandidates, panelStyle: 'default'});
    fireEvent.click(screen.getByRole('button', {name: 'Confirm'}));
    fireEvent.click(screen.getByRole('button', {name: 'Cancel'}));
    roundLog.push('4 default confirm/cancel stayed exclusive and sent cancel');
    expect(screen.getByRole('button', {name: 'Cancel'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'Confirm'})).not.toHaveClass('is-selected');
    await submitSelectedText('cancel', view.calls);
    cleanup();

    view = await renderChoiceApp({candidates: searchCandidates, panelStyle: 'forgiveMeGreen'});
    roundLog.push('5 switched theme to green and search choices appeared');
    expect(shell()).toHaveAttribute('data-theme', 'green');
    expect(screen.getByRole('button', {name: 'Local search'})).toBeInTheDocument();
    expect(screen.getByRole('button', {name: 'Web search'})).toBeInTheDocument();

    fireEvent.click(screen.getByRole('button', {name: 'Local search'}));
    fireEvent.click(screen.getByRole('button', {name: 'Web search'}));
    roundLog.push('6 green search multi-select highlighted and sent');
    expect(screen.getByRole('button', {name: 'Local search'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'Web search'})).toHaveClass('is-selected');
    expect(sendButton()).toHaveClass('send-btn-ready');
    await submitSelectedText('local search\nweb search', view.calls);
    cleanup();

    view = await renderChoiceApp({candidates: yesNoCandidates, panelStyle: 'forgiveMeGreen'});
    fireEvent.click(screen.getByRole('button', {name: 'No'}));
    fireEvent.click(screen.getByRole('button', {name: 'Yes'}));
    roundLog.push('7 green yes/no stayed exclusive and sent yes');
    expect(shell()).toHaveAttribute('data-theme', 'green');
    expect(screen.getByRole('button', {name: 'Yes'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'No'})).not.toHaveClass('is-selected');
    await submitSelectedText('yes', view.calls);
    cleanup();

    view = await renderChoiceApp({candidates: confirmCancelCandidates, panelStyle: 'forgiveMeGreen'});
    fireEvent.click(screen.getByRole('button', {name: 'Cancel'}));
    fireEvent.click(screen.getByRole('button', {name: 'Confirm'}));
    roundLog.push('8 green confirm/cancel stayed exclusive and sent confirm');
    expect(shell()).toHaveAttribute('data-theme', 'green');
    expect(screen.getByRole('button', {name: 'Confirm'})).toHaveClass('is-selected');
    expect(screen.getByRole('button', {name: 'Cancel'})).not.toHaveClass('is-selected');
    await submitSelectedText('confirm', view.calls);

    expect(roundLog).toEqual([
      '1 default search choices appeared',
      '2 default search multi-select highlighted and sent',
      '3 default yes/no stayed exclusive and sent no',
      '4 default confirm/cancel stayed exclusive and sent cancel',
      '5 switched theme to green and search choices appeared',
      '6 green search multi-select highlighted and sent',
      '7 green yes/no stayed exclusive and sent yes',
      '8 green confirm/cancel stayed exclusive and sent confirm',
    ]);
  });

  it('keeps yes/no choices mutually exclusive', async () => {
    await renderChoiceApp({candidates: yesNoCandidates, panelStyle: 'default'});
    const yes = screen.getByRole('button', {name: 'Yes'});
    const no = screen.getByRole('button', {name: 'No'});

    fireEvent.click(yes);
    expect(yes).toHaveClass('is-selected');
    fireEvent.click(no);
    expect(no).toHaveClass('is-selected');
    expect(yes).not.toHaveClass('is-selected');
  });

  it('keeps confirm/cancel choices mutually exclusive', async () => {
    await renderChoiceApp({candidates: confirmCancelCandidates, panelStyle: 'default'});
    const confirm = screen.getByRole('button', {name: 'Confirm'});
    const cancel = screen.getByRole('button', {name: 'Cancel'});

    fireEvent.click(confirm);
    expect(confirm).toHaveClass('is-selected');
    fireEvent.click(cancel);
    expect(cancel).toHaveClass('is-selected');
    expect(confirm).not.toHaveClass('is-selected');
  });

  it('keeps selection highlight and submit working after switching themes', async () => {
    const {calls} = await renderChoiceApp({candidates: searchCandidates, panelStyle: 'forgiveMeGreen'});
    expect(document.querySelector('.console-shell')).toHaveAttribute('data-theme', 'green');

    const local = screen.getByRole('button', {name: 'Local search'});
    const web = screen.getByRole('button', {name: 'Web search'});
    const send = sendButton();
    fireEvent.click(local);
    fireEvent.click(web);

    expect(local).toHaveClass('is-selected');
    expect(web).toHaveClass('is-selected');
    expect(send).toHaveClass('send-btn-ready');

    fireEvent.click(send);
    await waitFor(() => expect(calls.ExecuteSkillMessage?.[0]?.[2]).toBe('local search\nweb search'));
  });

  it('keeps selected and send-ready visual affordances in CSS', () => {
    const css = readFileSync('src/style.css', 'utf8');

    expect(css).toContain('.console-shell[data-theme] .floating-candidate-btn.is-selected');
    expect(css).toContain('--floating-candidate-selected-bg: #fff0a6');
    expect(css).toContain('--floating-candidate-selected-bg: #ffd0e8');
    expect(css).toContain('--floating-candidate-selected-bg: #eaffd2');
    expect(css).toContain('--floating-candidate-selected-bg: #e8f3ff');
    expect(css).toContain('animation: sendReadyRing 1.05s ease-in-out infinite');
    expect(css).toContain('@keyframes sendReadyRing');
  });

  it('bursts readiness polling after composer submit so late candidates surface quickly', () => {
    const app = readFileSync('src/App.jsx', 'utf8');

    expect(app).toContain('function scheduleReadinessGateBurstRefresh');
    expect(app).toContain('durationMs = 12000');
    expect(app).toContain('intervalMs = 700');
    expect(app).toContain('scheduleReadinessGateBurstRefresh();');
  });
});
