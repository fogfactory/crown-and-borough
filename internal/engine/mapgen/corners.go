package mapgen

import "sort"

// CornerContact is a vertex shared by two territories that have no frontier in
// common: they touch only at that point and are not adjacent. Between holds
// their codes, sorted.
type CornerContact struct {
	Point   [2]int    `json:"point"`
	Between [2]string `json:"between"`
}

// cornerContacts returns one contact per vertex where some pair of territories
// shares the vertex but no polygon edge, ordered by point. A vertex where every
// pair shares a frontier, such as an ordinary three-way junction, has none.
func cornerContacts(territories []Territory) []CornerContact {
	edges := make(map[string]map[[2][2]int]bool, len(territories))
	owners := make(map[[2]int][]string)
	for _, territory := range territories {
		set := make(map[[2][2]int]bool, len(territory.Points))
		for index, point := range territory.Points {
			next := territory.Points[(index+1)%len(territory.Points)]
			set[orderedEdge(point, next)] = true
			owners[point] = append(owners[point], territory.ID)
		}
		edges[territory.ID] = set
	}
	shareEdge := func(first, second string) bool {
		for edge := range edges[first] {
			if edges[second][edge] {
				return true
			}
		}
		return false
	}

	contacts := make([]CornerContact, 0)
	for point, ids := range owners {
		sort.Strings(ids)
		if contact, found := firstCornerPair(ids, shareEdge); found {
			contacts = append(contacts, CornerContact{Point: point, Between: contact})
		}
	}
	sort.Slice(contacts, func(i, j int) bool {
		first, second := contacts[i].Point, contacts[j].Point
		return first[0] < second[0] || (first[0] == second[0] && first[1] < second[1])
	})
	return contacts
}

func firstCornerPair(ids []string, shareEdge func(first, second string) bool) ([2]string, bool) {
	for i, first := range ids {
		for _, second := range ids[i+1:] {
			if !shareEdge(first, second) {
				return [2]string{first, second}, true
			}
		}
	}
	return [2]string{}, false
}

func orderedEdge(first, second [2]int) [2][2]int {
	if first[0] > second[0] || (first[0] == second[0] && first[1] > second[1]) {
		return [2][2]int{second, first}
	}
	return [2][2]int{first, second}
}
