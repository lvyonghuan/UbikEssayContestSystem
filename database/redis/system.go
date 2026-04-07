package redis

import (
	"errors"
	"main/model"
	_const "main/util/const"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

// CreateTrack 在redis中创建比赛缓存，在系统初始化时或创建比赛时调用
// 实际上只存储比赛的开始日期与结束日期，以时间戳的形式存储
func CreateTrack(track model.Track, contest model.Contest) error {
	start := time.Time(contest.ContestStartDate).Unix()
	end := time.Time(contest.ContestEndDate).Unix()

	err := rdb.client.Set(rdb.ctx, _const.RedisTrackStartDatePrefix+strconv.Itoa(track.TrackID), start, 0).Err()
	if err != nil {
		return uerr.NewError(err)
	}

	err = rdb.client.Set(rdb.ctx, _const.RedisTrackEndDatePrefix+strconv.Itoa(track.TrackID), end, 0).Err()
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteTrack(trackID int) error {
	err := rdb.client.Del(rdb.ctx, _const.RedisTrackEndDatePrefix+strconv.Itoa(trackID)).Err()
	if err != nil {
		return uerr.NewError(err)
	}

	err = rdb.client.Del(rdb.ctx, _const.RedisTrackEndDatePrefix+strconv.Itoa(trackID)).Err()
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetStartAndEndDate(trackID int) (start, end int64, err error) {
	startStr, err := rdb.client.Get(rdb.ctx, _const.RedisTrackEndDatePrefix+strconv.Itoa(trackID)).Result()
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	endStr, err := rdb.client.Get(rdb.ctx, _const.RedisTrackEndDatePrefix+strconv.Itoa(trackID)).Result()
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	start, err = strconv.ParseInt(startStr, 10, 64)
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	end, err = strconv.ParseInt(endStr, 10, 64)
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	return start, end, nil
}

func SetUploadFilePermission(authorID, trackID, workID int) error {
	err := rdb.client.Set(rdb.ctx, _const.RedisUploadPermissionPrefix+"-"+strconv.Itoa(workID), strconv.Itoa(trackID)+"-"+strconv.Itoa(authorID), 5*time.Minute).Err()
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetUploadFilePermission(workID int) (authorID, trackID int, err error) {
	result, err := rdb.client.Get(rdb.ctx, _const.RedisUploadPermissionPrefix+"-"+strconv.Itoa(workID)).Result()
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	parts := strings.Split(result, "-")
	if len(parts) != 2 {
		return 0, 0, uerr.NewError(errors.New("invalid permission format"))
	}

	authorID, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	trackID, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, uerr.NewError(err)
	}

	return authorID, trackID, nil
}
