package system

import (
	"context"
	"errors"
	"io"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/scriptflow"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type mockContestEndLogger struct {
	debugs []string
	warns  []string
}

func (m *mockContestEndLogger) Debug(v string)  { m.debugs = append(m.debugs, v) }
func (m *mockContestEndLogger) Info(v string)   {}
func (m *mockContestEndLogger) Warn(v string)   { m.warns = append(m.warns, v) }
func (m *mockContestEndLogger) Error(v error)   {}
func (m *mockContestEndLogger) Fatal(v error)   {}
func (m *mockContestEndLogger) System(v string) {}

func backupContestEndHookDeps(t *testing.T) {
	origGetTracksByContestForEndFn := getTracksByContestForEndFn
	origGetTrackByIDForEndFn := getTrackByIDForEndFn
	origGetContestEndExecutionForEndFn := getContestEndExecutionForEndFn
	origMarkTrackContestEndRunningForEndFn := markTrackContestEndRunningForEndFn
	origMarkContestEndRunningForEndFn := markContestEndRunningForEndFn
	origMarkTrackContestEndSuccessForEndFn := markTrackContestEndSuccessForEndFn
	origMarkContestEndSuccessForEndFn := markContestEndSuccessForEndFn
	origMarkTrackContestEndFailedForEndFn := markTrackContestEndFailedForEndFn
	origMarkContestEndFailedForEndFn := markContestEndFailedForEndFn
	origResolveFlowChainForEndFn := resolveFlowChainForEndFn
	origListReviewEventsForEndFn := listReviewEventsForEndFn
	origListReviewWorksForEndFn := listReviewWorksForEndFn
	origListReviewsForEndFn := listReviewsForEndFn
	origListJudgeIDsForEndFn := listJudgeIDsForEndFn
	origDeleteReviewResultsForEndFn := deleteReviewResultsForEndFn
	origUpsertReviewResultForEndFn := upsertReviewResultForEndFn
	origListWorksByTrackForEndFn := listWorksByTrackForEndFn
	origResolveSubmissionFileForEndFn := resolveSubmissionFileForEndFn
	origConvertDocxToPDFForEndFn := convertDocxToPDFForEndFn
	origReadDirForEndFn := readDirForEndFn
	origMkdirAllForEndFn := mkdirAllForEndFn
	origRenameFileForEndFn := renameFileForEndFn
	origOpenFileForEndFn := openFileForEndFn
	origCreateFileForEndFn := createFileForEndFn
	origCopyFileForEndFn := copyFileForEndFn
	origExecuteScriptChainForEndFn := executeScriptChainForEndFn

	getTrackByIDForEndFn = func(trackID int) (model.Track, error) {
		return model.Track{TrackID: trackID}, nil
	}
	getContestEndExecutionForEndFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		return contestEndExecutionState{}, errors.New("record not found")
	}
	markTrackContestEndRunningForEndFn = func(trackID int, triggerSource string) error { return nil }
	markContestEndRunningForEndFn = func(contestID int, trackID int, triggerSource string) error { return nil }
	markTrackContestEndSuccessForEndFn = func(trackID int, triggerSource string) error { return nil }
	markContestEndSuccessForEndFn = func(contestID int, trackID int, triggerSource string) error { return nil }
	markTrackContestEndFailedForEndFn = func(trackID int, triggerSource string, lastError string) error { return nil }
	markContestEndFailedForEndFn = func(contestID int, trackID int, triggerSource string, lastError string) error { return nil }

	t.Cleanup(func() {
		getTracksByContestForEndFn = origGetTracksByContestForEndFn
		getTrackByIDForEndFn = origGetTrackByIDForEndFn
		getContestEndExecutionForEndFn = origGetContestEndExecutionForEndFn
		markTrackContestEndRunningForEndFn = origMarkTrackContestEndRunningForEndFn
		markContestEndRunningForEndFn = origMarkContestEndRunningForEndFn
		markTrackContestEndSuccessForEndFn = origMarkTrackContestEndSuccessForEndFn
		markContestEndSuccessForEndFn = origMarkContestEndSuccessForEndFn
		markTrackContestEndFailedForEndFn = origMarkTrackContestEndFailedForEndFn
		markContestEndFailedForEndFn = origMarkContestEndFailedForEndFn
		resolveFlowChainForEndFn = origResolveFlowChainForEndFn
		listReviewEventsForEndFn = origListReviewEventsForEndFn
		listReviewWorksForEndFn = origListReviewWorksForEndFn
		listReviewsForEndFn = origListReviewsForEndFn
		listJudgeIDsForEndFn = origListJudgeIDsForEndFn
		deleteReviewResultsForEndFn = origDeleteReviewResultsForEndFn
		upsertReviewResultForEndFn = origUpsertReviewResultForEndFn
		listWorksByTrackForEndFn = origListWorksByTrackForEndFn
		resolveSubmissionFileForEndFn = origResolveSubmissionFileForEndFn
		convertDocxToPDFForEndFn = origConvertDocxToPDFForEndFn
		readDirForEndFn = origReadDirForEndFn
		mkdirAllForEndFn = origMkdirAllForEndFn
		renameFileForEndFn = origRenameFileForEndFn
		openFileForEndFn = origOpenFileForEndFn
		createFileForEndFn = origCreateFileForEndFn
		copyFileForEndFn = origCopyFileForEndFn
		executeScriptChainForEndFn = origExecuteScriptChainForEndFn
	})
}

