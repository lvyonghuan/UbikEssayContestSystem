package admin

import (
	"errors"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/password"
	"main/util/token"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

type testLogger struct{}

func (testLogger) Debug(string)  {}
func (testLogger) Info(string)   {}
func (testLogger) Warn(string)   {}
func (testLogger) Error(error)   {}
func (testLogger) Fatal(error)   {}
func (testLogger) System(string) {}

func backupSrcHooks(t *testing.T) {
	origLogger := log.Logger
	origFindAdminByUsernameFn := findAdminByUsernameFn
	origGenTokenAndRefreshTokenFn := genTokenAndRefreshTokenFn
	origCreateContestFn := createContestFn
	origUpdateContestFn := updateContestFn
	origDeleteContestFn := deleteContestFn
	origGetTracksByContestFn := getTracksByContestFn
	origGetContestByIDFn := getContestByIDFn
	origCreateTrackFn := createTrackFn
	origUpdateTrackFn := updateTrackFn
	origDeleteTrackFn := deleteTrackFn
	origCreateTrackCacheFn := createTrackCacheFn
	origDeleteTrackCacheFn := deleteTrackCacheFn
	origRegisterContestEndScheduleFn := registerContestEndScheduleFn
	origCancelContestEndScheduleFn := cancelContestEndScheduleFn
	origRequestContestEndReplayFn := requestContestEndReplayFn
	origResetContestEndStateByContestFn := resetContestEndStateByContestFn
	origListAuthorsFn := listAuthorsFn
	origGetAuthorByIDFn := getAuthorByIDFn
	origUpdateAuthorByIDFn := updateAuthorByIDFn
	origDeleteAuthorByIDFn := deleteAuthorByIDFn
	origGetWorkByIDFn := getWorkByIDFn
	origGetWorksByTrackFn := getWorksByTrackFn
	origGetWorksByAuthorFn := getWorksByAuthorFn
	origQueryWorksFn := queryWorksFn
	origDeleteWorkByIDFn := deleteWorkByIDFn
	origDeleteUploadPermFn := deleteUploadPermFn
	origCreateActionLogFn := createActionLogFn
	origReadDirFn := readDirFn
	origRemoveFn := removeFn
	origIsAdminActiveFn := isAdminActiveFn
	origIsAdminSuperFn := isAdminSuperFn
	origHasAdminPermissionFn := hasAdminPermissionFn
	origListPermissionNamesFn := listPermissionNamesFn
	origListSubAdminsFn := listSubAdminsFn
	origCreateSubAdminFn := createSubAdminFn
	origSetSubAdminPermsFn := setSubAdminPermsFn
	origDeleteSubAdminByIDFn := deleteSubAdminByIDFn
	origSetAdminActiveFn := setAdminActiveFn
	origHandoverSuperAdminFn := handoverSuperAdminFn
	origGetSystemEmailConfigFn := getSystemEmailConfigFn
	origSendSMTPMailFn := sendSMTPMailFn

	log.Logger = testLogger{}
	registerContestEndScheduleFn = func(contest model.Contest) {}
	cancelContestEndScheduleFn = func(contestID int) {}
	requestContestEndReplayFn = func(contestID int, trackID int) error { return nil }
	resetContestEndStateByContestFn = func(contestID int) error { return nil }

	t.Cleanup(func() {
		log.Logger = origLogger
		findAdminByUsernameFn = origFindAdminByUsernameFn
		genTokenAndRefreshTokenFn = origGenTokenAndRefreshTokenFn
		createContestFn = origCreateContestFn
		updateContestFn = origUpdateContestFn
		deleteContestFn = origDeleteContestFn
		getTracksByContestFn = origGetTracksByContestFn
		getContestByIDFn = origGetContestByIDFn
		createTrackFn = origCreateTrackFn
		updateTrackFn = origUpdateTrackFn
		deleteTrackFn = origDeleteTrackFn
		createTrackCacheFn = origCreateTrackCacheFn
		deleteTrackCacheFn = origDeleteTrackCacheFn
		registerContestEndScheduleFn = origRegisterContestEndScheduleFn
		cancelContestEndScheduleFn = origCancelContestEndScheduleFn
		requestContestEndReplayFn = origRequestContestEndReplayFn
		resetContestEndStateByContestFn = origResetContestEndStateByContestFn
		listAuthorsFn = origListAuthorsFn
		getAuthorByIDFn = origGetAuthorByIDFn
		updateAuthorByIDFn = origUpdateAuthorByIDFn
		deleteAuthorByIDFn = origDeleteAuthorByIDFn
		getWorkByIDFn = origGetWorkByIDFn
		getWorksByTrackFn = origGetWorksByTrackFn
		getWorksByAuthorFn = origGetWorksByAuthorFn
		queryWorksFn = origQueryWorksFn
		deleteWorkByIDFn = origDeleteWorkByIDFn
		deleteUploadPermFn = origDeleteUploadPermFn
		createActionLogFn = origCreateActionLogFn
		readDirFn = origReadDirFn
		removeFn = origRemoveFn
		isAdminActiveFn = origIsAdminActiveFn
		isAdminSuperFn = origIsAdminSuperFn
		hasAdminPermissionFn = origHasAdminPermissionFn
		listPermissionNamesFn = origListPermissionNamesFn
		listSubAdminsFn = origListSubAdminsFn
		createSubAdminFn = origCreateSubAdminFn
		setSubAdminPermsFn = origSetSubAdminPermsFn
		deleteSubAdminByIDFn = origDeleteSubAdminByIDFn
		setAdminActiveFn = origSetAdminActiveFn
		handoverSuperAdminFn = origHandoverSuperAdminFn
		getSystemEmailConfigFn = origGetSystemEmailConfigFn
		sendSMTPMailFn = origSendSMTPMailFn
	})
}

func TestLoginSrc(t *testing.T) {
	backupSrcHooks(t)

	hash, err := password.HashPassword("pass")
	if err != nil {
		t.Fatalf("hash password failed: %v", err)
	}

	findAdminByUsernameFn = func(username string) (model.Admin, error) {
		if username != "admin" {
			return model.Admin{}, errors.New("not found")
		}
		return model.Admin{AdminID: 7, Password: hash, IsActive: true}, nil
	}
	genTokenAndRefreshTokenFn = func(userID int64, role string) (token.ResponseToken, error) {
		if userID != 7 || role != _const.RoleAdmin {
			t.Fatalf("unexpected token args: %d, %s", userID, role)
		}
		return token.ResponseToken{Token: "token", RefreshToken: "refresh"}, nil
	}

	resp, err := loginSrc(model.Admin{AdminName: "admin", Password: "pass"})
	if err != nil {
		t.Fatalf("loginSrc should succeed: %v", err)
	}
	if resp.Token != "token" || resp.RefreshToken != "refresh" {
		t.Fatalf("unexpected token response: %+v", resp)
	}

	_, err = loginSrc(model.Admin{AdminName: "admin", Password: "wrong"})
	if err == nil {
		t.Fatal("loginSrc should fail on wrong password")
	}

	findAdminByUsernameFn = func(username string) (model.Admin, error) {
		return model.Admin{AdminID: 7, Password: hash, IsActive: false}, nil
	}
	_, err = loginSrc(model.Admin{AdminName: "admin", Password: "pass"})
	if err == nil {
		t.Fatal("loginSrc should fail when admin is disabled")
	}
}

func TestContestSrcCacheFlow(t *testing.T) {
	backupSrcHooks(t)

	updated := model.Contest{ContestID: 1, ContestName: "c"}
	cacheCount := 0
	logCount := 0
	getContestByIDFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestEndDate: time.Now().Add(24 * time.Hour)}, nil
	}

	updateContestFn = func(contestID int, contest *model.Contest) error {
		if contestID != 1 || contest.ContestName != "c" {
			t.Fatal("unexpected update contest input")
		}
		return nil
	}
	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 1}, {TrackID: 2}}, nil
	}
	createTrackCacheFn = func(track model.Track, contest model.Contest) error {
		cacheCount++
		if contest.ContestName != "c" {
			t.Fatal("contest data not passed to cache update")
		}
		return nil
	}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {
		logCount++
	}

	if err := updateContestSrc(8, 1, &updated); err != nil {
		t.Fatalf("updateContestSrc failed: %v", err)
	}
	if cacheCount != 2 {
		t.Fatalf("expected 2 cache updates, got %d", cacheCount)
	}
	if logCount != 1 {
		t.Fatalf("expected 1 action log, got %d", logCount)
	}

	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		return nil, errors.New("db error")
	}
	if err := updateContestSrc(8, 1, &updated); err == nil {
		t.Fatal("updateContestSrc should fail when track query fails")
	}
}

