import React from 'react';
import {act, fireEvent, render} from '@testing-library/react';
import {afterEach, describe, expect, it, vi} from 'vitest';

const inochiMock = vi.hoisted(() => ({instances: [], setParameter: vi.fn()}));

vi.mock('inochi-avatar', () => ({
  InochiAvatar: class MockInochiAvatar {
    constructor() {
      inochiMock.instances.push(this);
      this.model = {
        get_parameter_names: () => [
          'MouthOpen', 'HeadNod', 'TailSway', 'Breath', 'BodySway',
          'ArmNearSway', 'HandSway', 'LegNearSway', 'LegFarSway', 'WalkPhase',
        ],
      };
    }
    async load() {}
    setCameraZoom() {}
    setCameraPosition() {}
    play() {}
    stop() {}
    setParameter(...args) { inochiMock.setParameter(...args); }
    setParameter2D() {}
  },
}));

import InochiAvatarStage, {transitionFrames} from './InochiAvatarStage.jsx';
import yulesakuManifest from '../../assets/persona_motion/yulesaku/manifest.json';

const config = {
  runtimeUrl: '/runtime.js',
  transitionDurationMs: 1500,
  anchors: {
    front_000: {modelSrc: '/front.inp'},
    left_030: {modelSrc: '/left.inp'},
    right_030: {modelSrc: '/right.inp'},
  },
  transitions: {
    front_000_to_left_030: ['/left-010.png', '/left-020.png', '/left-030.png'],
    front_000_to_right_030: ['/right-010.png', '/right-020.png', '/right-030.png'],
  },
};

afterEach(() => {
  vi.useRealTimers();
  vi.restoreAllMocks();
  inochiMock.instances.length = 0;
  inochiMock.setParameter.mockClear();
});

describe('InochiAvatarStage turn transition', () => {
  it('uses 12 pelvis-anchored authored poses per side at 12 fps', () => {
    const transitions = yulesakuManifest.inochi2d.transitions;
    expect(yulesakuManifest.inochi2d.transitionDurationMs).toBe(1200);
    expect(yulesakuManifest.inochi2d.randomAnimationIntervalMs).toEqual([20000, 30000]);
    expect(yulesakuManifest.inochi2d.attention.returnDelayMs).toBe(8000);
    expect(yulesakuManifest.inochi2d.attention.returnDurationMs).toBe(3000);
    expect(yulesakuManifest.inochi2d.randomAnimations.map((item) => item.id))
      .toEqual(['working_reaction', 'walk']);
    // working_reaction 的 ArmRaise 實際出貨值為 0.5（manifest 與既有 bundle 一致）。
    expect(yulesakuManifest.inochi2d.expressions.working_reaction.params.ArmRaise).toBe(0.5);
    expect(transitions.front_000_to_left_030).toHaveLength(12);
    expect(transitions.front_000_to_right_030).toHaveLength(12);
    expect(transitions.front_000_to_left_030[0].src).toContain('turn_left_010_step2.png');
    expect(transitions.front_000_to_left_030[11].src).toContain('turn_left_030.png');
    expect(transitions.front_000_to_left_030.every((frame) => frame.hip)).toBe(true);
  });

  it('reverses the same normalized frames when returning to front', () => {
    expect(transitionFrames(config, 'left_030', 'front_000')).toEqual([
      '/left-030.png', '/left-020.png', '/left-010.png',
    ]);
  });

  it('shows one turn pose at a time and keeps the 30 degree anchor on pointer leave', async () => {
    vi.useFakeTimers();
    const view = render(<InochiAvatarStage config={config} width={200} height={360} />);
    await act(async () => {});
    const stage = view.container.querySelector('.floating-avatar-inochi-stage');
    stage.getBoundingClientRect = () => ({left: 0, top: 0, width: 200, height: 360});

    fireEvent.pointerMove(stage, {clientX: 0, clientY: 180});
    expect(view.container.querySelector('.floating-avatar-inochi-transition-current'))
      .toHaveAttribute('src', '/left-010.png');

    expect(view.container.querySelector('.floating-avatar-inochi-transition-previous')).toBeNull();
    expect(view.container.querySelectorAll('.floating-avatar-inochi-canvas[style*="opacity: 1"]')).toHaveLength(0);

    act(() => vi.advanceTimersByTime(1000));
    fireEvent.pointerLeave(stage);
    act(() => vi.advanceTimersByTime(1600));
    expect(view.container.querySelector('.floating-avatar-inochi-transition-current'))
      .toBeNull();
    expect(view.container.querySelector(`#${stage.querySelectorAll('canvas')[1].id}`))
      .toHaveStyle({opacity: '1'});
  });

  it('normalizes painted character height uniformly around the authored hip', async () => {
    vi.useFakeTimers();
    const proportionConfig = {
      ...config,
      transitionCanvas: [200, 360],
      transitionHip: [92, 196.25],
      targetColoredHeight: 352,
      transitions: {
        ...config.transitions,
        front_000_to_left_030: [
          {src: '/small-left-030.png', hip: [86, 200], coloredHeight: 306},
        ],
      },
    };
    const view = render(<InochiAvatarStage config={proportionConfig} width={200} height={360} />);
    await act(async () => {});
    const stage = view.container.querySelector('.floating-avatar-inochi-stage');
    stage.getBoundingClientRect = () => ({left: 0, top: 0, width: 200, height: 360});

    fireEvent.pointerMove(stage, {clientX: 0, clientY: 180});
    const frame = view.container.querySelector('.floating-avatar-inochi-transition-current');
    expect(frame).toHaveAttribute('src', '/small-left-030.png');
    expect(frame.style.transform).toContain(`scale(${352 / 306})`);
    expect(frame.style.transformOrigin).toContain('43% 55.555');
  });

  it('dispatches one shared random working reaction after the configured idle interval', async () => {
    vi.useFakeTimers();
    vi.spyOn(Math, 'random').mockReturnValue(0);
    const randomConfig = {
      ...config,
      randomAnimationIntervalMs: [1000, 1000],
      randomAnimationAllowedExpressions: ['idle'],
      randomAnimations: [
        {id: 'working_reaction', durationMs: 2500},
        {id: 'walk', durationMs: 5000},
      ],
      expressions: {
        idle: {},
        working_reaction: {
          params: {MouthOpen: 0.3, HeadNod: 0.15},
          motion: {TailSway: {speed: 3, min: -0.8, max: 0.8}},
        },
      },
    };
    render(<InochiAvatarStage config={randomConfig} expression="idle" width={200} height={360} />);
    await act(async () => {});

    act(() => vi.advanceTimersByTime(1000));
    inochiMock.instances.forEach((avatar) => avatar.onCustomAnimate?.(0.1));

    expect(inochiMock.instances).toHaveLength(3);
    expect(inochiMock.setParameter.mock.calls.some(
      ([name, value]) => name === 'MouthOpen' && value > 0,
    )).toBe(true);
  });
});
