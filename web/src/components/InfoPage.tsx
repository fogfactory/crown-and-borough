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
        <p className="text-xs font-semibold uppercase tracking-[0.2em] text-brass-700">
          Crown &amp; Borough
        </p>
        <h1 className="mt-1 font-display text-3xl font-semibold text-ink-900 sm:text-4xl">
          {isRules ? t('rules.pageTitle') : t('faq.title')}
        </h1>
        <p className="mt-2 text-sm leading-relaxed text-ink-500">
          {isRules ? t('rules.pageIntro') : t('faq.intro')}
        </p>
      </header>

      <Card className="border-line bg-paper-50 shadow-panel">
        <CardContent className="p-4 sm:p-8">
          {isRules ? <RulesPanel variant="page" /> : <FaqPanel showHeading={false} />}
        </CardContent>
      </Card>
    </div>
  )
}
