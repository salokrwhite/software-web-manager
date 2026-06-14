package handlers

import (
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

type regionRules struct {
	Mode             string             `json:"mode"`
	ActiveTemplateID string             `json:"active_template_id"`
	Templates        []regionRuleTemplate `json:"templates"`
	Allow            regionRuleGroup    `json:"allow"`
	Deny             regionRuleGroup    `json:"deny"`
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

func resolveRegion(h *Handler, attrs map[string]string, ip string) resolvedRegion {
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
