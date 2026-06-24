package handlers

import (
	"software-web-manager/backend/internal/auth"
	"software-web-manager/backend/internal/config"
	"software-web-manager/backend/internal/geo"
	"software-web-manager/backend/internal/storage"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Handler struct {
	DB      *gorm.DB
	Cfg     config.Config
	Storage storage.Driver
	ReplayStore *redis.Client
	Hub     *wsHub
	ClientUpdateHub *ClientUpdateHub
	OnlineTracker *OnlineTracker
	RegionResolver geo.Resolver
	AuthzSigner *auth.AuthzSigner
}


