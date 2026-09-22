package engine

import (
	"sort"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

// AnnouncementReport describes a scheduled calamity the players can still
// prepare for. Announcements are surfaced as a warning in the special-cards
// panel from the moment a calamity is drawn until it applies or is countered.
type AnnouncementReport struct {
	Kind   models.CardKind    `json:"kind"`
	Season models.Season      `json:"season"`
	Region models.TerritoryID `json:"region"`
	Year   int                `json:"year"`
}

func seasonRank(season models.Season) int {
	switch season {
	case models.SeasonSpring:
		return 0
	case models.SeasonSummer:
		return 1
	case models.SeasonAutumn:
		return 2
	case models.SeasonWinter:
		return 3
	}
	return 0
}

// SeedRegionEffects replaces the state's regional effects with the calamities
// scheduled for the current season. Called after the calendar advances, it
// makes the map overlay and the command post surface a calamity on the turn it
// applies, while players can still counter it with a bonus card.
func SeedRegionEffects(state *models.GameState) {
	if state == nil {
		return
	}
	effects := []models.ActiveRegionEffect{}
	seen := make(map[models.TerritoryID]map[models.CardKind]bool)
	if augury, exists := state.Auguries[state.Year()]; exists {
		for _, calamity := range augury.Calamities {
			if calamity.Season != state.Season {
				continue
			}
			if seen[calamity.RegionSeed] == nil {
				seen[calamity.RegionSeed] = make(map[models.CardKind]bool)
			}
			if seen[calamity.RegionSeed][calamity.Kind] {
				continue
			}
			seen[calamity.RegionSeed][calamity.Kind] = true
			effects = append(effects, models.ActiveRegionEffect{
				Kind: calamity.Kind, RegionSeed: calamity.RegionSeed,
				Season: state.Season, Year: state.Year(),
			})
		}
	}
	sort.Slice(effects, func(i, j int) bool {
		if effects[i].RegionSeed != effects[j].RegionSeed {
			return effects[i].RegionSeed < effects[j].RegionSeed
		}
		return effects[i].Kind < effects[j].Kind
	})
	state.ActiveRegionEffects = effects
}

// PendingAnnouncements lists scheduled calamities of the given year whose
// season is still upcoming (inclusive keeps the given season, exclusive drops
// it), followed by every calamity scheduled for the following year regardless
// of the augury reveal state.
func PendingAnnouncements(state *models.GameState, year int, season models.Season, inclusive bool) []AnnouncementReport {
	announcements := []AnnouncementReport{}
	if state == nil {
		return announcements
	}
	threshold := seasonRank(season)
	if augury, exists := state.Auguries[year]; exists {
		for _, calamity := range augury.Calamities {
			rank := seasonRank(calamity.Season)
			if rank > threshold || (inclusive && rank == threshold) {
				announcements = append(announcements, AnnouncementReport{
					Kind: calamity.Kind, Season: calamity.Season,
					Region: calamity.RegionSeed, Year: calamity.Year,
				})
			}
		}
	}
	if augury, exists := state.Auguries[year+1]; exists {
		for _, calamity := range augury.Calamities {
			announcements = append(announcements, AnnouncementReport{
				Kind: calamity.Kind, Season: calamity.Season,
				Region: calamity.RegionSeed, Year: calamity.Year,
			})
		}
	}
	sort.SliceStable(announcements, func(i, j int) bool {
		if announcements[i].Year != announcements[j].Year {
			return announcements[i].Year < announcements[j].Year
		}
		rankI, rankJ := seasonRank(announcements[i].Season), seasonRank(announcements[j].Season)
		if rankI != rankJ {
			return rankI < rankJ
		}
		if announcements[i].Region != announcements[j].Region {
			return announcements[i].Region < announcements[j].Region
		}
		return announcements[i].Kind < announcements[j].Kind
	})
	return announcements
}
