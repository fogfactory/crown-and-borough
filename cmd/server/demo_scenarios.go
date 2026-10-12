//go:build demo

package main

import (
	"fmt"
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
	"github.com/fogfactory/crown-and-borough/internal/store"
)

// Forged-state scenarios of the hotseat demo (scripts/demo.sh <scenario>).
// A scenario edits the state of the freshly created game (4 players, seed
// crown-and-borough-dev); the store validates the result. Add one here when a
// feature needs a situation that takes many turns to reach.
var demoScenarios = map[string]struct {
	description string
	apply       func(*models.GameState) error
}{
	"winter": {
		description: "winter, P1 is pope with a bishop-astrologer, a Witch, cards in hand; P2 holds a cardinal and an excommunicated noble",
		apply:       forgeWinter,
	},
	"election": {
		description: "winter (see the winter scenario), plus a vacant bishopric whose every territory is held by P1 or P2 (armies), with a free man of each player to run: a bishop election opens",
		apply:       forgeElection,
	},
	"king": {
		description: "winter (see the winter scenario), vacant throne, plus a baron of P1 (ZRA), P2 (ZRB) and P3 (ZRC) holding a barony each: the royal election opens (K R / V R), P1's pope and bishop and P2's cardinal can anoint",
		apply:       forgeKing,
	},
	"crown": {
		description: "winter (see the winter scenario): P2's excommunicated noble is the king, married to ZQU (P3), and holds a barony; claims on the couple: ZAN (P3, a woman) then ZGA (P3) on the king, ZPI (P2) on the queen. P1's pope and P2's cardinal can file the J trial that kills the king: the crown goes to ZPI, the barony to ZAN",
		apply:       forgeCrown,
	},
	"dissolution": {
		description: "winter (see the winter scenario), plus two marriages: ZEL (P1, the pope's house) with ZRO (P2), and ZCA (P2) with ZGU (P3); P1 can marry free nobles of P4",
		apply:       forgeDissolution,
	},
	"inquiry": {
		description: "winter (see the winter scenario), plus two ladies of P2 hiding a Witch and a Spy: P1's pope and P2's cardinal can investigate them",
		apply:       forgeInquiry,
	},
	"trial": {
		description: "winter (see the winter scenario): P1's pope (also a cardinal) and P2's cardinal can file the same J trial on P2's excommunicated noble",
		apply:       forgeWinter,
	},
	"annexation": {
		description: "winter (see the winter scenario), plus a barony of P1 held by its first noble, with three free territories next to it each held by a one-troop P1 army: try T X (annex) and T R (raise by one rank)",
		apply:       forgeAnnexation,
	},
	"royal": {
		description: "winter, annexation scenario plus a king (ZRK, P1), a barony of P2 held by ZRB, and a vacant crown fief of P3, who has no noble left: try W X, W R, W F, W G as P1 with the matching T X / T R / T F / T A from P1 and P2",
		apply:       forgeRoyal,
	},
	"appeasement": {
		description: "spring, P1 has a bishop, a cardinal and an abbess; rebel armies of 1 to 3 troops stand in their regions and elsewhere; P1 has plenty of R",
		apply:       forgeAppeasement,
	},
	"tithe": {
		description: "spring, P1 has a bishop and a cardinal (see the appeasement scenario) and two Tax cards: tithe orders on bishoprics",
		apply:       forgeTithe,
	},
	"tunnel": {
		description: "spring, P1 holds a Tunnel card and an army (with its noble) next to P2's capital castle, exactly as strong as the fortified defense: it fails alone, wins with P SO <origin> - <target>; the report shows the defense faced",
		apply:       forgeTunnel,
	},
	"action": {
		description: "spring, P1 holds bonus cards; a famine is active, a bad weather is announced (one bent by a ritual)",
		apply:       forgeAction,
	},
}

func init() {
	demoRegistered = func(name string, memory *store.MemoryStore) error {
		scenario, known := demoScenarios[name]
		if !known {
			names := make([]string, 0, len(demoScenarios))
			for known := range demoScenarios {
				names = append(names, known)
			}
			sort.Strings(names)
			return fmt.Errorf("unknown scenario (have %v)", names)
		}
		var failure error
		if err := memory.Forge(func(state *models.GameState) { failure = scenario.apply(state) }); err != nil {
			return err
		}
		return failure
	}
}

