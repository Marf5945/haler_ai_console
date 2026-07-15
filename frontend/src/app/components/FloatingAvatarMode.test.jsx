import React from 'react';
import {render, waitFor} from '@testing-library/react';
import {describe, expect, it, vi} from 'vitest';

vi.mock('./InochiAvatarStage.jsx', async () => {
  const ReactModule = await import('react');
  return {
    default: function MockInochiAvatarStage({onReady, expression}) {
      ReactModule.useEffect(() => {
        const timer = window.setTimeout(() => onReady?.(), 0);
        return () => window.clearTimeout(timer);
      }, [onReady]);
      return <span data-testid="inochi-stage" data-expression={expression} />;
    },
  };
});

vi.mock('./PixiAvatarStage.jsx', () => ({
  default: () => <span data-testid="pixi-stage" />,
}));

import FloatingAvatarMode from './FloatingAvatarMode.jsx';

const manifest = {
  id: 'yulesaku',
  renderer: 'inochi2d',
  baseSrc: '/static-fullbody.png',
  inochi2d: {
    anchors: {front_000: {modelSrc: '/front.inp'}},
  },
};

function props(expression) {
  return {
    active: true,
    t: (key) => key,
    avatarSrc: '/head.png',
    fullBodyAvatarSrc: '/static-fullbody.png',
    fullBodyAvatarKey: 'yulesaku-full',
    fullBodyMotionManifest: manifest,
    avatarExpression: expression,
    persona: {id: 'yulesaku', name: 'Yulesaku'},
    position: {x: 12, y: 180},
    onPositionChange: vi.fn(),
    bodyMode: 'full',
    dynamicImageEnabled: true,
  };
}

describe('FloatingAvatarMode Inochi backstop', () => {
  it('does not restore the static full-body image when only expression changes', async () => {
    const view = render(<FloatingAvatarMode {...props('idle')} />);

    await waitFor(() => {
      expect(view.container.querySelector('.floating-avatar-pixi-backstop')).toBeNull();
    });

    view.rerender(<FloatingAvatarMode {...props('happy')} />);

    expect(view.container.querySelector('.floating-avatar-pixi-backstop')).toBeNull();
    expect(view.getByTestId('inochi-stage')).toHaveAttribute('data-expression', 'happy');
  });
});
