// Package geo provides geo-region catalog loading that is independent of the
// HTTP layer (no gin, no response writing).
package geo

import (
	"bufio"
	"encoding/csv"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// RegionList is the flattened country/province/city catalog.
type RegionList struct {
	Countries []string `json:"countries"`
	Provinces []string `json:"provinces"`
	Cities    []string `json:"cities"`
}

var (
	geoRegionMu     sync.Mutex
	geoRegionLoaded bool
	geoRegionCache  RegionList
)

// LoadRegions returns the region catalog, building and caching it on first
// success. The cache is populated only on success so a transiently missing file
// is not cached permanently.
func LoadRegions() (RegionList, error) {
	geoRegionMu.Lock()
	defer geoRegionMu.Unlock()

	if geoRegionLoaded {
		return geoRegionCache, nil
	}

	list, err := buildGeoRegions()
	if err != nil {
		return RegionList{}, err
	}
	geoRegionCache = list
	geoRegionLoaded = true
	return geoRegionCache, nil
}

func buildGeoRegions() (RegionList, error) {
	var result RegionList

	path, err := resolveRegionCSVPath()
	if err != nil {
		return result, err
	}
	file, err := os.Open(path)
	if err != nil {
		return result, err
	}
	defer file.Close()

	reader := csv.NewReader(bufio.NewReader(file))
	reader.FieldsPerRecord = -1

	type node struct {
		ID       int
		ParentID int
		Name     string
		Level    int
	}
	nodes := make([]node, 0, 4096)
	for {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil || len(record) < 4 {
			continue
		}
		id := parseCSVInt(record[0])
		parentID := parseCSVInt(record[1])
		name := strings.TrimSpace(record[2])
		level := parseCSVInt(record[3])
		if id == 0 || name == "" || level == 0 {
			continue
		}
		nodes = append(nodes, node{
			ID:       id,
			ParentID: parentID,
			Name:     name,
			Level:    level,
		})
	}

	countryName := map[int]string{}
	provinceName := map[int]string{}
	provinceCountry := map[int]int{}

	for _, n := range nodes {
		if n.Level == 1 {
			countryName[n.ID] = n.Name
		}
	}
	for _, n := range nodes {
		if n.Level == 2 {
			provinceName[n.ID] = n.Name
			provinceCountry[n.ID] = n.ParentID
		}
	}

	countrySet := map[string]struct{}{}
	provinceSet := map[string]struct{}{}
	citySet := map[string]struct{}{}

	for _, n := range nodes {
		switch n.Level {
		case 1:
			if _, ok := countrySet[n.Name]; !ok {
				countrySet[n.Name] = struct{}{}
				result.Countries = append(result.Countries, n.Name)
			}
		case 2:
			country := countryName[n.ParentID]
			if country == "" {
				continue
			}
			value := country + "|" + n.Name
			if _, ok := provinceSet[value]; !ok {
				provinceSet[value] = struct{}{}
				result.Provinces = append(result.Provinces, value)
			}
		case 3:
			province := provinceName[n.ParentID]
			country := countryName[provinceCountry[n.ParentID]]
			if country == "" || province == "" {
				continue
			}
			value := country + "|" + province + "|" + n.Name
			if _, ok := citySet[value]; !ok {
				citySet[value] = struct{}{}
				result.Cities = append(result.Cities, value)
			}
		}
	}

	return result, nil
}

func resolveRegionCSVPath() (string, error) {
	candidate := []string{}
	if envPath := strings.TrimSpace(os.Getenv("IP2REGION_REGION_CSV_PATH")); envPath != "" {
		candidate = append(candidate, envPath)
	}
	candidate = append(candidate,
		filepath.Join(".", "backend", "third_party", "ip2region", "data", "global_region.csv"),
		filepath.Join(".", "third_party", "ip2region", "data", "global_region.csv"),
		filepath.Join("..", "backend", "third_party", "ip2region", "data", "global_region.csv"),
		filepath.Join("..", "third_party", "ip2region", "data", "global_region.csv"),
		filepath.Join("..", "..", "backend", "third_party", "ip2region", "data", "global_region.csv"),
		filepath.Join(".", "iplocation", "data", "global_region.csv"),
		filepath.Join("..", "iplocation", "data", "global_region.csv"),
		filepath.Join("..", "..", "iplocation", "data", "global_region.csv"),
	)
	for _, p := range candidate {
		if p == "" {
			continue
		}
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", errors.New("global_region.csv not found")
}

func parseCSVInt(raw string) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	val, err := strconv.Atoi(raw)
	if err != nil {
		return 0
	}
	return val
}
