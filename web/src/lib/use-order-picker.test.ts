import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { useOrderPicker } from '@/lib/use-order-picker'

function setup(options: {
  desktop: boolean
  ordersShown: boolean
  selectedId?: string | null
}) {
  const showOrders = vi.fn()
  const view = renderHook(
    (props: typeof options) =>
      useOrderPicker({ selectedId: 'ROS', ...props, showOrders }),
    { initialProps: options },
  )
  return { ...view, showOrders }
}

describe('useOrderPicker', () => {
  it('takes nothing while no field waits', () => {
    const { result } = setup({ desktop: true, ordersShown: true })
    expect(result.current.pickOnMap('ROS')).toBe(false)
    expect(result.current.fillWithSelection).toBeUndefined()
  })

  it('fills the waiting field from the map only while the orders are shown', () => {
    const picker = vi.fn()
    const { result, rerender } = setup({ desktop: true, ordersShown: true })
    act(() => result.current.onPickerChange(picker))

    expect(result.current.pickOnMap('ROS')).toBe(true)
    expect(picker).toHaveBeenCalledWith('ROS')

    rerender({ desktop: true, ordersShown: false })
    expect(result.current.pickOnMap('BRU')).toBe(false)
    expect(picker).toHaveBeenCalledTimes(1)
  })

  it('offers the card button on a desktop only while the orders are shown', () => {
    const picker = vi.fn()
    const { result, rerender, showOrders } = setup({ desktop: true, ordersShown: true })
    act(() => result.current.onPickerChange(picker))

    act(() => result.current.fillWithSelection?.())
    expect(picker).toHaveBeenCalledWith('ROS')
    expect(showOrders).not.toHaveBeenCalled()

    rerender({ desktop: true, ordersShown: false })
    expect(result.current.fillWithSelection).toBeUndefined()
  })

  it('offers the card button on a phone from the map tab, then shows the orders', () => {
    const picker = vi.fn()
    const { result, showOrders } = setup({ desktop: false, ordersShown: false })
    act(() => result.current.onPickerChange(picker))

    expect(result.current.fillWithSelection).toBeDefined()
    act(() => result.current.fillWithSelection?.())
    expect(picker).toHaveBeenCalledWith('ROS')
    expect(showOrders).toHaveBeenCalledOnce()
  })

  it('has nothing to fill with when no territory is selected', () => {
    const { result } = setup({ desktop: true, ordersShown: true, selectedId: null })
    act(() => result.current.onPickerChange(vi.fn()))
    expect(result.current.fillWithSelection).toBeUndefined()
  })

  it('forgets the field when it goes away', () => {
    const { result } = setup({ desktop: true, ordersShown: true })
    act(() => result.current.onPickerChange(vi.fn()))
    act(() => result.current.onPickerChange(null))
    expect(result.current.fillWithSelection).toBeUndefined()
  })
})
