package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
)

func TestGenerateSeedUsesReadableAssetSlugs(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}

	seed, err := GenerateSeed(assets)
	if err != nil {
		t.Fatalf("generate seed: %v", err)
	}
	if !regexp.MustCompile(`^[a-z0-9]+-de-[a-z0-9]+$`).MatchString(seed) {
		t.Fatalf("generated seed = %q, want a readable asset seed", seed)
	}
}

func TestSlugAssetNameRemovesAccents(t *testing.T) {
	if got := slugAssetName("Adélaïde"); got != "adelaide" {
		t.Fatalf("slugAssetName = %q, want adelaide", got)
	}
}

func TestSeedHandlerReturnsASeed(t *testing.T) {
	assets, err := assetgen.Load("../../assets")
	if err != nil {
		t.Fatalf("load assets: %v", err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/seed", nil)
	SeedHandler(assets).ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("seed status = %d: %s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Seed string `json:"seed"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode seed response: %v", err)
	}
	if response.Seed == "" {
		t.Fatal("seed response is empty")
	}
}
