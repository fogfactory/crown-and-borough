import changelogSource from '../../../CHANGELOG.md?raw'

import { REPOSITORY_URL } from '@/lib/repository'

export const RELEASES_URL = `${REPOSITORY_URL}/releases`

export interface ReleaseNoteSection {
  title: string
  items: string[]
}

export interface ReleaseNotesEntry {
  version: string
  date: string
  sections: ReleaseNoteSection[]
}

const VERSION_PATTERN = /^v?(\d+)\.(\d+)\.(\d+)(?:[-+].*)?$/
const ENTRY_HEADING = /^## \[?v?(\d+\.\d+\.\d+[^\]\s)]*)\]?(?:\([^)]*\))?\s*(?:\(([^)]*)\))?\s*$/

function parseVersion(value: string): [number, number, number] | null {
  const match = VERSION_PATTERN.exec(value.trim())
  return match ? [Number(match[1]), Number(match[2]), Number(match[3])] : null
}

/** True for a SemVer release; `dev`, `ci` and other build labels are not. */
export function isReleaseVersion(value: string): boolean {
  return parseVersion(value) !== null
}

/** Orders SemVer versions numerically (pre-release suffixes are ignored). */
export function compareVersions(a: string, b: string): number {
  const left = parseVersion(a)
  const right = parseVersion(b)
  if (!left || !right) return 0
  for (let index = 0; index < 3; index += 1) {
    if (left[index] !== right[index]) return left[index] < right[index] ? -1 : 1
  }
  return 0
}

function cleanItem(line: string): string {
  return line
    .replace(/^[*-]\s+/, '')
    .replace(/\s*\(\[[0-9a-f]{7,40}\]\([^)]*\)\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .trim()
}

/** Parses the release-please CHANGELOG.md, newest entry first. */
export function parseChangelog(source: string): ReleaseNotesEntry[] {
  const entries: ReleaseNotesEntry[] = []
  let entry: ReleaseNotesEntry | null = null
  let section: ReleaseNoteSection | null = null
  for (const line of source.split(/\r?\n/)) {
    const heading = ENTRY_HEADING.exec(line)
    if (heading) {
      entry = { version: heading[1], date: heading[2] ?? '', sections: [] }
      entries.push(entry)
      section = null
    } else if (entry && line.startsWith('### ')) {
      section = { title: line.slice(4).trim(), items: [] }
      entry.sections.push(section)
    } else if (section && /^[*-]\s/.test(line)) {
      const item = cleanItem(line)
      // release-please lists a change once per commit of its pull request.
      if (item && !section.items.includes(item)) section.items.push(item)
    }
  }
  return entries.filter((candidate) => isReleaseVersion(candidate.version))
}

/**
 * Entries newer than `lastSeen` up to `current` (newest first). A player who
 * has never seen a version gets the current entry only.
 */
export function releaseNotesBetween(
  entries: readonly ReleaseNotesEntry[],
  lastSeen: string,
  current: string,
): ReleaseNotesEntry[] {
  if (!isReleaseVersion(current)) return []
  const upToCurrent = entries.filter((entry) => compareVersions(entry.version, current) <= 0)
  if (!isReleaseVersion(lastSeen)) return upToCurrent.slice(0, 1)
  return upToCurrent.filter((entry) => compareVersions(entry.version, lastSeen) > 0)
}

const bundledEntries = parseChangelog(changelogSource)

/** Release notes to show a player, from the CHANGELOG embedded at build time. */
export function pendingReleaseNotes(lastSeen: string, current: string): ReleaseNotesEntry[] {
  return releaseNotesBetween(bundledEntries, lastSeen, current)
}
