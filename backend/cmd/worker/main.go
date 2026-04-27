package main

import (
	"context"
	"log"
	"time"

	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/db"
	"software-web-manager/backend/internal/jobs"
)

func main() {
	cfg := config.Load()
	conn, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	if err := db.SetConnPool(conn); err != nil {
		log.Fatalf("db pool: %v", err)
	}
	interval := time.Duration(cfg.WorkerIntervalSeconds) * time.Second
	jobs.Start(context.Background(), conn, interval, log.Default())
	log.Printf("独立 worker 已启动，间隔 %d 秒", cfg.WorkerIntervalSeconds)
	select {}
}

