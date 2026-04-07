package system

//import (
//	"context"
//	"errors"
//	"main/model"
//	_const "main/util/const"
//	"time"
//
//	"github.com/lvyonghuan/Ubik-Util/uerr"
//)
//
//// 倒计时总体上分为两个大类：比赛倒计时和评审deadline（这里的评审对应的是单次评审事件）
//// 比赛倒计时又可以细分为两种：比赛开始倒计时和比赛deadline
//
//type contestCutdown struct {
//	contest model.Contest
//	// 0: 未开始，1: 进行中，2: 已结束
//	status int
//	// 计时器
//	timer *time.Timer
//	// 取消context
//	context context.Context
//	// 取消函数
//	cancel context.CancelFunc
//}
//
//var contestsCutdown map[int]contestCutdown //K：ID
//
//// TODO 评审计时
//
//func initContestCutdown(contest model.Contest) {
//	// 判断当前比赛状态
//	nowTime := time.Now()
//	var statu int
//	var currentDeadLine time.Time
//
//	// 判断当前时间处在比赛的哪个时间区间（比赛前、 比赛中、比赛后）
//	switch {
//	case nowTime.Before(time.Time(contest.ContestStartDate)):
//		statu = _const.ContestNotBegin
//		currentDeadLine = time.Time(contest.ContestStartDate)
//	case nowTime.After(time.Time(contest.ContestEndDate)):
//		statu = _const.ContestEnded
//		currentDeadLine = time.Time(contest.ContestEndDate)
//	default:
//		statu = _const.ContestOngoing
//		currentDeadLine = time.Time(contest.ContestEndDate)
//
//	}
//
//	// 判断比赛是否结束
//	if statu == _const.ContestEnded {
//		c := contestCutdown{contest: contest, status: statu, timer: nil}
//		contestsCutdown[contest.ContestID] = c
//	}
//
//	// 创建计时器
//	timer := newTimer(currentDeadLine)
//
//	// 创建取消context
//	ctx, cancel := context.WithCancel(context.Background())
//
//	c := contestCutdown{contest: contest, status: statu, timer: timer, context: ctx, cancel: cancel}
//	contestsCutdown[contest.ContestID] = c
//
//	go c.contestStatuChanger() // 启用比赛状态改变器
//}
//
//// 创建计时器
//func newTimer(deadline time.Time) *time.Timer {
//	timer := time.NewTimer(deadline.Sub(time.Now()))
//	return timer
//}
//
//// 根据倒计时改变比赛状态
//func (contest *contestCutdown) contestStatuChanger() {
//	for {
//		select {
//		case <-contest.timer.C: //倒计时结束，触发状态改变
//			contest.timer.Stop()
//			contest.status++                           // 状态位自增
//			if contest.status == _const.ContestEnded { // 如果比赛结束则返回
//				return
//			}
//
//			if contest.status == _const.ContestOngoing {
//				// 计算比赛结束时间与当前时间的差值，重置计时器
//				duration := time.Time(contest.contest.ContestEndDate).Sub(time.Now())
//				contest.timer.Reset(duration)
//				// TODO 触发比赛开始事件（是否有？）
//			}
//
//		case <-contest.context.Done(): //外部取消信号，停止计时器并退出
//			return
//		}
//	}
//}
//
//func GetContestStatus(contestID int) (int, error) {
//	contest, ok := contestsCutdown[contestID]
//	if !ok {
//		return -1, uerr.NewError(errors.New("比赛ID不存在"))
//	}
//
//	return contest.status, nil
//}