// ownedNobles returns the indexes of the nobles of a player, in state order.
func ownedNobles(state *models.GameState, player models.PlayerID) []int {
	var indexes []int
	for index, noble := range state.Nobles {
		if noble.OwnerID == player {
			indexes = append(indexes, index)
		}
	}
	return indexes
}

// giveSpecialCards moves the first card of each kind from the draw pile to the
// player's hand.
func giveSpecialCards(state *models.GameState, player models.PlayerID, kinds ...models.CardKind) {
	deck := state.SpecialDeck
	if deck == nil {
		return
	}
	for _, kind := range kinds {
		for _, card := range deck.Cards {
			if card.Kind != kind {
				continue
			}
			index := -1
			for position, id := range deck.DrawPile {
				if id == card.ID {
					index = position
					break
				}
			}
			if index < 0 {
				continue
			}
			deck.DrawPile = append(deck.DrawPile[:index], deck.DrawPile[index+1:]...)
			deck.Hands[player] = append(deck.Hands[player], card.ID)
			break
		}
	}
}

func giveNobleCards(state *models.GameState, player models.PlayerID, cards ...models.NobleCard) {
	if state.NobleDeck == nil {
		return
	}
	for index, card := range cards {
		card.ID = models.NobleCardID(fmt.Sprintf("KD%d", index+1))
		state.NobleDeck.Cards = append(state.NobleDeck.Cards, card)
		state.NobleDeck.Hands[player] = append(state.NobleDeck.Hands[player], card.ID)
	}
}

func forgeWinter(state *models.GameState) error {
	state.Season = models.SeasonWinter
	state.Turn = 4
	p1, p2 := ownedNobles(state, "P1"), ownedNobles(state, "P2")
	if len(p1) < 2 || len(p2) < 2 || len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks nobles or regions")
	}
	pope, bishop := state.Nobles[p1[0]].ID, state.Nobles[p1[1]].ID
	cardinal, excommunicated := state.Nobles[p2[0]].ID, state.Nobles[p2[1]].ID
	state.Bishops = []models.Bishop{
		{Region: state.Regions[0].ID, Noble: pope},
		{Region: state.Regions[1].ID, Noble: bishop},
		{Region: state.Regions[2].ID, Noble: cardinal},
	}
	state.Cardinals = []models.NobleID{pope, cardinal}
	state.Pope = &pope
	state.Excommunications = []models.Excommunication{{Noble: excommunicated, Reason: models.ExcommunicationPapal, By: pope}}
	// P1's second noble becomes a lady astrologer; a Witch joins P1's court.
	state.Nobles[p1[1]].Sex = models.SexFemale
	state.Nobles[p1[1]].Dignities = []models.Dignity{models.DignityAstrologer}
	state.Nobles = append(state.Nobles, models.Noble{
		ID: "N900", Code: "ZOE", Name: "Zoé", Sex: models.SexFemale, OwnerID: "P1",
		LocationID: state.Nobles[p1[0]].LocationID, Status: models.NobleStatusFree,
		Dignities: []models.Dignity{models.DignityWitch},
	})
	// The winter orders (inquiry, cardinal purchase, fiefs) are paid in R.
	for territoryID, territoryState := range state.TerritoryStates {
		if owner, controlled := state.TerritoryController(territoryID); controlled && territoryState.Infrastructures != nil && (owner == "P1" || owner == "P2") {
			territoryState.Resources = 40
			state.TerritoryStates[territoryID] = territoryState
		}
	}
	giveSpecialCards(state, "P1", models.CardKindFairWeather, models.CardKindRevolt, models.CardKindSeigneurialTax)
	giveNobleCards(state, "P1",
		models.NobleCard{Kind: models.NobleCardKindNoble, Code: "ZAL", Name: "Albert", Sex: models.SexMale},
		models.NobleCard{Kind: models.NobleCardKindDignity, Code: models.DignityBastardCardCode, Dignity: models.DignityBastard},
	)
	return nil
}

