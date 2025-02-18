package captcha

import (
	"context"
	"github.com/zhanghaidi/zero-common/config"

	"time"

	"github.com/mojocn/base64Captcha"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

// NewRedisStore returns a redis store for captcha.
func NewRedisStore(r *redis.Redis) *RedisStore {
	return &RedisStore{
		Expiration: time.Minute * 5,
		PreKey:     config.RedisCaptchaPrefix,
		Redis:      r,
	}
}

// RedisStore stores captcha data.
type RedisStore struct {
	Expiration time.Duration
	PreKey     string
	Context    context.Context
	Redis      *redis.Redis
}

// UseWithCtx add context for captcha.
func (r *RedisStore) UseWithCtx(ctx context.Context) base64Captcha.Store {
	r.Context = ctx
	return r
}

// Set sets the captcha KV to redis.
func (r *RedisStore) Set(id string, value string) error {
	err := r.Redis.Setex(r.PreKey+id, value, int(r.Expiration.Seconds()))
	if err != nil {
		logx.Errorw("error occurs when captcha key sets to redis", logx.Field("detail", err))
		return err
	}
	return nil
}

// Get gets the captcha KV from redis.
func (r *RedisStore) Get(key string, clear bool) string {
	val, err := r.Redis.Get(key)
	if err != nil {
		logx.Errorw("error occurs when captcha key gets from redis", logx.Field("detail", err))
		return ""
	}
	if clear {
		_, err := r.Redis.Del(key)
		if err != nil {
			logx.Errorw("error occurs when captcha key deletes from redis", logx.Field("detail", err))
			return ""
		}
	}
	return val
}

// Verify verifies the captcha whether it is correct.
func (r *RedisStore) Verify(id, answer string, clear bool) bool {
	key := r.PreKey + id
	v := r.Get(key, clear)
	return v == answer
}