func TestRunContestEndHookForTrackRunsBuiltinPipeline(t *testing.T) {
	backupContestEndHookDeps(t)

	const (
		contestID = 9
		trackID   = 3
		eventID   = 77
		workID    = 11
		authorID  = 5
	)

	calls := make([]string, 0, 8)
	capturedPayload := map[string]any{}

	listReviewEventsForEndFn = func(inputTrackID int) ([]model.ReviewEvent, error) {
		if inputTrackID != trackID {
			t.Fatalf("unexpected trackID: %d", inputTrackID)
		}
		return []model.ReviewEvent{{EventID: eventID, TrackID: trackID}}, nil
	}
	deleteReviewResultsForEndFn = func(inputEventID int) error {
		if inputEventID != eventID {
			t.Fatalf("unexpected eventID for delete: %d", inputEventID)
		}
		calls = append(calls, "delete")
		return nil
	}
	listReviewWorksForEndFn = func(inputEventID int, offset int, limit int) ([]model.Work, error) {
		if inputEventID != eventID {
			t.Fatalf("unexpected eventID for list works: %d", inputEventID)
		}
		return []model.Work{{WorkID: workID, TrackID: trackID, AuthorID: authorID}}, nil
	}
	listReviewsForEndFn = func(inputWorkID int, inputEventID int) ([]model.Review, error) {
		if inputWorkID != workID || inputEventID != eventID {
			t.Fatalf("unexpected review query args: %d %d", inputWorkID, inputEventID)
		}
		return []model.Review{
			{JudgeID: 1, WorkReviews: map[string]any{"judgeScore": 90.0, "judgeComment": "A"}},
			{JudgeID: 2, WorkReviews: map[string]any{"judgeScore": 80.0, "judgeComment": "B"}},
		}, nil
	}
	listJudgeIDsForEndFn = func(inputEventID int) ([]int, error) {
		if inputEventID != eventID {
			t.Fatalf("unexpected eventID for list judges: %d", inputEventID)
		}
		return []int{1, 2}, nil
	}
	upsertReviewResultForEndFn = func(inputWorkID int, inputEventID int, reviews map[string]any) (model.ReviewResult, error) {
		if inputWorkID != workID || inputEventID != eventID {
			t.Fatalf("unexpected upsert args: %d %d", inputWorkID, inputEventID)
		}
		capturedPayload = reviews
		calls = append(calls, "upsert")
		return model.ReviewResult{WorkID: inputWorkID, ReviewEventID: inputEventID, Reviews: reviews}, nil
	}

	listWorksByTrackForEndFn = func(inputTrackID int) ([]model.Work, error) {
		if inputTrackID != trackID {
			t.Fatalf("unexpected trackID for list track works: %d", inputTrackID)
		}
		return []model.Work{{WorkID: workID, TrackID: trackID, AuthorID: authorID}}, nil
	}
	resolveSubmissionFileForEndFn = func(work model.Work) (string, error) {
		if work.WorkID != workID {
			t.Fatalf("unexpected work for resolve file: %+v", work)
		}
		return filepath.Join("tmp", strconv.Itoa(workID)+".docx"), nil
	}
	convertDocxToPDFForEndFn = func(ctx context.Context, srcDocxPath string, dstPDFPath string) error {
		if srcDocxPath != filepath.Join("tmp", strconv.Itoa(workID)+".docx") {
			t.Fatalf("unexpected src path: %s", srcDocxPath)
		}
		expectedDst := filepath.Join(_const.FileRootPath, "pdfs", strconv.Itoa(contestID), strconv.Itoa(trackID), strconv.Itoa(authorID), strconv.Itoa(workID)+".pdf")
		if filepath.Clean(dstPDFPath) != filepath.Clean(expectedDst) {
			t.Fatalf("unexpected dst path: got=%s want=%s", dstPDFPath, expectedDst)
		}
		calls = append(calls, "pdf")
		return nil
	}

	resolveFlowChainForEndFn = func(scope string, eventKey string, inputContestID int, inputTrackID int) ([]pgsql.ResolvedFlowChain, error) {
		if scope != scriptflow.ScopeSystem || eventKey != scriptflow.EventContestEnd || inputContestID != contestID || inputTrackID != trackID {
			t.Fatalf("unexpected resolve flow args: %s %s %d %d", scope, eventKey, inputContestID, inputTrackID)
		}
		return []pgsql.ResolvedFlowChain{{
			Flow: model.ScriptFlow{FlowKey: "contest-end"},
			Steps: []pgsql.ResolvedFlowStep{
				{
					Step:    model.FlowStep{StepID: 1, StepOrder: 1, StepName: "builtin_regenerate", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: contestEndBuiltinRegenerateScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: contestEndBuiltinRegenerateStepKey, IsActive: true},
				},
				{
					Step:    model.FlowStep{StepID: 2, StepOrder: 2, StepName: "builtin_export", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 2, ScriptKey: contestEndBuiltinExportPDFScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 2, ScriptID: 2, RelativePath: contestEndBuiltinExportPDFStepKey, IsActive: true},
				},
			},
		}}, nil
	}

	if err := runContestEndHookForTrack(contestID, trackID); err != nil {
		t.Fatalf("runContestEndHookForTrack failed: %v", err)
	}

	if got := strings.Join(calls, ","); got != "delete,upsert,pdf" {
		t.Fatalf("unexpected execution order: %s", got)
	}
	if got, ok := capturedPayload["finalScore"].(float64); !ok || got != 85 {
		t.Fatalf("unexpected finalScore payload: %+v", capturedPayload)
	}
	if got, ok := capturedPayload["reviewCount"].(int); !ok || got != 2 {
		t.Fatalf("unexpected reviewCount payload: %+v", capturedPayload)
	}
	judgeScores, ok := capturedPayload["judgeScores"].(map[string]float64)
	if !ok || judgeScores["1"] != 90 || judgeScores["2"] != 80 {
		t.Fatalf("unexpected judgeScores payload: %+v", capturedPayload)
	}
}

