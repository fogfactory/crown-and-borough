package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

func revealCurrentAugury(ctx *resolutionContext) {
	if ctx.state.Season != models.SeasonSpring {
		return
	}
	year := ctx.state.Year()
	augury, exists := ctx.state.Auguries[year]
	if !exists || augury.Revealed {
		return
	}
	augury.Revealed = true
	ctx.state.Auguries[year] = augury
	ctx.events = append(ctx.events, Event{Type: EventTypeAuguryRevealed, Phase: 0, Year: year, Season: models.SeasonSpring})
}
