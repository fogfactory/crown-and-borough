import { afterEach, describe, expect, it, vi } from 'vitest'

import {
  compareVersions,
  fetchReleaseNotes,
  type GitHubRelease,
  hasPendingReleaseNotes,
  isReleaseVersion,
  parseReleaseBody,
  RELEASES_API_URL,
  releaseFromGitHub,
  releaseNotesBetween,
  type ReleaseNotesEntry,
} from '@/lib/release-notes'

const BODY = `## [0.10.0](https://github.com/o/r/compare/v0.9.1...v0.10.0) (2026-09-23)


### Features

* **web:** winter overlay ([d09d500](https://github.com/o/r/commit/d09d50066010dfbf6bf433c097cd1fe7054df876))
* web: winter overlay ([aaaaaaa](https://github.com/o/r/commit/aaaaaaa))
* plain item with [link](https://x.test) ([#12](https://github.com/o/r/issues/12))

### Bug Fixes

* **map:** snowflakes ([d46f839](https://github.com/o/r/commit/d46f8391fc82e315d3b9d8c6469d5307943fe76e))
`

function release(tag: string, extra: Partial<GitHubRelease> = {}): GitHubRelease {
  return {
    tag_name: tag,
    published_at: '2026-09-23T14:13:39Z',
    body: '### Features\n\n* something\n',
    draft: false,
    ...extra,
  }
}

function entry(version: string): ReleaseNotesEntry {
  return { version, date: '', sections: [] }
}

describe('release notes', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('parses sections and cleans commit links', () => {
    const sections = parseReleaseBody(BODY)
    expect(sections.map((section) => section.title)).toEqual(['Features', 'Bug Fixes'])
    expect(sections[0].items).toEqual([
      'web: winter overlay',
      'plain item with link (#12)',
    ])
    expect(sections[1].items).toEqual(['map: snowflakes'])
  })

  it('converts GitHub releases and skips drafts and non-SemVer tags', () => {
    expect(releaseFromGitHub(release('v0.10.0'))).toEqual({
      version: '0.10.0',
      date: '2026-09-23',
      sections: [{ title: 'Features', items: ['something'] }],
    })
    expect(releaseFromGitHub(release('v1.0.0', { draft: true }))).toBeNull()
    expect(releaseFromGitHub(release('nightly'))).toBeNull()
    expect(
      releaseFromGitHub(release('v1.0.0', { body: null, published_at: null })),
    ).toEqual({
      version: '1.0.0',
      date: '',
      sections: [],
    })
  })

  it('compares versions numerically', () => {
    expect(compareVersions('0.9.1', '0.10.0')).toBe(-1)
    expect(compareVersions('v1.0.0', '1.0.0')).toBe(0)
    expect(compareVersions('2.0.0', '1.9.9')).toBe(1)
    expect(isReleaseVersion('dev')).toBe(false)
    expect(isReleaseVersion('ci')).toBe(false)
    expect(isReleaseVersion('1.2.3')).toBe(true)
  })

  it('opens only for an unseen release build', () => {
    expect(hasPendingReleaseNotes('0.9.0', '0.10.0')).toBe(true)
    expect(hasPendingReleaseNotes('', '0.10.0')).toBe(true)
    expect(hasPendingReleaseNotes('0.10.0', '0.10.0')).toBe(false)
    expect(hasPendingReleaseNotes('0.11.0', '0.10.0')).toBe(false)
    expect(hasPendingReleaseNotes('0.9.0', 'dev')).toBe(false)
  })

  it('returns every version between the last seen and the current one', () => {
    const entries = [entry('0.9.0'), entry('0.10.0'), entry('0.9.1'), entry('0.11.0')]
    expect(releaseNotesBetween(entries, '0.9.0', '0.10.0').map((e) => e.version)).toEqual(
      ['0.10.0', '0.9.1'],
    )
    expect(releaseNotesBetween(entries, '0.9.1', '0.9.1')).toEqual([])
    expect(releaseNotesBetween(entries, '', '0.10.0').map((e) => e.version)).toEqual([
      '0.10.0',
    ])
    expect(releaseNotesBetween(entries, '0.9.0', 'dev')).toEqual([])
  })

  it('fetches the releases from the GitHub API', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        new Response(
          JSON.stringify([release('v0.11.0'), release('v0.10.0'), release('v0.9.0')]),
        ),
      )
    vi.stubGlobal('fetch', fetchMock)
    const entries = await fetchReleaseNotes('0.9.0', '0.10.0')
    expect(entries.map((e) => e.version)).toEqual(['0.10.0'])
    expect(String(fetchMock.mock.calls[0][0])).toMatch(
      new RegExp(`^${RELEASES_API_URL}\\?`),
    )
  })

  it('rejects when GitHub answers with an error', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('{}', { status: 403 })))
    await expect(fetchReleaseNotes('0.9.0', '0.10.0')).rejects.toThrow(/403/)
  })
})
