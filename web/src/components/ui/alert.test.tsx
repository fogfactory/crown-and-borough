import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { Alert } from '@/components/ui/alert'
import { Input } from '@/components/ui/input'

describe('Alert', () => {
  it('is an error by default and can be a success or an information', () => {
    render(
      <>
        <Alert role="alert">Broken</Alert>
        <Alert tone="success">Sent</Alert>
        <Alert tone="info">Note</Alert>
      </>,
    )
    expect(screen.getByRole('alert').className).toContain('text-vermilion-700')
    expect(screen.getByText('Sent').className).toContain('text-success-700')
    expect(screen.getByText('Note').className).toContain('text-brass-700')
  })

  it('lets the caller add spacing', () => {
    render(<Alert className="mb-4">Spaced</Alert>)
    expect(screen.getByText('Spaced').className).toContain('mb-4')
  })
})

describe('Input', () => {
  it('replaces the default height instead of stacking it', () => {
    render(<Input aria-label="Email" className="h-11" />)
    const { className } = screen.getByLabelText('Email')
    expect(className).toContain('h-11')
    expect(className).not.toContain('h-10')
  })
})
