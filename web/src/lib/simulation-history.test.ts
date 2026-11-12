import { act, renderHook } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { simulationLog, useSimulationHistory } from '@/lib/simulation-history'
import type { SimulationAction } from '@/types'

describe('simulationLog', () => {
  it('names the message of each action and what it bears on', () => {
    expect(simulationLog({ type: 'marry', noble: 'A', other: 'B' })).toEqual({
      key: 'sim.log.marry',
      noble: 'A',
      other: 'B',
      fief: '',
    })
    expect(simulationLog({ type: 'conquer', noble: 'A', fief: 'ROC' })).toMatchObject({
      key: 'sim.log.conquer',
      noble: 'A',
      fief: 'ROC',
    })
    expect(simulationLog({ type: 'restore', fief: 'ROC' })).toMatchObject({
      key: 'sim.log.restore',
      noble: '',
      fief: 'ROC',
    })
    expect(simulationLog({ type: 'renounce_claim', noble: 'A' }).key).toBe('sim.log.renounce_claim')
  })

  it('reads the variants of a death', () => {
    expect(simulationLog({ type: 'kill', noble: 'A' }).key).toBe('sim.log.kill')
    expect(simulationLog({ type: 'kill', noble: 'A', alsoKill: 'B' })).toMatchObject({
      key: 'sim.log.killTogether',
      other: 'B',
    })
    expect(simulationLog({ type: 'kill', noble: 'A', garrison: 'C' })).toMatchObject({
      key: 'sim.log.killGarrison',
      other: 'C',
    })
    expect(simulationLog({ type: 'kill', noble: 'A', noGarrison: true }).key).toBe(
      'sim.log.killNoGarrison',
    )
  })
})

describe('useSimulationHistory', () => {
  it('undoes and redoes actions of every shape, keeping their fields', () => {
    const actions: SimulationAction[] = [
      { type: 'kill', noble: 'A', alsoKill: 'B' },
      { type: 'conquer', noble: 'C', fief: 'ROC' },
      { type: 'renounce_title', noble: 'D', fief: 'ROC' },
    ]
    const { result } = renderHook(() => useSimulationHistory())
    act(() => actions.forEach((action) => result.current.push(action)))
    expect(result.current.actions).toEqual(actions)

    act(() => result.current.undo())
    act(() => result.current.undo())
    expect(result.current.actions).toEqual([actions[0]])
    expect(result.current.canRedo).toBe(true)
    act(() => result.current.redo())
    expect(result.current.actions).toEqual(actions.slice(0, 2))

    act(() => result.current.push({ type: 'restore', fief: 'ROC' }))
    expect(result.current.canRedo).toBe(false)
    act(() => result.current.dropLast())
    expect(result.current.actions).toEqual(actions.slice(0, 2))
    act(() => result.current.reset())
    expect(result.current.actions).toEqual([])
  })
})
