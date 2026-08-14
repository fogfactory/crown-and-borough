import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

type JsonObject = Record<string, unknown>

const fixturesDirectory = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '../../../specs/fixtures',
)

function readFixture(name: string): string {
  return readFileSync(path.join(fixturesDirectory, name), 'utf8')
}

const mapRaw = readFixture('map.json')
const exactCombatRaw = readFixture('report-combat-exact.json')
const generalCombatRaw = readFixture('report-combat-general.json')
const hiddenChainRaw = readFixture('state-army-hidden-chain.json')
const noChainRaw = readFixture('state-army-no-chain.json')
const territoryStateRaw = readFixture('state-territory-id.json')

const fixtureRaws: Record<string, string> = {
  map: mapRaw,
  territoryState: territoryStateRaw,
  noChain: noChainRaw,
  hiddenChain: hiddenChainRaw,
  exactCombat: exactCombatRaw,
  generalCombat: generalCombatRaw,
}

function parseObject(raw: string, label: string): JsonObject {
  const value: unknown = JSON.parse(raw)
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be a JSON object`)
  }
  return value as JsonObject
}

function objectValue(value: unknown, label: string): JsonObject {
  if (value === null || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error(`${label} must be a JSON object`)
  }
  return value as JsonObject
}

function arrayValue(value: unknown, label: string): unknown[] {
  if (!Array.isArray(value)) {
    throw new Error(`${label} must be an array`)
  }
  return value
}

function territories(fixture: JsonObject): JsonObject[] {
  return arrayValue(fixture.territories, 'territories').map((value, index) =>
    objectValue(value, `territory ${index}`),
  )
}

function firstArmyChain(fixture: JsonObject): unknown {
  for (const [index, territoryValue] of territories(fixture).entries()) {
    const territory = objectValue(territoryValue, `territory ${index}`)
    if (territory.army === null || territory.army === undefined) {
      continue
    }
    const army = objectValue(territory.army, `territory ${index} army`)
    if (!('chain' in army)) {
      throw new Error(`territory ${index} army has no chain field`)
    }
    return army.chain
  }
  throw new Error('fixture has no army')
}

function firstCombat(fixture: JsonObject): JsonObject {
  const combats = arrayValue(fixture.combats, 'combats')
  if (combats.length === 0) {
    throw new Error('fixture has no combat')
  }
  return objectValue(combats[0], 'first combat')
}

describe('online contract fixtures', () => {
  it('round-trips every fixture as JSON without changing its shape', () => {
    for (const [label, raw] of Object.entries(fixtureRaws)) {
      const parsed: unknown = JSON.parse(raw)
      expect(JSON.parse(JSON.stringify(parsed)), label).toEqual(parsed)
    }
  })

  it('uses the territory trigram as the only public territory identifier', () => {
    const fixtureTerritories = [
      ['map', parseObject(mapRaw, 'map')],
      ['territory state', parseObject(territoryStateRaw, 'territory state')],
      ['no-chain state', parseObject(noChainRaw, 'no-chain state')],
      ['hidden-chain state', parseObject(hiddenChainRaw, 'hidden-chain state')],
    ] as const

    for (const [label, fixture] of fixtureTerritories) {
      const fixtureTerritories = territories(fixture)
      const ids = fixtureTerritories.map((territory) => territory.id)
      expect(new Set(ids).size, label).toBe(ids.length)
      for (const [index, territory] of fixtureTerritories.entries()) {
        expect(territory.id, `${label} territory ${index} id`).toMatch(/^[A-Z]{3}$/)
        expect(territory, `${label} territory ${index}`).not.toHaveProperty('code')
      }
    }
  })

  it('distinguishes an army without a chain from a hidden chain', () => {
    const noChain = firstArmyChain(parseObject(noChainRaw, 'no-chain state'))
    const hiddenChain = firstArmyChain(parseObject(hiddenChainRaw, 'hidden-chain state'))

    expect(noChain).toBeNull()
    expect(hiddenChain).toEqual({ visibility: 'hidden' })
    expect(hiddenChain).not.toBeNull()
  })

  it('keeps exact combat details separate from the general combat view', () => {
    const exactCombat = firstCombat(parseObject(exactCombatRaw, 'exact combat report'))
    expect(exactCombat.visibility).toBe('exact')

    const contenders = arrayValue(exactCombat.contenders, 'exact contenders')
    expect(contenders.length).toBeGreaterThan(0)
    for (const [index, value] of contenders.entries()) {
      const contender = objectValue(value, `exact contender ${index}`)
      expect(contender).toHaveProperty('army')
      expect(contender).toHaveProperty('owner')
      expect(contender).toHaveProperty('force')
    }

    const generalCombat = firstCombat(parseObject(generalCombatRaw, 'general combat report'))
    expect(generalCombat.visibility).toBe('general')
    for (const field of [
      'contenders',
      'force',
      'army',
      'owner',
      'baseDefense',
      'defense',
      'castleBonus',
      'winner',
      'dislodged',
    ]) {
      expect(generalCombat).not.toHaveProperty(field)
    }
  })
})
