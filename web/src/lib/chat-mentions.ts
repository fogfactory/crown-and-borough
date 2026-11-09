/**
 * Names of places and nobles found in a chat message, so the chat can link
 * them to the map (charte-graphique.md § Identifiants et liens vers la carte).
 * Detection runs on the client, on the plain text: the message stays text.
 */

/** What a written name stands for: a territory, nobles, or both. */
export interface MentionTarget {
  /** Territory whose commune bears the name. */
  territory?: string
  /** Codes of the living nobles who bear the name. */
  nobles: string[]
}

export type MentionSegment =
  | { kind: 'text'; text: string }
  | { kind: 'mention'; text: string; target: MentionTarget }

export interface MentionIndex {
  targets: ReadonlyMap<string, MentionTarget>
  /** Every indexed name, longest first; null when there is none. */
  pattern: RegExp | null
}

interface MentionSources {
  /** The map's territories: id and commune name. */
  territories: ReadonlyArray<{ id: string; name: string }>
  /**
   * Living nobles the reader's state shows, under their public identity:
   * a secret identity never enters the index.
   */
  nobles: ReadonlyArray<{ code: string; name: string; firstName?: string }>
}

// A name is not part of a longer word: letters, digits and hyphens around it
// (« Saint-Rochevent », « Rocheventais ») forbid the match. An apostrophe
// does not, so an elided name (« d'Hautebourg ») is found.
const WORD_CHAR = '[\\p{L}\\p{N}\\p{M}-]'

const ARTICLE = /^(Le|La|Les) |^L['’]/

function escapeRegExp(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Spellings of a name: itself, and for a commune opening on an article
 * (« Les Saules »), the article in lower case as written mid-sentence.
 */
function spellings(name: string): string[] {
  const match = ARTICLE.exec(name)
  if (!match) return [name]
  return [name, name[0].toLowerCase() + name.slice(1)]
}

/**
 * Builds the index of the names the reader may see linked: commune names of
 * the map and their codes (« RVE »), and for each living noble its public
 * code, its displayed name (« Baron Hugon de Rochevent »), that name from the
 * first name on (« Hugon de Rochevent ») and the first name alone. A name or
 * code borne by several nobles, or by a noble and a commune, points to all
 * of them.
 */
export function buildMentionIndex({ territories, nobles }: MentionSources): MentionIndex {
  const targets = new Map<string, MentionTarget>()
  const add = (name: string, apply: (target: MentionTarget) => void) => {
    const trimmed = name.trim()
    if (!trimmed) return
    for (const spelling of spellings(trimmed)) {
      let target = targets.get(spelling)
      if (!target) {
        target = { nobles: [] }
        targets.set(spelling, target)
      }
      apply(target)
    }
  }
  for (const territory of territories) {
    for (const name of [territory.name, territory.id]) {
      add(name, (target) => {
        target.territory ??= territory.id
      })
    }
  }
  for (const noble of nobles) {
    const firstName = noble.firstName || noble.name.split(' ')[0]
    const names = new Set([noble.name, firstName, noble.code])
    const at = noble.name.indexOf(`${firstName} `)
    if (at > 0) names.add(noble.name.slice(at))
    for (const name of names) {
      add(name, (target) => {
        if (!target.nobles.includes(noble.code)) target.nobles.push(noble.code)
      })
    }
  }
  if (targets.size === 0) return { targets, pattern: null }
  const alternatives = [...targets.keys()]
    .sort((a, b) => b.length - a.length || a.localeCompare(b))
    .map(escapeRegExp)
  const pattern = new RegExp(
    `(?<!${WORD_CHAR})(?:${alternatives.join('|')})(?!${WORD_CHAR})`,
    'gu',
  )
  return { targets, pattern }
}

/**
 * Splits a message into plain text and names of the index. Matching is exact
 * (case included, apart from a commune's leading article) and the longest
 * name wins: « Hugon de Rochevent » is one noble, not a first name and a
 * commune.
 */
export function tokenizeMentions(text: string, index: MentionIndex): MentionSegment[] {
  if (!index.pattern || !text) return text ? [{ kind: 'text', text }] : []
  const segments: MentionSegment[] = []
  let last = 0
  for (const match of text.matchAll(index.pattern)) {
    const start = match.index
    const target = index.targets.get(match[0])
    if (!target) continue
    if (start > last) segments.push({ kind: 'text', text: text.slice(last, start) })
    segments.push({ kind: 'mention', text: match[0], target })
    last = start + match[0].length
  }
  if (last < text.length) segments.push({ kind: 'text', text: text.slice(last) })
  return segments
}
