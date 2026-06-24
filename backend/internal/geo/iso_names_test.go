package geo

import "testing"

// Mainstream countries: CLDR (x/text) names verified to match the project's
// region catalog (global_region.csv level-1 names). These are the countries that
// realistically carry geo publish-rules, so name-path matching must hold.
func TestCountryNameByISO_Mainstream(t *testing.T) {
	cases := map[string]string{
		"CN": "中国",
		"US": "美国",
		"JP": "日本",
		"KR": "韩国",
		"KP": "朝鲜",
		"GB": "英国",
		"DE": "德国",
		"FR": "法国",
		"RU": "俄罗斯",
		"cn": "中国", // case-insensitive input
		" US ": "美国", // trims surrounding spaces
		// override-aligned (CLDR differs from catalog; override wins):
		"BA": "波黑",
		"CD": "刚果民主共和国",
		"CG": "刚果共和国",
		"AD": "安道尔共和国",
		"RE": "留尼汪岛",
		"PS": "巴勒斯坦",
	}
	for code, want := range cases {
		if got := CountryNameByISO(code); got != want {
			t.Errorf("CountryNameByISO(%q) = %q, want %q", code, got, want)
		}
	}
}

// Empty/invalid/unknown inputs must yield "" so the caller falls back to the ISO
// path (or attrs) instead of injecting a bogus name. Note ESA's country header is
// an ISO Alpha-2 code; its "XX" unknown-marker maps to "".
func TestCountryNameByISO_EmptyForInvalid(t *testing.T) {
	for _, code := range []string{"", " ", "X", "XXX", "1A", "@@", "XX"} {
		if got := CountryNameByISO(code); got != "" {
			t.Errorf("CountryNameByISO(%q) = %q, want empty", code, got)
		}
	}
}
