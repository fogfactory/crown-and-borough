/** Player counts of a game, online or local (mirrors models.MinimumGamePlayers / MaximumGamePlayers). */
export const MIN_PLAYERS = 2
export const MAX_PLAYERS = 12

/** Every player count a game can be created with. */
export const PLAYER_COUNT_OPTIONS = Array.from(
  { length: MAX_PLAYERS - MIN_PLAYERS + 1 },
  (_, index) => MIN_PLAYERS + index,
)