// forgeElection opens a bishop election: the first region without bishop and
// without any army has each of its territories occupied by a one-troop army of
// P1 or P2 (alternating, unless a player already controls it), and a free man
// of each player is added to run. The orders to try: `K E <noble> <seat>` to
// file a candidacy, `V E <noble> <seat>` to vote.
func forgeElection(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	armed := make(map[models.TerritoryID]bool)
	for _, army := range state.Armies {
		armed[army.TerritoryID] = true
	}
	var region *models.Region
	for index := range state.Regions {
		candidate := &state.Regions[index]
		if _, hasBishop := state.BishopOf(candidate.ID); hasBishop {
			continue
		}
		unarmed := true
		for _, id := range candidate.Territories {
			if armed[id] {
				unarmed = false
			}
		}
		if unarmed {
			region = candidate
			break
		}
	}
	if region == nil {
		return fmt.Errorf("no vacant region free of armies")
	}
	players := []models.PlayerID{"P1", "P2"}
	for index, id := range region.Territories {
		if _, controlled := state.TerritoryController(id); controlled {
			continue
		}
		armyID := models.ArmyID(fmt.Sprintf("ZE%d", index+1))
		state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: players[index%2], TerritoryID: id, Size: 1})
		territoryState := state.TerritoryStates[id]
		territoryState.Army = &armyID
		state.TerritoryStates[id] = territoryState
	}
	state.NextArmyID += len(state.Armies)
	for index, candidate := range []struct{ id, code, name string }{{"N910", "ZEL", "Aldric"}, {"N911", "ZEM", "Bertrand"}} {
		state.Nobles = append(state.Nobles, models.Noble{
			ID: models.NobleID(candidate.id), Code: candidate.code, Name: candidate.name, Sex: models.SexMale,
			OwnerID: players[index], LocationID: state.Nobles[ownedNobles(state, players[index])[0]].LocationID,
			Status: models.NobleStatusFree,
		})
	}
	fmt.Printf("demo election: bishopric %s (seat %s) is vacant and fully held; candidates ZEL (P1) and ZEM (P2)\n", region.ID, region.Seed)
	return nil
}

// forgeAnnexation gives P1 a barony on its capital castle and two neighbours,
// held by the first noble of P1 (head of its line of succession), and puts a
// one-troop P1 army on up to three free territories next to the barony. The
// orders to try: `T X <noble> <capital> <territory>` and `T R <noble> <capital>`.
func forgeAnnexation(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	var capitalID models.TerritoryID
	for _, player := range state.Players {
		if player.ID != "P1" || player.CapitalCastleID == nil {
			continue
		}
		for _, infrastructure := range state.Infrastructures {
			if infrastructure.ID == *player.CapitalCastleID {
				capitalID = infrastructure.TerritoryID
			}
		}
	}
	if capitalID == "" {
		return fmt.Errorf("P1 has no capital castle")
	}
	adjacencies := make(map[models.TerritoryID][]models.TerritoryID, len(state.Territories))
	for _, territory := range state.Territories {
		adjacencies[territory.ID] = territory.Adjacencies
	}
	free := func(id models.TerritoryID) bool {
		_, controlled := state.TerritoryController(id)
		return !controlled && state.TerritoryStates[id].Army == nil
	}
	ownedByP1 := func(id models.TerritoryID) bool {
		owner, controlled := state.TerritoryController(id)
		return controlled && owner == "P1"
	}
	members := []models.TerritoryID{capitalID}
	for _, wantFree := range []bool{false, true} {
		for _, id := range adjacencies[capitalID] {
			if len(members) == models.FiefMinTerritories {
				break
			}
			if containsTerritory(members, id) || (!wantFree && !ownedByP1(id)) || (wantFree && !free(id)) {
				continue
			}
			members = append(members, id)
		}
	}
	if len(members) < models.FiefMinTerritories {
		return fmt.Errorf("no room for a barony around %s", capitalID)
	}
	holder := state.Nobles[ownedNobles(state, "P1")[0]].ID
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "F1", Title: models.FiefTitleBarony, CapitalTerritoryID: capitalID,
		Territories: members, OwnerID: "P1", HolderNobleID: &holder,
	})
	var annexable []models.TerritoryID
	for _, member := range members {
		for _, id := range adjacencies[member] {
			if len(annexable) < 3 && !containsTerritory(members, id) && !containsTerritory(annexable, id) && free(id) {
				annexable = append(annexable, id)
			}
		}
	}
	for index, id := range annexable {
		armyID := models.ArmyID(fmt.Sprintf("ZX%d", index+1))
		state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: "P1", TerritoryID: id, Size: 1})
		territoryState := state.TerritoryStates[id]
		territoryState.Army = &armyID
		state.TerritoryStates[id] = territoryState
	}
	state.NextArmyID += len(annexable)
	fmt.Printf("demo annexation: barony %v (holder %s); annexable territories %v\n", members, state.Nobles[ownedNobles(state, "P1")[0]].Code, annexable)
	return nil
}

