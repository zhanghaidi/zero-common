package config

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zhanghaidi/zero-common/define"
	"strings"
	"time"
)

// A RedisConf is a redis config.
type RedisConf struct {
	Host     string `json:",env=REDIS_HOST"`
	Db       int    `json:",default=0,env=REDIS_DB"`
	Username string `json:",optional,env=REDIS_USERNAME"`
	Pass     string `json:",optional,env=REDIS_PASSWORD"`
	Tls      bool   `json:",optional,env=REDIS_TLS"`
	Master   string `json:",optional,env=REDIS_MASTER"`
}

func (r RedisConf) Validate() error {
	if len(r.Host) == 0 {
		return errors.New("host cannot be empty")
	}
	return nil
}

func (r RedisConf) NewUniversalRedis() (redis.UniversalClient, error) {
	err := r.Validate()
	if err != nil {
		return nil, err
	}

	opt := &redis.UniversalOptions{
		Addrs:    strings.Split(r.Host, ","),
		DB:       r.Db,
		Password: r.Pass,
		Username: r.Username,
	}

	if r.Master != "" {
		opt.MasterName = r.Master
	}

	if r.Tls {
		opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	rds := redis.NewUniversalClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = rds.Ping(ctx).Err()
	if err != nil {
		return nil, err
	}

	define.GlobalRedis = rds // 设置全局Redis变量

	return rds, nil
}

func (r RedisConf) MustNewUniversalRedis() redis.UniversalClient {
	rds, err := r.NewUniversalRedis()
	logx.Must(err)

	return rds
}
