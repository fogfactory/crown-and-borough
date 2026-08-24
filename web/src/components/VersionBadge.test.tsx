import { render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { VersionBadge } from '@/components/VersionBadge'

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('VersionBadge', () => {
  it('renders the configured application version', () => {
    vi.stubEnv('VITE_APP_VERSION', 'v0.3.1')

    render(<VersionBadge />)

    expect(screen.getByText('Version v0.3.1')).toBeInTheDocument()
  })

  it('falls back to dev when no version is configured', () => {
    render(<VersionBadge />)

    expect(screen.getByText('Version dev')).toBeInTheDocument()
  })
})
