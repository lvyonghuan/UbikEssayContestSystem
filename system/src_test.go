package system

import (
	"errors"
	"main/model"
	"main/util/log"
	"testing"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

type systemSrcTestLogger struct{}

func (systemSrcTestLogger) Debug(string)  {}
func (systemSrcTestLogger) Info(string)   {}
func (systemSrcTestLogger) Warn(string)   {}
func (systemSrcTestLogger) Error(error)   {}
func (systemSrcTestLogger) Fatal(error)   {}
func (systemSrcTestLogger) System(string) {}

func backupSystemSrcHooks(t *testing.T) {
	origLogger := log.Logger
	origGetContestListFn := getContestListFn
	origGetTrackListFn := getTrackListFn
	origGetContestByIDPg := getContestByIDPg
	origGetTrackByIDPg := getTrackByIDPg

	log.Logger = systemSrcTestLogger{}

	t.Cleanup(func() {
		log.Logger = origLogger
		getContestListFn = origGetContestListFn
		getTrackListFn = origGetTrackListFn
		getContestByIDPg = origGetContestByIDPg
		getTrackByIDPg = origGetTrackByIDPg
	})
}

func TestGetContestSrc(t *testing.T) {
	backupSystemSrcHooks(t)

	getContestListFn = func() ([]model.Contest, error) {
		return []model.Contest{{ContestID: 1, ContestName: "c"}}, nil
	}
	contests, err := getContestSrc()
	if err != nil || len(contests) != 1 {
		t.Fatalf("getContestSrc failed: %v, %+v", err, contests)
	}

	getContestListFn = func() ([]model.Contest, error) {
		return nil, errors.New("db error")
	}
	if _, err = getContestSrc(); err == nil {
		t.Fatal("getContestSrc should fail on db error")
	}
}

func TestGetTracksSrc(t *testing.T) {
	backupSystemSrcHooks(t)

	getTrackListFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 1, ContestID: contestID}}, nil
	}
	tracks, err := getTracksSrc(3)
	if err != nil || len(tracks) != 1 {
		t.Fatalf("getTracksSrc failed: %v, %+v", err, tracks)
	}

	getTrackListFn = func(contestID int) ([]model.Track, error) {
		return nil, errors.New("db error")
	}
	if _, err = getTracksSrc(1); err == nil {
		t.Fatal("getTracksSrc should fail on db error")
	}
}

func TestGetContestByIDSrc(t *testing.T) {
	backupSystemSrcHooks(t)

	getContestByIDPg = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestName: "c"}, nil
	}
	contest, err := getContestByIDSrc(8)
	if err != nil || contest.ContestID != 8 {
		t.Fatalf("getContestByIDSrc failed: %v, %+v", err, contest)
	}

	getContestByIDPg = func(contestID int) (model.Contest, error) {
		return model.Contest{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	if _, err = getContestByIDSrc(1); !errors.Is(err, errContestNotFound) {
		t.Fatalf("expected errContestNotFound, got %v", err)
	}

	getContestByIDPg = func(contestID int) (model.Contest, error) {
		return model.Contest{}, errors.New("db error")
	}
	if _, err = getContestByIDSrc(1); err == nil || errors.Is(err, errContestNotFound) {
		t.Fatalf("expected generic error, got %v", err)
	}
}

func TestGetTrackByIDSrc(t *testing.T) {
	backupSystemSrcHooks(t)

	getTrackByIDPg = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, TrackName: "t"}, nil
	}
	track, err := getTrackByIDSrc(6)
	if err != nil || track.TrackID != 6 {
		t.Fatalf("getTrackByIDSrc failed: %v, %+v", err, track)
	}

	getTrackByIDPg = func(trackID int) (model.Track, error) {
		return model.Track{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	if _, err = getTrackByIDSrc(1); !errors.Is(err, errTrackNotFound) {
		t.Fatalf("expected errTrackNotFound, got %v", err)
	}

	getTrackByIDPg = func(trackID int) (model.Track, error) {
		return model.Track{}, errors.New("db error")
	}
	if _, err = getTrackByIDSrc(1); err == nil || errors.Is(err, errTrackNotFound) {
		t.Fatalf("expected generic error, got %v", err)
	}
}

func TestNormalizeContestTimezoneKeepsWallClock(t *testing.T) {
	contest := model.Contest{
		ContestStartDate: time.Date(2026, 4, 13, 10, 30, 0, 0, time.UTC),
		ContestEndDate:   time.Date(2026, 4, 13, 18, 45, 0, 0, time.UTC),
	}

	normalizeContestTimezone(&contest)

	startName, startOffset := contest.ContestStartDate.Zone()
	endName, endOffset := contest.ContestEndDate.Zone()
	if startOffset != 8*3600 || endOffset != 8*3600 {
		t.Fatalf("expected +08:00 offset, got start=%s(%d) end=%s(%d)", startName, startOffset, endName, endOffset)
	}

	if contest.ContestStartDate.Hour() != 10 || contest.ContestStartDate.Minute() != 30 {
		t.Fatalf("start wall-clock changed unexpectedly: %s", contest.ContestStartDate.Format(time.RFC3339))
	}
	if contest.ContestEndDate.Hour() != 18 || contest.ContestEndDate.Minute() != 45 {
		t.Fatalf("end wall-clock changed unexpectedly: %s", contest.ContestEndDate.Format(time.RFC3339))
	}
}
