package system

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

var (
	getContestListFn = pgsql.GetContestList
	getTrackListFn   = pgsql.GetTrackList
	getContestByIDPg = pgsql.GetContestByID
	getTrackByIDPg   = pgsql.GetTrackByID
)

var (
	errContestNotFound = errors.New("contest not found")
	errTrackNotFound   = errors.New("track not found")
)

func chinaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}

func withLocationKeepWallClock(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

func normalizeContestTimezone(contest *model.Contest) {
	loc := chinaLocation()
	contest.ContestStartDate = withLocationKeepWallClock(contest.ContestStartDate, loc)
	contest.ContestEndDate = withLocationKeepWallClock(contest.ContestEndDate, loc)
}

func getContestSrc() ([]model.Contest, error) {
	contests, err := getContestListFn()
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Error(errors.New("GetContestList error: " + parsedErr.Error()))
		return nil, parsedErr
	}

	for i := range contests {
		normalizeContestTimezone(&contests[i])
	}

	return contests, nil
}

func getTracksSrc(contestID int) ([]model.Track, error) {
	tracks, err := getTrackListFn(contestID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Error(errors.New("GetTrackList error: " + parsedErr.Error()))
		return nil, parsedErr
	}

	return tracks, nil
}

func getContestByIDSrc(contestID int) (model.Contest, error) {
	contest, err := getContestByIDPg(contestID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Error(errors.New("GetContestByID error: " + errContestNotFound.Error()))
			return model.Contest{}, errContestNotFound
		}
		log.Logger.Error(errors.New("GetContestByID error: " + parsedErr.Error()))
		return model.Contest{}, parsedErr
	}

	normalizeContestTimezone(&contest)

	return contest, nil
}

func getTrackByIDSrc(trackID int) (model.Track, error) {
	track, err := getTrackByIDPg(trackID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Error(errors.New("GetTrackByID error: " + errTrackNotFound.Error()))
			return model.Track{}, errTrackNotFound
		}
		log.Logger.Error(errors.New("GetTrackByID error: " + parsedErr.Error()))
		return model.Track{}, parsedErr
	}

	return track, nil
}
