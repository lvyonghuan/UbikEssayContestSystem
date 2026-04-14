package submission

import (
	"errors"
	"fmt"
	"io/fs"
	"main/database/pgsql"
	"main/model"
	"main/util/scriptflow"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestCheckHookAllowedAndApplyPatchSuccess 测试Hook通过且应用Patch成功
func TestCheckHookAllowedAndApplyPatchSuccess(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "allow", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "allow", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/allow/v1/allow.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"score": 100}}, nil
	}

	work := &model.Work{
		WorkID: 1, AuthorID: 1, TrackID: 2,
		WorkInfos: map[string]any{"existing": "data"},
	}
	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		2,
		work,
		map[string]any{"test": "data"},
		"",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed=true")
	}
	if work.WorkInfos["score"] != 100 {
		t.Fatalf("expected patch applied, got %+v", work.WorkInfos)
	}
}

func TestCheckHookAllowedAndApplyPatchWithStatus(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "allow", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "allow", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/allow/v1/allow.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"workStatus": "review_round_1_passed"}}, nil
	}

	work := &model.Work{WorkID: 1, AuthorID: 1, TrackID: 2}
	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		2,
		work,
		map[string]any{"test": "data"},
		"",
	)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed=true")
	}
	if work.WorkStatus != "review_round_1_passed" {
		t.Fatalf("expected patched work status, got %q", work.WorkStatus)
	}
}

// TestCheckHookAllowedAndApplyPatchBlocked 测试Hook被拒绝
func TestCheckHookAllowedAndApplyPatchBlocked(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "deny", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "deny", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/deny/v1/deny.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: false, Reason: "quota exceeded"}, scriptflow.ErrExecutionBlocked
	}

	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		2,
		&model.Work{WorkID: 1, AuthorID: 1, TrackID: 2},
		map[string]any{},
		"",
	)
	if err == nil {
		t.Fatal("expected error when hook is rejected")
	}
	if allowed {
		t.Fatal("expected allowed=false")
	}
	if !strings.Contains(err.Error(), "quota exceeded") {
		t.Fatalf("expected custom reason in error, got %v", err)
	}
}

// TestCheckHookAllowedAndApplyPatchBlockedNoReason 测试Hook被拒绝但无自定义原因
func TestCheckHookAllowedAndApplyPatchBlockedNoReason(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "deny", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "deny", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/deny/v1/deny.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: false}, scriptflow.ErrExecutionBlocked
	}

	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		2,
		&model.Work{WorkID: 1, AuthorID: 1, TrackID: 2},
		map[string]any{},
		"update",
	)
	if err == nil {
		t.Fatal("expected error")
	}
	if allowed {
		t.Fatal("expected allowed=false")
	}
	if !strings.Contains(err.Error(), "submission update blocked by script flow") {
		t.Fatalf("expected generic blocked message, got %v", err)
	}
}

// TestCheckHookAllowedAndApplyPatchFlowNotMounted 测试Hook流未挂载
func TestCheckHookAllowedAndApplyPatchFlowNotMounted(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}

	work := &model.Work{WorkID: 1, WorkInfos: map[string]any{}}
	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		2,
		work,
		map[string]any{},
		"",
	)
	if err != nil {
		t.Fatalf("expected no error for unmounted flow, got %v", err)
	}
	if !allowed {
		t.Fatal("expected allowed=true when flow not mounted")
	}
}

// TestPerformWorkOperationWithPermissionSuccess 测试工作操作成功
func TestPerformWorkOperationWithPermissionSuccess(t *testing.T) {
	setupSubmissionRouteMocks(t)

	operationCalled := false
	submissionWorkFn = func(work *model.Work) error {
		operationCalled = true
		work.WorkID = 999
		return nil
	}

	permissionCalled := false
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		permissionCalled = true
		if workID != 999 {
			t.Fatalf("unexpected workID=%d", workID)
		}
		return nil
	}

	work := &model.Work{AuthorID: 1, TrackID: 2}
	err := performWorkOperationWithPermission(work, submissionWorkFn, "Test operation")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !operationCalled {
		t.Fatal("operation should be called")
	}
	if !permissionCalled {
		t.Fatal("permission setter should be called")
	}
}

