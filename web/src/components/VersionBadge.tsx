import { useLanguage } from '@/i18n/LanguageContext'

const fallbackVersion = 'dev'

export function deployedVersion(): string {
  return import.meta.env.VITE_APP_VERSION?.trim() || fallbackVersion
}

export function VersionBadge() {
  const { t } = useLanguage()
  const version = deployedVersion()

  return (
    <span
      className="block font-mono text-[10px] text-[#806f57]"
      title={t('app.versionTitle', { version })}
    >
      {t('app.version', { version })}
    </span>
  )
}
