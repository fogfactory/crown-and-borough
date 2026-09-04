import type { LiaisonMode, Order, OrderType } from '@/types'

const SYMBOL_TO_TYPE: Record<string, OrderType> = {
  A: 'attack',
  J: 'join',
  D: 'disperse',
  H: 'hold',
  P: 'pillage',
  S: 'support',
}

function stripComment(line: string): string {
  const commentIndex = line.indexOf('#')
  return (commentIndex >= 0 ? line.slice(0, commentIndex) : line).trim()
}

function splitLiaison(line: string): { content: string; liaison: LiaisonMode } | null {
  const opens = (line.match(/\(/g) ?? []).length
  const closes = (line.match(/\)/g) ?? []).length
  if (opens === 0 && closes === 0) {
    return { content: line, liaison: 'single' }
  }
  if (opens !== 1 || closes !== 1 || !line.startsWith('(') || !line.endsWith(')')) {
    return null
  }
  return { content: line.slice(1, -1).trim(), liaison: 'loop' }
}

function parseDisperseTarget(token: string): {
  territory: string
  assignments: string[]
} | null {
  const parts = token.split('*')
  const territory = parts[0] ?? ''
  if (territory === '') return null
  const assignments = parts.slice(1).filter((part) => part !== '')
  return { territory, assignments }
}

function parseLine(line: string): Order | null {
  const knownOrder = stripComment(line).toUpperCase()
  if (knownOrder === '') return null
  const liaisonParse = splitLiaison(knownOrder)
  if (!liaisonParse) return null
  const { content, liaison } = liaisonParse
  const tokens = content.split(/\s+/).filter((token) => token !== '')
  if (tokens.length === 0) return null

  const first = tokens[0]
  if (first === 'H' || first === 'P') {
    if (tokens.length !== 2) return null
    const type = first === 'H' ? 'hold' : 'pillage'
    return { type, position: tokens[1], targets: [], liaison }
  }

  if (tokens.length < 2) return null
  const position = tokens[0]
  const symbol = tokens[1]
  const type = SYMBOL_TO_TYPE[symbol]
  switch (type) {
    case 'attack':
    case 'join': {
      if (tokens.length !== 3) return null
      return { type, position, targets: [tokens[2]], liaison }
    }
    case 'support': {
      if (tokens.length === 3) {
        return { type, position, targets: [tokens[2]], liaison }
      }
      if (tokens.length === 5 && tokens[3] === '-') {
        return { type, position, targets: [tokens[2], tokens[4]], liaison }
      }
      return null
    }
    case 'disperse': {
      const targets: string[] = []
      const nobleAssignments: Record<string, string[]> = {}
      for (const token of tokens.slice(2)) {
        const parsed = parseDisperseTarget(token)
        if (!parsed) return null
        targets.push(parsed.territory)
        if (parsed.assignments.length > 0) {
          nobleAssignments[parsed.territory] = [
            ...(nobleAssignments[parsed.territory] ?? []),
            ...parsed.assignments,
          ]
        }
      }
      if (targets.length === 0) return null
      return { type, position, targets, nobleAssignments, liaison }
    }
    default:
      return null
  }
}

export function parseChainDraft(text: string): Order[] {
  const orders: Order[] = []
  for (const rawLine of text.replace(/\r\n/g, '\n').split('\n')) {
    const order = parseLine(rawLine)
    if (order) orders.push(order)
  }
  return orders
}
