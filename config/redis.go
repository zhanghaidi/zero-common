package config

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/redis/go-redis/v9"
	"strings"
	"time"
)

type RedisConf struct {
	Host       string `json:",default=127.0.0.1:6379"`
	Db         int    `json:",default=0"`
	Username   string `json:",optional"`
	Pass       string `json:",optional"` // 可选的密码字段
	Tls        bool   `json:",optional"`
	MasterName string `json:",optional"`
	PoolSize   int    `json:",optional,default=10"`
}

// Validate 验证 Redis 配置是否正确
func (r RedisConf) Validate() error {
	if r.Host == "" {
		return errors.New("RedisHost不能为空")
	}
	return nil
}

// NewRedisClient 创建 Redis 客户端
func (r RedisConf) NewRedisClient() (redis.UniversalClient, error) {
	err := r.Validate()
	if err != nil {
		return nil, err
	}

	opt := &redis.UniversalOptions{
		Addrs:        strings.Split(r.Host, ","),
		DB:           r.Db,
		Username:     r.Username,
		PoolSize:     r.PoolSize,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	}

	// 仅在 Pass 不为空时设置密码
	if r.Pass != "" {
		opt.Password = r.Pass
	}

	// 如果有 MasterName，则设置主节点名称
	if r.MasterName != "" {
		opt.MasterName = r.MasterName
	}

	// 启用 TLS 配置（可选）
	if r.Tls {
		opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}

	client := redis.NewUniversalClient(opt)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// MustNewRedisClient 必须创建 Redis 客户端，若失败则直接 panic
func (r RedisConf) MustNewRedisClient() redis.UniversalClient {
	client, err := r.NewRedisClient()
	if err != nil {
		panic(err)
	}
	return client
}
