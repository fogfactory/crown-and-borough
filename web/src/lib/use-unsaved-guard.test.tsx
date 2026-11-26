import { fireEvent, render } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { useUnsavedGuard } from '@/lib/use-unsaved-guard'

function Guarded({ active }: { active: boolean }) {
  useUnsavedGuard(active, 'Leave?')
  return (
    <>
      <a href="/games/other">Other game</a>
      <a href={window.location.pathname}>Same page</a>
    </>
  )
}

afterEach(() => vi.restoreAllMocks())

describe('useUnsavedGuard', () => {
  it('asks the browser to warn on unload only while active', () => {
    const { rerender } = render(<Guarded active={false} />)
    const idle = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(idle)
    expect(idle.defaultPrevented).toBe(false)

    rerender(<Guarded active />)
    const guarded = new Event('beforeunload', { cancelable: true })
    window.dispatchEvent(guarded)
    expect(guarded.defaultPrevented).toBe(true)
  })

  it('confirms before following a link to another page, and can cancel', () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const { getByText } = render(<Guarded active />)

    const stopped = fireEvent.click(getByText('Other game'))
    expect(confirm).toHaveBeenCalledWith('Leave?')
    expect(stopped).toBe(false)

    confirm.mockReturnValue(true)
    expect(fireEvent.click(getByText('Other game'))).toBe(true)
  })

  it('leaves links to the same page alone', () => {
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const { getByText } = render(<Guarded active />)
    fireEvent.click(getByText('Same page'))
    expect(confirm).not.toHaveBeenCalled()
  })
})
