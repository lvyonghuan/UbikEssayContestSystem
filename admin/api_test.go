package admin

import (
	"bytes"
	"encoding/json"
	"errors"
	"main/conf"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/token"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

type apiTestLogger struct{}

func (apiTestLogger) Debug(string)  {}
func (apiTestLogger) Info(string)   {}
func (apiTestLogger) Warn(string)   {}
func (apiTestLogger) Error(error)   {}
func (apiTestLogger) Fatal(error)   {}
func (apiTestLogger) System(string) {}

func backupAPIHooks(t *testing.T) {
	origLogger := log.Logger
	origCheckTokenFn := checkTokenFn
	origCheckRefreshTokenFn := checkRefreshTokenFn
	origRunServerFn := runServerFn
	origLoginSrcFn := loginSrcFn
	origRefreshTokenSrcFn := refreshTokenSrcFn
	origCreateContestSrcFn := createContestSrcFn
	origUpdateContestSrcFn := updateContestSrcFn
	origDeleteContestSrcFn := deleteContestSrcFn
	origCreateTrackSrcFn := createTrackSrcFn
	origUpdateTrackSrcFn := updateTrackSrcFn
	origDeleteTrackSrcFn := deleteTrackSrcFn
	origGetWorkByIDSrcFn := getWorkByIDSrcFn
	origGetWorkFilePathSrcFn := getWorkFilePathSrcFn
	origGetWorksByTrackIDSrcFn := getWorksByTrackIDSrcFn
	origGetWorksByAuthorIDSrcFn := getWorksByAuthorIDSrcFn
	origDeleteWorkSrcFn := deleteWorkSrcFn

	log.Logger = apiTestLogger{}

	t.Cleanup(func() {
		log.Logger = origLogger
		checkTokenFn = origCheckTokenFn
		checkRefreshTokenFn = origCheckRefreshTokenFn
		runServerFn = origRunServerFn
		loginSrcFn = origLoginSrcFn
		refreshTokenSrcFn = origRefreshTokenSrcFn
		createContestSrcFn = origCreateContestSrcFn
		updateContestSrcFn = origUpdateContestSrcFn
		deleteContestSrcFn = origDeleteContestSrcFn
		createTrackSrcFn = origCreateTrackSrcFn
		updateTrackSrcFn = origUpdateTrackSrcFn
		deleteTrackSrcFn = origDeleteTrackSrcFn
		getWorkByIDSrcFn = origGetWorkByIDSrcFn
		getWorkFilePathSrcFn = origGetWorkFilePathSrcFn
		getWorksByTrackIDSrcFn = origGetWorksByTrackIDSrcFn
		getWorksByAuthorIDSrcFn = origGetWorksByAuthorIDSrcFn
		deleteWorkSrcFn = origDeleteWorkSrcFn
	})
}

func mockAuthAndSources(t *testing.T) string {
	backupAPIHooks(t)

	checkTokenFn = func(tokenStr string) (int64, string, error) {
		switch tokenStr {
		case "Bearer admin":
			return 1, _const.RoleAdmin, nil
		case "Bearer author":
			return 2, _const.RoleAuthor, nil
		default:
			return 0, "", errors.New("bad token")
		}
	}
	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) {
		if tokenStr == "Bearer refresh-admin" {
			return 1, _const.RoleAdmin, nil
		}
		return 0, "", errors.New("bad refresh token")
	}

	loginSrcFn = func(admin model.Admin) (token.ResponseToken, error) {
		return token.ResponseToken{Token: "token", RefreshToken: "refresh"}, nil
	}
	refreshTokenSrcFn = func(adminID int64) (token.ResponseToken, error) {
		return token.ResponseToken{Token: "new-token", RefreshToken: "new-refresh"}, nil
	}
	createContestSrcFn = func(adminID int, contest *model.Contest) error { return nil }
	updateContestSrcFn = func(adminID int, contestID int, contest *model.Contest) error { return nil }
	deleteContestSrcFn = func(adminID int, contestID int) error { return nil }
	createTrackSrcFn = func(adminID int, track *model.Track) error { return nil }
	updateTrackSrcFn = func(adminID int, trackID int, track *model.Track) error { return nil }
	deleteTrackSrcFn = func(adminID int, trackID int) error { return nil }
	getWorkByIDSrcFn = func(workID int) (model.Work, error) {
		return model.Work{WorkID: workID, WorkTitle: "w", TrackID: 1, AuthorID: 1}, nil
	}
	getWorksByTrackIDSrcFn = func(trackID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1}}, nil
	}
	getWorksByAuthorIDSrcFn = func(authorID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1}}, nil
	}
	deleteWorkSrcFn = func(adminID, workID int) error { return nil }

	tmpFile := filepath.Join(t.TempDir(), "work.docx")
	if err := os.WriteFile(tmpFile, []byte("docx"), 0o644); err != nil {
		t.Fatalf("write tmp file failed: %v", err)
	}
	getWorkFilePathSrcFn = func(workID int) (string, error) {
		return tmpFile, nil
	}

	return tmpFile
}

