import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { HeaderPopover } from '@/components/ui/header-popover'

describe('HeaderPopover', () => {
  it('opens its content from the trigger and closes it back', () => {
    render(
      <HeaderPopover label="Scores" icon={<span>icon</span>} hint="12">
        <p>Scores content</p>
      </HeaderPopover>,
    )

    const trigger = screen.getByRole('button', { name: 'Scores' })
    expect(trigger).toHaveTextContent('12')
    expect(screen.queryByText('Scores content')).not.toBeInTheDocument()

    fireEvent.click(trigger)
    expect(screen.getByText('Scores content')).toBeInTheDocument()

    fireEvent.click(trigger)
    expect(screen.queryByText('Scores content')).not.toBeInTheDocument()
  })

  it('renders without a hint', () => {
    render(
      <HeaderPopover label="Lobby" icon={<span>icon</span>}>
        <p>Lobby content</p>
      </HeaderPopover>,
    )

    expect(screen.getByRole('button', { name: 'Lobby' })).toBeInTheDocument()
  })
})
