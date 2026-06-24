package client

import (
	"encoding/json"
	"strings"

	"software-web-manager/backend/internal/geo"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type regionRules struct {
	Mode             string               `json:"mode"`
	ActiveTemplateID string               `json:"active_template_id"`
	Templates        []regionRuleTemplate `json:"templates"`
	Allow            regionRuleGroup      `json:"allow"`
	Deny             regionRuleGroup      `json:"deny"`
}

type regionRuleGroup struct {
	Countries []string `json:"countries"`
	Provinces []string `json:"provinces"`
	Cities    []string `json:"cities"`
}

type regionRuleTemplate struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Allow regionRuleGroup `json:"allow"`
	Deny  regionRuleGroup `json:"deny"`
}

type resolvedRegion struct {
	ISO      string
	Country  string
	Province string
	City     string
}

// esaGeo holds the trusted geo signals extracted from ESA managed-transform
// headers. The zero value means "no ESA signal" (headers absent or not trusted).
type esaGeo struct {
	RealIP     string // ali-real-client-ip
	CountryISO string // ali-ip-country, upper-cased ISO 3166-1 Alpha-2 (e.g. "CN")
	CityCode   string // ali-ip-city: 省/州级地区码(非城市级)。Phase 2 才用，见方案 §4.9。
}

// readESAGeo extracts the ESA geo headers, but only when they are trusted
// (config.TrustESAGeoHeaders). Otherwise it returns the zero value, so on
// untrusted ingress we never honor client-forgeable headers.
func (h *Handler) readESAGeo(c *gin.Context) esaGeo {
	if h == nil || !h.Cfg.TrustESAGeoHeaders {
		return esaGeo{}
	}
	return esaGeo{
		RealIP:     strings.TrimSpace(c.GetHeader(h.Cfg.ESARealIPHeader)),
		CountryISO: strings.ToUpper(strings.TrimSpace(c.GetHeader(h.Cfg.ESACountryHeader))),
		CityCode:   strings.TrimSpace(c.GetHeader(h.Cfg.ESACityHeader)),
	}
}

// realIPOr returns the ESA-provided real client IP when present, else fallback.
func (e esaGeo) realIPOr(fallback string) string {
	if e.RealIP != "" {
		return e.RealIP
	}
	return fallback
}

// ResolveRegion derives the effective region used for rule matching.
//
// Source precedence is governed by config.PreferServerSideRegion:
//   - true (default, anti-forgery): server-side wins per field —
//     ESA > ip2region(ip) > client-reported attrs. Client self-report is only a
//     last-resort fallback, so a client cannot bypass geo rules by lying.
//   - false (legacy rollback): client-reported attrs short-circuit first, then
//     ip2region — byte-identical to the pre-ESA behavior.
//
// `esa` is the already trusted-extracted ESA signal; `ip` is the IP fed to
// ip2region (callers pass the real IP, i.e. esa.realIPOr(c.ClientIP())).
// matchesRegionRules is unchanged; only how `region` is produced changes.
func (h *Handler) ResolveRegion(esa esaGeo, attrs map[string]string, ip string) resolvedRegion {
	if h == nil || !h.Cfg.PreferServerSideRegion {
		return h.resolveRegionLegacy(attrs, ip)
	}

	var ipReg geo.Region
	if h.RegionResolver != nil && strings.TrimSpace(ip) != "" {
		if r, err := h.RegionResolver.Resolve(ip); err == nil {
			ipReg = r
		}
	}
	ipISO := strings.ToUpper(strings.TrimSpace(ipReg.ISO))
	ipCountry := strings.TrimSpace(ipReg.Country)
	ipProvince := strings.TrimSpace(ipReg.Province)
	ipCity := strings.TrimSpace(ipReg.City)

	attrISO := strings.ToUpper(strings.TrimSpace(attrs["country_iso"]))
	attrCountry := strings.TrimSpace(attrs["country"])
	attrProvince := strings.TrimSpace(attrs["province"])
	attrCity := strings.TrimSpace(attrs["city"])

	out := resolvedRegion{}

	// ISO: ESA > ip2region > attrs
	switch {
	case esa.CountryISO != "":
		out.ISO = esa.CountryISO
	case ipISO != "":
		out.ISO = ipISO
	default:
		out.ISO = attrISO
	}

	// 国家名（必须与 global_region.csv 写法一致才能命中名称路径，见方案 §4.10）。
	switch {
	case esa.CountryISO != "":
		if esa.CountryISO == ipISO && ipCountry != "" {
			out.Country = ipCountry // 同国家 → 复用 ip2region 的（目录对齐的）名称
		} else {
			out.Country = geo.CountryNameByISO(esa.CountryISO) // 可能为 "" → 靠 ISO 路径命中
		}
	case ipCountry != "":
		out.Country = ipCountry
	default:
		out.Country = attrCountry
	}

	// 省/州：ESA 中国省级码（Phase 2） > ip2region > attrs。
	// ESA ali-ip-city 是省/州级码；其中国省级码（GB/T 2260）映射成目录省名后覆盖。
	// 仅当国家判定为中国(CN)时启用——映射表只含中国省份，避免把中国省名挂到他国。
	esaProvince := ""
	if out.ISO == "CN" {
		esaProvince = geo.ProvinceNameByAliCityCode(esa.CityCode)
	}
	switch {
	case esaProvince != "":
		out.Province = esaProvince
	case ipProvince != "":
		out.Province = ipProvince
	default:
		out.Province = attrProvince
	}

	// 地级市：ip2region > attrs。ESA 永不提供城市。
	if ipCity != "" {
		out.City = ipCity
	} else {
		out.City = attrCity
	}

	return out
}

