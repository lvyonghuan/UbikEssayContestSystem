package submission

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"main/conf"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/document"
	"main/util/log"
	"main/util/scriptflow"
	"main/util/token"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

type submissionTestLogger struct{}

func (submissionTestLogger) Debug(string)  {}
func (submissionTestLogger) Info(string)   {}
func (submissionTestLogger) Warn(string)   {}
func (submissionTestLogger) Error(error)   {}
func (submissionTestLogger) Fatal(error)   {}
func (submissionTestLogger) System(string) {}

type fakeConverter struct{}

const defaultFileHashForTests = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func (fakeConverter) ConvertDocToDocx(_ context.Context, srcDocPath string, dstDocxPath string) error {
	input, err := os.ReadFile(srcDocPath)
	if err != nil {
		return err
	}
	return os.WriteFile(dstDocxPath, input, 0o644)
}

func backupSubmissionHooks(t *testing.T) {
	t.Helper()

	origLogger := log.Logger
	origCheckTokenFn := checkTokenFn
	origRunServerFn := runServerFn
	origCheckRefreshTokenFn := checkRefreshTokenFn
	origRegisterAuthorSrcFn := registerAuthorSrcFn
	origAuthorLoginSrcFn := authorLoginSrcFn
	origRefreshTokenSrcFn := refreshTokenSrcFn
	origUpdateAuthorSrcFn := updateAuthorSrcFn
	origSubmissionWorkSrcFn := submissionWorkSrcFn
	origFindSubmissionsByAuthorIDFn := findSubmissionsByAuthorIDFn
	origUpdateSubmissionSrcFn := updateSubmissionSrcFn
	origDeleteSubmissionSrcFn := deleteSubmissionSrcFn
	origGetUploadFilePermissionFn := getUploadFilePermissionFn
	origRunTrackHookFn := runTrackHookFn
	origPatchWorkInfosFn := patchWorkInfosFn
	origUpdateWorkStatusFn := updateWorkStatusFn
	origGetSubmissionByWorkIDFn := getSubmissionByWorkIDFn
	origResolveSubmissionFilePathFn := resolveSubmissionFilePathFn
	origComputeFileSHA256Fn := computeFileSHA256Fn
	origNewDocumentConverterFn := newDocumentConverterFn

	origGetAuthorByAuthorNameFn := getAuthorByAuthorNameFn
	origCreateAuthorFn := createAuthorFn
	origGetAuthorByAuthorIDFn := getAuthorByAuthorIDFn
	origUpdateAuthorFn := updateAuthorFn
	origSubmissionWorkFn := submissionWorkFn
	origUpdateWorkFn := updateWorkFn
	origDeleteWorkFn := deleteWorkFn
	origFindWorksByAuthorIDFn := findWorksByAuthorIDFn
	origCountWorksByAuthorAndTrackFn := countWorksByAuthorAndTrackFn
	origCountWorksByAuthorAndContestFn := countWorksByAuthorAndContestFn
	origGetTrackByIDFn := getTrackByIDFn
	origSetUploadFilePermissionFn := setUploadFilePermissionFn
	origGetStartAndEndDateFn := getStartAndEndDateFn
	origResolveFlowForExecutionFn := resolveFlowForExecutionFn
	origExecuteScriptChainFn := executeScriptChainFn
	origReadDirFn := readDirFn
	origRemoveFn := removeFn

	log.Logger = submissionTestLogger{}

	t.Cleanup(func() {
		log.Logger = origLogger
		checkTokenFn = origCheckTokenFn
		runServerFn = origRunServerFn
		checkRefreshTokenFn = origCheckRefreshTokenFn
		registerAuthorSrcFn = origRegisterAuthorSrcFn
		authorLoginSrcFn = origAuthorLoginSrcFn
		refreshTokenSrcFn = origRefreshTokenSrcFn
		updateAuthorSrcFn = origUpdateAuthorSrcFn
		submissionWorkSrcFn = origSubmissionWorkSrcFn
		findSubmissionsByAuthorIDFn = origFindSubmissionsByAuthorIDFn
		updateSubmissionSrcFn = origUpdateSubmissionSrcFn
		deleteSubmissionSrcFn = origDeleteSubmissionSrcFn
		getUploadFilePermissionFn = origGetUploadFilePermissionFn
		runTrackHookFn = origRunTrackHookFn
		patchWorkInfosFn = origPatchWorkInfosFn
		updateWorkStatusFn = origUpdateWorkStatusFn
		getSubmissionByWorkIDFn = origGetSubmissionByWorkIDFn
		resolveSubmissionFilePathFn = origResolveSubmissionFilePathFn
		computeFileSHA256Fn = origComputeFileSHA256Fn
		newDocumentConverterFn = origNewDocumentConverterFn

		getAuthorByAuthorNameFn = origGetAuthorByAuthorNameFn
		createAuthorFn = origCreateAuthorFn
		getAuthorByAuthorIDFn = origGetAuthorByAuthorIDFn
		updateAuthorFn = origUpdateAuthorFn
		submissionWorkFn = origSubmissionWorkFn
		updateWorkFn = origUpdateWorkFn
		deleteWorkFn = origDeleteWorkFn
		findWorksByAuthorIDFn = origFindWorksByAuthorIDFn
		countWorksByAuthorAndTrackFn = origCountWorksByAuthorAndTrackFn
		countWorksByAuthorAndContestFn = origCountWorksByAuthorAndContestFn
		getTrackByIDFn = origGetTrackByIDFn
		setUploadFilePermissionFn = origSetUploadFilePermissionFn
		getStartAndEndDateFn = origGetStartAndEndDateFn
		resolveFlowForExecutionFn = origResolveFlowForExecutionFn
		executeScriptChainFn = origExecuteScriptChainFn
		readDirFn = origReadDirFn
		removeFn = origRemoveFn
	})
}