// forgeKing leaves the throne vacant (no king in the base game) and gives P1,
// P2 and P3 a free baron each, holder of a barony built from three territories
// the player controls. With the winter titles that makes 6 voices, 3 of them
// from the clergy. The orders to try: `K R <noble>` to file a candidacy,
// `V R <noble>` to vote.
func forgeKing(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	state.King = nil
	inFief := make(map[models.TerritoryID]bool)
	for _, fief := range state.Fiefs {
		for _, id := range fief.Territories {
			inFief[id] = true
		}
	}
	barons := []struct {
		player   models.PlayerID
		id, code string
		name     string
	}{{"P1", "N920", "ZRA", "Aldric"}, {"P2", "N921", "ZRB", "Bertrand"}, {"P3", "N922", "ZRC", "Conrad"}}
	for _, baron := range barons {
		var group []models.TerritoryID
		for _, territory := range state.Territories {
			owner, controlled := state.TerritoryController(territory.ID)
			if controlled && owner == baron.player && !inFief[territory.ID] && len(group) < models.FiefMinTerritories {
				group = append(group, territory.ID)
			}
		}
		if len(group) < models.FiefMinTerritories {
			return fmt.Errorf("%s controls fewer than %d free territories", baron.player, models.FiefMinTerritories)
		}
		for _, id := range group {
			inFief[id] = true
		}
		holder := models.NobleID(baron.id)
		state.Nobles = append(state.Nobles, models.Noble{
			ID: holder, Code: baron.code, Name: baron.name, Sex: models.SexMale,
			OwnerID: baron.player, LocationID: state.Nobles[ownedNobles(state, baron.player)[0]].LocationID,
			Status: models.NobleStatusFree,
		})
		state.Fiefs = append(state.Fiefs, models.Fief{
			ID: models.FiefID("FZK" + baron.code), Title: models.FiefTitleBarony, CapitalTerritoryID: group[0],
			Territories: group, OwnerID: baron.player, HolderNobleID: &holder,
		})
	}
	fmt.Println("demo king: the throne is vacant; candidates ZRA (P1), ZRB (P2), ZRC (P3), or the pope and cardinal houses' nobles")
	return nil
}

// forgeCrown makes the excommunicated noble of P2 the king (an excommunicated
// king keeps the crown), gives him a barony and a queen of P3, and stakes three
// claims on the couple, each by a child of one of the spouses' houses: ZAN (P3,
// a woman) then ZGA (P3) on the king, ZPI (P2) on the queen. The order to try: `J <pope or cardinal> <king>` from both P1's
// pope and P2's cardinal. The trial is judged at the end of the winter; the
// king dies, the barony goes to ZAN (first claim on the king) and the crown to
// ZPI, the first heir who can reign (ZAN is a woman).
func forgeCrown(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	if len(state.Excommunications) == 0 {
		return fmt.Errorf("the winter scenario excommunicated nobody")
	}
	king := state.Excommunications[0].Noble
	state.King = &king
	location := state.Nobles[ownedNobles(state, "P1")[0]].LocationID
	for _, noble := range []struct {
		id, code, name, owner string
		sex                   models.Sex
	}{
		{"N930", "ZQU", "Quitterie", "P3", models.SexFemale},
		{"N931", "ZAN", "Anne", "P3", models.SexFemale},
		{"N932", "ZPI", "Pierre", "P2", models.SexMale},
		{"N933", "ZGA", "Gaspard", "P3", models.SexMale},
	} {
		state.Nobles = append(state.Nobles, models.Noble{
			ID: models.NobleID(noble.id), Code: noble.code, Name: noble.name, Sex: noble.sex,
			OwnerID: models.PlayerID(noble.owner), LocationID: location, Status: models.NobleStatusFree,
		})
	}
	state.Marriages = []models.Marriage{{NobleA: king, NobleB: "N930", Turn: 1}}
	queen := models.NobleID("N930")
	state.Queen = &queen
	state.Claims = []models.Claim{
		{Heir: "N931", Target: king, Spouse: "N930", Turn: 1},
		{Heir: "N932", Target: "N930", Spouse: king, Turn: 2},
		{Heir: "N933", Target: king, Spouse: "N930", Turn: 3},
	}
	inFief := make(map[models.TerritoryID]bool)
	for _, fief := range state.Fiefs {
		for _, id := range fief.Territories {
			inFief[id] = true
		}
	}
	var group []models.TerritoryID
	for _, territory := range state.Territories {
		owner, controlled := state.TerritoryController(territory.ID)
		if controlled && owner == "P2" && !inFief[territory.ID] && len(group) < models.FiefMinTerritories {
			group = append(group, territory.ID)
		}
	}
	if len(group) < models.FiefMinTerritories {
		return fmt.Errorf("P2 controls fewer than %d free territories", models.FiefMinTerritories)
	}
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "FZKKING", Title: models.FiefTitleBarony, CapitalTerritoryID: group[0],
		Territories: group, OwnerID: "P2", HolderNobleID: &king,
	})
	fmt.Println("demo crown: the king is P2's excommunicated noble; try a J trial on him from P1's pope and P2's cardinal")
	return nil
}

