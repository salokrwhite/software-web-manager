package common

import (
	"encoding/json"
	"strings"

	"gorm.io/datatypes"
)

// NormalizeRegionRules normalizes a raw region-rules JSON document into the
// canonical shape, returning nil for empty/null input.
func NormalizeRegionRules(raw json.RawMessage) datatypes.JSON {
	if len(raw) == 0 {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(string(raw)), "null") {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil
	}
	payload = normalizeRegionRulesMap(payload)
	b, _ := json.Marshal(payload)
	return datatypes.JSON(b)
}

// NormalizeRegionRulesValue normalizes an already-decoded region-rules value.
func NormalizeRegionRulesValue(value interface{}) datatypes.JSON {
	if value == nil {
		return nil
	}
	m, ok := value.(map[string]interface{})
	if ok {
		b, _ := json.Marshal(normalizeRegionRulesMap(m))
		return datatypes.JSON(b)
	}
	b, err := json.Marshal(value)
	if err != nil {
		return nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(b, &payload); err != nil {
		return nil
	}
	payload = normalizeRegionRulesMap(payload)
	out, _ := json.Marshal(payload)
	return datatypes.JSON(out)
}

func normalizeRegionRulesMap(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		return map[string]interface{}{
			"mode":  "allow_deny",
			"allow": normalizeRegionRuleGroup(nil),
			"deny":  normalizeRegionRuleGroup(nil),
		}
	}
	payload["mode"] = "allow_deny"
	allow := normalizeRegionRuleGroup(payload["allow"])
	deny := normalizeRegionRuleGroup(payload["deny"])
	payload["allow"] = allow
	payload["deny"] = deny
	if templates, ok := payload["templates"].([]interface{}); ok {
		normTemplates := make([]map[string]interface{}, 0, len(templates))
		for _, t := range templates {
			tm, ok := t.(map[string]interface{})
			if !ok {
				continue
			}
			id := strings.TrimSpace(ToString(tm["id"]))
			name := strings.TrimSpace(ToString(tm["name"]))
			normTemplates = append(normTemplates, map[string]interface{}{
				"id":    id,
				"name":  name,
				"allow": normalizeRegionRuleGroup(tm["allow"]),
				"deny":  normalizeRegionRuleGroup(tm["deny"]),
			})
		}
		payload["templates"] = normTemplates
		if _, ok := payload["active_template_id"]; !ok {
			payload["active_template_id"] = ""
		}
	}
	return payload
}

func normalizeRegionRuleGroup(value interface{}) map[string][]string {
	out := map[string][]string{
		"countries": {},
		"provinces": {},
		"cities":    {},
	}
	m, ok := value.(map[string]interface{})
	if !ok {
		return out
	}
	out["countries"] = normalizeStringList(m["countries"])
	out["provinces"] = normalizeStringList(m["provinces"])
	out["cities"] = normalizeStringList(m["cities"])
	return out
}

func normalizeStringList(value interface{}) []string {
	items := []string{}
	switch v := value.(type) {
	case []interface{}:
		for _, item := range v {
			s := strings.TrimSpace(ToString(item))
			if s != "" {
				items = append(items, s)
			}
		}
	case []string:
		for _, item := range v {
			s := strings.TrimSpace(item)
			if s != "" {
				items = append(items, s)
			}
		}
	}
	return items
}
