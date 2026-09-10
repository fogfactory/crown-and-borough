package api

import (
	"net/http"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

// WinterCostsView is the public subset of the balance needed to estimate
// winter orders in the client.
type WinterCostsView struct {
	Castle      int   `json:"castle"`
	MillLevels  []int `json:"millLevels"`
	Troop       int   `json:"troop"`
	Noble       int   `json:"noble"`
	SupplyDepot int   `json:"supplyDepot"`
	Liberation  int   `json:"liberation"`
}

// WinterCostsHandler serves the current winter investment costs. The handler
// is used by the hotseat API, where the session already provides the game
// context and does not need an additional game identifier.
func WinterCostsHandler(balance assetgen.Balance) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		writeJSON(w, http.StatusOK, winterCostsView(balance))
	}
}

func winterCostsView(balance assetgen.Balance) WinterCostsView {
	return WinterCostsView{
		Castle:      balance.Costs.Castle,
		MillLevels:  append([]int(nil), balance.Costs.MillLevels...),
		Troop:       balance.Costs.Troop,
		Noble:       balance.Costs.Noble,
		SupplyDepot: balance.Costs.SupplyDepot,
		Liberation:  balance.Costs.Liberation,
	}
}