func TestUpdateContestSrcResetsContestEndStateWhenEndDateChanged(t *testing.T) {
	backupSrcHooks(t)

	originalEnd := time.Now().Add(-24 * time.Hour).UTC()
	updatedEnd := time.Now().Add(24 * time.Hour).UTC()
	updated := model.Contest{ContestID: 1, ContestName: "restart", ContestEndDate: updatedEnd}

	getContestByIDFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestName: "old", ContestEndDate: originalEnd}, nil
	}
	updateContestFn = func(contestID int, contest *model.Contest) error { return nil }
	getTracksByContestFn = func(contestID int) ([]model.Track, error) { return nil, nil }

	resetCalls := 0
	resetContestEndStateByContestFn = func(contestID int) error {
		resetCalls++
		if contestID != 1 {
			t.Fatalf("unexpected contestID for reset: %d", contestID)
		}
		return nil
	}

	scheduleCalls := 0
	registerContestEndScheduleFn = func(contest model.Contest) {
		scheduleCalls++
		if !contest.ContestEndDate.UTC().Equal(updatedEnd) {
			t.Fatalf("unexpected scheduled contest end date: %s", contest.ContestEndDate)
		}
	}

	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	if err := updateContestSrc(8, 1, &updated); err != nil {
		t.Fatalf("updateContestSrc failed: %v", err)
	}
	if resetCalls != 1 {
		t.Fatalf("expected reset called once, got %d", resetCalls)
	}
	if scheduleCalls != 1 {
		t.Fatalf("expected schedule registration once, got %d", scheduleCalls)
	}
}

