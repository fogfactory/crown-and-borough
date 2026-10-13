package engine

import "github.com/fogfactory/crown-and-borough/internal/models"

// seatTinctures are the eight player tinctures of the graphic charter, in
// seat order, with the hex value published in players[].color. Gold is never
// a tincture: it only appears as a metal.
var seatTinctures = [...]struct {
	tincture models.Tincture
	color    string
}{
	{models.TinctureGules, "#b3322b"},
	{models.TinctureAzure, "#2b5c9e"},
	{models.TinctureVert, "#3f7d3c"},
	{models.TincturePurpure, "#6e3f8f"},
	{models.TinctureSable, "#2f2b27"},
	{models.TinctureTenne, "#c8651b"},
	{models.TinctureCeleste, "#4fa3c7"},
	{models.TinctureMurrey, "#8c2f5a"},
}

// seatCharges are the charges of the first eight seats, in order.
var seatCharges = [...]models.Charge{
	models.ChargeBend,
	models.ChargeFess,
	models.ChargeChevron,
	models.ChargeCross,
	models.ChargeSaltire,
	models.ChargePale,
	models.ChargeChief,
	models.ChargeBordure,
}

// seatChargeShift is how far the charge moves, in ranks, for the seats past
// the first eight.
const seatChargeShift = 4

// SeatArms returns the generated arms and tincture color of the seat at
// index (0 for P1). Seats 1 to 8 alternate or and argent; seat n past eight
// takes the tincture of seat n − 8, inverts its metal and shifts its charge
// by four ranks, so the sixteen hotseat seats never repeat a coat of arms.
func SeatArms(index int) (models.Arms, string) {
	count := len(seatTinctures)
	base := index % count
	round := index / count
	metal := models.MetalOr
	if base%2 == 1 {
		metal = models.MetalArgent
	}
	if round%2 == 1 {
		metal = invertMetal(metal)
	}
	tincture := seatTinctures[base]
	charge := seatCharges[(base+round*seatChargeShift)%len(seatCharges)]
	return models.Arms{Field: tincture.tincture, Metal: metal, Charge: charge}, tincture.color
}

func invertMetal(metal models.Metal) models.Metal {
	if metal == models.MetalOr {
		return models.MetalArgent
	}
	return models.MetalOr
}
