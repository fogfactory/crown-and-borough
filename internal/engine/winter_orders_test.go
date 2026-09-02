package engine

import (
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/models"
)

func TestNewExecutableWinterOrder(t *testing.T) {
	validTypes := []models.WinterOrderType{
		models.WinterOrderTypeRecruitNoble,
		models.WinterOrderTypeRecruitTroop,
		models.WinterOrderTypeBuild,
		models.WinterOrderTypeElectCapital,
		models.WinterOrderTypeLiberateNoble,
		models.WinterOrderTypeHostage,
		models.WinterOrderTypeDungeon,
	}
	for _, orderType := range validTypes {
		if executable := newExecutableWinterOrder(models.WinterOrder{Type: orderType}); executable == nil {
			t.Errorf("newExecutableWinterOrder(%q) returned nil", orderType)
		}
	}
	if executable := newExecutableWinterOrder(models.WinterOrder{Type: "invalid"}); executable != nil {
		t.Fatalf("newExecutableWinterOrder(invalid) = %T, want nil", executable)
	}
}
