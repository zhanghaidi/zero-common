package define

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	GlobalDatabase *gorm.DB
	GlobalRedis    redis.UniversalClient
)