// forgeRoyal makes a free noble of P1 (ZRK) the king, on top of the annexation
// scenario (P1 barony F1 with annexable territories), and adds a barony of P2
// held by ZRB and a crown fief: vacant, attached to P3 who has no noble left.
func forgeRoyal(state *models.GameState) error {
	if err := forgeAnnexation(state); err != nil {
		return err
	}
	location := state.Nobles[ownedNobles(state, "P1")[0]].LocationID
	for _, noble := range []struct {
		id, code, name, owner string
	}{{"N940", "ZRK", "Robert", "P1"}, {"N941", "ZRB", "Bertrand", "P2"}} {
		state.Nobles = append(state.Nobles, models.Noble{
			ID: models.NobleID(noble.id), Code: noble.code, Name: noble.name, Sex: models.SexMale,
			OwnerID: models.PlayerID(noble.owner), LocationID: location, Status: models.NobleStatusFree,
		})
	}
	king := models.NobleID("N940")
	state.King = &king
	inFief := make(map[models.TerritoryID]bool)
	for _, fief := range state.Fiefs {
		for _, id := range fief.Territories {
			inFief[id] = true
		}
	}
	group := func(player models.PlayerID) ([]models.TerritoryID, error) {
		var members []models.TerritoryID
		for _, territory := range state.Territories {
			owner, controlled := state.TerritoryController(territory.ID)
			if controlled && owner == player && !inFief[territory.ID] && len(members) < models.FiefMinTerritories {
				members = append(members, territory.ID)
			}
		}
		if len(members) < models.FiefMinTerritories {
			return nil, fmt.Errorf("%s controls fewer than %d free territories", player, models.FiefMinTerritories)
		}
		for _, id := range members {
			inFief[id] = true
		}
		return members, nil
	}
	p2Members, err := group("P2")
	if err != nil {
		return err
	}
	holder := models.NobleID("N941")
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "FZKROYAL2", Title: models.FiefTitleBarony, CapitalTerritoryID: p2Members[0],
		Territories: p2Members, OwnerID: "P2", HolderNobleID: &holder,
	})
	// The crown fief gathers free territories next to nothing in particular:
	// three one-troop armies of P3 take them, and P3 has no noble left.
	remaining := state.Nobles[:0]
	for _, noble := range state.Nobles {
		if noble.OwnerID != "P3" {
			remaining = append(remaining, noble)
		}
	}
	state.Nobles = remaining
	var crownMembers []models.TerritoryID
	for _, territory := range state.Territories {
		if _, controlled := state.TerritoryController(territory.ID); controlled || inFief[territory.ID] ||
			state.TerritoryStates[territory.ID].Army != nil || len(crownMembers) == models.FiefMinTerritories {
			continue
		}
		armyID := models.ArmyID(fmt.Sprintf("ZC%d", len(crownMembers)+1))
		state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: "P3", TerritoryID: territory.ID, Size: 1})
		territoryState := state.TerritoryStates[territory.ID]
		territoryState.Army = &armyID
		state.TerritoryStates[territory.ID] = territoryState
		crownMembers = append(crownMembers, territory.ID)
	}
	if len(crownMembers) < models.FiefMinTerritories {
		return fmt.Errorf("not enough free territories for the crown fief")
	}
	state.NextArmyID += len(crownMembers)
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "FZKCROWN", Title: models.FiefTitleBarony, CapitalTerritoryID: crownMembers[0],
		Territories: crownMembers, OwnerID: "P3",
	})
	fmt.Printf("demo royal: king ZRK (P1); P2 barony %v (ZRB); crown fief %v\n", p2Members, crownMembers)
	return nil
}

