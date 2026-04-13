package system

import (
	"errors"
	"main/conf"
	"main/database/pgsql"
	"main/database/redis"
	"main/util/log"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

var (
	initContestRedisCacheFn = initContestRedisCache
	initGlobalInfoRouterFn  = initGlobalInfoRouter

	getContestListForCacheFn = pgsql.GetContestList
	getTracksByContestFn     = pgsql.GetTracksByContestID
	createTrackCacheFn       = redis.CreateTrack
)

// SysStart 各项动作初始化（或中断恢复）
func SysStart(apiConf conf.APIConfig) {
	// 初始化比赛信息redis缓存
	err := initContestRedisCacheFn()
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Error(errors.New("System start init contest redis cache error: " + parsedErr.Error()))
		panic(parsedErr)
	}

	// 初始化比赛结束定时调度
	err = initContestEndSchedulesFn()
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Error(errors.New("System start init contest end schedule error: " + parsedErr.Error()))
		panic(parsedErr)
	}

	// 初始化路由
	go initGlobalInfoRouterFn(apiConf)
}

func initContestRedisCache() error {
	//1. 从数据库中获取全部的比赛信息
	contests, err := getContestListForCacheFn()
	if err != nil {
		return err
	}

	//2. 遍历比赛
	for _, contest := range contests {
		tracks, trackErr := getTracksByContestFn(contest.ContestID)
		if trackErr != nil {
			return trackErr
		}

		// 将赛道开始/结束时间写入Redis缓存
		for _, track := range tracks {
			if err = createTrackCacheFn(track, contest); err != nil {
				return err
			}
		}
	}

	return nil
}
