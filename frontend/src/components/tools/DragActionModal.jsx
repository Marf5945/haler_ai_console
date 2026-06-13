

export default function DragActionModal({ariaLabel, icon = '↗', title, detail = '', actions = []}) {
  return (
    <div className="tool-drag-overlay" role="dialog" aria-modal="true" aria-label={ariaLabel}>
      <section className="tool-drag-modal">
        <header>
          <span>{icon}</span>
          <div className="drag-action-title">
            <strong>{title}</strong>
            {detail && <small>{detail}</small>}
          </div>
        </header>
        <div
          className="tool-drag-actions"
          style={{gridTemplateColumns: `repeat(${Math.max(1, actions.length)}, minmax(0, 1fr))`}}
        >
          {actions.map((action) => (
            <button
              className={action.disabled ? 'tool-drag-action-disabled' : ''}
              type="button"
              disabled={action.disabled}
              key={action.label}
              onClick={action.onClick}
            >
              {action.label}
            </button>
          ))}
        </div>
      </section>
    </div>
  );
}