func decodeRespCode(t *testing.T, body []byte) int {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal failed: %v body=%s", err, string(body))
	}
	code, ok := resp["code"].(float64)
	if !ok {
		t.Fatalf("no code in response: %+v", resp)
	}
	return int(code)
}

func doJSONRequest(router http.Handler, method string, path string, auth string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doMultipartRequest(router http.Handler, path string, auth string, fields map[string]string, fileField string, fileName string, fileContent []byte) *httptest.ResponseRecorder {
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	if _, hasFileHash := fields["file_hash"]; !hasFileHash {
		sum := sha256.Sum256(fileContent)
		_ = writer.WriteField("file_hash", hex.EncodeToString(sum[:]))
	}
	part, _ := writer.CreateFormFile(fileField, fileName)
	_, _ = io.Copy(part, bytes.NewReader(fileContent))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, path, &payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func doMultipartWithoutFile(router http.Handler, path string, auth string, fields map[string]string) *httptest.ResponseRecorder {
	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	for k, v := range fields {
		_ = writer.WriteField(k, v)
	}
	if _, hasFileHash := fields["file_hash"]; !hasFileHash {
		_ = writer.WriteField("file_hash", defaultFileHashForTests)
	}
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, path, &payload)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func setupSubmissionRouteMocks(t *testing.T) {
	backupSubmissionHooks(t)

	checkTokenFn = func(tokenStr string) (int64, string, error) {
		switch tokenStr {
		case "Bearer author":
			return 1, _const.RoleAuthor, nil
		case "Bearer admin":
			return 2, _const.RoleAdmin, nil
		default:
			return 0, "", errors.New("bad token")
		}
	}
	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) {
		if tokenStr == "Bearer refresh-author" {
			return 1, _const.RoleAuthor, nil
		}
		return 0, "", errors.New("bad refresh token")
	}

	registerAuthorSrcFn = func(author *model.Author) error { return nil }
	authorLoginSrcFn = func(author *model.Author) (token.ResponseToken, error) {
		return token.ResponseToken{Token: "token", RefreshToken: "refresh"}, nil
	}
	refreshTokenSrcFn = func(authorID int64) (token.ResponseToken, error) {
		return token.ResponseToken{Token: "new-token", RefreshToken: "new-refresh"}, nil
	}
	updateAuthorSrcFn = func(author *model.Author) error { return nil }
	submissionWorkSrcFn = func(work *model.Work) error { return nil }
	findSubmissionsByAuthorIDFn = func(authorID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1, AuthorID: authorID, TrackID: 2, WorkTitle: "demo"}}, nil
	}
	updateSubmissionSrcFn = func(work *model.Work) error { return nil }
	deleteSubmissionSrcFn = func(work *model.Work) error { return nil }
	getTrackByIDFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 100}, nil
	}
	countWorksByAuthorAndContestFn = func(authorID int, contestID int) (int64, error) {
		return 0, nil
	}

	getUploadFilePermissionFn = func(workID int) (int, int, error) { return 1, 2, nil }
	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	patchWorkInfosFn = func(workID int, patch map[string]any) error { return nil }
	updateWorkStatusFn = func(workID int, status string) error { return nil }
	getSubmissionByWorkIDFn = func(work *model.Work) error {
		switch work.WorkID {
		case 404:
			return errors.New("record not found")
		case 500:
			return errors.New("db down")
		case 403:
			work.AuthorID = 2
			work.TrackID = 2
			work.WorkID = 403
			return nil
		default:
			work.AuthorID = 1
			work.TrackID = 2
			if work.WorkID == 0 {
				work.WorkID = 10
			}
			return nil
		}
	}
	downloadDir := t.TempDir()
	resolveSubmissionFilePathFn = func(work model.Work) (string, error) {
		switch work.WorkID {
		case 999:
			return "", os.ErrNotExist
		case 998:
			return "", errors.New("resolve fail")
		}
		filePath := filepath.Join(downloadDir, strconv.Itoa(work.WorkID)+".docx")
		if err := os.WriteFile(filePath, []byte("download-content"), 0o644); err != nil {
			return "", err
		}
		return filePath, nil
	}
	computeFileSHA256Fn = computeFileSHA256
	newDocumentConverterFn = func() document.Converter { return fakeConverter{} }
}

