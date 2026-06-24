package geo

import "testing"

func TestProvinceNameByAliCityCode(t *testing.T) {
	cases := map[string]string{
		"440000":  "广东省",
		"310000":  "上海市",
		"110000":  "北京市",
		"650000":  "新疆维吾尔自治区",
		"810000":  "香港特别行政区", // 港澳台按中国的省存（目录写法）
		"820000":  "澳门特别行政区",
		"710000":  "台湾省",
		" 440000 ": "广东省", // trims spaces
		// unknown / overseas / empty → "":
		"XX":    "",
		"":      "",
		"US_CA": "",
		"999999": "",
	}
	for code, want := range cases {
		if got := ProvinceNameByAliCityCode(code); got != want {
			t.Errorf("ProvinceNameByAliCityCode(%q) = %q, want %q", code, got, want)
		}
	}
}
