package system

import (
	"bytes"
	"encoding/json"
	"errors"
	"main/conf"
	"main/model"
	"main/util/log"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type systemTestLogger struct{}

func (systemTestLogger) Debug(string)  {}
func (systemTestLogger) Info(string)   {}
func (systemTestLogger) Warn(string)   {}
func (systemTestLogger) Error(error)   {}
func (systemTestLogger) Fatal(error)   {}
func (systemTestLogger) System(string) {}

func backupSystemAPIHooks(t *testing.T) {
	origLogger := log.Logger
	origCheckTokenFn := checkTokenFn
	origRunServerFn := runServerFn
	origGetContestSrcFn := getContestSrcFn
	origGetContestByIDFn := getContestByIDFn
	origGetTracksSrcFn := getTracksSrcFn
	origGetTrackByIDSrcFn := getTrackByIDSrcFn

	log.Logger = systemTestLogger{}

	t.Cleanup(func() {
		log.Logger = origLogger
		checkTokenFn = origCheckTokenFn
		runServerFn = origRunServerFn
		getContestSrcFn = origGetContestSrcFn
		getContestByIDFn = origGetContestByIDFn
		getTracksSrcFn = origGetTracksSrcFn
		getTrackByIDSrcFn = origGetTrackByIDSrcFn
	})
}

func mockSystemAuthAndSources(t *testing.T) {
	backupSystemAPIHooks(t)

	checkTokenFn = func(tokenStr string) (int64, string, error) {
		if tokenStr == "Bearer ok" {
			return 1, "admin", nil
		}
		return 0, "", errors.New("bad token")
	}

	getContestSrcFn = func() ([]model.Contest, error) {
		return []model.Contest{{ContestID: 1, ContestName: "c1"}}, nil
	}
	getContestByIDFn = func(contestID int) (model.Contest, error) {
		return model.Contest{ContestID: contestID, ContestName: "c"}, nil
	}
	getTracksSrcFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 1, ContestID: contestID, TrackName: "t1"}}, nil
	}
	getTrackByIDSrcFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 1, TrackName: "t"}, nil
	}
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
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func TestGlobalInfoRoutesSmokeSuccess(t *testing.T) {
	mockSystemAuthAndSources(t)
	router := buildGlobalInfoRouter()

	tests := []struct {
		path string
	}{
		{path: "/api/v1/contests"},
		{path: "/api/v1/contests/1"},
		{path: "/api/v1/tracks/1"},
		{path: "/api/v1/tracks/detail/1"},
	}

	for _, tc := range tests {
		w := doRequest(router, http.MethodGet, tc.path, nil, "Bearer ok")
		if w.Code != http.StatusOK {
			t.Fatalf("unexpected http status for %s: %d", tc.path, w.Code)
		}
		if got := reqCode(t, w.Body.Bytes()); got != 200 {
			t.Fatalf("unexpected business code for %s: %d body=%s", tc.path, got, w.Body.String())
		}
	}
}

func TestInitGlobalInfoRouterUsesRunServerFn(t *testing.T) {
	backupSystemAPIHooks(t)

	called := false
	runServerFn = func(r *gin.Engine, port string) error {
		called = true
		if port != "19082" {
			t.Fatalf("unexpected global info port: %s", port)
		}
		if r == nil {
			t.Fatal("router should not be nil")
		}
		return nil
	}

	initGlobalInfoRouter(conf.APIConfig{GlobalInfoPort: "19082"})
	if !called {
		t.Fatal("runServerFn should be called by initGlobalInfoRouter")
	}
}

func TestGlobalInfoAuthFailures(t *testing.T) {
	mockSystemAuthAndSources(t)
	router := buildGlobalInfoRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/contests", nil, "")
	if got := reqCode(t, w.Body.Bytes()); got != 401 {
		t.Fatalf("expected 401 for missing token, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/contests", nil, "Bearer invalid")
	if got := reqCode(t, w.Body.Bytes()); got != 401 {
		t.Fatalf("expected 401 for invalid token, got %d", got)
	}
}

func TestGlobalInfoErrorBranches(t *testing.T) {
	mockSystemAuthAndSources(t)
	router := buildGlobalInfoRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/contests/bad", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid contest id, got %d", got)
	}

	getContestSrcFn = func() ([]model.Contest, error) { return nil, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/contests", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get contests error, got %d", got)
	}

	getContestByIDFn = func(contestID int) (model.Contest, error) { return model.Contest{}, errContestNotFound }
	w = doRequest(router, http.MethodGet, "/api/v1/contests/1", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for contest not found, got %d", got)
	}

	getContestByIDFn = func(contestID int) (model.Contest, error) { return model.Contest{}, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/contests/1", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get contest by id error, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/tracks/bad", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid contest id in tracks list, got %d", got)
	}

	getTracksSrcFn = func(contestID int) ([]model.Track, error) { return nil, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/tracks/1", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get tracks error, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/tracks/detail/bad", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid track id, got %d", got)
	}

	getTrackByIDSrcFn = func(trackID int) (model.Track, error) { return model.Track{}, errTrackNotFound }
	w = doRequest(router, http.MethodGet, "/api/v1/tracks/detail/1", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for track not found, got %d", got)
	}

	getTrackByIDSrcFn = func(trackID int) (model.Track, error) { return model.Track{}, errors.New("x") }
	w = doRequest(router, http.MethodGet, "/api/v1/tracks/detail/1", nil, "Bearer ok")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for get track by id error, got %d", got)
	}
}
