package models

// MinimumGamePlayers and MaximumGamePlayers bound the number of players of a
// game, online or local alike: the engine, the stores, the server and the
// tables of the player rules all follow them.
const (
	MinimumGamePlayers = 2
	MaximumGamePlayers = 12
)
