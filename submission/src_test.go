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

func TestSubmissionWorkSrcWithHookPatch(t *testing.T) {
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

	dir := filepath.Join(tmp, "submissions", "7", "9")
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