func reqCode(t *testing.T, body []byte) int {
	t.Helper()
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response failed: %v, body=%s", err, string(body))
	}
	codeFloat, ok := resp["code"].(float64)
	if !ok {
		t.Fatalf("response has no code: %v", resp)
	}
	return int(codeFloat)
}

func doRequest(router http.Handler, method, path string, body []byte, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestAdminRoutesSmokeSuccess(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	tests := []struct {
		method string
		path   string
		body   []byte
		auth   string
		file   bool
	}{
		{http.MethodPost, "/api/v1/admin/login", []byte(`{"adminName":"a","password":"b"}`), "", false},
		{http.MethodPost, "/api/v1/admin/refresh", nil, "Bearer refresh-admin", false},
		{http.MethodPost, "/api/v1/admin/contest", []byte(`{"contestName":"c"}`), "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/contest/1", []byte(`{"contestName":"c2"}`), "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/contest/1", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/track", []byte(`{"trackName":"t"}`), "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/track/1", []byte(`{"trackName":"t2"}`), "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/track/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works/1/file", nil, "Bearer admin", true},
		{http.MethodGet, "/api/v1/admin/works/track/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works/author/1", nil, "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/works/1", nil, "Bearer admin", false},
	}

	for _, tc := range tests {
		w := doRequest(router, tc.method, tc.path, tc.body, tc.auth)
		if w.Code != http.StatusOK {
			t.Fatalf("unexpected http status for %s %s: %d", tc.method, tc.path, w.Code)
		}
		if tc.file {
			if len(w.Body.Bytes()) == 0 {
				t.Fatalf("expected file content for %s %s", tc.method, tc.path)
			}
			continue
		}
		if got := reqCode(t, w.Body.Bytes()); got != 200 {
			t.Fatalf("unexpected business code for %s %s: %d body=%s", tc.method, tc.path, got, w.Body.String())
		}
	}
}

func TestInitRouterUsesRunServerFn(t *testing.T) {
	backupAPIHooks(t)

	called := false
	runServerFn = func(r *gin.Engine, port string) error {
		called = true
		if port != "19081" {
			t.Fatalf("unexpected admin port: %s", port)
		}
		if r == nil {
			t.Fatal("router should not be nil")
		}
		return nil
	}

	InitRouter(conf.APIConfig{AdminPort: "19081"})
	if !called {
		t.Fatal("runServerFn should be called by InitRouter")
	}
}

func TestAuthFailurePaths(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "")
	if got := reqCode(t, w.Body.Bytes()); got != 401 {
		t.Fatalf("expected 401 for missing token, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer author")
	if got := reqCode(t, w.Body.Bytes()); got != 403 {
		t.Fatalf("expected 403 for non-admin token, got %d", got)
	}

	w = doRequest(router, http.MethodPost, "/api/v1/admin/refresh", nil, "Bearer invalid")
	if got := reqCode(t, w.Body.Bytes()); got != 401 {
		t.Fatalf("expected 401 for invalid refresh token, got %d", got)
	}
}

func TestWorksErrorPaths(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/admin/works/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid work_id, got %d", got)
	}

	getWorkByIDSrcFn = func(workID int) (model.Work, error) {
		return model.Work{}, errWorkNotFound
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for work not found, got %d", got)
	}

	getWorkFilePathSrcFn = func(workID int) (string, error) {
		return "", errWorkFileNotFound
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1/file", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for work file not found, got %d", got)
	}

	deleteWorkSrcFn = func(adminID, workID int) error {
		return errWorkNotFound
	}
	w = doRequest(router, http.MethodDelete, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for delete work not found, got %d", got)
	}
}

