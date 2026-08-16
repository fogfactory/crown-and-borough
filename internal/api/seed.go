package api

import (
	cryptorand "crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"strings"
	"unicode"

	"github.com/fogfactory/crown-and-borough/internal/db/assetgen"
	"golang.org/x/text/unicode/norm"
)

var errSeedAssetsUnavailable = errors.New("seed assets are unavailable")

// GenerateSeed creates a readable deterministic seed from the same names and
// communes used to initialize a game.
func GenerateSeed(assets assetgen.Assets) (string, error) {
	if len(assets.Prenoms) == 0 || len(assets.Communes) == 0 {
		return "", errSeedAssetsUnavailable
	}
	firstName, err := randomIndex(len(assets.Prenoms))
	if err != nil {
		return "", err
	}
	commune, err := randomIndex(len(assets.Communes))
	if err != nil {
		return "", err
	}
	seed := strings.Join(
		[]string{
			slugAssetName(assets.Prenoms[firstName].Name),
			"de",
			slugAssetName(assets.Communes[commune].Name),
		},
		"-",
	)
	if strings.Trim(seed, "-") == "de" {
		return "", errSeedAssetsUnavailable
	}
	return seed, nil
}

// SeedHandler exposes a new readable seed for the online create-game form.
func SeedHandler(assets assetgen.Assets) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			methodNotAllowed(w, http.MethodGet)
			return
		}
		seed, err := GenerateSeed(assets)
		if err != nil {
			writeAPIError(w, http.StatusInternalServerError, "seed_unavailable", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, struct {
			Seed string `json:"seed"`
		}{Seed: seed})
	})
}

func randomIndex(length int) (int, error) {
	value, err := cryptorand.Int(cryptorand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0, err
	}
	return int(value.Int64()), nil
}

func slugAssetName(value string) string {
	value = norm.NFD.String(strings.ToLower(strings.TrimSpace(value)))
	var builder strings.Builder
	pendingHyphen := false
	for _, character := range value {
		if unicode.Is(unicode.Mn, character) {
			continue
		}
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if pendingHyphen && builder.Len() > 0 {
				builder.WriteByte('-')
			}
			builder.WriteRune(character)
			pendingHyphen = false
			continue
		}
		if builder.Len() > 0 {
			pendingHyphen = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