func forgeDissolution(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	location := state.Nobles[ownedNobles(state, "P1")[0]].LocationID
	for _, noble := range []struct {
		id, code, name, owner string
		sex                   models.Sex
	}{
		{"N910", "ZEL", "Eléonore", "P1", models.SexFemale},
		{"N911", "ZRO", "Roland", "P2", models.SexMale},
		{"N912", "ZCA", "Catherine", "P2", models.SexFemale},
		{"N913", "ZGU", "Guibert", "P3", models.SexMale},
		{"N914", "ZAD", "Aldric", "P4", models.SexMale},
		{"N915", "ZLU", "Ludivine", "P1", models.SexFemale},
	} {
		state.Nobles = append(state.Nobles, models.Noble{
			ID: models.NobleID(noble.id), Code: noble.code, Name: noble.name, Sex: noble.sex,
			OwnerID: models.PlayerID(noble.owner), LocationID: location, Status: models.NobleStatusFree,
		})
	}
	state.Marriages = []models.Marriage{
		{NobleA: "N910", NobleB: "N911", Turn: 1},
		{NobleA: "N912", NobleB: "N913", Turn: 1},
	}
	return nil
}

func forgeInquiry(state *models.GameState) error {
	if err := forgeWinter(state); err != nil {
		return err
	}
	location := state.Nobles[ownedNobles(state, "P2")[0]].LocationID
	for _, lady := range []struct {
		id, code, name string
		dignity        models.Dignity
	}{
		{"N901", "ZMA", "Mabile", models.DignityWitch},
		{"N902", "ZIS", "Isabeau", models.DignitySpy},
	} {
		state.Nobles = append(state.Nobles, models.Noble{
			ID: models.NobleID(lady.id), Code: lady.code, Name: lady.name, Sex: models.SexFemale, OwnerID: "P2",
			LocationID: location, Status: models.NobleStatusFree, Dignities: []models.Dignity{lady.dignity},
		})
	}
	return nil
}

// forgeTunnel puts a P1 army, with P1's first noble, on a free territory next
// to P2's capital castle, with exactly the force of the defense there
// (troops, castle bonus and the commanding noble). The attack then ties
// against the fortifications and wins when the Tunnel card ignores them. The
// orders to try, with the codes printed by the game:
//
//	A <origin> <target>          lost on its own (tie)
//	P SO <origin> - <target>     the same attack, tunneled
func forgeTunnel(state *models.GameState) error {
	var capitalID models.TerritoryID
	for _, player := range state.Players {
		if player.ID != "P2" || player.CapitalCastleID == nil {
			continue
		}
		for _, infrastructure := range state.Infrastructures {
			if infrastructure.ID == *player.CapitalCastleID {
				capitalID = infrastructure.TerritoryID
			}
		}
	}
	p1 := ownedNobles(state, "P1")
	if capitalID == "" || len(p1) == 0 {
		return fmt.Errorf("the base game lacks a P2 capital or a P1 noble")
	}
	defense := 1 // castle bonus
	for _, army := range state.Armies {
		if army.TerritoryID == capitalID {
			defense += army.Size
		}
	}
	for _, noble := range state.Nobles {
		if noble.OwnerID == "P2" && noble.LocationID == capitalID && noble.Status == models.NobleStatusFree {
			defense++
			break
		}
	}
	var originID models.TerritoryID
	for _, territory := range state.Territories {
		if territory.ID != capitalID {
			continue
		}
		for _, id := range territory.Adjacencies {
			if _, controlled := state.TerritoryController(id); !controlled && state.TerritoryStates[id].Army == nil {
				originID = id
				break
			}
		}
	}
	if originID == "" {
		return fmt.Errorf("no free territory next to %s", capitalID)
	}
	armyID := models.ArmyID("ZT1")
	state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: "P1", TerritoryID: originID, Size: defense - 1})
	territoryState := state.TerritoryStates[originID]
	territoryState.Army = &armyID
	state.TerritoryStates[originID] = territoryState
	state.NextArmyID++
	// A village with a large stock keeps the army supplied: a famished army
	// attacks with no force at all.
	state.Infrastructures = append(state.Infrastructures, models.Infrastructure{ID: "ZTV", Type: models.InfraTypeVillage, Level: 1, TerritoryID: originID})
	villageID := models.InfraID("ZTV")
	territoryState = state.TerritoryStates[originID]
	territoryState.Infrastructures = &villageID
	territoryState.Resources = 40
	state.TerritoryStates[originID] = territoryState
	state.Nobles[p1[0]].LocationID = originID
	giveSpecialCards(state, "P1", models.CardKindTunnel)
	fmt.Printf("demo tunnel: P1 army of %d (+ noble) on %s attacks the P2 castle on %s (defense %d)\n", defense-1, originID, capitalID, defense)
	return nil
}

