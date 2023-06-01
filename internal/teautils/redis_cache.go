package teautils

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis/v8"
	"github.com/tidwall/gjson"
	"golang.org/x/sync/singleflight"

	"errors"
	"strconv"
	"time"
)

var (
	RedisCli      *redis.Client
	lockG         = &singleflight.Group{}
	ErrRedisEmpty = errors.New("redis client cannot be empty.")
	RedisCliPing  = false
)

func SetRedis(addr string, db int, size int, minConn int, password string) error {
	RedisCli = redis.NewClient(&redis.Options{
		Addr:         addr,
		DB:           db,
		PoolSize:     size,
		MinIdleConns: minConn,
		Password:     password,
	})

	_, err := RedisCli.Ping(context.Background()).Result()
	if err != nil {
		return err
	}
	RedisCliPing = true
	return nil
}

/*
*
设置缓存
返回参数,,第一个数据,,第二个数据执行结果
*/
func CheckCache(key string, fn func() (interface{}, error), duration int64, needCache bool) (interface{}, error) {
	s, err := GetCache(key)
	if needCache && err == nil {
		return s, nil
	} else {
		var re interface{}
		//Num, ok := fn()
		//同一时间只有一个带相同key的函数执行 防击穿
		Num, ok, _ := lockG.Do(key, fn)
		if ok == nil {
			SetCache(key, Num, time.Duration(duration)*time.Second)
			re = Num
		} else {
			re = Num
		}

		return re, ok
	}

}

func SetCache(key string, data interface{}, duration time.Duration) error {
	key = Md5Str(key)
	if RedisCli == nil {
		return ErrRedisEmpty
	}
	dataMap := make(map[string]interface{})
	dataMap["data"] = data
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return err
	}
	err = RedisCli.Set(context.Background(), key, jsonData, duration).Err()
	if err != nil {
		return err
	}
	return nil
}

func GetCache(key string) (interface{}, error) {
	key = Md5Str(key)
	if RedisCli == nil {
		return nil, ErrRedisEmpty
	}
	data, err := RedisCli.Get(context.Background(), key).Result()
	if err == nil && data != "" {
		dom := gjson.Parse(data)
		return dom.Get("data").Value(), err
	}

	return "", err
}

func DelCache(key string) error {
	key = Md5Str(key)
	if RedisCli == nil {
		return ErrRedisEmpty
	}
	_ = RedisCli.Del(context.Background(), key).Err()
	//fmt.Println(err)
	return nil
}

// 加锁key 值为1
func SetNx(key string, t time.Duration) (res bool, err error) {
	key = Md5Str(key)
	ctx := context.Background()
	res, err = RedisCli.SetNX(ctx, key, 1, t).Result()
	return
}

// 为锁续期
func WatchDog(ctx context.Context, key string, t time.Duration) (err error) {
	key = Md5Str(key)
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			//续期
			_, err := RedisCli.Expire(ctx, key, t).Result()
			if err != nil {
				return err
			}
			//等待
			<-time.Tick(t / 2)
		}
	}
}

// key 值+1
func Incr(key string, t time.Duration) (res int64, err error) {
	key = Md5Str(key)
	ctx := context.Background()
	//return Rdb.Incr(ctx, key).Result()

	pipe := RedisCli.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, t)

	// Execute
	//
	//     MULTI
	//     INCR pipeline_counter
	//     EXPIRE pipeline_counts 3600
	//     EXEC
	//
	// using one rdb-server roundtrip.
	_, err = pipe.Exec(ctx)
	//fmt.Println(incr.Val(), err)
	res = incr.Val()
	return
}

// 获取key的int值
func GetInt(key string) (res int, err error) {
	key = Md5Str(key)
	ctx := context.Background()
	var result string
	result, err = RedisCli.Get(ctx, key).Result()
	if err == redis.Nil {
		res, err = 0, nil
	} else {
		res, _ = strconv.Atoi(result)
	}

	return
}

/*
*
md5
*/
func Md5Str(str string) string {
	//return str
	data := []byte(str)
	has := md5.Sum(data)
	md5str := fmt.Sprintf("%x", has)
	return md5str
}

// 获取key的有效时间
func GetTtl(key string) (res time.Duration, err error) {
	key = Md5Str(key)
	ctx := context.Background()
	res, err = RedisCli.TTL(ctx, key).Result()
	if err == redis.Nil {
		res, err = 0, nil
	}

	return res, nil
}
