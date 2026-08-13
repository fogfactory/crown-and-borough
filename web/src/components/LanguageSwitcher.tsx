import { useLanguage } from '@/i18n/LanguageContext'

export function LanguageSwitcher() {
  const { language, setLanguage, t } = useLanguage()

  return (
    <div
      role="group"
      aria-label={t('language.label')}
      className="inline-flex rounded-lg border border-[#b7a786] bg-[#f3ead9] p-0.5"
    >
      {(['en', 'fr'] as const).map((option) => (
        <button
          key={option}
          type="button"
          aria-pressed={language === option}
          className={`rounded-md px-2 py-1 text-[10px] font-bold uppercase tracking-[0.12em] transition ${language === option ? 'bg-[#fffaf0] text-[#a84632] shadow-sm' : 'text-[#806f57] hover:text-[#30291f]'}`}
          onClick={() => setLanguage(option)}
        >
          {option}
        </button>
      ))}
    </div>
  )
}
