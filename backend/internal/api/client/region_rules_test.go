package client

import (
	"errors"
	"net/http/httptest"
	"testing"

	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/geo"

	"github.com/gin-gonic/gin"
)

// fakeResolver is a stub geo.Resolver for deterministic ResolveRegion tests.
type fakeResolver struct {
	region geo.Region
	err    error
}

func (f fakeResolver) Resolve(string) (geo.Region, error) { return f.region, f.err }
func (f fakeResolver) Close() error                       { return nil }

func baseCfg() config.Config {
	return config.Config{
		PreferServerSideRegion: true,
		TrustESAGeoHeaders:     true,
		ESARealIPHeader:        "ali-real-client-ip",
		ESACountryHeader:       "ali-ip-country",
		ESACityHeader:          "ali-ip-city",
	}
}

func newHandler(cfg config.Config, res geo.Resolver) *Handler {
	return &Handler{Handler: &core.Handler{Cfg: cfg, RegionResolver: res}}
}

func testCtx(headers map[string]string) *gin.Context {
	w := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(w)
	ctx.Request = httptest.NewRequest("POST", "/", nil)
	for k, v := range headers {
		ctx.Request.Header.Set(k, v)
	}
	return ctx
}

// Legacy mode: client self-report short-circuits and wins (byte-identical to pre-ESA).
func TestResolveRegion_Legacy_ClientWins(t *testing.T) {
	cfg := baseCfg()
	cfg.PreferServerSideRegion = false
	h := newHandler(cfg, fakeResolver{region: geo.Region{ISO: "US", Country: "美国"}})

	got := h.ResolveRegion(esaGeo{}, map[string]string{"country_iso": "CN", "country": "中国"}, "1.2.3.4")
	if got.ISO != "CN" || got.Country != "中国" {
		t.Fatalf("legacy must keep client-reported CN, got ISO=%q Country=%q", got.ISO, got.Country)
	}
}

// Aggressive + ESA off: ip2region overrides a lying client (anti-forgery without ESA).
func TestResolveRegion_Aggressive_IP2RegionOverridesClient(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{region: geo.Region{ISO: "CN", Country: "中国", Province: "广东省", City: "深圳市"}})

	got := h.ResolveRegion(esaGeo{}, map[string]string{"country_iso": "US", "country": "美国"}, "1.2.3.4")
	if got.ISO != "CN" || got.Country != "中国" {
		t.Fatalf("ip2region must override lying client, got ISO=%q Country=%q", got.ISO, got.Country)
	}
	if got.Province != "广东省" || got.City != "深圳市" {
		t.Fatalf("province/city must come from ip2region, got Province=%q City=%q", got.Province, got.City)
	}
}

// Aggressive + ESA on: ESA country wins over ip2region.
func TestResolveRegion_Aggressive_ESAOverridesIP2Region(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{region: geo.Region{ISO: "US", Country: "美国"}})

	got := h.ResolveRegion(esaGeo{CountryISO: "CN"}, map[string]string{}, "1.2.3.4")
	if got.ISO != "CN" {
		t.Fatalf("ESA country must win, got ISO=%q", got.ISO)
	}
	// names differ from ip2region → CountryNameByISO fallback; assert it is non-empty & not the US name.
	if got.Country == "美国" || got.Country == "" {
		t.Fatalf("expected ESA-mapped CN name, got %q", got.Country)
	}
}

// Aggressive: when neither ESA nor ip2region resolves, client self-report is the last-resort fallback.
func TestResolveRegion_Aggressive_AttrsFallback(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{err: errors.New("unresolved")})

	got := h.ResolveRegion(esaGeo{}, map[string]string{"country_iso": "US", "country": "美国"}, "1.2.3.4")
	if got.ISO != "US" || got.Country != "美国" {
		t.Fatalf("should fall back to client attrs, got ISO=%q Country=%q", got.ISO, got.Country)
	}
}

// ESA never supplies a city; ali-ip-city must not leak into City (Phase 1).
func TestResolveRegion_CityNeverFromESA(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{region: geo.Region{ISO: "CN", Country: "中国"}})

	got := h.ResolveRegion(esaGeo{CountryISO: "CN", CityCode: "440000"}, map[string]string{}, "1.2.3.4")
	if got.City != "" {
		t.Fatalf("City must stay empty (ESA city code must not fill City), got %q", got.City)
	}
}

// Phase 2: ESA ali-ip-city (China province code) overrides ip2region's province.
func TestResolveRegion_Phase2_ESAProvinceChina(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{region: geo.Region{ISO: "CN", Country: "中国", Province: "浙江省", City: "杭州市"}})

	got := h.ResolveRegion(esaGeo{CountryISO: "CN", CityCode: "440000"}, map[string]string{}, "1.2.3.4")
	if got.Province != "广东省" {
		t.Fatalf("ESA province code must override ip2region province, got %q", got.Province)
	}
	if got.City != "杭州市" {
		t.Fatalf("city must still come from ip2region, got %q", got.City)
	}
}

// Phase 2 guard: the China province map must NOT apply when country != CN.
func TestResolveRegion_Phase2_SkippedForNonChina(t *testing.T) {
	h := newHandler(baseCfg(), fakeResolver{region: geo.Region{ISO: "US", Province: "加利福尼亚州"}})

	got := h.ResolveRegion(esaGeo{CountryISO: "US", CityCode: "440000"}, map[string]string{}, "1.2.3.4")
	if got.Province == "广东省" {
		t.Fatalf("China province map must not apply for non-CN country, got %q", got.Province)
	}
	if got.Province != "加利福尼亚州" {
		t.Fatalf("non-CN province should come from ip2region, got %q", got.Province)
	}
}

// readESAGeo must ignore headers unless TrustESAGeoHeaders is set.
func TestReadESAGeo_Gating(t *testing.T) {
	headers := map[string]string{"ali-ip-country": "cn", "ali-real-client-ip": "9.9.9.9", "ali-ip-city": "440000"}

	cfg := baseCfg()
	cfg.TrustESAGeoHeaders = false
	if e := newHandler(cfg, nil).readESAGeo(testCtx(headers)); e != (esaGeo{}) {
		t.Fatalf("untrusted ingress must yield empty esaGeo, got %+v", e)
	}

	cfg.TrustESAGeoHeaders = true
	e := newHandler(cfg, nil).readESAGeo(testCtx(headers))
	if e.CountryISO != "CN" || e.RealIP != "9.9.9.9" || e.CityCode != "440000" {
		t.Fatalf("trusted read mismatch: %+v", e)
	}
}
