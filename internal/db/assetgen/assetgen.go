// Package assetgen loads the static world assets (commune names, noble
// first names and territory qualifiers) from CSV files and validates the
// invariants they must satisfy at startup.
//
// The CSV files live in the assets directory and use a semicolon separator
// with a header line. Codes are strict uppercase trigrams for communes and
// first names, and a single uppercase letter for qualifier prefixes. Codes
// and names must be unique within each file (no cross-file uniqueness: the
// entity type disambiguates).
package assetgen

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Asset is a named entity (commune or first name) with its trigram code.
type Asset struct {
	Code string
	Name string
}

// Qualificatif is a territory qualifier: a single-letter prefix, a French
// display name and the terrain type it applies to.
type Qualificatif struct {
	Prefix  string
	Name    string
	Terrain string
}

// Assets holds all the world data loaded from the assets directory.
type Assets struct {
	Communes, Prenoms []Asset
	Qualificatifs     []Qualificatif
}

var validTerrains = map[string]bool{
	"plain":    true,
	"forest":   true,
	"hill":     true,
	"mountain": true,
	"swamp":    true,
	"any":      true,
}

// Load reads communes.csv, prenoms.csv and qualificatifs.csv from dir and
// validates them, failing fast at the first invalid record. A missing or
// empty file is an explicit error.
func Load(dir string) (Assets, error) {
	var a Assets
	var err error

	if a.Communes, err = loadAssets(filepath.Join(dir, "communes.csv")); err != nil {
		return Assets{}, err
	}
	if a.Prenoms, err = loadAssets(filepath.Join(dir, "prenoms.csv")); err != nil {
		return Assets{}, err
	}
	if a.Qualificatifs, err = loadQualificatifs(filepath.Join(dir, "qualificatifs.csv")); err != nil {
		return Assets{}, err
	}
	return a, nil
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

func loadQualificatifs(path string) ([]Qualificatif, error) {
	rows, err := loadCSV(path, 3)
	if err != nil {
		return nil, err
	}
	quals := make([]Qualificatif, 0, len(rows))
	seenPrefixes := make(map[string]bool, len(rows))
	seenNames := make(map[string]bool, len(rows))
	for i, row := range rows {
		line := i + 2 // header is line 1
		prefix, name, terrain := strings.TrimSpace(row[0]), strings.TrimSpace(row[1]), strings.TrimSpace(row[2])
		if !isPrefix(prefix) {
			return nil, fmt.Errorf("assetgen: %s: line %d: invalid prefix %q (want a single uppercase letter)", path, line, prefix)
		}
		if name == "" {
			return nil, fmt.Errorf("assetgen: %s: line %d: empty qualificatif name", path, line)
		}
		if !validTerrains[terrain] {
			return nil, fmt.Errorf("assetgen: %s: line %d: invalid terrain %q (want one of plain|forest|hill|mountain|swamp|any)", path, line, terrain)
		}
		if seenPrefixes[prefix] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate prefix %q", path, line, prefix)
		}
		if seenNames[name] {
			return nil, fmt.Errorf("assetgen: %s: line %d: duplicate qualificatif %q", path, line, name)
		}
		seenPrefixes[prefix] = true
		seenNames[name] = true
		quals = append(quals, Qualificatif{Prefix: prefix, Name: name, Terrain: terrain})
	}
	return quals, nil
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

func isPrefix(s string) bool {
	return len(s) == 1 && s[0] >= 'A' && s[0] <= 'Z'
}
