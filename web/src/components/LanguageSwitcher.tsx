import { useLanguage } from '@/i18n/LanguageContext'

export function LanguageSwitcher() {
  const { language, setLanguage, t } = useLanguage()

  return (
    <div
      role="group"
      aria-label={t('language.label')}
      className="inline-flex rounded-lg border border-line bg-paper-100 p-0.5"
    >
      {(['en', 'fr'] as const).map((option) => (
        <button
          key={option}
          type="button"
          aria-pressed={language === option}
          className={`rounded-md px-2 py-1 text-3xs font-bold uppercase tracking-[0.12em] transition ${language === option ? 'bg-paper-50 text-brass-700 shadow-sm' : 'text-ink-500 hover:text-ink-900'}`}
          onClick={() => setLanguage(option)}
        >
          {option}
        </button>
      ))}
    </div>
  )
}
