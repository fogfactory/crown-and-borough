package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

func TestWinterCostsHandlerServesConfiguredCosts(t *testing.T) {
	balance, err := assetgen.LoadBalance("../../assets")
	if err != nil {
		t.Fatalf("LoadBalance = %v", err)
	}

	recorder := httptest.NewRecorder()
	WinterCostsHandler(balance).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/balance", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /api/balance = %d, want %d", recorder.Code, http.StatusOK)
	}

	var got WinterCostsView
	if err := json.Unmarshal(recorder.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode costs: %v", err)
	}
	want := winterCostsView(balance)
	if got.Castle != want.Castle || got.Troop != want.Troop || got.Noble != want.Noble ||
		got.SupplyDepot != want.SupplyDepot || got.Liberation != want.Liberation {
		t.Fatalf("costs = %#v, want %#v", got, want)
	}
	if len(got.MillLevels) != len(want.MillLevels) {
		t.Fatalf("mill levels = %#v, want %#v", got.MillLevels, want.MillLevels)
	}
	for index := range want.MillLevels {
		if got.MillLevels[index] != want.MillLevels[index] {
			t.Errorf("mill level %d = %d, want %d", index, got.MillLevels[index], want.MillLevels[index])
		}
	}
}

func TestWinterCostsHandlerRejectsNonGet(t *testing.T) {
	recorder := httptest.NewRecorder()
	WinterCostsHandler(assetgen.Balance{}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodPost, "/api/balance", nil),
	)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST /api/balance = %d, want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}
