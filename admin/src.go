package admin

import (
	"errors"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	"main/system"
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

	createTrackFn                   = pgsql.CreateTrack
	updateTrackFn                   = pgsql.UpdateTrack
	deleteTrackFn                   = pgsql.DeleteTrack
	createTrackCacheFn              = redis.CreateTrack
	deleteTrackCacheFn              = redis.DeleteTrack
	registerContestEndScheduleFn    = system.RegisterContestEndSchedule
	cancelContestEndScheduleFn      = system.CancelContestEndSchedule
	requestContestEndReplayFn       = system.RequestContestEndReplay
	resetContestEndStateByContestFn = pgsql.ResetContestEndExecutionByContest
	listAuthorsFn                   = pgsql.ListAuthors
	getAuthorByIDFn                 = pgsql.GetAuthorByAuthorID
	updateAuthorByIDFn              = pgsql.UpdateAuthorByID
	deleteAuthorByIDFn              = pgsql.DeleteAuthorByID

	getWorkByIDFn      = pgsql.GetWorkByID
	getWorksByTrackFn  = pgsql.GetWorksByTrackID
	getWorksByAuthorFn = pgsql.GetWorksByAuthorID
	queryWorksFn       = pgsql.QueryWorks
	deleteWorkByIDFn   = pgsql.DeleteWorkByID
	deleteUploadPermFn = redis.DeleteUploadFilePermission

	createActionLogFn = newAdminActionLog

	readDirFn = os.ReadDir
	removeFn  = os.Remove
)

var (
	errAuthorNotFound   = errors.New("author not found")
	errWorkNotFound     = errors.New("work not found")
	errWorkFileNotFound = errors.New("work file not found")
)

