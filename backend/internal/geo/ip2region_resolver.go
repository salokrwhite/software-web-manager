package geo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"software-web-manager/backend/internal/config"

	"github.com/lionsoul2014/ip2region/binding/golang/service"
)

type ip2RegionResolver struct {
	svc *service.Ip2Region
}

func NewIP2RegionResolver(cfg config.Config) (Resolver, error) {
	v4Path := resolveXDBPath(cfg.IP2RegionV4Path, "ip2region_v4.xdb")
	v6Path := resolveXDBPath(cfg.IP2RegionV6Path, "ip2region_v6.xdb")
	if v4Path == "" && v6Path == "" {
		return nil, nil
	}
	policy, err := service.CachePolicyFromName(cfg.IP2RegionCachePolicy)
	if err != nil {
		return nil, err
	}
	var v4Config *service.Config
	if v4Path != "" {
		v4Config, err = service.NewV4Config(policy, v4Path, cfg.IP2RegionPoolSize)
		if err != nil {
			return nil, fmt.Errorf("ip2region v4 config: %w", err)
		}
	}
	var v6Config *service.Config
	if v6Path != "" {
		v6Config, err = service.NewV6Config(policy, v6Path, cfg.IP2RegionPoolSize)
		if err != nil {
			return nil, fmt.Errorf("ip2region v6 config: %w", err)
		}
	}
	svc, err := service.NewIp2Region(v4Config, v6Config)
	if err != nil {
		return nil, err
	}
	return &ip2RegionResolver{svc: svc}, nil
}

func (r *ip2RegionResolver) Resolve(ip string) (Region, error) {
	if r == nil || r.svc == nil {
		return Region{}, errors.New("ip2region not initialized")
	}
	raw, err := r.svc.SearchByStr(ip)
	if err != nil {
		return Region{}, err
	}
	return parseIP2Region(raw), nil
}

func (r *ip2RegionResolver) Close() error {
	if r == nil || r.svc == nil {
		return nil
	}
	r.svc.Close()
	return nil
}

func resolveXDBPath(inputPath string, filename string) string {
	trimmed := strings.TrimSpace(inputPath)
	if trimmed != "" {
		if fileExists(trimmed) {
			return trimmed
		}
	}
	candidates := []string{
		filepath.Join(".", "backend", "third_party", "ip2region", "data", filename),
		filepath.Join(".", "third_party", "ip2region", "data", filename),
		filepath.Join("..", "backend", "third_party", "ip2region", "data", filename),
		filepath.Join("..", "third_party", "ip2region", "data", filename),
		filepath.Join("..", "..", "backend", "third_party", "ip2region", "data", filename),
		filepath.Join(".", "iplocation", "data", filename),
		filepath.Join("..", "iplocation", "data", filename),
		filepath.Join("..", "..", "iplocation", "data", filename),
	}
	for _, p := range candidates {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func parseIP2Region(raw string) Region {
	parts := strings.Split(raw, "|")
	region := Region{}
	if len(parts) > 0 {
		region.Country = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		region.Province = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		region.City = strings.TrimSpace(parts[2])
	}
	if len(parts) > 4 {
		region.ISO = strings.TrimSpace(parts[4])
	}
	return region
}
