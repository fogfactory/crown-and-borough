import { FaqPanel } from '@/components/FaqPanel'
import { RulesPanel } from '@/components/RulesPanel'
import { Card, CardContent } from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'

export type InfoPageKind = 'rules' | 'faq'

export function InfoPage({ kind }: { kind: InfoPageKind }) {
  const { t } = useLanguage()
  const isRules = kind === 'rules'

  return (
    <div className="space-y-6">
      <header className="max-w-3xl">
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-[#a84632]">
          Crown &amp; Borough
        </p>
        <h1 className="mt-1 font-serif text-3xl font-semibold text-[#30291f] sm:text-4xl">
          {isRules ? t('rules.pageTitle') : t('faq.title')}
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-[#806f57]">
          {isRules ? t('rules.pageIntro') : t('faq.intro')}
        </p>
      </header>

      <Card className="border-[#b7a786] bg-[#fffaf0] shadow-[0_18px_50px_-30px_rgba(67,46,24,0.7)]">
        <CardContent className="p-4 sm:p-8">
          {isRules ? <RulesPanel variant="page" /> : <FaqPanel showHeading={false} />}
        </CardContent>
      </Card>
    </div>
  )
}
