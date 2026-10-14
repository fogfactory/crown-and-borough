import { act, renderHook } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { FOCUS_FRAME_PADDING, type MapFrame } from '@/lib/map-frame'
import { useMapLinkState } from '@/lib/use-map-link-state'
import type { MapData, StateData } from '@/types'

const map = {
  territories: [
    { id: 'ROS', name: 'Rosemont', points: [] },
    { id: 'BRU', name: 'Bruyères', points: [] },
  ],
} as unknown as MapData

const state = {
  turn: 1,
  season: 'spring',
  players: [],
  territories: [],
  nobles: [
    {
      id: 'JEA',
      code: 'JEA',
      name: 'Jean',
      owner: 'P1',
      location: 'BRU',
      status: 'free',
    },
  ],
  deceased: [{ code: 'OLD', name: 'Ancien', owner: 'P1', cause: 'natural', turn: 1 }],
} as unknown as StateData

describe('useMapLinkState', () => {
  it('frames, selects and haloes what a link points to', () => {
    const onSelect = vi.fn()
    const { result, rerender } = renderHook(
      ({ selectedId, baseFrame }: { selectedId: string | null; baseFrame?: MapFrame }) =>
        useMapLinkState({ map, state, selectedId, baseFrame, onSelect }),
      { initialProps: { selectedId: null as string | null } },
    )

    expect(result.current.links.territoryName('BRU')).toBe('Bruyères')
    expect(result.current.links.nobleTerritory('JEA')).toBe('BRU')
    expect(result.current.links.isDeceased('OLD')).toBe(true)
    expect(result.current.links.nobleName('OLD')).toBe('Ancien')

    act(() => result.current.links.highlight('ROS'))
    expect(result.current.highlightedId).toBe('ROS')

    act(() => result.current.links.focus('BRU', 'JEA'))
    expect(onSelect).toHaveBeenCalledWith('BRU')
    expect(result.current.highlightedId).toBeNull()
    expect(result.current.frame).toMatchObject({
      territories: ['BRU'],
      padding: FOCUS_FRAME_PADDING,
    })

    rerender({ selectedId: 'BRU' })
    expect(result.current.selectedNoble).toBe('JEA')
    // Selecting elsewhere drops the pawn halo.
    rerender({ selectedId: 'ROS' })
    expect(result.current.selectedNoble).toBeNull()
  })

  it('gives the frame back to a new base frame', () => {
    const own = (key: string): MapFrame => ({ territories: ['ROS'], key })
    const { result, rerender } = renderHook(
      ({ baseFrame }: { baseFrame: MapFrame }) =>
        useMapLinkState({ map, state, selectedId: null, baseFrame, onSelect: vi.fn() }),
      { initialProps: { baseFrame: own('g:P1') } },
    )

    act(() => result.current.links.focus('BRU'))
    expect(result.current.frame?.territories).toEqual(['BRU'])

    rerender({ baseFrame: own('g:P2') })
    expect(result.current.frame).toEqual(own('g:P2'))
  })
})

describe('useMapLinkState report events', () => {
  it('frames an event, draws it, and drops it at the next selection', () => {
    const onShowEvent = vi.fn()
    const { result, rerender } = renderHook(
      ({ selectedId }: { selectedId: string | null }) =>
        useMapLinkState({ map, state, selectedId, onSelect: vi.fn(), onShowEvent }),
      { initialProps: { selectedId: null as string | null } },
    )

    act(() =>
      result.current.links.showEvent({
        territories: ['ROS', 'BRU'],
        arrows: [['ROS', 'BRU']],
      }),
    )
    expect(onShowEvent).toHaveBeenCalledOnce()
    expect(result.current.frame?.territories).toEqual(['ROS', 'BRU'])
    expect(result.current.eventMarks?.arrows).toEqual([['ROS', 'BRU']])

    rerender({ selectedId: 'ROS' })
    expect(result.current.eventMarks).toBeNull()
  })

  it('drops the event once the turn is resolved', () => {
    const { result, rerender } = renderHook(
      ({ turn }: { turn: number }) =>
        useMapLinkState({
          map,
          state: { ...state, turn },
          selectedId: null,
          onSelect: vi.fn(),
        }),
      { initialProps: { turn: state.turn } },
    )

    act(() => result.current.links.showEvent({ territories: ['ROS'] }))
    expect(result.current.eventMarks).not.toBeNull()

    rerender({ turn: state.turn + 1 })
    expect(result.current.eventMarks).toBeNull()
  })
})
