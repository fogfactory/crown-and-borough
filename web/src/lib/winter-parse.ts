import type { MapData, Noble } from '@/types'

export type WinterOrderType =
  | 'recruit_noble'
  | 'recruit_troop'
  | 'build'
  | 'elect_capital'
  | 'liberate_noble'
  | 'hostage'
  | 'dungeon'
  | 'transfer'

export interface ParsedWinterOrder {
  line: number
  type: WinterOrderType
  territory?: string
  source?: string
  target?: string
  amount?: number
  infrastructure?: 'mill' | 'castle' | 'supply_depot'
  noble?: string
}

export type WinterParseErrorKey =
  | 'error.winter.order_shape'
  | 'error.winter.target_only_one'
  | 'error.winter.transfer_shape'
  | 'error.winter.transfer_amount'
  | 'error.winter.unknown_symbol'
  | 'error.winter.unknown_subtype'
  | 'error.winter.territory_code_format'
  | 'error.winter.territory_unknown'
  | 'error.winter.noble_code_format'
  | 'error.winter.noble_unknown'

export interface WinterParseError {
  line: number
  key: WinterParseErrorKey
  values?: Record<string, string>
}

export interface WinterDraftParseOptions {
  map?: MapData
  nobles?: ReadonlyArray<Pick<Noble, 'code'>>
}

export interface WinterDraftParseResult {
  orders: ParsedWinterOrder[]
  errors: WinterParseError[]
}

function isCode(value: string): boolean {
  return /^[A-Z]{3}$/.test(value)
}

function positiveAmount(value: string): number | null {
  const amount = Number(value)
  if (!Number.isSafeInteger(amount) || amount < 1) return null
  return amount
}

function winterError(
  line: number,
  key: WinterParseErrorKey,
  values?: Record<string, string>,
): WinterParseError {
  return { line, key, values }
}

function territoryError(
  code: string,
  line: number,
  map: MapData | undefined,
): WinterParseError | null {
  if (!isCode(code)) {
    return winterError(line, 'error.winter.territory_code_format', { code })
  }
  if (map && !map.territories.some((territory) => territory.id === code)) {
    return winterError(line, 'error.winter.territory_unknown', { code })
  }
  return null
}

function nobleError(
  code: string,
  line: number,
  nobles: ReadonlyArray<Pick<Noble, 'code'>> | undefined,
): WinterParseError | null {
  if (!isCode(code)) {
    return winterError(line, 'error.winter.noble_code_format', { code })
  }
  if (nobles && !nobles.some((noble) => noble.code === code)) {
    return winterError(line, 'error.winter.noble_unknown', { code })
  }
  return null
}

type ParsedWinterLine = { order: ParsedWinterOrder } | { error: WinterParseError }

function parseLine(
  rawLine: string,
  line: number,
  options: WinterDraftParseOptions,
): ParsedWinterLine | null {
  const comment = rawLine.indexOf('#')
  const normalized = (comment >= 0 ? rawLine.slice(0, comment) : rawLine)
    .trim()
    .toUpperCase()
  if (normalized === '') return null

  const fields = normalized.split(/\s+/)
  if (fields.length < 3) {
    return { error: winterError(line, 'error.winter.order_shape') }
  }
  if (fields[0] === 'G') {
    if (fields.length !== 4) {
      return { error: winterError(line, 'error.winter.transfer_shape') }
    }
    const sourceError = territoryError(fields[1], line, options.map)
    if (sourceError) return { error: sourceError }
    const targetError = territoryError(fields[2], line, options.map)
    if (targetError) return { error: targetError }
    const amount = positiveAmount(fields[3])
    if (amount === null) {
      return {
        error: winterError(line, 'error.winter.transfer_amount', {
          amount: fields[3],
        }),
      }
    }
    return {
      order: { line, type: 'transfer', source: fields[1], target: fields[2], amount },
    }
  }

  if (fields.length > 3) {
    return { error: winterError(line, 'error.winter.target_only_one') }
  }
  const first = fields[0]
  const subtype = fields[1]
  const reference = fields[2]

  if (first === 'R') {
    const referenceError = territoryError(reference, line, options.map)
    if (referenceError) return { error: referenceError }
    if (subtype === 'N') {
      return { order: { line, type: 'recruit_noble', territory: reference } }
    }
    if (subtype === 'T') {
      return { order: { line, type: 'recruit_troop', territory: reference } }
    }
    return {
      error: winterError(line, 'error.winter.unknown_subtype', {
        symbol: first,
        subtype,
      }),
    }
  }

  if (first === 'C') {
    const referenceError = territoryError(reference, line, options.map)
    if (referenceError) return { error: referenceError }
    const infrastructure = {
      M: 'mill',
      C: 'castle',
      D: 'supply_depot',
    }[subtype] as ParsedWinterOrder['infrastructure'] | undefined
    if (!infrastructure) {
      return {
        error: winterError(line, 'error.winter.unknown_subtype', {
          symbol: first,
          subtype,
        }),
      }
    }
    return { order: { line, type: 'build', territory: reference, infrastructure } }
  }

  if (first === 'E') {
    if (subtype !== 'C') {
      return {
        error: winterError(line, 'error.winter.unknown_subtype', {
          symbol: first,
          subtype,
        }),
      }
    }
    const referenceError = territoryError(reference, line, options.map)
    if (referenceError) return { error: referenceError }
    return { order: { line, type: 'elect_capital', territory: reference } }
  }

  if (first === 'L' || first === 'O' || first === 'P') {
    if (subtype !== 'N') {
      return {
        error: winterError(line, 'error.winter.unknown_subtype', {
          symbol: first,
          subtype,
        }),
      }
    }
    const referenceError = nobleError(reference, line, options.nobles)
    if (referenceError) return { error: referenceError }
    if (first === 'L') {
      return { order: { line, type: 'liberate_noble', noble: reference } }
    }
    return {
      order: {
        line,
        type: first === 'O' ? 'hostage' : 'dungeon',
        noble: reference,
      },
    }
  }

  return {
    error: winterError(line, 'error.winter.unknown_symbol', { symbol: first }),
  }
}

export function parseWinterDraftDetailed(
  text: string,
  options: WinterDraftParseOptions = {},
): WinterDraftParseResult {
  const result: WinterDraftParseResult = { orders: [], errors: [] }
  for (const [index, line] of text.replace(/\r\n/g, '\n').split('\n').entries()) {
    const parsed = parseLine(line, index + 1, options)
    if (!parsed) continue
    if ('error' in parsed) {
      result.errors.push(parsed.error)
    } else {
      result.orders.push(parsed.order)
    }
  }
  return result
}

export function parseWinterDraft(text: string): ParsedWinterOrder[] {
  return parseWinterDraftDetailed(text).orders
}
