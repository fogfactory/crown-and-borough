import type { ReactNode } from 'react'

import { Input } from '@/components/ui/input'
import { useLanguage } from '@/i18n/LanguageContext'

/** The labelled house name input, with its note; `action` sits beside the input. */
export function HouseNameField({
  value,
  onChange,
  action,
}: {
  value: string
  onChange: (value: string) => void
  action?: ReactNode
}) {
  const { t } = useLanguage()
  return (
    <>
      <label className="block space-y-1.5">
        <span className="text-sm font-semibold text-ink-700">{t('online.houseName')}</span>
        <div className="flex gap-2">
          <Input
            type="text"
            value={value}
            maxLength={32}
            onChange={(event) => onChange(event.target.value)}
          />
          {action}
        </div>
      </label>
      <p className="text-xs text-ink-500">{t('online.houseNameNote')}</p>
    </>
  )
}
