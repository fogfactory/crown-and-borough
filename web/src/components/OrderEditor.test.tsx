import { fireEvent, render, screen } from '@testing-library/react'
import { createRef } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { OrderEditor, tokenizeOrderLine } from '@/components/OrderEditor'

describe('tokenizeOrderLine', () => {
  it('classifies the words by their shape, in either case', () => {
    expect(tokenizeOrderLine('a ABC (def) 12 - x')).toEqual([
      { kind: 'verb', text: 'a' },
      { kind: 'text', text: ' ' },
      { kind: 'trigram', text: 'ABC' },
      { kind: 'text', text: ' ' },
      { kind: 'punctuation', text: '(' },
      { kind: 'trigram', text: 'def' },
      { kind: 'punctuation', text: ')' },
      { kind: 'text', text: ' ' },
      { kind: 'number', text: '12' },
      { kind: 'text', text: ' ' },
      { kind: 'punctuation', text: '-' },
      { kind: 'text', text: ' ' },
      { kind: 'verb', text: 'x' },
    ])
  })

  it('makes a comment of everything after the hash', () => {
    expect(tokenizeOrderLine('H AAA # stay (3)')).toEqual([
      { kind: 'verb', text: 'H' },
      { kind: 'text', text: ' ' },
      { kind: 'trigram', text: 'AAA' },
      { kind: 'text', text: ' ' },
      { kind: 'comment', text: '# stay (3)' },
    ])
  })

  it('leaves other word lengths uncolored and gives the line back', () => {
    const line = 'hello 42 é ab-€ #x'
    const tokens = tokenizeOrderLine(line)
    expect(tokens.map((token) => token.text).join('')).toBe(line)
    expect(tokens[0]).toEqual({ kind: 'text', text: 'hello' })
  })
})

describe('OrderEditor', () => {
  it('edits the text and mirrors it for the colors', () => {
    const onChange = vi.fn()
    render(<OrderEditor value="M AAA" onChange={onChange} aria-label="orders" />)
    const textarea = screen.getByLabelText('orders')
    fireEvent.change(textarea, { target: { value: 'M AAAx' } })
    expect(onChange).toHaveBeenCalledWith('M AAAx')
    expect(document.querySelector('[data-order-mirror]')?.textContent).toBe('M AAA ')
    expect(textarea).not.toHaveAttribute('aria-invalid')
  })

  it('underlines the faulty lines and flags the textarea', () => {
    render(
      <OrderEditor
        value={'A\nB\nC'}
        onChange={() => {}}
        errorLines={[2]}
        errorsId="errors"
        invalid
        aria-label="orders"
      />,
    )
    const underlined = document.querySelectorAll('[data-order-error-line]')
    expect(underlined).toHaveLength(1)
    expect(underlined[0].textContent).toBe('B')
    const textarea = screen.getByLabelText('orders')
    expect(textarea).toHaveAttribute('aria-invalid', 'true')
    expect(textarea).toHaveAttribute('aria-describedby', 'errors')
  })

  it('forwards its ref to the textarea', () => {
    const ref = createRef<HTMLTextAreaElement>()
    render(<OrderEditor ref={ref} value="" onChange={() => {}} aria-label="orders" />)
    expect(ref.current).toBe(screen.getByLabelText('orders'))
  })
})
