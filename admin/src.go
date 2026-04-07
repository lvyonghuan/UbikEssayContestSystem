package admin

import (
	"errors"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/password"
	"main/util/token"
	"strconv"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

func loginSrc(admin model.Admin) (token.ResponseToken, error) {
	dbAdmin, err := pgsql.FindAdminByUsername(admin.AdminName)
	if err != nil {
		return token.ResponseToken{}, err
	}

	isSame := password.CheckPasswordHash(admin.Password, dbAdmin.Password)
	if !isSame {
		return token.ResponseToken{}, uerr.NewError(errors.New("login error"))
	}

	return token.GenTokenAndRefreshToken(int64(dbAdmin.AdminID), _const.RoleAdmin)
}

func refreshTokenSrc(adminID int64) (token.ResponseToken, error) {
	return token.GenTokenAndRefreshToken(adminID, _const.RoleAdmin)
}

func createContestSrc(adminID int, contest *model.Contest) error {
	err := pgsql.CreateContest(contest)
	if err != nil {
		log.Logger.Warn("Create contest error: " + err.Error())

		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Contests, _const.Create,
		genDetails([]string{"contest_name", "contest_id"}, []string{contest.ContestName, strconv.Itoa(contest.ContestID)}))

	// TODO 激活比赛倒计时

	return nil
}

func updateContestSrc(adminID int, contestID int, updatedContest *model.Contest) error {
	err := pgsql.UpdateContest(contestID, updatedContest)
	if err != nil {
		log.Logger.Warn("Update contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	//FIXME 修改所有赛道的开始结束日期

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Contests, _const.Update,
		genDetails([]string{"contest_name", "contest_id"}, []string{updatedContest.ContestName, strconv.Itoa(contestID)}))

	return nil
}

func deleteContestSrc(adminID int, contestID int) error {
	contest, err := pgsql.DeleteContest(contestID)
	if err != nil {
		log.Logger.Warn("Delete contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	//FIXME 删除所有赛道缓存

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Contests, _const.Delete,
		genDetails([]string{"contest_name", "contest_id"}, []string{contest.ContestName, strconv.Itoa(contestID)}))

	return nil
}

func createTrackSrc(adminID int, track *model.Track) error {
	//在redis中写缓存
	//1. 获取contest信息
	contest, err := pgsql.GetContestByID(track.ContestID)
	if err != nil {
		log.Logger.Warn("Get contest error: " + err.Error())
		return uerr.ExtractError(err)
	}
	//2. 写redis缓存
	err = redis.CreateTrack(*track, contest)
	if err != nil {
		log.Logger.Warn("Create track cache error: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = pgsql.CreateTrack(track)
	if err != nil {
		//删除redis缓存
		_ = redis.DeleteTrack(track.TrackID)

		log.Logger.Warn("Create track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Tracks, _const.Create,
		genDetails([]string{"track_name", "track_id"}, []string{track.TrackName, strconv.Itoa(track.TrackID)}))

	return nil
}

func updateTrackSrc(adminID int, trackID int, updatedTrack *model.Track) error {
	err := pgsql.UpdateTrack(trackID, updatedTrack)
	if err != nil {
		log.Logger.Warn("Update track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Tracks, _const.Update,
		genDetails([]string{"track_name", "track_id"}, []string{updatedTrack.TrackName, strconv.Itoa(trackID)}))

	return nil
}

func deleteTrackSrc(adminID int, trackID int) error {
	track, err := pgsql.DeleteTrack(trackID)
	if err != nil {
		log.Logger.Warn("Delete track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	newAdminActionLog(adminID, _const.Tracks, _const.Delete,
		genDetails([]string{"track_name", "track_id"}, []string{track.TrackName, strconv.Itoa(trackID)}))

	// 删除redis赛道缓存
	err = redis.DeleteTrack(track.TrackID)
	if err != nil {
		log.Logger.Warn("Delete track error: " + err.Error())
	}

	return nil
}
