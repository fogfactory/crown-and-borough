import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { PageHeader } from '@/components/PageHeader'

describe('PageHeader', () => {
  it('renders one h1 with the label, the intro, the leading mark and the actions', () => {
    render(
      <PageHeader
        title="Rules"
        intro="How to play"
        leading={<span>seal</span>}
        actions={<button type="button">Create</button>}
      />,
    )
    expect(screen.getAllByRole('heading', { level: 1 })).toHaveLength(1)
    expect(screen.getByRole('heading', { name: 'Rules' })).toBeTruthy()
    expect(screen.getByText('Crown & Borough')).toBeTruthy()
    expect(screen.getByText('How to play')).toBeTruthy()
    expect(screen.getByText('seal')).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Create' })).toBeTruthy()
  })

  it('uses the larger title on landing pages', () => {
    render(<PageHeader size="landing" title="Home" />)
    expect(screen.getByRole('heading', { name: 'Home' }).className).toContain('text-3xl')
  })
})