func TestUpdateContestSrcSkipsResetWhenEndDateUnchanged(t *testing.T) {
	backupSrcHooks(t)

	sharedEnd := time.Now().Add(24 * time.Hour).UTC()
	updated := model.Contest{ContestID: 1, ContestName: "no-change", ContestEndDate: sharedEnd}

	getContestByIDFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestName: "old", ContestEndDate: sharedEnd}, nil
	}
	updateContestFn = func(contestID int, contest *model.Contest) error { return nil }
	getTracksByContestFn = func(contestID int) ([]model.Track, error) { return nil, nil }

	resetCalls := 0
	resetContestEndStateByContestFn = func(contestID int) error {
		resetCalls++
		return nil
	}
	registerContestEndScheduleFn = func(contest model.Contest) {}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	if err := updateContestSrc(8, 1, &updated); err != nil {
		t.Fatalf("updateContestSrc failed: %v", err)
	}
	if resetCalls != 0 {
		t.Fatalf("expected reset not called, got %d", resetCalls)
	}
}

func TestDeleteContestSrcCacheFlow(t *testing.T) {
	backupSrcHooks(t)

	deletedCacheIDs := make(map[int]bool)
	deleteContestFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestName: "x"}, nil
	}
	getTracksByContestFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 10}, {TrackID: 11}}, nil
	}
	deleteTrackCacheFn = func(trackID int) error {
		deletedCacheIDs[trackID] = true
		return nil
	}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	if err := deleteContestSrc(1, 99); err != nil {
		t.Fatalf("deleteContestSrc failed: %v", err)
	}
	if !deletedCacheIDs[10] || !deletedCacheIDs[11] {
		t.Fatalf("expected both track caches deleted: %+v", deletedCacheIDs)
	}
}

func TestReplayContestEndSrc(t *testing.T) {
	backupSrcHooks(t)

	replayCalls := 0
	requestContestEndReplayFn = func(contestID int, trackID int) error {
		replayCalls++
		if contestID != 7 || trackID != 0 {
			t.Fatalf("unexpected replay args: %d %d", contestID, trackID)
		}
		return nil
	}

	logged := false
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {
		logged = true
		if adminID != 3 || res != _const.Contests || act != _const.Update {
			t.Fatalf("unexpected action log args: %d %s %s", adminID, res, act)
		}
		if details["contest_id"] != "7" {
			t.Fatalf("unexpected details: %+v", details)
		}
	}

	if err := replayContestEndSrc(3, 7, 0); err != nil {
		t.Fatalf("replayContestEndSrc failed: %v", err)
	}
	if replayCalls != 1 {
		t.Fatalf("expected one replay call, got %d", replayCalls)
	}
	if !logged {
		t.Fatal("expected action log to be written")
	}
}

func TestCreateTrackSrcRollbackOnCacheError(t *testing.T) {
	backupSrcHooks(t)

	rollbackID := 0
	getContestByIDFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID}, nil
	}
	createTrackFn = func(track *model.Track) error {
		track.TrackID = 123
		return nil
	}
	createTrackCacheFn = func(track model.Track, contest model.Contest) error {
		return errors.New("cache fail")
	}
	deleteTrackFn = func(trackID int) (model.Track, error) {
		rollbackID = trackID
		return model.Track{TrackID: trackID}, nil
	}

	err := createTrackSrc(1, &model.Track{TrackName: "t", ContestID: 2})
	if err == nil {
		t.Fatal("createTrackSrc should fail when cache write fails")
	}
	if rollbackID != 123 {
		t.Fatalf("expected rollback on track 123, got %d", rollbackID)
	}
}

