import {useEffect, useRef} from 'react';

export default function FloatingPreviewPopover({
  open,
  boundaryRef,
  className = '',
  titleClassName = '',
  title,
  ariaLabel,
  onClose,
  children,
}) {
  const popoverRef = useRef(null);

  useEffect(() => {
    if (!open) return undefined;
    const onKeyDown = (event) => {
      if (event.key === 'Escape') onClose?.();
    };
    const onPointerDown = (event) => {
      const target = event.target;
      if (popoverRef.current?.contains(target)) return;
      if (boundaryRef?.current?.contains?.(target)) return;
      onClose?.();
    };
    document.addEventListener('keydown', onKeyDown);
    document.addEventListener('pointerdown', onPointerDown);
    return () => {
      document.removeEventListener('keydown', onKeyDown);
      document.removeEventListener('pointerdown', onPointerDown);
    };
  }, [open, boundaryRef, onClose]);

  if (!open) return null;

  return (
    <div ref={popoverRef} className={className} role="dialog" aria-label={ariaLabel || title}>
      {title && <strong className={titleClassName}>{title}</strong>}
      {children}
    </div>
  );
}
