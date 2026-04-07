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
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

var (
	findAdminByUsernameFn     = pgsql.FindAdminByUsername
	genTokenAndRefreshTokenFn = token.GenTokenAndRefreshToken

	createContestFn      = pgsql.CreateContest
	updateContestFn      = pgsql.UpdateContest
	deleteContestFn      = pgsql.DeleteContest
	getTracksByContestFn = pgsql.GetTracksByContestID
	getContestByIDFn     = pgsql.GetContestByID

	createTrackFn      = pgsql.CreateTrack
	updateTrackFn      = pgsql.UpdateTrack
	deleteTrackFn      = pgsql.DeleteTrack
	createTrackCacheFn = redis.CreateTrack
	deleteTrackCacheFn = redis.DeleteTrack

	getWorkByIDFn      = pgsql.GetWorkByID
	getWorksByTrackFn  = pgsql.GetWorksByTrackID
	getWorksByAuthorFn = pgsql.GetWorksByAuthorID
	deleteWorkByIDFn   = pgsql.DeleteWorkByID
	deleteUploadPermFn = redis.DeleteUploadFilePermission

	createActionLogFn = newAdminActionLog

	readDirFn = os.ReadDir
	removeFn  = os.Remove
)

var (
	errWorkNotFound     = errors.New("work not found")
	errWorkFileNotFound = errors.New("work file not found")
)

func loginSrc(admin model.Admin) (token.ResponseToken, error) {
	dbAdmin, err := findAdminByUsernameFn(admin.AdminName)
	if err != nil {
		return token.ResponseToken{}, err
	}

	isSame := password.CheckPasswordHash(admin.Password, dbAdmin.Password)
	if !isSame {
		return token.ResponseToken{}, uerr.NewError(errors.New("login error"))
	}

	return genTokenAndRefreshTokenFn(int64(dbAdmin.AdminID), _const.RoleAdmin)
}

func refreshTokenSrc(adminID int64) (token.ResponseToken, error) {
	return genTokenAndRefreshTokenFn(adminID, _const.RoleAdmin)
}

func createContestSrc(adminID int, contest *model.Contest) error {
	err := createContestFn(contest)
	if err != nil {
		log.Logger.Warn("Create contest error: " + err.Error())

		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Contests, _const.Create,
		genDetails([]string{"contest_name", "contest_id"}, []string{contest.ContestName, strconv.Itoa(contest.ContestID)}))

	return nil
}

