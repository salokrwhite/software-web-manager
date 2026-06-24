// Command backfillauthzkeys seeds every existing app with a first active
// authorization signing key, reusing the platform key (AUTHZ_SIGNING_PRIVATE_KEY
// / AUTHZ_KEY_ID) as that key. This is the zero-downtime migration step: clients
// already in the field embed only the platform public key, so signing with the
// same key_id keeps them verifying. Apps are later rotated to independent keys
// via the console (pending -> publish new client -> activate).
//
// Run ONCE after migration 0006 is applied. It is idempotent: apps that already
// have an active, non-revoked key are skipped.
//
//	go run ./cmd/backfillauthzkeys            # backfill
//	go run ./cmd/backfillauthzkeys -dry-run   # list apps that would be backfilled
package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/db"
	"software-web-manager/backend/internal/db/schema"
	"software-web-manager/backend/internal/models"
	appsvc "software-web-manager/backend/internal/services/app"

	"github.com/google/uuid"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "list apps that would be backfilled without writing")
	flag.Parse()

	cfg := config.Load()
	dbConn, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	if !schema.HasAppAuthzKeysTable(dbConn) {
		log.Fatalf("app_authz_keys 表不存在，请先执行迁移 0006_app_authz_keys")
	}

	keyID := strings.TrimSpace(cfg.AuthzKeyID)
	if keyID == "" {
		log.Fatalf("AUTHZ_KEY_ID 为空，无法回填")
	}
	seedHex, err := appsvc.NormalizeSeedHex(cfg.AuthzSigningKey)
	if err != nil {
		log.Fatalf("平台签名私钥(AUTHZ_SIGNING_PRIVATE_KEY)无效: %v", err)
	}
	pubHex, err := appsvc.PublicKeyHexFromSeed(cfg.AuthzSigningKey)
	if err != nil {
		log.Fatalf("无法从平台私钥推导公钥: %v", err)
	}

	// Apps with no active, non-revoked authz key yet.
	var appIDs []uuid.UUID
	if err := dbConn.
		Model(&models.App{}).
		Where("id NOT IN (SELECT app_id FROM app_authz_keys WHERE status = ? AND revoked_at IS NULL)", appsvc.AuthzKeyStatusActive).
		Pluck("id", &appIDs).Error; err != nil {
		log.Fatalf("查询待回填应用失败: %v", err)
	}

	log.Printf("待回填应用数: %d (key_id=%s, pub=%s)", len(appIDs), keyID, pubHex)
	if *dryRun {
		for _, id := range appIDs {
			fmt.Println(id.String())
		}
		log.Printf("dry-run，未写入")
		return
	}

	created := 0
	for _, appID := range appIDs {
		row, err := appsvc.BuildAuthzKeyRow(appID, keyID, seedHex, pubHex, appsvc.AuthzKeyStatusActive, cfg.AppSecretMasterKey)
		if err != nil {
			log.Printf("[skip] app=%s 构造失败: %v", appID, err)
			continue
		}
		if err := dbConn.Create(&row).Error; err != nil {
			// Likely a concurrent backfill or an existing (app_id, key_id) row; skip.
			log.Printf("[skip] app=%s 写入失败: %v", appID, err)
			continue
		}
		created++
	}
	log.Printf("回填完成: 新增 %d 把 active 密钥", created)
}