func TestTrackSrcUpdateDelete(t *testing.T) {
	backupSrcHooks(t)

	updateTrackFn = func(trackID int, updatedTrack *model.Track) error { return nil }
	deleteTrackFn = func(trackID int) (model.Track, error) { return model.Track{TrackID: trackID, TrackName: "t"}, nil }
	deleteTrackCacheFn = func(trackID int) error { return nil }
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {}

	if err := updateTrackSrc(1, 1, &model.Track{TrackName: "x"}); err != nil {
		t.Fatalf("updateTrackSrc failed: %v", err)
	}
	if err := deleteTrackSrc(1, 1); err != nil {
		t.Fatalf("deleteTrackSrc failed: %v", err)
	}
}

func TestAuthorSrcPaths(t *testing.T) {
	backupSrcHooks(t)

	listAuthorsFn = func(authorName string, offset int, limit int) ([]model.Author, error) {
		if authorName != "alpha" || offset != 1 || limit != 10 {
			t.Fatalf("unexpected list authors args: %s %d %d", authorName, offset, limit)
		}
		return []model.Author{{AuthorID: 1, AuthorName: "alpha"}}, nil
	}
	authors, err := listAuthorsSrc("alpha", 1, 10)
	if err != nil || len(authors) != 1 {
		t.Fatalf("listAuthorsSrc failed: %v %+v", err, authors)
	}

	getAuthorByIDFn = func(author *model.Author) error {
		author.AuthorName = "author_1"
		author.AuthorEmail = "a1@example.com"
		return nil
	}
	author, err := getAuthorByIDSrc(1)
	if err != nil || author.AuthorName != "author_1" {
		t.Fatalf("getAuthorByIDSrc failed: %v %+v", err, author)
	}

	getAuthorByIDFn = func(author *model.Author) error {
		return uerr.NewError(gorm.ErrRecordNotFound)
	}
	_, err = getAuthorByIDSrc(2)
	if !errors.Is(err, errAuthorNotFound) {
		t.Fatalf("expected errAuthorNotFound, got %v", err)
	}

	logged := false
	updateAuthorByIDFn = func(authorID int, updated *model.Author) (model.Author, error) {
		return model.Author{AuthorID: authorID, AuthorName: "updated"}, nil
	}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {
		logged = true
	}
	updated, err := updateAuthorSrc(7, 3, &model.Author{AuthorName: "updated"})
	if err != nil || updated.AuthorID != 3 || !logged {
		t.Fatalf("updateAuthorSrc failed: err=%v author=%+v logged=%v", err, updated, logged)
	}

	updateAuthorByIDFn = func(authorID int, updated *model.Author) (model.Author, error) {
		return model.Author{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	_, err = updateAuthorSrc(7, 4, &model.Author{AuthorName: "x"})
	if !errors.Is(err, errAuthorNotFound) {
		t.Fatalf("expected errAuthorNotFound on update, got %v", err)
	}

	deleteAuthorByIDFn = func(authorID int) (model.Author, error) {
		return model.Author{AuthorID: authorID, AuthorName: "del"}, nil
	}
	logged = false
	err = deleteAuthorSrc(7, 5)
	if err != nil || !logged {
		t.Fatalf("deleteAuthorSrc failed: err=%v logged=%v", err, logged)
	}

	deleteAuthorByIDFn = func(authorID int) (model.Author, error) {
		return model.Author{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	err = deleteAuthorSrc(7, 6)
	if !errors.Is(err, errAuthorNotFound) {
		t.Fatalf("expected errAuthorNotFound on delete, got %v", err)
	}
}

func TestWorksSrcPaths(t *testing.T) {
	backupSrcHooks(t)

	getWorkByIDFn = func(workID int) (model.Work, error) {
		return model.Work{}, uerr.NewError(gorm.ErrRecordNotFound)
	}
	_, err := getWorkByIDSrc(1)
	if !errors.Is(err, errWorkNotFound) {
		t.Fatalf("expected errWorkNotFound, got %v", err)
	}

	getWorksByTrackFn = func(trackID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1}}, nil
	}
	works, err := getWorksByTrackIDSrc(1)
	if err != nil || len(works) != 1 {
		t.Fatalf("getWorksByTrackIDSrc failed: %v, %+v", err, works)
	}

	getWorksByAuthorFn = func(authorID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 2}}, nil
	}
	works, err = getWorksByAuthorIDSrc(1)
	if err != nil || len(works) != 1 {
		t.Fatalf("getWorksByAuthorIDSrc failed: %v, %+v", err, works)
	}

	queryWorksFn = func(trackID *int, workStatus string, workTitle string, authorName string, offset int, limit int) ([]model.Work, error) {
		if trackID == nil || *trackID != 1 {
			t.Fatalf("unexpected trackID input: %+v", trackID)
		}
		if workStatus != "reviewing" {
			t.Fatalf("unexpected workStatus input: %s", workStatus)
		}
		if workTitle != "w" || authorName != "a" || offset != 2 || limit != 10 {
			t.Fatalf("unexpected query args: %s %s %d %d", workTitle, authorName, offset, limit)
		}
		return []model.Work{{WorkID: 3}}, nil
	}
	works, err = queryWorksSrc(func() *int { v := 1; return &v }(), "reviewing", "w", "a", 2, 10)
	if err != nil || len(works) != 1 || works[0].WorkID != 3 {
		t.Fatalf("queryWorksSrc failed: %v, %+v", err, works)
	}
}

