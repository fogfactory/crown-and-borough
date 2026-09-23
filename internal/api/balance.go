package api

import "github.com/fogfactory/crown-and-borough/internal/db/assetgen"

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
