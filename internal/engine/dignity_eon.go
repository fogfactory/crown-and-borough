package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// unmaskEon reveals a chevalier d'Éon's dignity and secret identity to every
// player once she is captured, imprisoned or targeted by a Claim while
// married (specs/dames.md § Chevalier d'Éon). It is a no-op for any other
// noble, or once she is already unmasked.
func (ctx *resolutionContext) unmaskEon(noble *models.Noble, phase int) {
	if noble == nil || noble.EonUnmasked || !noble.Has(models.DignityChevalierDEon) {
		return
	}
	noble.EonUnmasked = true
	ctx.events = append(ctx.events, Event{
		Type:      EventTypeEonUnmasked,
		Phase:     phase,
		OwnerID:   noble.OwnerID,
		NobleID:   noble.ID,
		NobleCode: models.NobleCode(noble.Code),
		NobleName: ctx.state.NobleDisplayName(*noble),
		Dignity:   models.DignityChevalierDEon,
	})
}
