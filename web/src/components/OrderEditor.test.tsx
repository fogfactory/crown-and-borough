import { fireEvent, render, screen } from '@testing-library/react'
import { createRef, useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { OrderEditor, tokenizeOrderLine } from '@/components/OrderEditor'

describe('tokenizeOrderLine', () => {
  it('colors card codes of two letters like the other codes', () => {
    expect(tokenizeOrderLine('P BT RLG').map((token) => token.kind)).toEqual([
      'verb',
      'text',
      'trigram',
      'text',
      'trigram',
    ])
  })

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

  it('completes the typed word from the suggestions with the keyboard', () => {
    // Like suggestCodes, nothing for a whole code.
    const suggest = vi.fn((word: string) =>
      word.length === 3
        ? []
        : [
            { code: 'ECP', name: 'Escarpelle', kind: 'territory' as const },
            { code: 'ESC', name: 'Escalade', kind: 'territory' as const },
          ],
    )
    function Editor() {
      const [value, setValue] = useState('RVE A ')
      return <OrderEditor value={value} onChange={setValue} suggest={suggest} aria-label="orders" />
    }
    render(<Editor />)
    const textarea = screen.getByLabelText('orders') as HTMLTextAreaElement
    textarea.focus()
    fireEvent.change(textarea, { target: { value: 'RVE A es', selectionStart: 8, selectionEnd: 8 } })

    expect(suggest).toHaveBeenCalledWith('es', 'RVE A ')
    expect(screen.getByRole('option', { name: 'Escarpelle (ECP)' })).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(textarea, { key: 'ArrowDown' })
    expect(screen.getByRole('option', { name: 'Escalade (ESC)' })).toHaveAttribute('aria-selected', 'true')
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(textarea.value).toBe('RVE A ESC')
    expect(textarea.selectionStart).toBe(9)
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('closes the suggestions with Escape until the word changes', () => {
    const suggest = () => [{ code: 'ECP', name: 'Escarpelle', kind: 'territory' as const }]
    render(<OrderEditor value="es" onChange={() => {}} suggest={suggest} aria-label="orders" />)
    const textarea = screen.getByLabelText('orders') as HTMLTextAreaElement
    textarea.focus()
    textarea.setSelectionRange(2, 2)
    fireEvent.select(textarea)
    expect(screen.getByRole('listbox')).toBeInTheDocument()
    fireEvent.keyDown(textarea, { key: 'Escape' })
    expect(screen.queryByRole('listbox')).toBeNull()
  })

  it('opens the suggestions on Ctrl+Space without any letter', () => {
    const suggest = vi.fn(() => [{ code: 'ECP', name: 'Escarpelle', kind: 'territory' as const }])
    render(<OrderEditor value="RVE A " onChange={() => {}} suggest={suggest} aria-label="orders" />)
    const textarea = screen.getByLabelText('orders') as HTMLTextAreaElement
    textarea.focus()
    textarea.setSelectionRange(6, 6)
    fireEvent.select(textarea)
    expect(screen.queryByRole('listbox')).toBeNull()

    fireEvent.keyDown(textarea, { key: ' ', ctrlKey: true })
    expect(suggest).toHaveBeenCalledWith('', 'RVE A ')
    expect(screen.getByRole('option', { name: 'Escarpelle (ECP)' })).toBeInTheDocument()
  })

  it('leaves Enter and Tab to the textarea while the list only opened by itself', () => {
    const suggest = () => [{ code: 'PEL', name: 'Pellerin', kind: 'territory' as const }]
    const onChange = vi.fn()
    render(<OrderEditor value="S R ABC PE" onChange={onChange} suggest={suggest} aria-label="orders" />)
    const textarea = screen.getByLabelText('orders') as HTMLTextAreaElement
    textarea.focus()
    textarea.setSelectionRange(10, 10)
    fireEvent.select(textarea)
    expect(screen.getByRole('listbox')).toBeInTheDocument()

    expect(fireEvent.keyDown(textarea, { key: 'Enter' })).toBe(true)
    expect(fireEvent.keyDown(textarea, { key: 'Tab' })).toBe(true)
    expect(onChange).not.toHaveBeenCalled()
  })

  it('opens the list on Alt+Down and on its button, and forgets the request elsewhere', () => {
    const suggest = vi.fn(() => [{ code: 'ECP', name: 'Escarpelle', kind: 'territory' as const }])
    render(<OrderEditor value={'RVE A \nH'} onChange={() => {}} suggest={suggest} aria-label="orders" />)
    const textarea = screen.getByLabelText('orders') as HTMLTextAreaElement
    textarea.focus()
    textarea.setSelectionRange(6, 6)
    fireEvent.select(textarea)
    fireEvent.keyDown(textarea, { key: 'ArrowDown', altKey: true })
    expect(screen.getByRole('listbox')).toBeInTheDocument()

    // Moving away drops the request: coming back does not reopen the list.
    textarea.setSelectionRange(9, 9)
    fireEvent.select(textarea)
    textarea.setSelectionRange(6, 6)
    fireEvent.select(textarea)
    expect(screen.queryByRole('listbox')).toBeNull()

    fireEvent.click(screen.getByRole('button', { name: /Ctrl\+Space/ }))
    expect(screen.getByRole('listbox')).toBeInTheDocument()
  })
})
