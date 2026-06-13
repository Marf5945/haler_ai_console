import useI18n from '../../locales/useI18n';

export default function PackageErrorToast({message, onClose}) {
  const t = useI18n(s => s.t);
  return (
    <div className="pkg-error-toast" role="alert">
      <strong>{t("persona.installFail")}</strong>
      <span>{message}</span>
      <button type="button" onClick={onClose}>{t('common.close')}</button>
    </div>
  );
}