func TestSubmissionRoutesSmokeSuccess(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	router := buildSubmissionRouter()

	cases := []struct {
		method string
		path   string
		auth   string
		body   []byte
		isFile bool
	}{
		{http.MethodPost, "/api/v1/author/register", "", []byte(`{"authorName":"u","password":"p","authorEmail":"u@example.com"}`), false},
		{http.MethodPost, "/api/v1/author/login", "", []byte(`{"authorName":"u","password":"p"}`), false},
		{http.MethodGet, "/api/v1/author/refresh", "Bearer refresh-author", nil, false},
		{http.MethodPut, "/api/v1/author", "Bearer author", []byte(`{"authorID":1,"authorName":"u2"}`), false},
		{http.MethodPost, "/api/v1/author/submission", "Bearer author", []byte(`{"authorID":1,"trackID":2,"workTitle":"w"}`), false},
		{http.MethodPut, "/api/v1/author/submission", "Bearer author", []byte(`{"workID":1,"authorID":1,"trackID":2,"workTitle":"w2"}`), false},
		{http.MethodDelete, "/api/v1/author/submission", "Bearer author", []byte(`{"workID":1,"authorID":1,"trackID":2,"workTitle":"w2"}`), false},
		{http.MethodGet, "/api/v1/author/submission", "Bearer author", nil, false},
	}

	for _, tc := range cases {
		w := doJSONRequest(router, tc.method, tc.path, tc.auth, tc.body)
		if w.Code != http.StatusOK {
			t.Fatalf("unexpected http status: %s %s => %d", tc.method, tc.path, w.Code)
		}
		if code := decodeRespCode(t, w.Body.Bytes()); code != 200 {
			t.Fatalf("unexpected business code: %s %s => %d body=%s", tc.method, tc.path, code, w.Body.String())
		}
	}

	fileResp := doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "10"},
		"article_file",
		"paper.docx",
		[]byte("docx-content"),
	)
	if code := decodeRespCode(t, fileResp.Body.Bytes()); code != 200 {
		t.Fatalf("unexpected file upload code: %d body=%s", code, fileResp.Body.String())
	}

	downloadResp := doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/10", "Bearer author", nil)
	if downloadResp.Code != http.StatusOK {
		t.Fatalf("unexpected download status code: %d", downloadResp.Code)
	}
	if len(downloadResp.Body.Bytes()) == 0 {
		t.Fatal("download should return file body")
	}
	if downloadResp.Header().Get("X-File-SHA256") == "" {
		t.Fatal("download should include X-File-SHA256 header")
	}

	savedPath := filepath.Join(tmp, "files", "submissions", "2", "1", "10.docx")
	if _, err := os.Stat(savedPath); err != nil {
		t.Fatalf("expected saved docx file: %v", err)
	}
}