// TestPerformWorkOperationWithPermissionOperationFails 测试操作失败
func TestPerformWorkOperationWithPermissionOperationFails(t *testing.T) {
	setupSubmissionRouteMocks(t)

	submissionWorkFn = func(work *model.Work) error {
		return errors.New("operation failed")
	}

	work := &model.Work{AuthorID: 1, TrackID: 2}
	err := performWorkOperationWithPermission(work, submissionWorkFn, "Test operation")
	if err == nil {
		t.Fatal("expected error when operation fails")
	}
}

// TestPerformWorkOperationWithPermissionPermissionFails 测试权限设置失败
func TestPerformWorkOperationWithPermissionPermissionFails(t *testing.T) {
	setupSubmissionRouteMocks(t)

	submissionWorkFn = func(work *model.Work) error {
		work.WorkID = 123
		return nil
	}
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		return errors.New("permission cache failed")
	}

	work := &model.Work{AuthorID: 1, TrackID: 2}
	err := performWorkOperationWithPermission(work, submissionWorkFn, "Test operation")
	if err == nil {
		t.Fatal("expected error when permission setting fails")
	}
}

// TestSubmissionWorkSrcWithHookPatchIntegration 测试Hook Patch集成
func TestSubmissionWorkSrcWithHookPatchIntegration(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}
	getTrackByIDFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 42}, nil
	}
	countWorksByAuthorAndContestFn = func(authorID int, contestID int) (int64, error) {
		return 2, nil
	}
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "submission-flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "check", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "demo", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/demo/v1/hook.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"hook_score": 99}}, nil
	}

	uploadPermissionCalled := false
	submissionWorkFn = func(work *model.Work) error {
		work.WorkID = 123
		return nil
	}
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		uploadPermissionCalled = true
		if workID != 123 {
			t.Fatalf("unexpected workID=%d", workID)
		}
		return nil
	}

	work := &model.Work{
		AuthorID:  1,
		TrackID:   10,
		WorkTitle: "demo work",
		WorkInfos: map[string]any{"origin": true},
	}
	if err := submissionWorkSrc(work); err != nil {
		t.Fatalf("submissionWorkSrc failed: %v", err)
	}

	if !uploadPermissionCalled {
		t.Fatal("upload permission cache should be set")
	}
	if work.WorkInfos["hook_score"] != 99 {
		t.Fatalf("expected hook patch in work infos, got %+v", work.WorkInfos)
	}
	if work.WorkStatus != defaultSubmissionWorkStatus {
		t.Fatalf("expected default work status %q, got %q", defaultSubmissionWorkStatus, work.WorkStatus)
	}
	if work.WorkInfos["workStatus"] != defaultSubmissionWorkStatus {
		t.Fatalf("expected workInfos workStatus %q, got %+v", defaultSubmissionWorkStatus, work.WorkInfos)
	}
}

func TestSubmissionWorkSrcWithHookStatusOverride(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}
	getTrackByIDFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 42}, nil
	}
	countWorksByAuthorAndContestFn = func(authorID int, contestID int) (int64, error) {
		return 2, nil
	}
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "submission-flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "check", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "demo", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/demo/v1/hook.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"workStatus": "first_round_passed"}}, nil
	}

	submissionWorkFn = func(work *model.Work) error {
		work.WorkID = 321
		return nil
	}
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		return nil
	}

	work := &model.Work{AuthorID: 1, TrackID: 10, WorkTitle: "demo work"}
	if err := submissionWorkSrc(work); err != nil {
		t.Fatalf("submissionWorkSrc failed: %v", err)
	}

	if work.WorkStatus != "first_round_passed" {
		t.Fatalf("expected status from hook patch, got %q", work.WorkStatus)
	}
}

