package system

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	"main/util/log"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

func getContestSrc() ([]model.Contest, error) {
	contests, err := pgsql.GetContestList()
	if err != nil {
		log.Logger.Error(errors.New("GetContestList error: " + err.Error()))
		return nil, uerr.ExtractError(err)
	}

	return contests, nil
}

func getTracksSrc(contestID int) ([]model.Track, error) {
	tracks, err := pgsql.GetTrackList(contestID)
	if err != nil {
		log.Logger.Error(errors.New("GetTrackList error: " + err.Error()))
		return nil, uerr.ExtractError(err)
	}

	return tracks, nil
}