func TestRunContestEndHookForTrackSkipsMissingSubmissionFiles(t *testing.T) {
	backupContestEndHookDeps(t)

	const (
		contestID = 10
		trackID   = 4
	)

	listReviewEventsForEndFn = func(inputTrackID int) ([]model.ReviewEvent, error) {
		return nil, nil
	}
	listWorksByTrackForEndFn = func(inputTrackID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 1, TrackID: trackID, AuthorID: 2}}, nil
	}
	resolveSubmissionFileForEndFn = func(work model.Work) (string, error) {
		return "", os.ErrNotExist
	}

	convertCalls := 0
	convertDocxToPDFForEndFn = func(ctx context.Context, srcDocxPath string, dstPDFPath string) error {
		convertCalls++
		return nil
	}

	resolveFlowChainForEndFn = func(scope string, eventKey string, inputContestID int, inputTrackID int) ([]pgsql.ResolvedFlowChain, error) {
		return []pgsql.ResolvedFlowChain{{
			Flow: model.ScriptFlow{FlowKey: "contest-end"},
			Steps: []pgsql.ResolvedFlowStep{
				{
					Step:    model.FlowStep{StepID: 1, StepOrder: 1, StepName: "builtin_regenerate", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: contestEndBuiltinRegenerateScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: contestEndBuiltinRegenerateStepKey, IsActive: true},
				},
				{
					Step:    model.FlowStep{StepID: 2, StepOrder: 2, StepName: "builtin_export", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 2, ScriptKey: contestEndBuiltinExportPDFScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 2, ScriptID: 2, RelativePath: contestEndBuiltinExportPDFStepKey, IsActive: true},
				},
			},
		}}, nil
	}

	if err := runContestEndHookForTrack(contestID, trackID); err != nil {
		t.Fatalf("runContestEndHookForTrack failed: %v", err)
	}
	if convertCalls != 0 {
		t.Fatalf("pdf converter should not be called for missing files, got %d", convertCalls)
	}
}