// resolveRegionLegacy reproduces the pre-ESA precedence (client attrs first,
// then ip2region) verbatim. Used when PreferServerSideRegion is disabled, so
// operators can roll back to byte-identical legacy behavior.
func (h *Handler) resolveRegionLegacy(attrs map[string]string, ip string) resolvedRegion {
	iso := strings.ToUpper(strings.TrimSpace(attrs["country_iso"]))
	country := strings.TrimSpace(attrs["country"])
	province := strings.TrimSpace(attrs["province"])
	city := strings.TrimSpace(attrs["city"])
	if iso != "" || country != "" {
		return resolvedRegion{ISO: iso, Country: country, Province: province, City: city}
	}
	if h != nil && h.RegionResolver != nil && strings.TrimSpace(ip) != "" {
		if region, err := h.RegionResolver.Resolve(ip); err == nil {
			return resolvedRegion{
				ISO:      strings.ToUpper(strings.TrimSpace(region.ISO)),
				Country:  strings.TrimSpace(region.Country),
				Province: strings.TrimSpace(region.Province),
				City:     strings.TrimSpace(region.City),
			}
		}
	}
	return resolvedRegion{}
}

// regionRulesHasContent 判断一条区域规则是否真的配置了限制。
// 通道级规则若为空（无模板、allow/deny 全空），应被视为「继承应用级规则」，
// 而不是当成通道自定义后放行所有地区。也用于兼容历史上已存的空规则脏数据。
func regionRulesHasContent(raw datatypes.JSON) bool {
	if len(raw) == 0 {
		return false
	}
	var rules regionRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return false
	}
	groupHasContent := func(g regionRuleGroup) bool {
		return len(g.Countries) > 0 || len(g.Provinces) > 0 || len(g.Cities) > 0
	}
	for _, tpl := range rules.Templates {
		if groupHasContent(tpl.Allow) || groupHasContent(tpl.Deny) {
			return true
		}
	}
	return groupHasContent(rules.Allow) || groupHasContent(rules.Deny)
}

func matchesRegionRules(raw datatypes.JSON, region resolvedRegion) bool {
	if len(raw) == 0 {
		return true
	}
	var rules regionRules
	if err := json.Unmarshal(raw, &rules); err != nil {
		return false
	}
	if len(rules.Templates) > 0 {
		chosen := rules.Templates[0]
		if rules.ActiveTemplateID != "" {
			for _, tpl := range rules.Templates {
				if tpl.ID == rules.ActiveTemplateID {
					chosen = tpl
					break
				}
			}
		}
		rules.Allow = chosen.Allow
		rules.Deny = chosen.Deny
	}
	iso := strings.ToUpper(strings.TrimSpace(region.ISO))
	countryName := strings.TrimSpace(region.Country)
	province := strings.TrimSpace(region.Province)
	city := strings.TrimSpace(region.City)
	countryKey := iso
	countryNameKey := countryName
	provinceKey := ""
	provinceNameKey := ""
	cityKey := ""
	cityNameKey := ""
	if iso != "" && province != "" {
		provinceKey = iso + "|" + province
	}
	if iso != "" && province != "" && city != "" {
		cityKey = iso + "|" + province + "|" + city
	}
	if countryName != "" && province != "" {
		provinceNameKey = countryName + "|" + province
	}
	if countryName != "" && province != "" && city != "" {
		cityNameKey = countryName + "|" + province + "|" + city
	}
	if matchRegionGroup(rules.Deny, countryKey, provinceKey, cityKey) || matchRegionGroup(rules.Deny, countryNameKey, provinceNameKey, cityNameKey) {
		return false
	}
	allowEmpty := len(rules.Allow.Countries) == 0 && len(rules.Allow.Provinces) == 0 && len(rules.Allow.Cities) == 0
	if allowEmpty {
		return true
	}
	if iso == "" && countryName == "" {
		return false
	}
	return matchRegionGroup(rules.Allow, countryKey, provinceKey, cityKey) || matchRegionGroup(rules.Allow, countryNameKey, provinceNameKey, cityNameKey)
}

func matchRegionGroup(group regionRuleGroup, countryKey, provinceKey, cityKey string) bool {
	if countryKey != "" && matchList(countryKey, group.Countries) {
		return true
	}
	if provinceKey != "" && matchList(provinceKey, group.Provinces) {
		return true
	}
	if cityKey != "" && matchList(cityKey, group.Cities) {
		return true
	}
	return false
}
