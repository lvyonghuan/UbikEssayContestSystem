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
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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
	origListAuthorsSrcFn := listAuthorsSrcFn
	origGetAuthorByIDSrcFn := getAuthorByIDSrcFn
	origUpdateAuthorSrcFn := updateAuthorSrcFn
	origDeleteAuthorSrcFn := deleteAuthorSrcFn
	origGetWorkByIDSrcFn := getWorkByIDSrcFn
	origGetWorkFilePathSrcFn := getWorkFilePathSrcFn
	origQueryWorksSrcFn := queryWorksSrcFn
	origDeleteWorkSrcFn := deleteWorkSrcFn
	origCheckAdminActiveSrcFn := checkAdminActiveSrcFn
	origHasPermissionSrcFn := hasPermissionSrcFn
	origIsSuperAdminSrcFn := isSuperAdminSrcFn
	origCreateSubAdminSrcFn := createSubAdminSrcFn
	origBatchCreateSubAdminsSrcFn := batchCreateSubAdminsSrcFn
	origListSubAdminsSrcFn := listSubAdminsSrcFn
	origUpdateSubAdminPermissionsFn := updateSubAdminPermissionsFn
	origDisableSubAdminSrcFn := disableSubAdminSrcFn
	origDeleteSubAdminSrcFn := deleteSubAdminSrcFn
	origHandoverSuperAdminSrcFn := handoverSuperAdminSrcFn
	origCreateScriptDefinitionSrcFn := createScriptDefinitionSrcFn
	origListScriptDefinitionsSrcFn := listScriptDefinitionsSrcFn
	origGetScriptDefinitionByIDSrcFn := getScriptDefinitionByIDSrcFn
	origUpdateScriptDefinitionSrcFn := updateScriptDefinitionSrcFn
	origSetScriptDefinitionEnabledSrcFn := setScriptDefinitionEnabledSrcFn
	origUploadScriptVersionSrcFn := uploadScriptVersionSrcFn
	origListScriptVersionsSrcFn := listScriptVersionsSrcFn
	origActivateScriptVersionSrcFn := activateScriptVersionSrcFn
	origCreateScriptFlowSrcFn := createScriptFlowSrcFn
	origListScriptFlowsSrcFn := listScriptFlowsSrcFn
	origGetScriptFlowByIDSrcFn := getScriptFlowByIDSrcFn
	origUpdateScriptFlowSrcFn := updateScriptFlowSrcFn
	origSetScriptFlowEnabledSrcFn := setScriptFlowEnabledSrcFn
	origReplaceFlowStepsSrcFn := replaceFlowStepsSrcFn
	origListFlowStepsSrcFn := listFlowStepsSrcFn
	origCreateFlowMountSrcFn := createFlowMountSrcFn
	origDeleteFlowMountSrcFn := deleteFlowMountSrcFn
	origListFlowMountsByFlowSrcFn := listFlowMountsByFlowSrcFn

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
		listAuthorsSrcFn = origListAuthorsSrcFn
		getAuthorByIDSrcFn = origGetAuthorByIDSrcFn
		updateAuthorSrcFn = origUpdateAuthorSrcFn
		deleteAuthorSrcFn = origDeleteAuthorSrcFn
		getWorkByIDSrcFn = origGetWorkByIDSrcFn
		getWorkFilePathSrcFn = origGetWorkFilePathSrcFn
		queryWorksSrcFn = origQueryWorksSrcFn
		deleteWorkSrcFn = origDeleteWorkSrcFn
		checkAdminActiveSrcFn = origCheckAdminActiveSrcFn
		hasPermissionSrcFn = origHasPermissionSrcFn
		isSuperAdminSrcFn = origIsSuperAdminSrcFn
		createSubAdminSrcFn = origCreateSubAdminSrcFn
		batchCreateSubAdminsSrcFn = origBatchCreateSubAdminsSrcFn
		listSubAdminsSrcFn = origListSubAdminsSrcFn
		updateSubAdminPermissionsFn = origUpdateSubAdminPermissionsFn
		disableSubAdminSrcFn = origDisableSubAdminSrcFn
		deleteSubAdminSrcFn = origDeleteSubAdminSrcFn
		handoverSuperAdminSrcFn = origHandoverSuperAdminSrcFn
		createScriptDefinitionSrcFn = origCreateScriptDefinitionSrcFn
		listScriptDefinitionsSrcFn = origListScriptDefinitionsSrcFn
		getScriptDefinitionByIDSrcFn = origGetScriptDefinitionByIDSrcFn
		updateScriptDefinitionSrcFn = origUpdateScriptDefinitionSrcFn
		setScriptDefinitionEnabledSrcFn = origSetScriptDefinitionEnabledSrcFn
		uploadScriptVersionSrcFn = origUploadScriptVersionSrcFn
		listScriptVersionsSrcFn = origListScriptVersionsSrcFn
		activateScriptVersionSrcFn = origActivateScriptVersionSrcFn
		createScriptFlowSrcFn = origCreateScriptFlowSrcFn
		listScriptFlowsSrcFn = origListScriptFlowsSrcFn
		getScriptFlowByIDSrcFn = origGetScriptFlowByIDSrcFn
		updateScriptFlowSrcFn = origUpdateScriptFlowSrcFn
		setScriptFlowEnabledSrcFn = origSetScriptFlowEnabledSrcFn
		replaceFlowStepsSrcFn = origReplaceFlowStepsSrcFn
		listFlowStepsSrcFn = origListFlowStepsSrcFn
		createFlowMountSrcFn = origCreateFlowMountSrcFn
		deleteFlowMountSrcFn = origDeleteFlowMountSrcFn
		listFlowMountsByFlowSrcFn = origListFlowMountsByFlowSrcFn
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
	listAuthorsSrcFn = func(authorName string, offset int, limit int) ([]model.Author, error) {
		return []model.Author{{AuthorID: 1, AuthorName: "author_1", PenName: "pen", AuthorEmail: "a1@example.com"}}, nil
	}
	getAuthorByIDSrcFn = func(authorID int) (model.Author, error) {
		return model.Author{AuthorID: authorID, AuthorName: "author", PenName: "pen", AuthorEmail: "a@example.com"}, nil
	}
	updateAuthorSrcFn = func(adminID int, authorID int, author *model.Author) (model.Author, error) {
		author.AuthorID = authorID
		return *author, nil
	}
	deleteAuthorSrcFn = func(adminID int, authorID int) error { return nil }
	getWorkByIDSrcFn = func(workID int) (model.Work, error) {
		return model.Work{WorkID: workID, WorkTitle: "w", TrackID: 1, AuthorID: 1}, nil
	}
	queryWorksSrcFn = func(trackID *int, workTitle string, authorName string, offset int, limit int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1, WorkTitle: "w", TrackID: 1, AuthorID: 1}}, nil
	}
	deleteWorkSrcFn = func(adminID, workID int) error { return nil }
	checkAdminActiveSrcFn = func(adminID int) (bool, error) { return true, nil }
	hasPermissionSrcFn = func(adminID int, permissionName string) (bool, error) { return true, nil }
	isSuperAdminSrcFn = func(adminID int) (bool, error) { return adminID == 1, nil }
	createSubAdminSrcFn = func(adminID int, req model.CreateSubAdminRequest) (model.SubAdminCreateResult, error) {
		return model.SubAdminCreateResult{AdminID: 9, AdminName: "sub_9", AdminEmail: req.AdminEmail, TempPassword: "temp", EmailSent: true}, nil
	}
	batchCreateSubAdminsSrcFn = func(adminID int, emails []string, permissionNames []string) (model.BatchCreateSubAdminsResponse, error) {
		created := make([]model.SubAdminCreateResult, 0, len(emails))
		for idx, email := range emails {
			created = append(created, model.SubAdminCreateResult{AdminID: idx + 10, AdminName: "sub", AdminEmail: email, TempPassword: "temp", EmailSent: true})
		}
		return model.BatchCreateSubAdminsResponse{Created: created, Failed: nil}, nil
	}
	listSubAdminsSrcFn = func() ([]model.SubAdminInfo, error) {
		return []model.SubAdminInfo{{AdminID: 9, AdminName: "sub_9", AdminEmail: "sub@example.com", IsActive: true, PermissionNames: []string{"works.read"}}}, nil
	}
	updateSubAdminPermissionsFn = func(adminID int, targetAdminID int, permissionNames []string) error { return nil }
	disableSubAdminSrcFn = func(adminID int, targetAdminID int) error { return nil }
	deleteSubAdminSrcFn = func(adminID int, targetAdminID int) error { return nil }
	handoverSuperAdminSrcFn = func(currentAdminID int, newSuperAdminID int) error { return nil }
	createScriptDefinitionSrcFn = func(adminID int, def *model.ScriptDefinition) error {
		if def.ScriptID == 0 {
			def.ScriptID = 1
		}
		if def.ScriptKey == "" {
			def.ScriptKey = "script_key"
		}
		if def.ScriptName == "" {
			def.ScriptName = "script_name"
		}
		if def.Interpreter == "" {
			def.Interpreter = "python3"
		}
		return nil
	}
	listScriptDefinitionsSrcFn = func() ([]model.ScriptDefinition, error) {
		return []model.ScriptDefinition{{ScriptID: 1, ScriptKey: "script_key", ScriptName: "script_name", Interpreter: "python3", IsEnabled: true}}, nil
	}
	getScriptDefinitionByIDSrcFn = func(scriptID int) (model.ScriptDefinition, error) {
		return model.ScriptDefinition{ScriptID: scriptID, ScriptKey: "script_key", ScriptName: "script_name", Interpreter: "python3", IsEnabled: true}, nil
	}
	updateScriptDefinitionSrcFn = func(adminID int, scriptID int, req *model.ScriptDefinition) error {
		req.ScriptID = scriptID
		return nil
	}
	setScriptDefinitionEnabledSrcFn = func(adminID int, scriptID int, enabled bool) error { return nil }
	uploadScriptVersionSrcFn = func(adminID int, scriptID int, fileHeader *multipart.FileHeader) (model.ScriptVersion, error) {
		return model.ScriptVersion{VersionID: 1, ScriptID: scriptID, VersionNum: 1, FileName: "script.py", RelativePath: "scripts/script_key/v1/script.py", IsActive: true, CreatedBy: adminID}, nil
	}
	listScriptVersionsSrcFn = func(scriptID int) ([]model.ScriptVersion, error) {
		return []model.ScriptVersion{{VersionID: 1, ScriptID: scriptID, VersionNum: 1, FileName: "script.py", RelativePath: "scripts/script_key/v1/script.py", IsActive: true, CreatedBy: 1}}, nil
	}
	activateScriptVersionSrcFn = func(adminID int, scriptID int, versionID int) error { return nil }
	createScriptFlowSrcFn = func(adminID int, flow *model.ScriptFlow) error {
		if flow.FlowID == 0 {
			flow.FlowID = 1
		}
		if flow.FlowKey == "" {
			flow.FlowKey = "flow_key"
		}
		if flow.FlowName == "" {
			flow.FlowName = "flow_name"
		}
		return nil
	}
	listScriptFlowsSrcFn = func() ([]model.ScriptFlow, error) {
		return []model.ScriptFlow{{FlowID: 1, FlowKey: "flow_key", FlowName: "flow_name", IsEnabled: true}}, nil
	}
	getScriptFlowByIDSrcFn = func(flowID int) (model.ScriptFlow, error) {
		return model.ScriptFlow{FlowID: flowID, FlowKey: "flow_key", FlowName: "flow_name", IsEnabled: true}, nil
	}
	updateScriptFlowSrcFn = func(adminID int, flowID int, flow *model.ScriptFlow) error {
		flow.FlowID = flowID
		return nil
	}
	setScriptFlowEnabledSrcFn = func(adminID int, flowID int, enabled bool) error { return nil }
	replaceFlowStepsSrcFn = func(adminID int, flowID int, steps []model.FlowStep) error { return nil }
	listFlowStepsSrcFn = func(flowID int) ([]model.FlowStep, error) {
		return []model.FlowStep{{StepID: 1, FlowID: flowID, StepOrder: 1, StepName: "step", ScriptID: 1, IsEnabled: true}}, nil
	}
	createFlowMountSrcFn = func(adminID int, mount *model.FlowMount) error {
		if mount.MountID == 0 {
			mount.MountID = 1
		}
		if mount.FlowID == 0 {
			mount.FlowID = 1
		}
		if mount.Scope == "" {
			mount.Scope = "submission"
		}
		if mount.EventKey == "" {
			mount.EventKey = "after_submit"
		}
		if mount.TargetType == "" {
			mount.TargetType = "track"
		}
		if mount.TargetID == 0 {
			mount.TargetID = 1
		}
		mount.IsEnabled = true
		return nil
	}
	deleteFlowMountSrcFn = func(adminID int, mountID int) error { return nil }
	listFlowMountsByFlowSrcFn = func(flowID int) ([]model.FlowMount, error) {
		return []model.FlowMount{{MountID: 1, FlowID: flowID, Scope: "submission", EventKey: "after_submit", TargetType: "track", TargetID: 1, IsEnabled: true}}, nil
	}

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
		{http.MethodGet, "/api/v1/admin/authors?author_name=author_1&offset=0&limit=20", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/authors/1", nil, "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/authors/1", []byte(`{"authorName":"a1","penName":"p1","authorEmail":"a1@example.com"}`), "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/authors/1", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/scripts", []byte(`{"scriptKey":"k","scriptName":"n","interpreter":"python3"}`), "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/scripts", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/scripts/1", nil, "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/scripts/1", []byte(`{"scriptName":"n2","interpreter":"python3"}`), "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/scripts/1/status", []byte(`{"isEnabled":true}`), "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/scripts/1/versions", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/scripts/1/versions/1/activate", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/script-flows", []byte(`{"flowKey":"f","flowName":"flow"}`), "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/script-flows", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/script-flows/1", nil, "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/script-flows/1", []byte(`{"flowName":"flow-2"}`), "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/script-flows/1/status", []byte(`{"isEnabled":true}`), "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/script-flows/1/steps", []byte(`[]`), "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/script-flows/1/steps", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/script-flows/mounts", []byte(`{"flowID":1,"scope":"submission","eventKey":"after_submit","targetType":"track","targetID":1}`), "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/script-flows/mounts/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/script-flows/1/mounts", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works?track_id=1&offset=0&limit=20", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/works/1/file", nil, "Bearer admin", true},
		{http.MethodDelete, "/api/v1/admin/works/1", nil, "Bearer admin", false},
		{http.MethodGet, "/api/v1/admin/sub-admins", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/sub-admins", []byte(`{"adminEmail":"sub@example.com","permissionNames":["works.read"]}`), "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/sub-admins/batch", []byte(`{"emails":["a@example.com","b@example.com"],"permissionNames":["works.read"]}`), "Bearer admin", false},
		{http.MethodPut, "/api/v1/admin/sub-admins/9/permissions", []byte(`{"permissionNames":["works.read","works.delete"]}`), "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/sub-admins/9/disable", nil, "Bearer admin", false},
		{http.MethodDelete, "/api/v1/admin/sub-admins/9", nil, "Bearer admin", false},
		{http.MethodPost, "/api/v1/admin/sub-admins/handover-super", []byte(`{"newSuperAdminID":9}`), "Bearer admin", false},
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

	hasPermissionSrcFn = func(adminID int, permissionName string) (bool, error) { return false, nil }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 403 {
		t.Fatalf("expected 403 for missing permission, got %d", got)
	}
	hasPermissionSrcFn = func(adminID int, permissionName string) (bool, error) { return true, nil }
	checkAdminActiveSrcFn = func(adminID int) (bool, error) { return false, nil }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 403 {
		t.Fatalf("expected 403 for disabled admin, got %d", got)
	}
	checkAdminActiveSrcFn = func(adminID int) (bool, error) { return true, nil }

	isSuperAdminSrcFn = func(adminID int) (bool, error) { return false, nil }
	w = doRequest(router, http.MethodGet, "/api/v1/admin/sub-admins", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 403 {
		t.Fatalf("expected 403 for non-super access, got %d", got)
	}
}

func TestWorksErrorPaths(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/admin/works?offset=-1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid offset, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/works?limit=101", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid limit, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/works?track_id=bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid track_id query, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/works/bad", nil, "Bearer admin")
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

func TestAuthorsErrorPaths(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	w := doRequest(router, http.MethodGet, "/api/v1/admin/authors?offset=-1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid offset, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/authors?limit=101", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid limit, got %d", got)
	}

	w = doRequest(router, http.MethodGet, "/api/v1/admin/authors/bad", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid author_id, got %d", got)
	}

	listAuthorsSrcFn = func(authorName string, offset int, limit int) ([]model.Author, error) {
		return nil, errors.New("x")
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/authors", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for list authors src error, got %d", got)
	}

	getAuthorByIDSrcFn = func(authorID int) (model.Author, error) {
		return model.Author{}, errAuthorNotFound
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/authors/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for author not found, got %d", got)
	}

	updateAuthorSrcFn = func(adminID int, authorID int, author *model.Author) (model.Author, error) {
		return model.Author{}, errAuthorNotFound
	}
	w = doRequest(router, http.MethodPut, "/api/v1/admin/authors/1", []byte(`{"authorName":"a"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for update author not found, got %d", got)
	}

	deleteAuthorSrcFn = func(adminID int, authorID int) error {
		return errAuthorNotFound
	}
	w = doRequest(router, http.MethodDelete, "/api/v1/admin/authors/1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for delete author not found, got %d", got)
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

	queryWorksSrcFn = func(trackID *int, workTitle string, authorName string, offset int, limit int) ([]model.Work, error) {
		return nil, errors.New("x")
	}
	w = doRequest(router, http.MethodGet, "/api/v1/admin/works?track_id=1", nil, "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for query works src error, got %d", got)
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

func TestSubAdminErrorPaths(t *testing.T) {
	mockAuthAndSources(t)
	router := buildAdminRouter()

	createSubAdminSrcFn = func(adminID int, req model.CreateSubAdminRequest) (model.SubAdminCreateResult, error) {
		return model.SubAdminCreateResult{}, errors.New("invalid adminEmail")
	}
	w := doRequest(router, http.MethodPost, "/api/v1/admin/sub-admins", []byte(`{"adminEmail":"bad"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid create payload, got %d", got)
	}

	createSubAdminSrcFn = func(adminID int, req model.CreateSubAdminRequest) (model.SubAdminCreateResult, error) {
		return model.SubAdminCreateResult{}, errors.New("db down")
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/sub-admins", []byte(`{"adminEmail":"ok@example.com"}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 500 {
		t.Fatalf("expected 500 for create src error, got %d", got)
	}

	w = doRequest(router, http.MethodPut, "/api/v1/admin/sub-admins/bad/permissions", []byte(`{"permissionNames":["works.read"]}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid admin_id, got %d", got)
	}

	updateSubAdminPermissionsFn = func(adminID int, targetAdminID int, permissionNames []string) error {
		return gorm.ErrRecordNotFound
	}
	w = doRequest(router, http.MethodPut, "/api/v1/admin/sub-admins/9/permissions", []byte(`{"permissionNames":["works.read"]}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 404 {
		t.Fatalf("expected 404 for target sub admin not found, got %d", got)
	}

	handoverSuperAdminSrcFn = func(currentAdminID int, newSuperAdminID int) error {
		return errors.New("invalid handover request")
	}
	w = doRequest(router, http.MethodPost, "/api/v1/admin/sub-admins/handover-super", []byte(`{"newSuperAdminID":1}`), "Bearer admin")
	if got := reqCode(t, w.Body.Bytes()); got != 400 {
		t.Fatalf("expected 400 for invalid handover, got %d", got)
	}
}
