package system

import (
	"context"
	"errors"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/scriptflow"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func backupContestEndHookDeps(t *testing.T) {
	origGetTracksByContestForEndFn := getTracksByContestForEndFn
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
	origExecuteScriptChainForEndFn := executeScriptChainForEndFn

	t.Cleanup(func() {
		getTracksByContestForEndFn = origGetTracksByContestForEndFn
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
		return []pgsql.ResolvedFlowChain{{Flow: model.ScriptFlow{FlowKey: "contest-end"}}}, nil
	}
	executeScriptChainForEndFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		calls = append(calls, "flow")
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	if err := runContestEndHookForTrack(contestID, trackID); err != nil {
		t.Fatalf("runContestEndHookForTrack failed: %v", err)
	}

	if got := strings.Join(calls, ","); got != "delete,upsert,pdf,flow" {
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
		return []pgsql.ResolvedFlowChain{{Flow: model.ScriptFlow{FlowKey: "contest-end"}}}, nil
	}

	flowCalled := false
	executeScriptChainForEndFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		flowCalled = true
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	if err := runContestEndHookForTrack(contestID, trackID); err != nil {
		t.Fatalf("runContestEndHookForTrack failed: %v", err)
	}
	if convertCalls != 0 {
		t.Fatalf("pdf converter should not be called for missing files, got %d", convertCalls)
	}
	if !flowCalled {
		t.Fatal("script flow should still execute when files are missing")
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

	flowCalled := false
	executeScriptChainForEndFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		flowCalled = true
		return scriptflow.ChainResult{Allowed: true}, nil
	}

	err := runContestEndHookForTrack(contestID, trackID)
	if err == nil || !strings.Contains(err.Error(), "upsert failed") {
		t.Fatalf("expected upsert error, got %v", err)
	}
	if flowCalled {
		t.Fatal("script flow should not execute when builtin stage fails")
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
