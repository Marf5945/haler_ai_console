

export default function SettingSelect({icon, label, value, onNext}) {
  return (
    <label className="settings-select-row">
      <span className="settings-select-label"><i>{icon}</i>{label}</span>
      <button type="button" onClick={onNext}>
        <span className="settings-select-value">{value}</span>
        <span className="settings-select-caret">⌄</span>
      </button>
    </label>
  );
}
