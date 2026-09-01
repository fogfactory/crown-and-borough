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
        <label className="block space-y-1.5 text-sm font-semibold text-[#594b3c]">
          <span>{t('online.reportHistory')}</span>
          <select
            value={
              report
                ? (summaries.find((item) => item.header.turn === report.header.turn)
                    ?.index ?? '')
                : ''
            }
            onChange={(event) => onSelectReport(Number(event.target.value))}
            className="h-10 w-full rounded-lg border border-[#b7a786] bg-[#f8f0e2] px-3 text-sm"
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
        <p className="text-sm italic text-[#806f57]">{t('online.loading')}</p>
      ) : error ? (
        <p role="alert" className="text-sm text-[#8d321e]">
          {error}
        </p>
      ) : report ? (
        <ReportPanel report={report} map={map} players={players} />
      ) : (
        <p className="rounded-lg border border-dashed border-[#b7a786] bg-[#f8f0e2] px-4 py-8 text-center font-serif italic text-[#806f57]">
          {t('app.noReport')}
        </p>
      )}
    </div>
  )
}