func forgeAction(state *models.GameState) error {
	if len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks regions")
	}
	year := state.Year()
	state.Auguries[year] = models.YearAugury{
		Year:       year,
		Capacities: map[models.Season]int{models.SeasonSpring: 1, models.SeasonSummer: 1, models.SeasonAutumn: 1},
		Revealed:   true,
		Calamities: []models.Calamity{
			{CardID: "ZF1", Kind: models.CardKindFamine, Year: year, Season: models.SeasonSpring, RegionSeed: state.Regions[0].Seed},
			{CardID: "ZB1", Kind: models.CardKindBadWeather, Year: year, Season: models.SeasonSummer, RegionSeed: state.Regions[1].Seed, Ritual: true},
			{CardID: "ZB2", Kind: models.CardKindBadWeather, Year: year, Season: models.SeasonAutumn, RegionSeed: state.Regions[2].Seed},
		},
	}
	if state.SpecialDeck != nil {
		for _, card := range []models.SpecialCard{
			{ID: "ZF1", Kind: models.CardKindFamine},
			{ID: "ZB1", Kind: models.CardKindBadWeather},
			{ID: "ZB2", Kind: models.CardKindBadWeather},
		} {
			state.SpecialDeck.Cards = append(state.SpecialDeck.Cards, card)
		}
	}
	giveSpecialCards(state, "P1", models.CardKindFairWeather, models.CardKindAbundantHarvest, models.CardKindRevolt)
	return nil
}

