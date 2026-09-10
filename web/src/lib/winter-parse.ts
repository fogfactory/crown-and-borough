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

function isCode(value: string): boolean {
  return /^[A-Z]{3}$/.test(value)
}

function positiveAmount(value: string): number | null {
  const amount = Number(value)
  if (!Number.isSafeInteger(amount) || amount < 1) return null
  return amount
}

function parseLine(rawLine: string, line: number): ParsedWinterOrder | null {
  const comment = rawLine.indexOf('#')
  const normalized = (comment >= 0 ? rawLine.slice(0, comment) : rawLine)
    .trim()
    .toUpperCase()
  if (normalized === '') return null

  const fields = normalized.split(/\s+/)
  if (fields[0] === 'G') {
    if (fields.length !== 4 || !isCode(fields[1]) || !isCode(fields[2])) return null
    const amount = positiveAmount(fields[3])
    if (amount === null) return null
    return { line, type: 'transfer', source: fields[1], target: fields[2], amount }
  }

  if (fields.length !== 3) return null
  const first = fields[0]
  const subtype = fields[1]
  const reference = fields[2]

  if (first === 'R' && isCode(reference)) {
    if (subtype === 'N') return { line, type: 'recruit_noble', territory: reference }
    if (subtype === 'T') return { line, type: 'recruit_troop', territory: reference }
    return null
  }

  if (first === 'C' && isCode(reference)) {
    const infrastructure = {
      M: 'mill',
      C: 'castle',
      D: 'supply_depot',
    }[subtype] as ParsedWinterOrder['infrastructure'] | undefined
    if (!infrastructure) return null
    return { line, type: 'build', territory: reference, infrastructure }
  }

  if (first === 'E' && subtype === 'C' && isCode(reference)) {
    return { line, type: 'elect_capital', territory: reference }
  }

  if (first === 'L' && subtype === 'N' && isCode(reference)) {
    return { line, type: 'liberate_noble', noble: reference }
  }

  if (first === 'O' && subtype === 'N' && isCode(reference)) {
    return { line, type: 'hostage', noble: reference }
  }

  if (first === 'P' && subtype === 'N' && isCode(reference)) {
    return { line, type: 'dungeon', noble: reference }
  }

  return null
}

export function parseWinterDraft(text: string): ParsedWinterOrder[] {
  return text
    .replace(/\r\n/g, '\n')
    .split('\n')
    .flatMap((line, index) => {
      const parsed = parseLine(line, index + 1)
      return parsed ? [parsed] : []
    })
}