func TestRunContestEndHookForTrackStopsWhenBuiltinFails(t *testing.T) {
	backupContestEndHookDeps(t)

	const (
		contestID = 1
		trackID   = 2
		eventID   = 3
	)

	listReviewEventsForEndFn = func(inputTrackID int) ([]model.ReviewEvent, error) {
		return []model.ReviewEvent{{EventID: eventID, TrackID: trackID}}, nil
	}
	deleteReviewResultsForEndFn = func(inputEventID int) error { return nil }
	listReviewWorksForEndFn = func(inputEventID int, offset int, limit int) ([]model.Work, error) {
		return []model.Work{{WorkID: 100, TrackID: trackID, AuthorID: 50}}, nil
	}
	listReviewsForEndFn = func(inputWorkID int, inputEventID int) ([]model.Review, error) {
		return nil, nil
	}
	listJudgeIDsForEndFn = func(inputEventID int) ([]int, error) {
		return nil, nil
	}
	upsertReviewResultForEndFn = func(inputWorkID int, inputEventID int, reviews map[string]any) (model.ReviewResult, error) {
		return model.ReviewResult{}, errors.New("upsert failed")
	}

	convertCalls := 0
	listWorksByTrackForEndFn = func(inputTrackID int) ([]model.Work, error) {
		return []model.Work{{WorkID: 100, TrackID: trackID, AuthorID: 50}}, nil
	}
	resolveSubmissionFileForEndFn = func(work model.Work) (string, error) {
		return filepath.Join("tmp", "100.docx"), nil
	}
	convertDocxToPDFForEndFn = func(ctx context.Context, srcDocxPath string, dstPDFPath string) error {
		convertCalls++
		return nil
	}

	resolveFlowChainForEndFn = func(scope string, eventKey string, inputContestID int, inputTrackID int) ([]pgsql.ResolvedFlowChain, error) {
		return []pgsql.ResolvedFlowChain{{
			Flow: model.ScriptFlow{FlowKey: "contest-end"},
			Steps: []pgsql.ResolvedFlowStep{
				{
					Step:    model.FlowStep{StepID: 1, StepOrder: 1, StepName: "builtin_regenerate", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: contestEndBuiltinRegenerateScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: contestEndBuiltinRegenerateStepKey, IsActive: true},
				},
				{
					Step:    model.FlowStep{StepID: 2, StepOrder: 2, StepName: "builtin_export", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 2, ScriptKey: contestEndBuiltinExportPDFScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 2, ScriptID: 2, RelativePath: contestEndBuiltinExportPDFStepKey, IsActive: true},
				},
			},
		}}, nil
	}

	err := runContestEndHookForTrack(contestID, trackID)
	if err == nil || !strings.Contains(err.Error(), "upsert failed") {
		t.Fatalf("expected upsert error, got %v", err)
	}
	if convertCalls != 0 {
		t.Fatalf("export step should not execute when regenerate fails, got convert calls: %d", convertCalls)
	}
}

func TestRunContestEndHookForTrackSkipsWhenAlreadySuccessful(t *testing.T) {
	backupContestEndHookDeps(t)

	getContestEndExecutionForEndFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		return contestEndExecutionState{Status: contestEndExecutionStatusSuccess}, nil
	}

	builtinCalled := false
	listReviewEventsForEndFn = func(inputTrackID int) ([]model.ReviewEvent, error) {
		builtinCalled = true
		return nil, nil
	}

	markRunningCalls := 0
	markContestEndRunningForEndFn = func(contestID int, trackID int, triggerSource string) error {
		markRunningCalls++
		return nil
	}

	if err := runContestEndHookForTrack(9, 8); err != nil {
		t.Fatalf("runContestEndHookForTrack should skip success state: %v", err)
	}
	if builtinCalled {
		t.Fatal("builtin pipeline should not run when state is success")
	}
	if markRunningCalls != 0 {
		t.Fatalf("running marker should not be called when skipped, got %d", markRunningCalls)
	}
}

