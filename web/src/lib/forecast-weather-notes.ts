import type { MessageKey } from '@/i18n/messages'
import type { ForecastEffectKind, PlayerId, StateData } from '@/types'

/** One weather effect that explains a difference in the treasury forecast. */
export interface WeatherNote {
  key: MessageKey
  /** Seed of the affected region. */
  region: string
  /** Bad for the player, good for them, or a canceled calamity. */
  tone: 'negative' | 'positive' | 'neutral'
  /** Income (R) and rations the effect adds (+) or costs (−); 0 when none. */
  income: number
  rations: number
}

const NOTES: Record<ForecastEffectKind, { key: MessageKey; tone: WeatherNote['tone'] }> =
  {
    famine: { key: 'treasury.cause.famine', tone: 'negative' },
    famine_canceled: { key: 'treasury.cause.famineCanceled', tone: 'neutral' },
    abundant_harvest: { key: 'treasury.cause.abundantHarvest', tone: 'positive' },
    bad_weather: { key: 'treasury.cause.badWeather', tone: 'negative' },
    bad_weather_canceled: { key: 'treasury.cause.badWeatherCanceled', tone: 'neutral' },
    fair_weather: { key: 'treasury.cause.fairWeather', tone: 'positive' },
  }

/**
 * The weather effects behind the income, mill and ration forecast, for the
 * regions where the player holds a territory. The server lists them from the
 * very computation that produces the figures (see `forecastEffects`), so they
 * follow the drafted cards exactly: the first card cancels the calamity, a
 * second one, or a card with no calamity to cancel, doubles the production.
 */
export function forecastWeatherNotes(
  state: StateData,
  playerId: PlayerId,
): WeatherNote[] {
  if (state.season === 'winter') return []
  const bishoprics = state.bishoprics ?? []
  const held = new Set(
    state.territories
      .filter((territory) => territory.owner === playerId)
      .map((territory) => territory.id),
  )
  const heldRegions = new Set(
    bishoprics
      .filter((bishopric) => bishopric.territories.some((id) => held.has(id)))
      .map((bishopric) => bishopric.region),
  )
  return (state.forecastEffects ?? [])
    .filter((effect) => bishoprics.length === 0 || heldRegions.has(effect.region))
    .map((effect) => ({
      ...NOTES[effect.kind],
      region: effect.region,
      income: effect.income ?? 0,
      rations: effect.rations ?? 0,
    }))
}
