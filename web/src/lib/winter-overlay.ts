import type { WinterLinePreview } from '@/types'

export type WinterIntentionKind =
  | 'build'
  | 'recruit_troop'
  | 'recruit_noble'
  | 'liberate'
  | 'hostage'
  | 'dungeon'
  | 'capital'
  | 'transfer'
  | 'error'

export type WinterIntentionSource = 'draft' | 'submitted'

export interface WinterIntention {
  kind: WinterIntentionKind
  line: number
  valid: boolean
  source: WinterIntentionSource
  color?: string
  territory?: string
  sourceTerritory?: string
  targetTerritory?: string
  amount?: number
  noble?: string
  infrastructure?: 'mill' | 'castle' | 'supply_depot'
  level?: number
  /** Engine rejection reason, translated with `reports.reason.*`. */
  reason?: string
  reasonValues?: Record<string, string | number>
  /** Already localized explanation of a line that does not parse. */
  message?: string
  warning?: boolean
  label: string
}

export interface WinterOverlayOptions {
  source?: WinterIntentionSource
  color?: string
}

const INSUFFICIENT_RESOURCES = 'insufficient_resources'

function lineText(draft: string, line: number): string {
  return draft.replace(/\r\n/g, '\n').split('\n')[line - 1]?.trim() || `Line ${line}`
}

function kindOf(preview: WinterLinePreview): WinterIntentionKind {
  switch (preview.type) {
    case 'build':
      return 'build'
    case 'recruit_troop':
      return 'recruit_troop'
    case 'recruit_noble':
      return 'recruit_noble'
    case 'liberate_noble':
      return 'liberate'
    case 'hostage':
      return 'hostage'
    case 'dungeon':
      return 'dungeon'
    case 'elect_capital':
      return 'capital'
    case 'transfer':
      return 'transfer'
    default:
      return 'error'
  }
}

/**
 * Turns the server's simulated winter lines into map intentions. A line the
 * engine would refuse only for lack of resources stays drawn as a warning;
 * any other refusal or a malformed line becomes an error marker.
 */
export function buildWinterIntentions(
  lines: WinterLinePreview[],
  draft: string,
  options: WinterOverlayOptions = {},
): WinterIntention[] {
  const source = options.source ?? 'draft'
  return lines.flatMap((preview): WinterIntention[] => {
    if (preview.status === 'discard') return []
    const label = lineText(draft, preview.line)
    const common = { line: preview.line, source, color: options.color, label }
    if (preview.status === 'invalid') {
      return [{ ...common, kind: 'error', valid: false, message: preview.message }]
    }
    const warning =
      preview.status === 'rejected' && preview.reason === INSUFFICIENT_RESOURCES
    const refused = preview.status === 'rejected' && !warning
    const transfer = preview.type === 'transfer'
    return [
      {
        ...common,
        kind: refused ? 'error' : kindOf(preview),
        valid: !refused,
        warning: warning || undefined,
        reason: preview.reason,
        territory: transfer && !refused ? undefined : preview.territory,
        sourceTerritory: transfer ? preview.source : undefined,
        targetTerritory: transfer ? preview.target : undefined,
        amount: transfer ? preview.amount : undefined,
        noble: preview.noble,
        infrastructure: preview.infrastructure,
        level: preview.level,
      },
    ]
  })
}
