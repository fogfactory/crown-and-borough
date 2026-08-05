// Package assetgen loads the static world assets (communes with terrain
// affinities and noble first names) from CSV files and validates the
// invariants they must satisfy at startup.
//
// The two CSV files live in the assets directory and use a semicolon separator
// with a header line. Codes are strict uppercase trigrams. Codes and names
// must be unique within each file (no cross-file uniqueness: the entity type
// disambiguates).
package assetgen

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Asset is a noble first name with its trigram code.
type Asset struct {
	Code string
	Name string
}

// Commune is a territory name with its trigram code and preferred terrain.
type Commune struct {
	Code    string
	Name    string
	Terrain string
}

// Assets holds all the world data loaded from the assets directory.
type Assets struct {
	Communes []Commune
	Prenoms  []Asset
}

var validTerrains = map[string]bool{
	"plain":    true,
	"forest":   true,
	"hill":     true,
	"mountain": true,
	"swamp":    true,
	"any":      true,
}

var communeTerrains = [...]string{
	"plain",
	"forest",
	"hill",
	"mountain",
	"swamp",
	"any",
}

// Load reads communes.csv and prenoms.csv from dir and validates them, failing
// fast at the first invalid record. A missing or empty file is an explicit
// error.
func Load(dir string) (Assets, error) {
	var a Assets
	var err error

	if a.Communes, err = loadCommunes(filepath.Join(dir, "communes.csv")); err != nil {
		return Assets{}, err
	}
	if a.Prenoms, err = loadAssets(filepath.Join(dir, "prenoms.csv")); err != nil {
		return Assets{}, err
	}
	return a, nil
}

func loadCommunes(path string) ([]Commune, error) {
	rows, err := loadCSV(path, 3)
	if err != nil {
		return nil, err
	}
	communes := make([]Commune, 0, len(rows))
	seenCodes := make(map[string]bool, len(rows))
	seenNames := make(map[string]bool, len(rows))
	terrainCounts := make(map[string]int, len(communeTerrains))
	for i, row := range rows {
		line := i + 2 // header is line 1
		code, name, terrain := strings.TrimSpace(row[0]), strings.TrimSpace(row[1]), strings.TrimSpace(row[2])
		if !isTrigram(code) {
			return nil, fmt.Errorf("assetgen: %s: line %d: invalid code %q (want exactly 3 uppercase letters)", path, line, code)
		}
		if name == "" {
			return nil, fmt.Errorf("assetgen: %s: line %d: empty name", path, line)
		}
		if !validTerrains[terrain] {
			return nil, fmt.Errorf("assetgen: %s: line %d: invalid terrain %q (want one of plain|forest|hill|mountain|swamp|any)", path, line, terrain)
		}
		if seenCodes[code] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate code %q", path, line, code)
		}
		if seenNames[name] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate name %q", path, line, name)
		}
		seenCodes[code] = true
		seenNames[name] = true
		terrainCounts[terrain]++
		communes = append(communes, Commune{Code: code, Name: name, Terrain: terrain})
	}
	for _, terrain := range communeTerrains {
		if terrainCounts[terrain] == 0 {
			return nil, fmt.Errorf("assetgen: %s: no commune with terrain %q (want at least one per affinity)", path, terrain)
		}
	}
	return communes, nil
}

func loadAssets(path string) ([]Asset, error) {
	rows, err := loadCSV(path, 2)
	if err != nil {
		return nil, err
	}
	assets := make([]Asset, 0, len(rows))
	seenCodes := make(map[string]bool, len(rows))
	seenNames := make(map[string]bool, len(rows))
	for i, row := range rows {
		line := i + 2 // header is line 1
		code, name := strings.TrimSpace(row[0]), strings.TrimSpace(row[1])
		if !isTrigram(code) {
			return nil, fmt.Errorf("assetgen: %s: line %d: invalid code %q (want exactly 3 uppercase letters)", path, line, code)
		}
		if name == "" {
			return nil, fmt.Errorf("assetgen: %s: line %d: empty name", path, line)
		}
		if seenCodes[code] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate code %q", path, line, code)
		}
		if seenNames[name] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate name %q", path, line, name)
		}
		seenCodes[code] = true
		seenNames[name] = true
		assets = append(assets, Asset{Code: code, Name: name})
	}
	return assets, nil
}

func loadCSV(path string, fieldsPerRecord int) ([][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("assetgen: %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.FieldsPerRecord = fieldsPerRecord

	if _, err := r.Read(); err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("assetgen: %s: empty file", path)
		}
		return nil, fmt.Errorf("assetgen: %s: %w", path, err)
	}

	var rows [][]string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("assetgen: %s: %w", path, err)
		}
		rows = append(rows, record)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("assetgen: %s: empty file", path)
	}
	return rows, nil
}

func isTrigram(s string) bool {
	if len(s) != 3 {
		return false
	}
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}
