package geo

import "strings"

// aliCityProvince maps ESA `ali-ip-city` codes to the province name exactly as
// written in this project's region catalog (global_region.csv, level-2 under
// 中国). ESA's `ali-ip-city` for mainland China is a GB/T 2260 province-level
// code (e.g. 440000=广东省); "XX" = unknown.
//
// Important alignment notes (verified against global_region.csv):
//   - Mainland province names match the ESA region table verbatim (广东省, 上海市…).
//   - 港澳台 use the CATALOG writing — 香港特别行政区 / 澳门特别行政区 / 台湾省 —
//     NOT the ESA table's "中国香港/中国澳门/中国台湾". Per project decision 港澳台
//     are stored as provinces of China (country = CN), so name-path keys are
//     中国|香港特别行政区 etc.
//   - Overseas `ali-ip-city` codes (e.g. "US_CA") are intentionally absent;
//     overseas province/state resolution stays with ip2region.
var aliCityProvince = map[string]string{
	"110000": "北京市",
	"120000": "天津市",
	"130000": "河北省",
	"140000": "山西省",
	"150000": "内蒙古自治区",
	"210000": "辽宁省",
	"220000": "吉林省",
	"230000": "黑龙江省",
	"310000": "上海市",
	"320000": "江苏省",
	"330000": "浙江省",
	"340000": "安徽省",
	"350000": "福建省",
	"360000": "江西省",
	"370000": "山东省",
	"410000": "河南省",
	"420000": "湖北省",
	"430000": "湖南省",
	"440000": "广东省",
	"450000": "广西壮族自治区",
	"460000": "海南省",
	"500000": "重庆市",
	"510000": "四川省",
	"520000": "贵州省",
	"530000": "云南省",
	"540000": "西藏自治区",
	"610000": "陕西省",
	"620000": "甘肃省",
	"630000": "青海省",
	"640000": "宁夏回族自治区",
	"650000": "新疆维吾尔自治区",
	"710000": "台湾省",
	"810000": "香港特别行政区",
	"820000": "澳门特别行政区",
}

// ProvinceNameByAliCityCode returns the catalog province name for an ESA
// `ali-ip-city` code, or "" for empty, unknown ("XX"), or overseas codes.
func ProvinceNameByAliCityCode(code string) string {
	return aliCityProvince[strings.TrimSpace(code)]
}