func updateContestSrc(adminID int, contestID int, updatedContest *model.Contest) error {
	err := updateContestFn(contestID, updatedContest)
	if err != nil {
		log.Logger.Warn("Update contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	tracks, err := getTracksByContestFn(contestID)
	if err != nil {
		log.Logger.Warn("Get tracks by contest id error: " + err.Error())
		return uerr.ExtractError(err)
	}

	for _, track := range tracks {
		cacheErr := createTrackCacheFn(track, *updatedContest)
		if cacheErr != nil {
			log.Logger.Warn("Update contest cache error: " + cacheErr.Error())
		}
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Contests, _const.Update,
		genDetails([]string{"contest_name", "contest_id"}, []string{updatedContest.ContestName, strconv.Itoa(contestID)}))

	return nil
}

func deleteContestSrc(adminID int, contestID int) error {
	tracks, err := getTracksByContestFn(contestID)
	if err != nil {
		log.Logger.Warn("Get tracks by contest id error: " + err.Error())
		return uerr.ExtractError(err)
	}

	contest, err := deleteContestFn(contestID)
	if err != nil {
		log.Logger.Warn("Delete contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	for _, track := range tracks {
		cacheErr := deleteTrackCacheFn(track.TrackID)
		if cacheErr != nil {
			log.Logger.Warn("Delete contest cache error: " + cacheErr.Error())
		}
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Contests, _const.Delete,
		genDetails([]string{"contest_name", "contest_id"}, []string{contest.ContestName, strconv.Itoa(contestID)}))

	return nil
}

func createTrackSrc(adminID int, track *model.Track) error {
	// 先写数据库，确保拿到真实track_id后再写redis。
	contest, err := getContestByIDFn(track.ContestID)
	if err != nil {
		log.Logger.Warn("Get contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = createTrackFn(track)
	if err != nil {
		log.Logger.Warn("Create track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = createTrackCacheFn(*track, contest)
	if err != nil {
		_, rollbackErr := deleteTrackFn(track.TrackID)
		if rollbackErr != nil {
			log.Logger.Warn("Rollback track on cache error failed: " + rollbackErr.Error())
		}

		log.Logger.Warn("Create track cache error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Tracks, _const.Create,
		genDetails([]string{"track_name", "track_id"}, []string{track.TrackName, strconv.Itoa(track.TrackID)}))

	return nil
}

func updateTrackSrc(adminID int, trackID int, updatedTrack *model.Track) error {
	err := updateTrackFn(trackID, updatedTrack)
	if err != nil {
		log.Logger.Warn("Update track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Tracks, _const.Update,
		genDetails([]string{"track_name", "track_id"}, []string{updatedTrack.TrackName, strconv.Itoa(trackID)}))

	return nil
}

func deleteTrackSrc(adminID int, trackID int) error {
	track, err := deleteTrackFn(trackID)
	if err != nil {
		log.Logger.Warn("Delete track error: " + err.Error())
		return uerr.ExtractError(err)
	}

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Tracks, _const.Delete,
		genDetails([]string{"track_name", "track_id"}, []string{track.TrackName, strconv.Itoa(trackID)}))

	// 删除redis赛道缓存
	err = deleteTrackCacheFn(track.TrackID)
	if err != nil {
		log.Logger.Warn("Delete track error: " + err.Error())
	}

	return nil
}

func getWorkByIDSrc(workID int) (model.Work, error) {
	work, err := getWorkByIDFn(workID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return model.Work{}, errWorkNotFound
		}
		log.Logger.Warn("Get work by id error: " + err.Error())
		return model.Work{}, parsedErr
	}

	return work, nil
}

func getWorksByTrackIDSrc(trackID int) ([]model.Work, error) {
	works, err := getWorksByTrackFn(trackID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Get works by track id error: " + err.Error())
		return nil, parsedErr
	}

	return works, nil
}

func getWorksByAuthorIDSrc(authorID int) ([]model.Work, error) {
	works, err := getWorksByAuthorFn(authorID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Get works by author id error: " + err.Error())
		return nil, parsedErr
	}

	return works, nil
}

func getWorkFilePathSrc(workID int) (string, error) {
	work, err := getWorkByIDSrc(workID)
	if err != nil {
		return "", err
	}

	return resolveWorkFilePath(work)
}

func resolveWorkFilePath(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.FileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errWorkFileNotFound
		}
		return "", uerr.ExtractError(err)
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	selectedName := ""
	selectedTime := time.Time{}
	hasDocx := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		isDocx := ext == ".docx"

		if isDocx {
			if !hasDocx || selectedName == "" || info.ModTime().After(selectedTime) {
				hasDocx = true
				selectedName = name
				selectedTime = info.ModTime()
			}
			continue
		}

		if hasDocx {
			continue
		}

		if selectedName == "" || info.ModTime().After(selectedTime) {
			selectedName = name
			selectedTime = info.ModTime()
		}
	}

	if selectedName == "" {
		return "", errWorkFileNotFound
	}

	return filepath.Join(dstDir, selectedName), nil
}

func deleteWorkSrc(adminID, workID int) error {
	work, err := getWorkByIDSrc(workID)
	if err != nil {
		return err
	}

	err = deleteWorkFiles(work)
	if err != nil {
		log.Logger.Warn("Delete work files error: " + err.Error())
		return err
	}

	err = deleteWorkByIDFn(workID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Delete work by id error: " + err.Error())
		return parsedErr
	}

	permErr := deleteUploadPermFn(workID)
	if permErr != nil {
		log.Logger.Warn("Delete work upload permission cache error: " + permErr.Error())
	}

	createActionLogFn(adminID, _const.Works, _const.Delete,
		genDetails(
			[]string{"work_id", "work_title", "track_id", "author_id"},
			[]string{strconv.Itoa(work.WorkID), work.WorkTitle, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID)},
		),
	)

	return nil
}

func deleteWorkFiles(work model.Work) error {
	dstDir := filepath.Join(_const.FileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	prefix := strconv.Itoa(work.WorkID) + "."

	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return uerr.ExtractError(err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		if rmErr := removeFn(filepath.Join(dstDir, name)); rmErr != nil {
			return uerr.ExtractError(rmErr)
		}
	}

	return nil
}