func TestRunContestEndHookForTrackWarnsWhenFlowNotMounted(t *testing.T) {
	backupContestEndHookDeps(t)

	loggerBackup := log.Logger
	mockLogger := &mockContestEndLogger{}
	log.Logger = mockLogger
	t.Cleanup(func() {
		log.Logger = loggerBackup
	})

	successCalls := 0
	markContestEndSuccessForEndFn = func(contestID int, trackID int, triggerSource string) error {
		successCalls++
		return nil
	}
	resolveFlowChainForEndFn = func(scope string, eventKey string, inputContestID int, inputTrackID int) ([]pgsql.ResolvedFlowChain, error) {
		return nil, pgsql.ErrFlowNotMounted
	}

	err := runContestEndHookForTrackWithSource(7, 9, contestEndTriggerSourceTimer)
	if err != nil {
		t.Fatalf("runContestEndHookForTrackWithSource failed: %v", err)
	}
	if successCalls != 1 {
		t.Fatalf("expected mark success once, got %d", successCalls)
	}

	warned := false
	for _, warn := range mockLogger.warns {
		if strings.Contains(warn, "contest_end_flow_not_mounted") {
			warned = true
			break
		}
	}
	if !warned {
		t.Fatalf("expected missing mount warning, got warns=%v", mockLogger.warns)
	}
}

func TestRunContestEndHookForTrackMarksFailedWhenBuiltinFails(t *testing.T) {
	backupContestEndHookDeps(t)

	markFailed := 0
	markContestEndFailedForEndFn = func(contestID int, trackID int, triggerSource string, lastError string) error {
		markFailed++
		if !strings.Contains(lastError, "boom-upsert") {
			t.Fatalf("unexpected lastError: %s", lastError)
		}
		return nil
	}

	listReviewEventsForEndFn = func(inputTrackID int) ([]model.ReviewEvent, error) {
		return []model.ReviewEvent{{EventID: 1, TrackID: inputTrackID}}, nil
	}
	deleteReviewResultsForEndFn = func(inputEventID int) error { return nil }
	listReviewWorksForEndFn = func(inputEventID int, offset int, limit int) ([]model.Work, error) {
		return []model.Work{{WorkID: 100, TrackID: 2, AuthorID: 3}}, nil
	}
	listReviewsForEndFn = func(inputWorkID int, inputEventID int) ([]model.Review, error) {
		return nil, nil
	}
	listJudgeIDsForEndFn = func(inputEventID int) ([]int, error) {
		return nil, nil
	}
	upsertReviewResultForEndFn = func(inputWorkID int, inputEventID int, reviews map[string]any) (model.ReviewResult, error) {
		return model.ReviewResult{}, errors.New("boom-upsert")
	}
	resolveFlowChainForEndFn = func(scope string, eventKey string, inputContestID int, inputTrackID int) ([]pgsql.ResolvedFlowChain, error) {
		return []pgsql.ResolvedFlowChain{{
			Flow: model.ScriptFlow{FlowKey: "contest-end"},
			Steps: []pgsql.ResolvedFlowStep{
				{
					Step:    model.FlowStep{StepID: 1, StepOrder: 1, StepName: "builtin_regenerate", TimeoutMs: 5000, FailureStrategy: "fail_close", IsEnabled: true},
					Script:  model.ScriptDefinition{ScriptID: 1, ScriptKey: contestEndBuiltinRegenerateScriptKey, Interpreter: scriptflow.InterpreterBuiltinGo, IsEnabled: true},
					Version: model.ScriptVersion{VersionID: 1, ScriptID: 1, RelativePath: contestEndBuiltinRegenerateStepKey, IsActive: true},
				},
			},
		}}, nil
	}

	err := runContestEndHookForTrack(1, 2)
	if err == nil || !strings.Contains(err.Error(), "boom-upsert") {
		t.Fatalf("expected upsert failure, got %v", err)
	}
	if markFailed != 1 {
		t.Fatalf("expected one failed marker call, got %d", markFailed)
	}
}

