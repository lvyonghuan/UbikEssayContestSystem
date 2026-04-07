package redis

import (
	"context"
	"main/conf"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"github.com/redis/go-redis/v9"
)

type rdbStruct struct {
	client *redis.Client
	ctx    context.Context
}

var rdb *rdbStruct

func InitRedis(redisConf conf.RedisConfig) error {
	rdb = &rdbStruct{}

	rdb.ctx = context.Background()
	rdb.client = redis.NewClient(&redis.Options{
		Addr:     redisConf.Host + ":" + redisConf.Port,
		Password: redisConf.Password,
		DB:       redisConf.DB,
	})

	// Test connection
	_, err := rdb.client.Ping(rdb.ctx).Result()
	if err != nil {
		return uerr.NewError(err)
	}
	return nil
}
