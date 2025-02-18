package config

import (
	"context"
	"testing"
	"time"
)

func TestRedis(t *testing.T) {
	t.Skip("skip test")

	// Test must redis
	conf := &RedisConf{
		Host:     "localhost:6379",
		Db:       0,
		Username: "",
		Pass:     "",
		Tls:      false,
	}

	rds := conf.MustNewUniversalRedis()

	rds.Set(context.Background(), "testKeyDb0", "testVal", 2*time.Minute)

	conf1 := &RedisConf{
		Host:     "localhost:6379",
		Db:       1,
		Username: "",
		Pass:     "",
		Tls:      false,
	}

	rds1 := conf1.MustNewUniversalRedis()

	rds1.Set(context.Background(), "testKeyDb1", "testVal", 2*time.Minute)

	conf2 := &RedisConf{
		Host:     "localhost:6379,localhost:6380",
		Db:       1,
		Username: "",
		Pass:     "",
		Tls:      false,
	}

	rds2 := conf2.MustNewUniversalRedis()

	rds2.Set(context.Background(), "testCluster", "testCluster", 2*time.Minute)
}
