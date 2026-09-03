package engine

import (
	"math/rand/v2"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

type ExecutableOrder interface {
	Apply(*ExecutionContext)
}

type ExecutionContext struct {
	resolution   *resolutionContext
	playerID     models.PlayerID
	season       models.Season
	firstNameRNG *rand.Rand
}

type winterOrderFactory func(models.WinterOrder) ExecutableOrder

var winterOrderFactories = map[models.WinterOrderType]winterOrderFactory{
	models.WinterOrderTypeRecruitNoble:  func(order models.WinterOrder) ExecutableOrder { return recruitNobleOrder{order: order} },
	models.WinterOrderTypeRecruitTroop:  func(order models.WinterOrder) ExecutableOrder { return recruitTroopOrder{order: order} },
	models.WinterOrderTypeBuild:         func(order models.WinterOrder) ExecutableOrder { return buildOrder{order: order} },
	models.WinterOrderTypeElectCapital:  func(order models.WinterOrder) ExecutableOrder { return electCapitalOrder{order: order} },
	models.WinterOrderTypeLiberateNoble: func(order models.WinterOrder) ExecutableOrder { return liberateNobleOrder{order: order} },
	models.WinterOrderTypeHostage:       func(order models.WinterOrder) ExecutableOrder { return hostageOrder{order: order} },
	models.WinterOrderTypeDungeon:       func(order models.WinterOrder) ExecutableOrder { return dungeonOrder{order: order} },
}

func newExecutableWinterOrder(order models.WinterOrder) ExecutableOrder {
	factory := winterOrderFactories[order.Type]
	if factory == nil {
		return nil
	}
	return factory(order)
}

func executeWinterOrder(ctx *resolutionContext, playerID models.PlayerID, order models.WinterOrder, firstNameRNG *rand.Rand) {
	executable := newExecutableWinterOrder(order)
	if executable == nil {
		ctx.rejectWinterOrder(playerID, order, "invalid_winter_order")
		return
	}
	executable.Apply(&ExecutionContext{
		resolution:   ctx,
		playerID:     playerID,
		firstNameRNG: firstNameRNG,
	})
}
