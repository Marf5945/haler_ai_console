import {afterEach, beforeEach, describe, expect, it, vi} from 'vitest';
import {createInochiParamDriver} from './inochiParamDriver.js';

const PARAMS = [
  'Blink', 'HeadYaw', 'HeadNod', 'EyeTracking', 'MouthOpen',
  'Breath', 'BodySway', 'TailSway', 'ArmNearSway', 'HandSway',
  'LegNearSway', 'LegFarSway', 'EarWiggle',
  'WalkPhase', 'ArmRaise', 'PetResponse', 'Tickle',
];

function fakeAvatar() {
  return {
    values: new Map(),
    values2d: new Map(),
    setParameter(name, value) { this.values.set(name, value); },
    setParameter2D(name, x, y) { this.values2d.set(name, [x, y]); },
  };
}

describe('createInochiParamDriver', () => {
  beforeEach(() => vi.spyOn(Math, 'random').mockReturnValue(0));
  afterEach(() => vi.restoreAllMocks());

  it('drives pet and tickle through Inochi parameters', () => {
    const avatar = fakeAvatar();
    const driver = createInochiParamDriver({paramNames: PARAMS});

    expect(driver.matched).toEqual(expect.arrayContaining([
      'WalkPhase', 'ArmRaise', 'PetResponse', 'Tickle',
    ]));

    driver.setInteraction({pet: 1, tickle: 1});
    driver.update(avatar, 0.1);

    expect(avatar.values.get('ArmRaise')).toBeGreaterThan(0);
    expect(avatar.values.get('PetResponse')).toBeGreaterThan(0);
    expect(Math.abs(avatar.values.get('Tickle'))).toBeGreaterThan(0);
    expect(avatar.values.get('WalkPhase')).toBe(0);
  });

  it('runs a synchronized ambient walk and returns to neutral on interaction', () => {
    const avatar = fakeAvatar();
    const driver = createInochiParamDriver({paramNames: PARAMS});
    const observed = [];
    const observedLeg = [];

    driver.triggerAmbient('walk', 5);
    for (let index = 0; index < 10; index += 1) {
      driver.update(avatar, 0.1);
      observed.push(avatar.values.get('WalkPhase'));
      observedLeg.push(Math.abs(avatar.values.get('LegNearSway')));
    }
    expect(observed.some((value) => value > 0)).toBe(true);
    expect(observedLeg.some((value) => value > 0.4)).toBe(true);

    driver.setInteraction({pet: 1});
    driver.update(avatar, 0.1);
    expect(avatar.values.get('WalkPhase')).toBe(0);
  });

  it('temporarily applies the existing working_reaction expression', () => {
    const avatar = fakeAvatar();
    const driver = createInochiParamDriver({
      paramNames: PARAMS,
      expressions: {
        working_reaction: {
          params: {MouthOpen: 0.3, HeadNod: 0.15},
          motion: {TailSway: {speed: 3, min: -0.8, max: 0.8}},
        },
      },
    });

    driver.triggerAmbient('working_reaction', 2.5);
    driver.update(avatar, 0.1);
    expect(avatar.values.get('MouthOpen')).toBeGreaterThan(0);
    expect(avatar.values.get('HeadNod')).toBeGreaterThan(0);
  });
});
