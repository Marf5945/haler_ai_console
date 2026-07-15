import {describe, expect, it} from 'vitest';
import {resolveAvatarMotionManifest} from './avatarMotionRegistry.js';

describe('yulesaku Inochi2D assets', () => {
  it('hydrates every normalized stepping frame and all three INP anchors', () => {
    const manifest = resolveAvatarMotionManifest('yulesaku', 'idle');
    expect(manifest.renderer).toBe('inochi2d');
    expect(Object.values(manifest.inochi2d.anchors).every((anchor) => anchor.modelSrc)).toBe(true);
    expect(manifest.inochi2d.transitions.front_000_to_left_030).toHaveLength(12);
    expect(manifest.inochi2d.transitions.front_000_to_right_030).toHaveLength(12);
    expect(manifest.inochi2d.transitions.front_000_to_left_030.every((frame) => frame.src && frame.hip)).toBe(true);
    expect(manifest.inochi2d.transitions.front_000_to_right_030.every((frame) => frame.src && frame.hip)).toBe(true);
    expect(manifest.inochi2d.transitions.front_000_to_left_030.every((frame) => frame.coloredHeight > 0)).toBe(true);
    expect(manifest.inochi2d.transitions.front_000_to_right_030.every((frame) => frame.coloredHeight > 0)).toBe(true);
  });
});
