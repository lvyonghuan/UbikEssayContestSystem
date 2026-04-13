package judge

import (
	"main/model"
	"strings"
	"testing"
)

func backupJudgeSrcHooks(t *testing.T) {
	t.Helper()

	origIsJudgeAssignedToEventFn := isJudgeAssignedToEventFn
	origGetReviewWorksByEventFn := getReviewWorksByEventFn
	origGetReviewEventByIDFn := getReviewEventByIDFn
	origGetWorkByIDFn := getWorkByIDFn
	origHasReviewedWorkInOtherEventsFn := hasReviewedWorkInOtherEventsFn
	origUpsertReviewFn := upsertReviewFn
	origCountAssignedWorksForJudgeInEvent := countAssignedWorksForJudgeInEvent
	origCountSubmittedReviewsForJudgeFn := countSubmittedReviewsForJudgeFn

	t.Cleanup(func() {
		isJudgeAssignedToEventFn = origIsJudgeAssignedToEventFn
		getReviewWorksByEventFn = origGetReviewWorksByEventFn
		getReviewEventByIDFn = origGetReviewEventByIDFn
		getWorkByIDFn = origGetWorkByIDFn
		hasReviewedWorkInOtherEventsFn = origHasReviewedWorkInOtherEventsFn
		upsertReviewFn = origUpsertReviewFn
		countAssignedWorksForJudgeInEvent = origCountAssignedWorksForJudgeInEvent
		countSubmittedReviewsForJudgeFn = origCountSubmittedReviewsForJudgeFn
	})
}

func TestListJudgeEventWorksSrcUsesJudgeScopedQuery(t *testing.T) {
	backupJudgeSrcHooks(t)

	isJudgeAssignedToEventFn = func(eventID int, judgeID int) (bool, error) {
		return true, nil
	}
	capturedJudgeID := 0
	capturedEventID := 0
	getReviewWorksByEventFn = func(eventID int, judgeID int, offset int, limit int) ([]model.Work, error) {
		capturedJudgeID = judgeID
		capturedEventID = eventID
		return []model.Work{{WorkID: 1}}, nil
	}

	works, err := listJudgeEventWorksSrc(7, 11, 0, 20)
	if err != nil {
		t.Fatalf("listJudgeEventWorksSrc failed: %v", err)
	}
	if len(works) != 1 || works[0].WorkID != 1 {
		t.Fatalf("unexpected works: %+v", works)
	}
	if capturedJudgeID != 7 || capturedEventID != 11 {
		t.Fatalf("expected judge-scoped query args, got judge=%d event=%d", capturedJudgeID, capturedEventID)
	}
}

func TestSubmitJudgeReviewSrcRejectsCrossEventRepeat(t *testing.T) {
	backupJudgeSrcHooks(t)

	isJudgeAssignedToEventFn = func(eventID int, judgeID int) (bool, error) {
		return true, nil
	}
	getReviewEventByIDFn = func(eventID int) (model.ReviewEvent, error) {
		return model.ReviewEvent{EventID: eventID, TrackID: 1, WorkStatus: "submission_success"}, nil
	}
	countAssignedWorksForJudgeInEvent = func(judgeID int, eventID int) (int64, error) {
		return 1, nil
	}
	countSubmittedReviewsForJudgeFn = func(judgeID int, eventID int) (int64, error) {
		return 0, nil
	}
	getWorkByIDFn = func(workID int) (model.Work, error) {
		return model.Work{WorkID: workID, TrackID: 1, WorkStatus: "submission_success"}, nil
	}
	hasReviewedWorkInOtherEventsFn = func(judgeID int, workID int, eventID int) (bool, error) {
		return true, nil
	}
	upsertReviewFn = func(workID int, reviewEventID int, judgeID int, workReviews map[string]any) (model.Review, error) {
		t.Fatal("upsertReviewFn should not be called when conflict is detected")
		return model.Review{}, nil
	}

	_, err := submitJudgeReviewSrc(7, ReviewSubmitInput{WorkID: 1001, EventID: 2001, JudgeScore: 90})
	if err == nil {
		t.Fatal("submitJudgeReviewSrc should reject cross-event repeated review")
	}
	if !strings.Contains(err.Error(), "already reviewed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
