import { FaqPanel } from '@/components/FaqPanel'
import { PageHeader } from '@/components/PageHeader'
import { RulesPanel } from '@/components/RulesPanel'
import { Card, CardContent } from '@/components/ui/card'
import { useLanguage } from '@/i18n/LanguageContext'

export type InfoPageKind = 'rules' | 'faq'

export function InfoPage({ kind }: { kind: InfoPageKind }) {
  const { t } = useLanguage()
  const isRules = kind === 'rules'

  return (
    <div className="space-y-6">
      <PageHeader
        title={isRules ? t('rules.pageTitle') : t('faq.title')}
        intro={isRules ? t('rules.pageIntro') : t('faq.intro')}
      />

      <Card className="border-line">
        <CardContent className="p-4 sm:p-8">
          {isRules ? <RulesPanel variant="page" /> : <FaqPanel showHeading={false} />}
        </CardContent>
      </Card>
    </div>
  )
}
