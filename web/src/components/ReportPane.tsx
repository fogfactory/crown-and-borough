import { useLanguage } from '@/i18n/LanguageContext'
import { SEASON_LABEL_KEYS } from '@/lib/season'
import { ReportPanel } from '@/components/ReportPanel'
import type { MapData, Player, ReportHeader, TurnReport } from '@/types'

export interface ReportSummary {
  index: number
  header: ReportHeader
}

interface ReportPaneProps {
  report: TurnReport | null
  map: MapData | null
  players: Player[]
  summaries?: ReportSummary[]
  loading?: boolean
  error?: string | null
  onSelectReport?: (index: number) => void
}

export function ReportPane({
  report,
  map,
  players,
  summaries = [],
  loading = false,
  error = null,
  onSelectReport,
}: ReportPaneProps) {
  const { t } = useLanguage()

  return (
    <div className="space-y-4">
      {summaries.length > 0 && onSelectReport && (
        <label className="block space-y-1.5 text-sm font-semibold text-ink-700">
          <span>{t('online.reportHistory')}</span>
          <select
            value={
              report
                ? (summaries.find((item) => item.header.turn === report.header.turn)
                    ?.index ?? '')
                : ''
            }
            onChange={(event) => onSelectReport(Number(event.target.value))}
            className="h-10 w-full rounded-lg border border-line bg-paper-100 px-3 text-sm"
          >
            {summaries.map((item) => (
              <option key={item.index} value={item.index}>
                {t('online.currentTurn', {
                  turn: item.header.turn,
                  season: t(SEASON_LABEL_KEYS[item.header.season]),
                })}
              </option>
            ))}
          </select>
        </label>
      )}

      {loading ? (
        <p className="text-sm italic text-ink-500">{t('online.loading')}</p>
      ) : error ? (
        <p role="alert" className="text-sm text-vermilion-700">
          {error}
        </p>
      ) : report ? (
        <ReportPanel report={report} map={map} players={players} />
      ) : (
        <p className="rounded-lg border border-dashed border-line bg-paper-100 px-4 py-8 text-center font-display italic text-ink-500">
          {t('app.noReport')}
        </p>
      )}
    </div>
  )
}
