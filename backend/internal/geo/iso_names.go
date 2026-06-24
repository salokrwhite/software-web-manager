package geo

import (
	"strings"

	"golang.org/x/text/language"
	"golang.org/x/text/language/display"
)

// zhRegions renders ISO 3166-1 region codes to Simplified-Chinese names (CLDR).
var zhRegions = display.Regions(language.SimplifiedChinese)

// isoNameOverrides aligns the handful of ISO codes whose CLDR Simplified-Chinese
// name differs from this project's region catalog (global_region.csv) writing.
// CLDR matches the catalog for all mainstream countries; these are the verified
// exceptions — each target name was confirmed present in the catalog. Override is
// checked before CLDR so these win.
var isoNameOverrides = map[string]string{
	"AD": "安道尔共和国",       // CLDR: 安道尔
	"BA": "波黑",            // CLDR: 波斯尼亚和黑塞哥维那
	"BL": "圣巴泰勒米岛",       // CLDR: 圣巴泰勒米
	"BQ": "博内尔岛",         // CLDR: 荷属加勒比区
	"CD": "刚果民主共和国",      // CLDR: 刚果（金）
	"CG": "刚果共和国",        // CLDR: 刚果（布）
	"IM": "英属马恩岛",        // CLDR: 马恩岛
	"MS": "蒙特塞拉特岛",       // CLDR: 蒙特塞拉特
	"PS": "巴勒斯坦",         // CLDR: 巴勒斯坦领土
	"RE": "留尼汪岛",         // CLDR: 留尼汪
	"SX": "圣马丁岛",         // CLDR: 荷属圣马丁
	"VU": "瓦努阿图共和国",      // CLDR: 瓦努阿图
	"WF": "瓦利斯和富图纳群岛",   // CLDR: 瓦利斯和富图纳
}

// CountryNameByISO maps an ISO 3166-1 Alpha-2 country code to its Simplified
// Chinese name (e.g. "CN" -> "中国", "US" -> "美国"). It returns "" for empty,
// non-2-letter, unknown, or non-ISO inputs (e.g. ESA's "XX"/"IANA"), so callers
// can fall back gracefully without panicking.
//
// NOTE: the returned name must match how countries are stored in the geo catalog
// (global_region.csv) for name-path rule matching to hit. A consistency test
// guards this (see plan §4.10 / naming_consistency_test.go). In the common path
// the merge prefers ip2region's own (catalog-aligned) name, so this mapping is
// only a fallback for the rare ESA-vs-ip2region disagreement / no-IP case.
func CountryNameByISO(iso string) string {
	code := strings.ToUpper(strings.TrimSpace(iso))
	if len(code) != 2 {
		return ""
	}
	if name, ok := isoNameOverrides[code]; ok {
		return name
	}
	region, err := language.ParseRegion(code)
	if err != nil {
		return ""
	}
	name := strings.TrimSpace(zhRegions.Name(region))
	// Guard against x/text echoing the code back (or naming it an "unknown
	// region") when there is no real localized name.
	if name == "" || strings.EqualFold(name, code) {
		return ""
	}
	return name
}
