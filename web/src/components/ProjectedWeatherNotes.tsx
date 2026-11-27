import { RegionRef } from '@/components/MapLinks'
import { useLanguage } from '@/i18n/LanguageContext'
import { forecastWeatherNotes, type WeatherNote } from '@/lib/forecast-weather-notes'
import { richTranslate } from '@/lib/rich-text'
import type { PlayerId, StateData } from '@/types'

interface ProjectedWeatherNotesProps {
  state: StateData
  playerId: PlayerId
}

const TONE_CLASSES: Record<WeatherNote['tone'], string> = {
  negative: 'text-vermilion-700',
  positive: 'text-success-700',
  neutral: 'text-ink-900',
}

function signed(amount: number): string {
  return amount > 0 ? `+${amount}` : `−${-amount}`
}

/**
 * Names the weather effects (announced calamity, drafted bonus cards) that
 * change the income, mill and ration forecast, so a difference with the
 * normal figures never goes unexplained.
 */
export function ProjectedWeatherNotes({ state, playerId }: ProjectedWeatherNotesProps) {
  const { t } = useLanguage()
  const notes = forecastWeatherNotes(state, playerId)
  if (notes.length === 0) return null
  return (
    <div className="text-sm" data-weather-notes>
      <p className="text-xs font-bold uppercase tracking-[0.16em] text-ink-500">
        {t('treasury.cause.title')}
      </p>
      <ul className="mt-1 list-disc space-y-0.5 pl-4">
        {notes.map((note) => (
          <li key={`${note.key}-${note.region}`} className={TONE_CLASSES[note.tone]}>
            {richTranslate(t, note.key, { region: <RegionRef id={note.region} /> })}
            {(note.income !== 0 || note.rations !== 0) && (
              <strong className="ml-1 font-bold">
                {[
                  note.income !== 0 &&
                    t('treasury.cause.incomeAmount', { amount: signed(note.income) }),
                  note.rations !== 0 &&
                    t('treasury.cause.rationsAmount', { amount: signed(note.rations) }),
                ]
                  .filter(Boolean)
                  .join(' · ')}
              </strong>
            )}
          </li>
        ))}
      </ul>
    </div>
  )
}