func TestSubmissionWorkResponseContainsWorkStatus(t *testing.T) {
	setupSubmissionRouteMocks(t)
	router := buildSubmissionRouter()

	submissionWorkSrcFn = func(work *model.Work) error {
		work.WorkID = 77
		work.WorkStatus = "submission_success"
		return nil
	}

	w := doJSONRequest(
		router,
		http.MethodPost,
		"/api/v1/author/submission",
		"Bearer author",
		[]byte(`{"authorID":1,"trackID":2,"workTitle":"w"}`),
	)
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected http status: %d", w.Code)
	}

	var resp struct {
		Code int `json:"code"`
		Msg  struct {
			WorkStatus string `json:"workStatus"`
		} `json:"msg"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v body=%s", err, w.Body.String())
	}
	if resp.Code != 200 {
		t.Fatalf("unexpected business code: %d", resp.Code)
	}
	if resp.Msg.WorkStatus != "submission_success" {
		t.Fatalf("expected workStatus in response, got %q", resp.Msg.WorkStatus)
	}
}

func TestSaveSubmissionFileDocConversionAndPatch(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	patchCalls := 0
	patchWorkInfosFn = func(workID int, patch map[string]any) error {
		patchCalls++
		if workID != 11 {
			t.Fatalf("unexpected workID for patch: %d", workID)
		}
		return nil
	}
	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePre {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"pre": true}}, nil
		}
		if eventKey == scriptflow.EventFilePost {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"post": true}}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	newDocumentConverterFn = func() document.Converter { return fakeConverter{} }

	router := buildSubmissionRouter()
	w := doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "11"},
		"article_file",
		"paper.doc",
		[]byte("doc-content"),
	)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 200 {
		t.Fatalf("upload doc should success, got code=%d body=%s", code, w.Body.String())
	}
	if patchCalls != 3 {
		t.Fatalf("expected patch called three times(pre+integrity+post), got %d", patchCalls)
	}

	docxPath := filepath.Join(tmp, "files", "submissions", "2", "1", "11.docx")
	if _, err := os.Stat(docxPath); err != nil {
		t.Fatalf("expected converted docx exists: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "files", "submissions", "2", "1", "11.doc")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("temp doc should be removed, stat err=%v", err)
	}
}

func TestSaveSubmissionFilePatchCanUpdateWorkStatus(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	statusUpdates := make([]string, 0, 2)
	updateWorkStatusFn = func(workID int, status string) error {
		if workID != 12 {
			t.Fatalf("unexpected workID for update status: %d", workID)
		}
		statusUpdates = append(statusUpdates, status)
		return nil
	}
	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePre {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"workStatus": "reviewing"}}, nil
		}
		if eventKey == scriptflow.EventFilePost {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"work_status": "approved"}}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	router := buildSubmissionRouter()
	w := doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "12"},
		"article_file",
		"paper.docx",
		[]byte("docx-content"),
	)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 200 {
		t.Fatalf("upload docx should success, got code=%d body=%s", code, w.Body.String())
	}

	if len(statusUpdates) != 2 {
		t.Fatalf("expected 2 status updates from hooks, got %d", len(statusUpdates))
	}
	if statusUpdates[0] != "reviewing" || statusUpdates[1] != "approved" {
		t.Fatalf("unexpected status updates: %+v", statusUpdates)
	}
}

func TestGetSubmissionFileErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)
	router := buildSubmissionRouter()

	w := doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/bad", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400 for bad submission_id, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/404", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 404 {
		t.Fatalf("expected 404 for missing submission, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/403", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403 for ownership violation, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/999", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 404 {
		t.Fatalf("expected 404 for missing file, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/998", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500 for resolve file error, got %d", code)
	}

	computeFileSHA256Fn = func(filePath string) (string, int64, error) {
		return "", 0, errors.New("hash fail")
	}
	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission/file/10", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500 for hash compute error, got %d", code)
	}
}

func TestInitRouterUsesRunServerFn(t *testing.T) {
	setupSubmissionRouteMocks(t)

	called := false
	runServerFn = func(r *gin.Engine, port string) error {
		called = true
		if port != "19082" {
			t.Fatalf("unexpected port: %s", port)
		}
		if r == nil {
			t.Fatal("router should not be nil")
		}
		return nil
	}

	InitRouter(conf.APIConfig{SubmissionsPort: "19082"})
	if !called {
		t.Fatal("runServerFn should be called")
	}
}

func TestSubmissionAuthFailurePaths(t *testing.T) {
	setupSubmissionRouteMocks(t)
	router := buildSubmissionRouter()

	w := doJSONRequest(router, http.MethodGet, "/api/v1/author/submission", "", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 401 {
		t.Fatalf("expected 401, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission", "Bearer admin", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}

	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/refresh", "Bearer bad", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 401 {
		t.Fatalf("expected 401 for bad refresh token, got %d", code)
	}
}

func TestSubmissionFileBadSuffix(t *testing.T) {
	setupSubmissionRouteMocks(t)
	router := buildSubmissionRouter()

	w := doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "12"},
		"article_file",
		"paper.txt",
		[]byte("x"),
	)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400 for bad suffix, got %d", code)
	}
}

func TestRunTrackHookNoMount(t *testing.T) {
	setupSubmissionRouteMocks(t)
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}

	result, err := runTrackHook(scriptflow.ScopeSubmission, scriptflow.EventSubmissionPre, 1, map[string]any{"k": "v"})
	if err != nil {
		t.Fatalf("runTrackHook should ignore missing mount: %v", err)
	}
	if !result.Allowed {
		t.Fatal("result should be allowed when no mount")
	}
}

func TestSubmissionHandlersErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)
	router := buildSubmissionRouter()

	registerAuthorSrcFn = func(author *model.Author) error { return errors.New("register failed") }
	w := doJSONRequest(router, http.MethodPost, "/api/v1/author/register", "", []byte(`{"authorName":"u","password":"p","authorEmail":"u@example.com"}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	w = doJSONRequest(router, http.MethodPost, "/api/v1/author/register", "", []byte(`{`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400, got %d", code)
	}

	w = doJSONRequest(router, http.MethodPost, "/api/v1/author/register", "", []byte(`{"authorName":"u","password":"p"}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400 when authorEmail is missing, got %d", code)
	}

	authorLoginSrcFn = func(author *model.Author) (token.ResponseToken, error) {
		return token.ResponseToken{}, errors.New("login failed")
	}
	w = doJSONRequest(router, http.MethodPost, "/api/v1/author/login", "", []byte(`{"authorName":"u","password":"x"}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) { return 1, _const.RoleAdmin, nil }
	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/refresh", "Bearer refresh-author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}

	checkRefreshTokenFn = func(tokenStr string) (int64, string, error) { return 1, _const.RoleAuthor, nil }
	refreshTokenSrcFn = func(authorID int64) (token.ResponseToken, error) {
		return token.ResponseToken{}, errors.New("refresh failed")
	}
	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/refresh", "Bearer refresh-author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	updateAuthorSrcFn = func(author *model.Author) error { return errors.New("update failed") }
	w = doJSONRequest(router, http.MethodPut, "/api/v1/author", "Bearer author", []byte(`{"authorID":2}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}
	w = doJSONRequest(router, http.MethodPut, "/api/v1/author", "Bearer author", []byte(`{"authorID":1}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	submissionWorkSrcFn = func(work *model.Work) error { return errors.New("submission failed") }
	w = doJSONRequest(router, http.MethodPost, "/api/v1/author/submission", "Bearer author", []byte(`{"authorID":2,"trackID":1}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}
	w = doJSONRequest(router, http.MethodPost, "/api/v1/author/submission", "Bearer author", []byte(`{"authorID":1,"trackID":1}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	updateSubmissionSrcFn = func(work *model.Work) error { return errors.New("update failed") }
	w = doJSONRequest(router, http.MethodPut, "/api/v1/author/submission", "Bearer author", []byte(`{"authorID":1,"trackID":1}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	deleteSubmissionSrcFn = func(work *model.Work) error { return errors.New("delete failed") }
	w = doJSONRequest(router, http.MethodDelete, "/api/v1/author/submission", "Bearer author", []byte(`{"authorID":1,"trackID":1}`))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	findSubmissionsByAuthorIDFn = func(authorID int) ([]model.Work, error) { return nil, errors.New("query failed") }
	w = doJSONRequest(router, http.MethodGet, "/api/v1/author/submission", "Bearer author", nil)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}
}

func TestSaveSubmissionFileErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	router := buildSubmissionRouter()

	w := doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "bad"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400, got %d", code)
	}

	w = doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "1", "file_hash": "bad-hash"},
		"article_file",
		"a.docx",
		[]byte("x"),
	)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400 for malformed file_hash, got %d", code)
	}

	getUploadFilePermissionFn = func(workID int) (int, int, error) { return 0, 0, errors.New("perm fail") }
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "1"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	getUploadFilePermissionFn = func(workID int) (int, int, error) { return 1, 2, nil }
	w = doMultipartWithoutFile(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "1"})
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400, got %d", code)
	}

	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePre {
			return scriptflow.ChainResult{}, errors.New("hook fail")
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "1"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePre {
			return scriptflow.ChainResult{Allowed: false, Reason: "denied"}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "2"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}

	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePre {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"x": true}}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	patchWorkInfosFn = func(workID int, patch map[string]any) error { return errors.New("patch fail") }
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "3"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	patchWorkInfosFn = func(workID int, patch map[string]any) error { return nil }
	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	if err := os.MkdirAll(filepath.Join(tmp, "files", "submissions", "2"), 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "files", "submissions", "2", "1"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "4"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	_ = os.Remove(filepath.Join(tmp, "files", "submissions", "2", "1"))
	newDocumentConverterFn = func() document.Converter {
		return converterFunc(func(ctx context.Context, srcDocPath string, dstDocxPath string) error {
			return errors.New("convert fail")
		})
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "5"}, "article_file", "a.doc", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	newDocumentConverterFn = func() document.Converter { return fakeConverter{} }
	mismatchHash := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	w = doMultipartRequest(
		router,
		"/api/v1/author/submission/file",
		"Bearer author",
		map[string]string{"work_id": "6", "file_hash": mismatchHash},
		"article_file",
		"a.docx",
		[]byte("x"),
	)
	if code := decodeRespCode(t, w.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400 for hash mismatch, got %d", code)
	}
	if _, statErr := os.Stat(filepath.Join(tmp, "files", "submissions", "2", "1", "6.docx")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("hash mismatch should remove uploaded file, stat err=%v", statErr)
	}

	newDocumentConverterFn = func() document.Converter { return fakeConverter{} }
	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePost {
			return scriptflow.ChainResult{}, errors.New("post hook fail")
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "6"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}

	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePost {
			return scriptflow.ChainResult{Allowed: false, Reason: "post denied"}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "7"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}

	runTrackHookFn = func(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
		if eventKey == scriptflow.EventFilePost {
			return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"y": true}}, nil
		}
		return scriptflow.ChainResult{Allowed: true}, nil
	}
	patchWorkInfosFn = func(workID int, patch map[string]any) error {
		if _, ok := patch["y"]; ok {
			return errors.New("post patch fail")
		}
		return nil
	}
	w = doMultipartRequest(router, "/api/v1/author/submission/file", "Bearer author", map[string]string{"work_id": "8"}, "article_file", "a.docx", []byte("x"))
	if code := decodeRespCode(t, w.Body.Bytes()); code != 500 {
		t.Fatalf("expected 500, got %d", code)
	}
}

func TestSubmissionHandlersMissingWorkBranches(t *testing.T) {
	setupSubmissionRouteMocks(t)

	checkBusinessCode := func(rec *httptest.ResponseRecorder, expected int) {
		if rec.Code != http.StatusOK {
			t.Fatalf("unexpected http code %d", rec.Code)
		}
		if got := decodeRespCode(t, rec.Body.Bytes()); got != expected {
			t.Fatalf("unexpected business code: got=%d expected=%d body=%s", got, expected, rec.Body.String())
		}
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	submissionWork(c)
	checkBusinessCode(rec, 400)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	c.Set("work", "bad")
	submissionWork(c)
	checkBusinessCode(rec, 400)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)
	updateSubmission(c)
	checkBusinessCode(rec, 400)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPut, "/", nil)
	c.Set("work", "bad")
	updateSubmission(c)
	checkBusinessCode(rec, 400)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	deleteSubmission(c)
	checkBusinessCode(rec, 400)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodDelete, "/", nil)
	c.Set("work", 1)
	deleteSubmission(c)
	checkBusinessCode(rec, 400)
}

func TestCheckWorkSubmissionValidMiddlewareBranches(t *testing.T) {
	setupSubmissionRouteMocks(t)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("author_token_id", 1)
	checkWorkSubmissionValid(c)
	if code := decodeRespCode(t, rec.Body.Bytes()); code != 400 {
		t.Fatalf("expected 400, got %d", code)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"authorID":2,"trackID":1}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("author_token_id", 1)
	checkWorkSubmissionValid(c)
	if code := decodeRespCode(t, rec.Body.Bytes()); code != 403 {
		t.Fatalf("expected 403, got %d", code)
	}

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader([]byte(`{"authorID":1,"trackID":1,"workTitle":"ok"}`)))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Set("author_token_id", 1)
	checkWorkSubmissionValid(c)
	if _, exists := c.Get("work"); !exists {
		t.Fatal("work should be set in context on success")
	}
}

func TestCleanupSubmissionFileVariants(t *testing.T) {
	setupSubmissionRouteMocks(t)

	if err := cleanupSubmissionFileVariants(filepath.Join(t.TempDir(), "not-exists"), 1); err != nil {
		t.Fatalf("cleanup should ignore non-existing directory: %v", err)
	}
}

type converterFunc func(ctx context.Context, srcDocPath string, dstDocxPath string) error

func (f converterFunc) ConvertDocToDocx(ctx context.Context, srcDocPath string, dstDocxPath string) error {
	return f(ctx, srcDocPath, dstDocxPath)
}
