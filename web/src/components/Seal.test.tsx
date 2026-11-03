import { render } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import favicon from '../../public/favicon.svg?raw'
import { Seal, sealVersion, sealWaxPath } from '@/components/Seal'
import { SEAL_MONOGRAM_PATH } from '@/components/seal-monogram'
import {
  SEAL_LETTER,
  SEAL_WAX_DARK,
  SEAL_WAX_LIGHT,
  SEAL_WAX_RIM,
  SEAL_WAX_SHADOW,
} from '@/lib/map-palette'

describe('sealVersion', () => {
  it('keeps the version and drops build number and suffix', () => {
    expect(sealVersion('v1.2.3')).toBe('v1.2.3')
    expect(sealVersion('1.2.3')).toBe('v1.2.3')
    expect(sealVersion('v0.10.0-rc.1+build.123')).toBe('v0.10.0')
    expect(sealVersion('v1.2.3.456')).toBe('v1.2.3')
  })

  it('keeps "dev" and truncates anything else', () => {
    expect(sealVersion('dev')).toBe('dev')
    expect(sealVersion('3f9c2a1b7d8e')).toBe('3f9c2a1b')
    expect(sealVersion('abcdef0123456789')).toBe('abcdef01')
  })
})

describe('Seal', () => {
  it('shows the name and the version in the full variant', () => {
    vi.stubEnv('VITE_APP_VERSION', 'v1.2.3-rc.1')
    const { container } = render(<Seal variant="full" />)
    const text = container.textContent
    expect(text).toContain('Crown')
    expect(text).toContain('and')
    expect(text).toContain('Borough')
    expect(text).toContain('v1.2.3')
    expect(text).not.toContain('rc')
    vi.unstubAllEnvs()
  })

  it('has no legend in the compact variant', () => {
    const { container } = render(<Seal />)
    expect(container.querySelector('text')).toBeNull()
  })

  it('gives each seal its own gradient and path ids', () => {
    const { container } = render(
      <>
        <Seal variant="full" />
        <Seal variant="full" />
      </>,
    )
    const ids = [...container.querySelectorAll('[id]')].map((el) => el.id)
    expect(new Set(ids).size).toBe(ids.length)
  })
})

describe('favicon', () => {
  it('embeds the same wax outline, monogram and colors as the component', () => {
    for (const part of [
      sealWaxPath(),
      SEAL_MONOGRAM_PATH,
      SEAL_WAX_LIGHT,
      SEAL_WAX_DARK,
      SEAL_WAX_RIM,
      SEAL_WAX_SHADOW,
      SEAL_LETTER,
    ]) {
      expect(favicon).toContain(part)
    }
  })
})
