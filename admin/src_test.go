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
	origGetWorkByIDFn := getWorkByIDFn
	origGetWorksByTrackFn := getWorksByTrackFn
	origGetWorksByAuthorFn := getWorksByAuthorFn
	origDeleteWorkByIDFn := deleteWorkByIDFn
	origDeleteUploadPermFn := deleteUploadPermFn
	origCreateActionLogFn := createActionLogFn
	origReadDirFn := readDirFn
	origRemoveFn := removeFn

	log.Logger = testLogger{}

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
		getWorkByIDFn = origGetWorkByIDFn
		getWorksByTrackFn = origGetWorksByTrackFn
		getWorksByAuthorFn = origGetWorksByAuthorFn
		deleteWorkByIDFn = origDeleteWorkByIDFn
		deleteUploadPermFn = origDeleteUploadPermFn
		createActionLogFn = origCreateActionLogFn
		readDirFn = origReadDirFn
		removeFn = origRemoveFn
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
		return model.Admin{AdminID: 7, Password: hash}, nil
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
}

func TestContestSrcCacheFlow(t *testing.T) {
	backupSrcHooks(t)

	updated := model.Contest{ContestID: 1, ContestName: "c"}
	cacheCount := 0
	logCount := 0

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