func TestUpdateSubmissionSrcHydrateExistingWorkStatus(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}
	getSubmissionByWorkIDFn = func(work *model.Work) error {
		work.AuthorID = 1
		work.TrackID = 2
		work.WorkStatus = "review_round_1_done"
		return nil
	}
	updateWorkFn = func(work *model.Work) error {
		if work.WorkStatus != "review_round_1_done" {
			return fmt.Errorf("unexpected status %q", work.WorkStatus)
		}
		return nil
	}
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		return nil
	}

	work := &model.Work{WorkID: 7, AuthorID: 1, TrackID: 2, WorkTitle: "updated"}
	if err := updateSubmissionSrc(work); err != nil {
		t.Fatalf("updateSubmissionSrc failed: %v", err)
	}
	if work.WorkStatus != "review_round_1_done" {
		t.Fatalf("expected hydrated status, got %q", work.WorkStatus)
	}
}

func TestSubmissionWorkSrcBlockedByHook(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}
	getTrackByIDFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID, ContestID: 42}, nil
	}
	countWorksByAuthorAndContestFn = func(authorID int, contestID int) (int64, error) { return 0, nil }
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "deny", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "deny", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/deny/v1/deny.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: false, Reason: "policy denied"}, scriptflow.ErrExecutionBlocked
	}

	err := submissionWorkSrc(&model.Work{AuthorID: 1, TrackID: 2, WorkTitle: "x"})
	if err == nil {
		t.Fatal("submission should be blocked by hook")
	}
	if !strings.Contains(err.Error(), "policy denied") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDeleteSubmissionSrcRemovesFiles(t *testing.T) {
	setupSubmissionRouteMocks(t)

	wd, _ := os.Getwd()
	tmp := t.TempDir()
	_ = os.Chdir(tmp)
	t.Cleanup(func() { _ = os.Chdir(wd) })

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}
	deleteWorkFn = func(work *model.Work) error { return nil }
	readDirFn = os.ReadDir
	removeFn = os.Remove

	dir := filepath.Join(tmp, "files", "submissions", "7", "9")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "100.docx"), []byte("a"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "100.doc"), []byte("b"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "101.docx"), []byte("c"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}

	work := &model.Work{WorkID: 100, TrackID: 7, AuthorID: 9}
	if err := deleteSubmissionSrc(work); err != nil {
		t.Fatalf("deleteSubmissionSrc failed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "100.docx")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("100.docx should be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "100.doc")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("100.doc should be removed, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "101.docx")); err != nil {
		t.Fatalf("101.docx should remain, err=%v", err)
	}
}

func TestCheckSubmissionTimeValid(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now + 60, now + 3600, nil
	}
	if err := checkSubmissionTimeValid(1); err == nil {
		t.Fatal("should reject before start")
	}

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 3600, now - 60, nil
	}
	if err := checkSubmissionTimeValid(1); err == nil {
		t.Fatal("should reject after end")
	}

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 60, now + 60, nil
	}
	if err := checkSubmissionTimeValid(1); err != nil {
		t.Fatalf("should allow valid submission window, got %v", err)
	}
}

func TestAuthorLoginAndUpdateAuthorErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getAuthorByAuthorNameFn = func(author *model.Author) error { return errors.New("query failed") }
	if _, err := authorLoginSrc(&model.Author{AuthorName: "demo", Password: "x"}); err == nil {
		t.Fatal("authorLoginSrc should fail when querying author fails")
	}

	updateAuthorFn = func(author *model.Author) error { return errors.New("update failed") }
	if err := updateAuthorSrc(&model.Author{AuthorID: 1}); err == nil {
		t.Fatal("updateAuthorSrc should fail on update error")
	}

	findWorksByAuthorIDFn = func(authorID int) ([]model.Work, error) { return nil, errors.New("query failed") }
	if _, err := findSubmissionsByAuthorIDSrc(1); err == nil {
		t.Fatal("findSubmissionsByAuthorIDSrc should fail on query error")
	}
}

func TestUpdateSubmissionSrcErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)
	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "deny", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "deny", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/deny/v1/deny.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: false, Reason: "blocked"}, scriptflow.ErrExecutionBlocked
	}
	if err := updateSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 1}); err == nil {
		t.Fatal("updateSubmissionSrc should be blocked by hook")
	}

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}
	updateWorkFn = func(work *model.Work) error { return errors.New("update failed") }
	if err := updateSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 1}); err == nil {
		t.Fatal("updateSubmissionSrc should fail on updateWork error")
	}

	updateWorkFn = func(work *model.Work) error { return nil }
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error { return errors.New("cache failed") }
	if err := updateSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 1}); err == nil {
		t.Fatal("updateSubmissionSrc should fail on cache error")
	}
}

func TestDeleteSubmissionSrcErrorPaths(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "deny", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "deny", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/deny/v1/deny.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: false, Reason: "blocked"}, scriptflow.ErrExecutionBlocked
	}
	if err := deleteSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 1}); err == nil {
		t.Fatal("deleteSubmissionSrc should be blocked by hook")
	}

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{}, nil, pgsql.ErrFlowNotMounted
	}
	deleteWorkFn = func(work *model.Work) error { return errors.New("delete failed") }
	if err := deleteSubmissionSrc(&model.Work{WorkID: 1, AuthorID: 1, TrackID: 1}); err == nil {
		t.Fatal("deleteSubmissionSrc should fail when deleteWork fails")
	}
}

func TestRemoveSubmissionFilesErrorBranches(t *testing.T) {
	setupSubmissionRouteMocks(t)

	readDirFn = func(path string) ([]fs.DirEntry, error) { return nil, fmt.Errorf("boom") }
	if err := removeSubmissionFiles(model.Work{WorkID: 1, TrackID: 1, AuthorID: 1}); err == nil {
		t.Fatal("removeSubmissionFiles should fail on readDir error")
	}
}

// TestUpdateSubmissionSrcWithHookPatchIntegration 测试Update操作的Hook Patch集成
func TestUpdateSubmissionSrcWithHookPatchIntegration(t *testing.T) {
	setupSubmissionRouteMocks(t)

	getStartAndEndDateFn = func(trackID int) (int64, int64, error) {
		now := time.Now().Unix()
		return now - 100, now + 100, nil
	}
	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "update", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "update", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/update/v1/hook.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true, Patch: map[string]any{"updated": true}}, nil
	}

	permissionCalled := false
	updateWorkFn = func(work *model.Work) error { return nil }
	setUploadFilePermissionFn = func(authorID int, trackID int, workID int) error {
		permissionCalled = true
		return nil
	}

	work := &model.Work{
		WorkID:    1,
		AuthorID:  1,
		TrackID:   2,
		WorkTitle: "updated",
	}
	if err := updateSubmissionSrc(work); err != nil {
		t.Fatalf("updateSubmissionSrc failed: %v", err)
	}

	if !permissionCalled {
		t.Fatal("upload permission should be set after update")
	}
	if work.WorkInfos["updated"] != true {
		t.Fatalf("hook patch should be applied")
	}
}

// TestDeleteSubmissionSrcWithHookSuccess 测试Delete操作成功完成
func TestDeleteSubmissionSrcWithHookSuccess(t *testing.T) {
	setupSubmissionRouteMocks(t)

	resolveFlowForExecutionFn = func(scope string, eventKey string, targetType string, targetID int) (model.ScriptFlow, []pgsql.ResolvedFlowStep, error) {
		return model.ScriptFlow{FlowID: 1, FlowKey: "flow"}, []pgsql.ResolvedFlowStep{{
			Step:    model.FlowStep{StepID: 1, StepName: "delete", TimeoutMs: 1000, FailureStrategy: "fail_close"},
			Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: "delete", Interpreter: "python3", IsEnabled: true},
			Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: "scripts/delete/v1/hook.py", IsActive: true},
		}}, nil
	}
	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	deleteWorkFn = func(work *model.Work) error { return nil }
	readDirFn = func(path string) ([]fs.DirEntry, error) { return nil, os.ErrNotExist }

	work := &model.Work{WorkID: 5, AuthorID: 2, TrackID: 3}
	if err := deleteSubmissionSrc(work); err != nil {
		t.Fatalf("deleteSubmissionSrc should succeed, got %v", err)
	}
}
