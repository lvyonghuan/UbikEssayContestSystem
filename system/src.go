package system

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"strings"

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

func getContestSrc() ([]model.Contest, error) {
	contests, err := getContestListFn()
	if err != nil {
		log.Logger.Error(errors.New("GetContestList error: " + err.Error()))
		return nil, uerr.ExtractError(err)
	}

	return contests, nil
}

func getTracksSrc(contestID int) ([]model.Track, error) {
	tracks, err := getTrackListFn(contestID)
	if err != nil {
		log.Logger.Error(errors.New("GetTrackList error: " + err.Error()))
		return nil, uerr.ExtractError(err)
	}

	return tracks, nil
}

func getContestByIDSrc(contestID int) (model.Contest, error) {
	contest, err := getContestByIDPg(contestID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return model.Contest{}, errContestNotFound
		}
		log.Logger.Error(errors.New("GetContestByID error: " + parsedErr.Error()))
		return model.Contest{}, parsedErr
	}

	return contest, nil
}

func getTrackByIDSrc(trackID int) (model.Track, error) {
	track, err := getTrackByIDPg(trackID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return model.Track{}, errTrackNotFound
		}
		log.Logger.Error(errors.New("GetTrackByID error: " + parsedErr.Error()))
		return model.Track{}, parsedErr
	}

	return track, nil
}
