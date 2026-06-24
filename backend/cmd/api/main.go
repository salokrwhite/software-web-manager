package main

import (
	"context"
	"log"
	"strings"
	"time"

	"software-web-manager/backend/internal/api"
	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/core"
	"software-web-manager/backend/internal/db"
	"software-web-manager/backend/internal/geo"
	"software-web-manager/backend/internal/jobs"
	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/clientupdate"
	"software-web-manager/backend/internal/services/online"
	"software-web-manager/backend/internal/storage"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	if err := cfg.Validate(); err != nil {
		log.Fatalf("配置校验失败，拒绝启动: %v", err)
	}
	authzSigner, err := auth.NewAuthzSigner(cfg.AuthzSigningKey, cfg.AuthzKeyID)
	if err != nil {
		log.Fatalf("授权签名密钥加载失败: %v", err)
	}
	log.Printf("授权签名已就绪 (key_id=%s, pub=%s)", authzSigner.KeyID(), authzSigner.PublicKeyHex())
	dbConn, err := db.Open(cfg.DatabaseURL)
	installMode := false
	if err != nil {
		log.Printf("数据库连接失败，进入安装模式: %v", err)
		installMode = true
		dbConn = nil
	} else {
		if err := db.SetConnPool(dbConn); err != nil {
			log.Printf("数据库池设置失败，进入安装模式: %v", err)
			installMode = true
			dbConn = nil
		}
	}

	var store storage.Driver
	var resolver geo.Resolver
	var replayStore *redis.Client
	if !installMode && dbConn != nil {
		if cfg.RunMigrations {
			if err := db.Migrate(dbConn, "./migrations"); err != nil {
				log.Printf("数据库迁移失败: %v", err)
			}
		}
		if err := dbConn.AutoMigrate(&models.User{}, &models.SystemSetting{}); err != nil {
			log.Printf("基础表自动迁移失败: %v", err)
		}

		var err error
		store, err = storage.New(context.Background(), cfg)
		if err != nil {
			log.Fatalf("storage: %v", err)
		}
		resolver, err = geo.NewIP2RegionResolver(cfg)
		if err != nil {
			log.Printf("ip2region 初始化失败: %v", err)
		}
		if cfg.EnableEmbeddedWorker {
			jobs.Start(context.Background(), dbConn, time.Duration(cfg.WorkerIntervalSeconds)*time.Second, log.Default())
			log.Printf("内置聚合任务已启动，间隔 %d 秒", cfg.WorkerIntervalSeconds)
		} else {
			log.Printf("内置聚合任务已禁用")
		}

		opt, err := redis.ParseURL(cfg.RedisURL)
		if err != nil {
			log.Printf("Redis URL 解析失败: %v", err)
		} else {
			replayStore = redis.NewClient(opt)
			pingCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			if pingErr := replayStore.Ping(pingCtx).Err(); pingErr != nil {
				log.Printf("Redis 连接失败，受保护接口签名重放校验将返回 503: %v", pingErr)
			}
		}
	}

	r := gin.New()
	// 接入阿里云 ESA 后（确认回源流量必经 ESA，ESA_GEO_HEADERS_TRUSTED=true）才设置：
	// 把 gin 的「可信平台头」指向 ESA 真实 IP 头，使 c.ClientIP() 全站返回不可伪造的真实
	// 客户端 IP —— 限流、IP 白名单、审计、地域解析一致受益。默认关闭（开关 false）时不设置，
	// 行为与改造前完全一致；该头缺失时 gin 会自动回退到原有 RemoteIP/X-Forwarded-For 逻辑
	// （不会变空），故不会误伤。
	if cfg.TrustESAGeoHeaders && cfg.ESARealIPHeader != "" {
		r.TrustedPlatform = cfg.ESARealIPHeader
	}
	r.Use(gin.Recovery())
	origins := corsOrigins(cfg.CORSOrigins)
	allowCredentials := true
	if len(origins) == 1 && origins[0] == "*" {
		allowCredentials = false
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins: origins,
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{
			"Authorization", "Content-Type", "Accept", "Origin", "X-Requested-With",
			"X-App-Id", "X-Timestamp", "X-Nonce", "X-Signature", "X-Sign-Version",
			"x-app-id", "x-timestamp", "x-nonce", "x-signature", "x-sign-version",
		},
		AllowCredentials: allowCredentials,
	}))

	onlineTracker := online.NewTracker(time.Duration(cfg.OnlineWindowSeconds) * time.Second)
	h := core.Handler{
		DB:              dbConn,
		Cfg:             cfg,
		Storage:         store,
		ReplayStore:     replayStore,
		RegionResolver:  resolver,
		OnlineTracker:   onlineTracker,
		ClientUpdateHub: clientupdate.NewHub(),
		AuthzSigner:     authzSigner,
	}
	if !installMode && dbConn != nil {
		clientupdate.NewService(h.DB, h.ClientUpdateHub).StartReleaseActivationWatcher(context.Background(), 20*time.Second, log.Default())
		log.Printf("发布计划激活扫描已启动，间隔 %d 秒", 20)
	}
	api.RegisterRoutes(r, &h, installMode)

	if installMode {
		log.Println("=====================================")
		log.Println("系统未安装，请访问前端页面进行安装")
		log.Println("=====================================")
	}

	log.Printf("服务将运行在 http://localhost%s", cfg.HTTPAddr)
	if err := r.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

func corsOrigins(origins []string) []string {
	if len(origins) == 0 {
		return []string{"*"}
	}
	if len(origins) == 1 && strings.TrimSpace(origins[0]) == "*" {
		return []string{"*"}
	}
	return origins
}
