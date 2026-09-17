import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { SubmissionDots } from '@/components/SubmissionDots'

const players = [
  { id: 'P1', name: 'One', color: '#a84632', submitted: true, isYou: true },
  { id: 'P2', name: 'Two', color: '#2d5f9e', submitted: false },
]

describe('SubmissionDots', () => {
  it('renders one dot per player with an accessible status label', () => {
    render(<SubmissionDots players={players} />)

    expect(screen.getByLabelText('One · Submitted')).toBeInTheDocument()
    expect(screen.getByLabelText('Two · Waiting')).toBeInTheDocument()
    expect(
      screen.getByRole('group', { name: 'Order submission status' }),
    ).toBeInTheDocument()
  })

  it('marks the submitted dot with the player color and the waiting dot as hollow', () => {
    render(<SubmissionDots players={players} />)

    const submitted = screen.getByLabelText('One · Submitted')
    expect(submitted).toHaveStyle({ backgroundColor: '#a84632' })

    const waiting = screen.getByLabelText('Two · Waiting')
    expect(waiting).toHaveStyle({ borderColor: '#2d5f9e' })
    expect(waiting.className).toContain('border-dashed')
  })

  it('renders nothing without players', () => {
    const { container } = render(<SubmissionDots players={[]} />)

    expect(container).toBeEmptyDOMElement()
  })
})
