import { REPOSITORY_SLUG, REPOSITORY_URL } from '@/lib/repository'

export const RELEASES_URL = `${REPOSITORY_URL}/releases`

/** Public GitHub API listing the releases published by release-please. */
export const RELEASES_API_URL = `https://api.github.com/repos/${REPOSITORY_SLUG}/releases`

/** Releases fetched per dialog; older ones stay reachable through RELEASES_URL. */
const RELEASES_PAGE_SIZE = 50

export interface ReleaseNoteSection {
  title: string
  items: string[]
}

export interface ReleaseNotesEntry {
  version: string
  date: string
  sections: ReleaseNoteSection[]
}

/** The fields of a GitHub release the dialog reads. */
export interface GitHubRelease {
  tag_name: string
  published_at: string | null
  body: string | null
  draft: boolean
}

// Keep aligned with releaseVersionPattern in internal/store/profile.go.
const VERSION_PATTERN = /^v?(\d+)\.(\d+)\.(\d+)(?:[-+][0-9A-Za-z.+-]*)?$/

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

/**
 * True when the deployed version is a release the player has not seen yet.
 * Decided locally, before any notes are fetched.
 */
export function hasPendingReleaseNotes(lastSeen: string, current: string): boolean {
  if (!isReleaseVersion(current)) return false
  return !isReleaseVersion(lastSeen) || compareVersions(current, lastSeen) > 0
}

function cleanItem(line: string): string {
  return line
    .replace(/^[*-]\s+/, '')
    .replace(/\s*\(\[[0-9a-f]{7,40}\]\([^)]*\)\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .trim()
}

/** Parses the `###` sections of a release-please release body. */
export function parseReleaseBody(body: string): ReleaseNoteSection[] {
  const sections: ReleaseNoteSection[] = []
  let section: ReleaseNoteSection | null = null
  for (const line of body.split(/\r?\n/)) {
    if (line.startsWith('### ')) {
      section = { title: line.slice(4).trim(), items: [] }
      sections.push(section)
    } else if (section && /^[*-]\s/.test(line)) {
      const item = cleanItem(line)
      // release-please lists a change once per commit of its pull request.
      if (item && !section.items.includes(item)) section.items.push(item)
    }
  }
  return sections.filter((candidate) => candidate.items.length > 0)
}

/** Converts a published GitHub release; drafts and non-SemVer tags give null. */
export function releaseFromGitHub(release: GitHubRelease): ReleaseNotesEntry | null {
  if (release.draft || !isReleaseVersion(release.tag_name)) return null
  return {
    version: release.tag_name.replace(/^v/, ''),
    date: release.published_at?.slice(0, 10) ?? '',
    sections: parseReleaseBody(release.body ?? ''),
  }
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
  const upToCurrent = entries
    .filter((entry) => compareVersions(entry.version, current) <= 0)
    .sort((a, b) => compareVersions(b.version, a.version))
  if (!isReleaseVersion(lastSeen)) return upToCurrent.slice(0, 1)
  return upToCurrent.filter((entry) => compareVersions(entry.version, lastSeen) > 0)
}

/** Fetches the release notes to show a player from the GitHub releases API. */
export async function fetchReleaseNotes(
  lastSeen: string,
  current: string,
  signal?: AbortSignal,
): Promise<ReleaseNotesEntry[]> {
  const response = await fetch(`${RELEASES_API_URL}?per_page=${RELEASES_PAGE_SIZE}`, {
    headers: { Accept: 'application/vnd.github+json' },
    signal,
  })
  if (!response.ok) throw new Error(`GitHub releases request failed (${response.status})`)
  const releases = (await response.json()) as GitHubRelease[]
  const entries = releases
    .map(releaseFromGitHub)
    .filter((entry): entry is ReleaseNotesEntry => entry !== null)
  return releaseNotesBetween(entries, lastSeen, current)
}