// forgeAppeasement sets up the revolt appeasement cases (specs/religieux.md):
// P1's first noble is the bishop of the region it stands in, its second noble
// a cardinal (and bishop of another region), and a new abbess stands beside
// the cardinal. Rebel armies wait in four places: 2 troops and 1 troop in the
// bishop's region, 3 troops in the cardinal's, 1 troop in a region where P1 has
// no cleric. The orders to try, with the noble codes printed by the game:
//
//	P AG <bishop> <rebels in his region>      free rite, may kill
//	P AP <bishop> <rebels in his region>      paid, 2^size R
//	P AP <bishop> <rebels elsewhere>          rejected: not his bishopric
//	P AP <cardinal> <rebels anywhere>         paid, anywhere
//	P AG <abbess> <rebels where she stands>   the only way open to her
func forgeAppeasement(state *models.GameState) error {
	p1 := ownedNobles(state, "P1")
	if len(p1) < 2 || len(state.Regions) < 3 {
		return fmt.Errorf("the base game lacks nobles or regions")
	}
	regionOf := func(territoryID models.TerritoryID) int {
		for index, region := range state.Regions {
			for _, id := range region.Territories {
				if id == territoryID {
					return index
				}
			}
		}
		return -1
	}
	bishop, cardinal := state.Nobles[p1[0]], state.Nobles[p1[1]]
	bishopRegion, cardinalRegion := regionOf(bishop.LocationID), regionOf(cardinal.LocationID)
	if bishopRegion < 0 || cardinalRegion < 0 {
		return fmt.Errorf("a noble stands outside every region")
	}
	if cardinalRegion == bishopRegion {
		// Their own bishoprics must differ: the cardinal takes the next region.
		cardinalRegion = (bishopRegion + 1) % len(state.Regions)
	}
	state.Bishops = []models.Bishop{
		{Region: state.Regions[bishopRegion].ID, Noble: bishop.ID},
		{Region: state.Regions[cardinalRegion].ID, Noble: cardinal.ID},
	}
	state.Cardinals = []models.NobleID{cardinal.ID}
	state.Nobles = append(state.Nobles, models.Noble{
		ID: "N901", Code: "ZAB", Name: "Aliénor", Sex: models.SexFemale, OwnerID: "P1",
		LocationID: cardinal.LocationID, Status: models.NobleStatusFree,
		Dignities: []models.Dignity{models.DignityAbbess}, AbbeyRegion: state.Regions[cardinalRegion].Seed,
	})
	occupied := make(map[models.TerritoryID]bool)
	for _, army := range state.Armies {
		occupied[army.TerritoryID] = true
	}
	freeIn := func(region int) (models.TerritoryID, bool) {
		for _, id := range state.Regions[region].Territories {
			if !occupied[id] {
				occupied[id] = true
				return id, true
			}
		}
		return "", false
	}
	other := 0
	for other == bishopRegion || other == cardinalRegion {
		other++
	}
	next := 1
	for _, rebel := range []struct {
		region int
		size   int
	}{{bishopRegion, 2}, {bishopRegion, 1}, {cardinalRegion, 3}, {other, 1}} {
		territoryID, found := freeIn(rebel.region)
		if !found {
			return fmt.Errorf("region %d has no free territory for a rebel army", rebel.region)
		}
		armyID := models.ArmyID(fmt.Sprintf("ZR%d", next))
		next++
		state.Armies = append(state.Armies, models.Army{ID: armyID, OwnerID: models.NeutralPlayerID, TerritoryID: territoryID, Size: rebel.size})
		territoryState := state.TerritoryStates[territoryID]
		territoryState.Army = &armyID
		state.TerritoryStates[territoryID] = territoryState
		fmt.Printf("demo appeasement: %d rebel troops at %s (region %s)\n", rebel.size, territoryID, state.Regions[rebel.region].Seed)
	}
	state.NextArmyID += len(state.Armies)
	for territoryID, territoryState := range state.TerritoryStates {
		if owner, controlled := state.TerritoryController(territoryID); controlled && owner == "P1" && territoryState.Infrastructures != nil {
			territoryState.Resources = 40
			state.TerritoryStates[territoryID] = territoryState
		}
	}
	fmt.Printf("demo appeasement: bishop %s (region %s), cardinal %s (bishop of %s), abbess ZAB stands at %s\n",
		bishop.Code, state.Regions[bishopRegion].Seed, cardinal.Code, state.Regions[cardinalRegion].Seed, cardinal.LocationID)
	return nil
}

// forgeTithe reuses the appeasement clerics (a bishop and a cardinal of P1),
// hands P1 two Tax cards and makes the bishop also the baron of a fief whose
// capital is the seed of his own bishopric, so the same territory can be taxed
// as a fief (seigneurial tax) or as a bishopric (tithe). The orders to try:
//
//	P TX <bishop> <seed>     seigneurial tax on his fief
//	P DI <bishop> <seed>     tithe on his bishopric
//	P DI <cardinal> <seed>   tithe on any bishopric
func forgeTithe(state *models.GameState) error {
	if err := forgeAppeasement(state); err != nil {
		return err
	}
	giveSpecialCards(state, "P1", models.CardKindSeigneurialTax, models.CardKindSeigneurialTax)
	p1 := ownedNobles(state, "P1")
	bishop := state.Nobles[p1[0]]
	region, found := state.RegionOfTerritory(bishop.LocationID)
	if !found {
		return fmt.Errorf("the bishop stands outside every region")
	}
	territories := []models.TerritoryID{region.Seed}
	for _, territory := range state.Territories {
		if territory.ID != region.Seed && len(territories) < models.FiefMinTerritories && containsTerritory(territory.Adjacencies, region.Seed) {
			territories = append(territories, territory.ID)
		}
	}
	if len(territories) < models.FiefMinTerritories {
		return fmt.Errorf("the bishopric seed %s has too few neighbors for a fief", region.Seed)
	}
	title, _ := models.FiefTitleForSize(len(territories))
	holder := bishop.ID
	state.Fiefs = append(state.Fiefs, models.Fief{
		ID: "F901", Title: title, CapitalTerritoryID: region.Seed,
		Territories: territories, OwnerID: "P1", HolderNobleID: &holder,
	})
	fmt.Printf("demo tithe: %s is bishop of %s and holds a fief whose capital is %s\n", bishop.Code, region.Seed, region.Seed)
	return nil
}

func containsTerritory(list []models.TerritoryID, target models.TerritoryID) bool {
	for _, id := range list {
		if id == target {
			return true
		}
	}
	return false
}