func TestHandlerErrorBranches(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	loginSrcFn = func(admin model.Admin) (token.ResponseToken, error) {
		return token.ResponseToken{}, errors.New("login failed")
	}
	w := doRequest(router, http.MethodPost, "/api/v1/admin/login", []byte(`{"adminName":"a","password":"b"}`), "")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for login error, got %d", got)
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/login", []byte(`{"adminName":`), "")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for login bind error, got %d", got)
	}

	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) {
		return 2, _const.RoleAuthor, nil
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/refresh", nil, "Bearer refresh-admin")
	if got := reqCode(t, w.Body.Bytes()); got != 403 {
		t.Fatalf("expected 403 for non-admin refresh role, got %d", got)
	}
	refreshTokenSrcFn = func(adminID int64) (token.ResponseToken, error) {
		return token.ResponseToken{}, errors.New("refresh fail")
	}
	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) {
		return 1, _const.RoleAdmin, nil
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/refresh", nil, "Bearer refresh-admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for refresh src error, got %d", got)
	}

	createContestSrcFn = func(adminID int, contest *model.Contest) error { return errors.New("x") }
	w = doRequest(router, http.MethodPost, "/api/v1/admin/contest", []byte(`{"contestName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for create contest src error, got %d", got)
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/contest", []byte(`{"contestName":`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for create contest bind error, got %d", got)
	}

	w = doRequest(router, http.MethodPut, "/api/v1/admin/contest/bad", []byte(`{"contestName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid contest_id, got %d", got)
	}
	updateContestSrcFn = func(adminID int, contestID int, contest *model.Contest) error { return errors.New("x") }
	w = doRequest(router, http.MethodPut, "/api/v1/admin/contest/1", []byte(`{"contestName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for update contest src error, got %d", got)
	}
	w = doRequest(router, http.MethodPut, "/api/v1/admin/contest/1", []byte(`{"contestName":`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for update contest bind error, got %d", got)
	}

	w = doRequest(router, http.MethodDelete, "/api/v1/admin/contest/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid contest_id on delete, got %d", got)
	}
	deleteContestSrcFn = func(adminID int, contestID int) error { return errors.New("x") }
	w = doRequest(router, http.MethodDelete, "/api/v1/admin/contest/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for delete contest src error, got %d", got)
	}

	createTrackSrcFn = func(adminID int, track *model.Track) error { return errors.New("x") }
	w = doRequest(router, http.MethodPost, "/api/v1/admin/track", []byte(`{"trackName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for create track src error, got %d", got)
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/track", []byte(`{"trackName":`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for create track bind error, got %d", got)
	}

	w = doRequest(router, http.MethodPut, "/api/v1/admin/track/bad", []byte(`{"trackName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid track_id, got %d", got)
	}
	updateTrackSrcFn = func(adminID int, trackID int, track *model.Track) error { return errors.New("x") }
	w = doRequest(router, http.MethodPut, "/api/v1/admin/track/1", []byte(`{"trackName":"x"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for update track src error, got %d", got)
	}
	w = doRequest(router, http.MethodPut, "/api/v1/admin/track/1", []byte(`{"trackName":`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for update track bind error, got %d", got)
	}

	w = doRequest(router, http.MethodDelete, "/api/v1/admin/track/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid track_id on delete, got %d", got)
	}
	deleteTrackSrcFn = func(adminID int, trackID int) error { return errors.New("x") }
	w = doRequest(router, http.MethodDelete, "/api/v1/admin/track/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for delete track src error, got %d", got)
	}

	getWorksByTrackIDSrcFn = func(trackID int) ([]model.Work, error) { return nil, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/track/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get works by track src error, got %d", got)
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/track/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for bad track_id in works list, got %d", got)
	}

	getWorksByAuthorIDSrcFn = func(authorID int) ([]model.Work, error) { return nil, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/author/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get works by author src error, got %d", got)
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/author/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for bad author_id in works list, got %d", got)
	}

	getWorkByIDSrcFn = func(workID int) (model.Work, error) { return model.Work{}, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get work src error, got %d", got)
	}

	getWorkFilePathSrcFn = func(workID int) (string, error) { return "", errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1/file", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get work file src error, got %d", got)
	}

	deleteWorkSrcFn = func(adminID, workID int) error { return errors.New("x") }
	w = doRequest(router, http.MethodDelete, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for delete work src error, got %d", got)
	}
}