func loginSrc(admin model.Admin) (token.ResponseToken, error) {
	dbAdmin, err := findAdminByUsernameFn(admin.AdminName)
	if err != nil {
		log.Logger.Warn("Admin login query error: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}
	if !dbAdmin.IsActive {
		err = errors.New("admin account is disabled")
		log.Logger.Warn("Admin login error: " + err.Error())
		return token.ResponseToken{}, err
	}

	isSame := password.CheckPasswordHash(admin.Password, dbAdmin.Password)
	if !isSame {
		err = errors.New("login error")
		log.Logger.Warn("Admin login error: " + err.Error())
		return token.ResponseToken{}, err
	}

	tokens, err := genTokenAndRefreshTokenFn(int64(dbAdmin.AdminID), _const.RoleAdmin)
	if err != nil {
		log.Logger.Warn("Admin login generate token error: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	return tokens, nil
}

func refreshTokenSrc(adminID int64) (token.ResponseToken, error) {
	tokens, err := genTokenAndRefreshTokenFn(adminID, _const.RoleAdmin)
	if err != nil {
		log.Logger.Warn("Admin refresh token src error: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	return tokens, nil
}

func createContestSrc(adminID int, contest *model.Contest) error {
	err := createContestFn(contest)
	if err != nil {
		log.Logger.Warn("Create contest error: " + err.Error())

		return uerr.ExtractError(err)
	}

	registerContestEndScheduleFn(*contest)

	// 记录管理行为日志
	createActionLogFn(adminID, _const.Contests, _const.Create,
		genDetails([]string{"contest_name", "contest_id"}, []string{contest.ContestName, strconv.Itoa(contest.ContestID)}))

	return nil
}

func updateContestSrc(adminID int, contestID int, updatedContest *model.Contest) error {
	originalContest, err := getContestByIDFn(contestID)
	if err != nil {
		log.Logger.Warn("Get contest before update error: " + err.Error())
		return uerr.ExtractError(err)
	}
	if log.Logger != nil {
		log.Logger.Debug(
			"contest_update_received: contestID=" + strconv.Itoa(contestID) +
				" originalEnd=" + formatContestTimeForDebug(originalContest.ContestEndDate) +
				" updatedEnd=" + formatContestTimeForDebug(updatedContest.ContestEndDate) +
				" originalStart=" + formatContestTimeForDebug(originalContest.ContestStartDate) +
				" updatedStart=" + formatContestTimeForDebug(updatedContest.ContestStartDate),
		)
	}

	updatedContest.ContestID = contestID
	err = updateContestFn(contestID, updatedContest)
	if err != nil {
		log.Logger.Warn("Update contest error: " + err.Error())
		return uerr.ExtractError(err)
	}

	endDateChanged := contestEndDateChanged(originalContest, *updatedContest)
	if log.Logger != nil {
		log.Logger.Debug(
			"contest_update_enddate_check: contestID=" + strconv.Itoa(contestID) +
				" changed=" + strconv.FormatBool(endDateChanged) +
				" originalEnd=" + formatContestTimeForDebug(originalContest.ContestEndDate) +
				" updatedEnd=" + formatContestTimeForDebug(updatedContest.ContestEndDate),
		)
	}

	if endDateChanged {
		if log.Logger != nil {
			log.Logger.Debug("contest_update_reset_contest_end_state: contestID=" + strconv.Itoa(contestID))
		}
		if err = resetContestEndStateByContestFn(contestID); err != nil {
			log.Logger.Warn("Reset contest_end execution state error: " + err.Error())
			return uerr.ExtractError(err)
		}
	} else if log.Logger != nil {
		log.Logger.Debug("contest_update_skip_reset_contest_end_state: contestID=" + strconv.Itoa(contestID) + " reason=end-date-not-changed-or-zero")
	}

	if log.Logger != nil {
		log.Logger.Debug(
			"contest_update_register_schedule: contestID=" + strconv.Itoa(contestID) +
				" end=" + formatContestTimeForDebug(updatedContest.ContestEndDate),
		)
	}
	registerContestEndScheduleFn(*updatedContest)

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

func contestEndDateChanged(originalContest model.Contest, updatedContest model.Contest) bool {
	if originalContest.ContestEndDate.IsZero() || updatedContest.ContestEndDate.IsZero() {
		return false
	}

	return !originalContest.ContestEndDate.UTC().Equal(updatedContest.ContestEndDate.UTC())
}

func formatContestTimeForDebug(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
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

	cancelContestEndScheduleFn(contestID)

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

func replayContestEndSrc(adminID int, contestID int, trackID int) error {
	err := requestContestEndReplayFn(contestID, trackID)
	if err != nil {
		log.Logger.Warn("Replay contest_end error: " + err.Error())
		return uerr.ExtractError(err)
	}

	detailsKeys := []string{"contest_id"}
	detailsValues := []string{strconv.Itoa(contestID)}
	if trackID > 0 {
		detailsKeys = append(detailsKeys, "track_id")
		detailsValues = append(detailsValues, strconv.Itoa(trackID))
	}

	createActionLogFn(adminID, _const.Contests, _const.Update, genDetails(detailsKeys, detailsValues))

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

func listAuthorsSrc(authorName string, offset int, limit int) ([]model.Author, error) {
	authors, err := listAuthorsFn(authorName, offset, limit)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("List authors error: " + err.Error())
		return nil, parsedErr
	}

	return authors, nil
}

func getAuthorByIDSrc(authorID int) (model.Author, error) {
	author := model.Author{AuthorID: authorID}
	err := getAuthorByIDFn(&author)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Warn("Get author by id error: " + errAuthorNotFound.Error())
			return model.Author{}, errAuthorNotFound
		}
		log.Logger.Warn("Get author by id error: " + err.Error())
		return model.Author{}, parsedErr
	}

	return author, nil
}

func updateAuthorSrc(adminID int, authorID int, updatedAuthor *model.Author) (model.Author, error) {
	author, err := updateAuthorByIDFn(authorID, updatedAuthor)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Warn("Update author error: " + errAuthorNotFound.Error())
			return model.Author{}, errAuthorNotFound
		}
		log.Logger.Warn("Update author error: " + err.Error())
		return model.Author{}, parsedErr
	}

	createActionLogFn(adminID, _const.Authors, _const.Update,
		genDetails(
			[]string{"author_id", "author_name"},
			[]string{strconv.Itoa(author.AuthorID), author.AuthorName},
		),
	)

	return author, nil
}

func deleteAuthorSrc(adminID int, authorID int) error {
	author, err := deleteAuthorByIDFn(authorID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Warn("Delete author error: " + errAuthorNotFound.Error())
			return errAuthorNotFound
		}
		log.Logger.Warn("Delete author error: " + err.Error())
		return parsedErr
	}

	createActionLogFn(adminID, _const.Authors, _const.Delete,
		genDetails(
			[]string{"author_id", "author_name"},
			[]string{strconv.Itoa(author.AuthorID), author.AuthorName},
		),
	)

	return nil
}

func getWorkByIDSrc(workID int) (model.Work, error) {
	work, err := getWorkByIDFn(workID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			log.Logger.Warn("Get work by id error: " + errWorkNotFound.Error())
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

func queryWorksSrc(trackID *int, workStatus string, workTitle string, authorName string, offset int, limit int) ([]model.Work, error) {
	works, err := queryWorksFn(trackID, workStatus, workTitle, authorName, offset, limit)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		log.Logger.Warn("Query works error: " + err.Error())
		return nil, parsedErr
	}

	return works, nil
}

func getWorkFilePathSrc(workID int) (string, error) {
	work, err := getWorkByIDSrc(workID)
	if err != nil {
		log.Logger.Warn("Get work file path get work error: " + err.Error())
		return "", uerr.ExtractError(err)
	}

	filePath, err := resolveWorkFilePath(work)
	if err != nil {
		log.Logger.Warn("Get work file path resolve file error: " + err.Error())
		return "", uerr.ExtractError(err)
	}

	return filePath, nil
}

func resolveWorkFilePath(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Logger.Warn("Resolve work file path error: " + errWorkFileNotFound.Error())
			return "", errWorkFileNotFound
		}
		wrappedErr := uerr.NewError(err)
		log.Logger.Warn("Resolve work file path read dir error: " + wrappedErr.Error())
		return "", uerr.ExtractError(wrappedErr)
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
		log.Logger.Warn("Resolve work file path error: " + errWorkFileNotFound.Error())
		return "", errWorkFileNotFound
	}

	return filepath.Join(dstDir, selectedName), nil
}

func deleteWorkSrc(adminID, workID int) error {
	work, err := getWorkByIDSrc(workID)
	if err != nil {
		log.Logger.Warn("Delete work get work error: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = deleteWorkFiles(work)
	if err != nil {
		log.Logger.Warn("Delete work files error: " + err.Error())
		return uerr.ExtractError(err)
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
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	prefix := strconv.Itoa(work.WorkID) + "."

	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		wrappedErr := uerr.NewError(err)
		log.Logger.Warn("Delete work files read dir error: " + wrappedErr.Error())
		return uerr.ExtractError(wrappedErr)
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
			wrappedErr := uerr.NewError(rmErr)
			log.Logger.Warn("Delete work files remove error: " + wrappedErr.Error())
			return uerr.ExtractError(wrappedErr)
		}
	}

	return nil
}
