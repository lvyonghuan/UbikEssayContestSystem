package system

import (
	"main/conf"
	"main/database/pgsql"
	"main/database/redis"
)

// SysStart 各项动作初始化（或中断恢复）
func SysStart(apiConf conf.APIConfig) {
	//初始化全局变量
	//contestsCutdown = make(map[int]contestCutdown)

	// 初始化比赛倒计时
	//err := initContestCountdown()
	//if err != nil {
	//	panic(err)
	//}

	//TODO 初始化评审倒计时

	// 初始化比赛信息redis缓存
	err := initContestRedisCache()
	if err != nil {
		panic(err)
	}

	// 初始化路由
	go initGlobalInfoRouter(apiConf)
}

func initContestRedisCache() error {
	//1. 从数据库中获取全部的比赛信息
	contests, err := pgsql.GetContestList()
	if err != nil {
		return err
	}

	//2. 遍历比赛
	for _, contest := range contests {
		// 将比赛信息存入Redis缓存
		err := redis.CreateContest(contest)
		if err != nil {
			return err
		}
	}

	return nil
}

//func initContestCountdown() error {
//	//1. 从数据库中获取全部的比赛信息
//	contests, err := pgsql.GetContestList()
//	if err != nil {
//		return err
//	}
//
//	//2. 遍历比赛
//	for _, contest := range contests {
//		// 初始化比赛状态改变器
//		initContestCutdown(contest)
//	}
//
//	return nil
//}
