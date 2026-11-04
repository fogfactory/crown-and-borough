import { describe, expect, it } from 'vitest'

import {
  compareVersions,
  isReleaseVersion,
  parseChangelog,
  pendingReleaseNotes,
  releaseNotesBetween,
} from '@/lib/release-notes'

const SOURCE = `# Changelog

## [0.10.0](https://github.com/o/r/compare/v0.9.1...v0.10.0) (2026-09-23)


### Features

* **web:** winter overlay ([d09d500](https://github.com/o/r/commit/d09d50066010dfbf6bf433c097cd1fe7054df876))
* web: winter overlay ([aaaaaaa](https://github.com/o/r/commit/aaaaaaa))
* plain item with [link](https://x.test) ([#12](https://github.com/o/r/issues/12))

### Bug Fixes

* **map:** snowflakes ([d46f839](https://github.com/o/r/commit/d46f8391fc82e315d3b9d8c6469d5307943fe76e))

## [0.9.1](https://github.com/o/r/compare/v0.9.0...v0.9.1) (2026-09-18)

### Bug Fixes

* old fix ([0f9b1d0](https://github.com/o/r/commit/0f9b1d0f9e75189063656b859b44dca77ba2028d))

## 0.9.0 (2026-09-17)

### Features

* first
`

describe('release notes', () => {
  const entries = parseChangelog(SOURCE)

  it('parses entries, sections and cleans commit links', () => {
    expect(entries.map((entry) => entry.version)).toEqual(['0.10.0', '0.9.1', '0.9.0'])
    expect(entries[0].date).toBe('2026-09-23')
    expect(entries[0].sections.map((section) => section.title)).toEqual([
      'Features',
      'Bug Fixes',
    ])
    expect(entries[0].sections[0].items).toEqual([
      'web: winter overlay',
      'plain item with link (#12)',
    ])
    expect(entries[2].sections[0].items).toEqual(['first'])
  })

  it('compares versions numerically', () => {
    expect(compareVersions('0.9.1', '0.10.0')).toBe(-1)
    expect(compareVersions('v1.0.0', '1.0.0')).toBe(0)
    expect(compareVersions('2.0.0', '1.9.9')).toBe(1)
    expect(isReleaseVersion('dev')).toBe(false)
    expect(isReleaseVersion('ci')).toBe(false)
    expect(isReleaseVersion('1.2.3')).toBe(true)
  })

  it('returns every version between the last seen and the current one', () => {
    expect(releaseNotesBetween(entries, '0.9.0', '0.10.0').map((e) => e.version)).toEqual([
      '0.10.0',
      '0.9.1',
    ])
    expect(releaseNotesBetween(entries, '0.9.1', '0.9.1')).toEqual([])
    expect(releaseNotesBetween(entries, '0.9.0', '0.9.1').map((e) => e.version)).toEqual([
      '0.9.1',
    ])
  })

  it('shows only the current version to a player who has seen none', () => {
    expect(releaseNotesBetween(entries, '', '0.10.0').map((e) => e.version)).toEqual(['0.10.0'])
  })

  it('never shows notes for dev or non-release builds', () => {
    expect(releaseNotesBetween(entries, '0.9.0', 'dev')).toEqual([])
    expect(pendingReleaseNotes('0.0.1', 'dev')).toEqual([])
    expect(pendingReleaseNotes('0.0.1', 'ci')).toEqual([])
  })

  it('bundles the repository changelog', () => {
    expect(pendingReleaseNotes('0.0.1', '0.10.0').length).toBeGreaterThan(0)
  })
})