func TestResolveWorkFilePathPreferDocx(t *testing.T) {
	backupSrcHooks(t)

	tmpDir := t.TempDir()
	writeFile := func(name string, mod time.Time) {
		full := filepath.Join(tmpDir, name)
		if err := os.WriteFile(full, []byte(name), 0o644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
		if err := os.Chtimes(full, mod, mod); err != nil {
			t.Fatalf("chtimes failed: %v", err)
		}
	}

	now := time.Now()
	writeFile("1.doc", now.Add(2*time.Hour))
	writeFile("1.docx", now)
	writeFile("1_v2.docx", now.Add(3*time.Hour))
	writeFile("2.docx", now)

	readDirFn = func(string) ([]os.DirEntry, error) {
		return os.ReadDir(tmpDir)
	}

	path, err := resolveWorkFilePath(model.Work{WorkID: 1, TrackID: 100, AuthorID: 200})
	if err != nil {
		t.Fatalf("resolveWorkFilePath failed: %v", err)
	}
	if filepath.Base(path) != "1.docx" {
		t.Fatalf("expected 1.docx, got %s", filepath.Base(path))
	}

	_, err = resolveWorkFilePath(model.Work{WorkID: 9, TrackID: 100, AuthorID: 200})
	if !errors.Is(err, errWorkFileNotFound) {
		t.Fatalf("expected errWorkFileNotFound, got %v", err)
	}
}

func TestDeleteWorkFilesAndDeleteWorkSrc(t *testing.T) {
	backupSrcHooks(t)

	tmpDir := t.TempDir()
	for _, name := range []string{"1.docx", "1.doc", "2.docx"} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(name), 0o644); err != nil {
			t.Fatalf("write file failed: %v", err)
		}
	}

	removed := make(map[string]bool)
	readDirFn = func(string) ([]os.DirEntry, error) {
		return os.ReadDir(tmpDir)
	}
	removeFn = func(name string) error {
		removed[filepath.Base(name)] = true
		return nil
	}

	err := deleteWorkFiles(model.Work{WorkID: 1, TrackID: 1, AuthorID: 1})
	if err != nil {
		t.Fatalf("deleteWorkFiles failed: %v", err)
	}
	if !removed["1.docx"] || !removed["1.doc"] || removed["2.docx"] {
		t.Fatalf("unexpected removed files: %+v", removed)
	}

	deleted := false
	logged := false
	getWorkByIDFn = func(workID int) (model.Work, error) {
		return model.Work{WorkID: workID, WorkTitle: "w", TrackID: 1, AuthorID: 1}, nil
	}
	readDirFn = func(string) ([]os.DirEntry, error) {
		return nil, os.ErrNotExist
	}
	deleteWorkByIDFn = func(workID int) error {
		deleted = true
		return nil
	}
	deleteUploadPermFn = func(workID int) error {
		return errors.New("cache delete error")
	}
	createActionLogFn = func(adminID int, res, act string, details map[string]interface{}) {
		logged = true
		if details["work_id"] != strconv.Itoa(3) {
			t.Fatal("unexpected action log details")
		}
	}

	err = deleteWorkSrc(9, 3)
	if err != nil {
		t.Fatalf("deleteWorkSrc failed: %v", err)
	}
	if !deleted || !logged {
		t.Fatalf("deleteWorkSrc should delete and log, deleted=%v logged=%v", deleted, logged)
	}
}
