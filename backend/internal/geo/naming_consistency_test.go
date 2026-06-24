package geo

import (
	"encoding/csv"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

//
// 地域规则（前端 picker → 规则）里存的字符串、以及 matchesRegionRules 比对的，
// 都来自 global_region.csv（单一事实源 SSOT）。本测试加载真实 csv，断言本包对外
// 可能产出的每一个名称都能在目录中命中——否则名称路径会"静默漏匹配"。
//
// 与 iso_names_test.go / ali_region_test.go 的区别：那两个是硬编码期望值的单测，
// 不读 csv，无法发现"映射名 ≠ csv 名"的漂移；本测试直接以 csv 为基准比对。

// loadCatalogNames 解析 global_region.csv，返回 level-1（国家）与 level-2（省/州）
// 名称集合——即规则构建与匹配所依据的写法。
func loadCatalogNames(t *testing.T) (countries, provinces map[string]struct{}) {
	t.Helper()
	path := locateRegionCSV(t)
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open region csv %q: %v", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.FieldsPerRecord = -1
	countries = map[string]struct{}{}
	provinces = map[string]struct{}{}
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil || len(rec) < 4 {
			continue
		}
		name := strings.TrimSpace(rec[2])
		if name == "" {
			continue
		}
		switch strings.TrimSpace(rec[3]) {
		case "1":
			countries[name] = struct{}{}
		case "2":
			provinces[name] = struct{}{}
		}
	}
	if len(countries) == 0 || len(provinces) == 0 {
		t.Fatalf("region csv parsed empty (countries=%d provinces=%d) at %s", len(countries), len(provinces), path)
	}
	return countries, provinces
}

// locateRegionCSV 稳定定位 global_region.csv：优先用测试源码目录相对定位
// （backend/internal/geo → backend/third_party/ip2region/data），再退到 env 与 cwd。
func locateRegionCSV(t *testing.T) string {
	t.Helper()
	if _, thisFile, _, ok := runtime.Caller(0); ok {
		cand := filepath.Join(filepath.Dir(thisFile), "..", "..", "third_party", "ip2region", "data", "global_region.csv")
		if _, err := os.Stat(cand); err == nil {
			return cand
		}
	}
	for _, c := range []string{
		strings.TrimSpace(os.Getenv("IP2REGION_REGION_CSV_PATH")),
		"./third_party/ip2region/data/global_region.csv",
		"../../third_party/ip2region/data/global_region.csv",
	} {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	t.Fatalf("global_region.csv not found (set IP2REGION_REGION_CSV_PATH to point at it)")
	return ""
}

// 每个 ali-ip-city 省级码映射出的省名必须是目录里的 level-2 省名。
func TestNamingConsistency_AliCityProvince(t *testing.T) {
	_, provinces := loadCatalogNames(t)
	for code, name := range aliCityProvince {
		if _, ok := provinces[name]; !ok {
			t.Errorf("aliCityProvince[%q]=%q 不在 global_region.csv 的 level-2 省名中 —— 名称漂移，省级名称路径将漏匹配", code, name)
		}
	}
}

// 每个国家名 override 必须是目录里的 level-1 国家名。
func TestNamingConsistency_ISOOverrides(t *testing.T) {
	countries, _ := loadCatalogNames(t)
	for code, name := range isoNameOverrides {
		if _, ok := countries[name]; !ok {
			t.Errorf("isoNameOverrides[%q]=%q 不在 global_region.csv 的 level-1 国家名中 —— 名称漂移，国家级名称路径将漏匹配", code, name)
		}
	}
}

// 常见会被地域规则限定的国家：CountryNameByISO 的输出必须非空且命中目录国家名。
func TestNamingConsistency_MainstreamCountries(t *testing.T) {
	countries, _ := loadCatalogNames(t)
	mainstream := []string{
		"CN", "US", "JP", "KR", "KP", "GB", "DE", "FR", "RU",
		"CA", "AU", "SG", "IN", "BR", "IT", "ES", "NL",
	}
	for _, code := range mainstream {
		name := CountryNameByISO(code)
		if name == "" {
			t.Errorf("CountryNameByISO(%q) 为空", code)
			continue
		}
		if _, ok := countries[name]; !ok {
			t.Errorf("CountryNameByISO(%q)=%q 不在 global_region.csv 的 level-1 国家名中 —— 需在 isoNameOverrides 对齐写法", code, name)
		}
	}
}