func TestConvertDocxToPDFValidation(t *testing.T) {
	backupContestEndHookDeps(t)

	if err := convertDocxToPDF(context.Background(), "demo.txt", "demo.pdf"); err == nil {
		t.Fatal("expected source extension validation error")
	}
	if err := convertDocxToPDF(context.Background(), "demo.docx", "demo.txt"); err == nil {
		t.Fatal("expected destination extension validation error")
	}
}

func TestMoveFileForEndCopiesWhenRenameFails(t *testing.T) {
	backupContestEndHookDeps(t)

	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "src.pdf")
	dstPath := filepath.Join(tempDir, "dst.pdf")

	content := []byte("contest-end-pdf")
	if err := os.WriteFile(srcPath, content, 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	renameFileForEndFn = func(oldpath string, newpath string) error {
		return errors.New("rename cross-device")
	}

	if err := moveFileForEnd(srcPath, dstPath); err != nil {
		t.Fatalf("moveFileForEnd should copy when rename fails, got: %v", err)
	}

	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("read destination file failed: %v", err)
	}
	if string(dstContent) != string(content) {
		t.Fatalf("unexpected destination content: got=%q want=%q", string(dstContent), string(content))
	}
}

func TestMoveFileForEndReturnsCopyError(t *testing.T) {
	backupContestEndHookDeps(t)

	tempDir := t.TempDir()
	srcPath := filepath.Join(tempDir, "src.pdf")
	dstPath := filepath.Join(tempDir, "dst.pdf")

	if err := os.WriteFile(srcPath, []byte("payload"), 0o644); err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	renameFileForEndFn = func(oldpath string, newpath string) error {
		return errors.New("rename cross-device")
	}
	copyFileForEndFn = func(dst io.Writer, src io.Reader) (int64, error) {
		return 0, errors.New("copy failed")
	}

	err := moveFileForEnd(srcPath, dstPath)
	if err == nil || !strings.Contains(err.Error(), "copy failed") {
		t.Fatalf("expected copy failure, got: %v", err)
	}
}

func TestResolveSubmissionFileForEndUsesSubmissionRoot(t *testing.T) {
	backupContestEndHookDeps(t)

	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err = os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	newDir := filepath.Join(_const.SubmissionFileRootPath, "7", "9")
	if err = os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("mkdir new dir failed: %v", err)
	}

	expectedPath := filepath.Join(newDir, "101.docx")
	if err = os.WriteFile(expectedPath, []byte("new"), 0o644); err != nil {
		t.Fatalf("write new file failed: %v", err)
	}

	resolvedPath, err := resolveSubmissionFileForEnd(model.Work{WorkID: 101, TrackID: 7, AuthorID: 9})
	if err != nil {
		t.Fatalf("resolveSubmissionFileForEnd failed: %v", err)
	}
	if filepath.Clean(resolvedPath) != filepath.Clean(expectedPath) {
		t.Fatalf("unexpected resolved path: got=%s want=%s", resolvedPath, expectedPath)
	}
}

func TestResolveSubmissionFileForEndLogsDiagnosticsWhenFileMissing(t *testing.T) {
	backupContestEndHookDeps(t)

	tempDir := t.TempDir()
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err = os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})
	loggerBackup := log.Logger
	mockLogger := &mockContestEndLogger{}
	log.Logger = mockLogger
	t.Cleanup(func() {
		log.Logger = loggerBackup
	})

	newDir := filepath.Join(_const.SubmissionFileRootPath, "7", "9")
	if err = os.MkdirAll(newDir, 0o755); err != nil {
		t.Fatalf("mkdir new dir failed: %v", err)
	}
	if err = os.WriteFile(filepath.Join(newDir, "other.docx"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write non-matching file failed: %v", err)
	}
	if err = os.WriteFile(filepath.Join(newDir, "203.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write txt file failed: %v", err)
	}

	_, err = resolveSubmissionFileForEnd(model.Work{WorkID: 202, TrackID: 7, AuthorID: 9})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("resolveSubmissionFileForEnd should return not-exist, got: %v", err)
	}
	if len(mockLogger.warns) == 0 {
		t.Fatal("expected missing-file diagnostics warn log")
	}
	found := false
	for _, warn := range mockLogger.warns {
		if strings.Contains(warn, "contest_end_submission_file_not_found") && strings.Contains(warn, "prefix=202.") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected missing-file diagnostics warning, got warns=%v", mockLogger.warns)
	}
}
