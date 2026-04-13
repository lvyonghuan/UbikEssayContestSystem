package admin

import (
	"main/model"
	"strconv"
	"strings"
	"testing"
	"time"
)

func backupReviewSrcHooks(t *testing.T) {
	t.Helper()

	origGetReviewEventByIDDBFn := getReviewEventByIDDBFn
	origGetJudgeByIDDBFn := getJudgeByIDDBFn
	origReplaceReviewEventJudgesDBFn := replaceReviewEventJudgesDBFn
	origCountAssignableWorksForJudgeInEvent := countAssignableWorksForJudgeInEvent
	origListJudgeIDsByReviewEventDBFn := listJudgeIDsByReviewEventDBFn
	origGetReviewWorksByEventDBFn := getReviewWorksByEventDBFn
	origGetAssignableJudgeIDsForWorkInEvent := getAssignableJudgeIDsForWorkInEvent
	origListReviewsByWorkAndEventDBFn := listReviewsByWorkAndEventDBFn
	origCountAssignedWorksForJudgeInEvent := countAssignedWorksForJudgeInEvent
	origCountSubmittedReviewsForJudgeEvent := countSubmittedReviewsForJudgeEvent
	origGetWorkByIDFn := getWorkByIDFn
	origListReviewEventsByTrackIDDBFn := listReviewEventsByTrackIDDBFn
	origCreateActionLogFn := createActionLogFn

	t.Cleanup(func() {
		getReviewEventByIDDBFn = origGetReviewEventByIDDBFn
		getJudgeByIDDBFn = origGetJudgeByIDDBFn
		replaceReviewEventJudgesDBFn = origReplaceReviewEventJudgesDBFn
		countAssignableWorksForJudgeInEvent = origCountAssignableWorksForJudgeInEvent
		listJudgeIDsByReviewEventDBFn = origListJudgeIDsByReviewEventDBFn
		getReviewWorksByEventDBFn = origGetReviewWorksByEventDBFn
		getAssignableJudgeIDsForWorkInEvent = origGetAssignableJudgeIDsForWorkInEvent
		listReviewsByWorkAndEventDBFn = origListReviewsByWorkAndEventDBFn
		countAssignedWorksForJudgeInEvent = origCountAssignedWorksForJudgeInEvent
		countSubmittedReviewsForJudgeEvent = origCountSubmittedReviewsForJudgeEvent
		getWorkByIDFn = origGetWorkByIDFn
		listReviewEventsByTrackIDDBFn = origListReviewEventsByTrackIDDBFn
		createActionLogFn = origCreateActionLogFn
	})
}

func TestAssignReviewEventJudgesSrcRejectsJudgeWithoutAssignableWorks(t *testing.T) {
	backupReviewSrcHooks(t)

	getReviewEventByIDDBFn = func(eventID int) (model.ReviewEvent, error) {
		return model.ReviewEvent{EventID: eventID, TrackID: 1, EventName: "event"}, nil
	}
	getJudgeByIDDBFn = func(judgeID int) (model.Judge, error) {
		return model.Judge{JudgeID: judgeID, JudgeName: "judge"}, nil
	}
	countAssignableWorksForJudgeInEvent = func(judgeID int, eventID int) (int64, error) {
		return 0, nil
	}
	replaceReviewEventJudgesDBFn = func(eventID int, judgeIDs []int) error {
		t.Fatal("replaceReviewEventJudgesDBFn should not be called when assignable count is zero")
		return nil
	}
	createActionLogFn = func(adminID int, resource string, action string, details map[string]interface{}) {}

	err := assignReviewEventJudgesSrc(1, 10, []int{101})
	if err == nil {
		t.Fatal("assignReviewEventJudgesSrc should reject judges with zero assignable works")
	}
	if !strings.Contains(err.Error(), "no assignable works") {
		t.Fatalf("expected no assignable works error, got %v", err)
	}
}

func TestGetReviewEventProgressSrcUsesAssignableJudgeSet(t *testing.T) {
	backupReviewSrcHooks(t)

	getReviewEventByIDDBFn = func(eventID int) (model.ReviewEvent, error) {
		return model.ReviewEvent{EventID: eventID, TrackID: 1, EventName: "event"}, nil
	}
	listJudgeIDsByReviewEventDBFn = func(eventID int) ([]int, error) {
		return []int{1, 2}, nil
	}
	getReviewWorksByEventDBFn = func(eventID int, offset int, limit int) ([]model.Work, error) {
		return []model.Work{{WorkID: 201, TrackID: 1, WorkStatus: "submission_success"}}, nil
	}
	getAssignableJudgeIDsForWorkInEvent = func(eventID int, workID int) ([]int, error) {
		return []int{2}, nil
	}
	listReviewsByWorkAndEventDBFn = func(workID int, eventID int) ([]model.Review, error) {
		return []model.Review{{ReviewID: 1, WorkID: workID, ReviewEventID: eventID, JudgeID: 2}}, nil
	}
	countAssignedWorksForJudgeInEvent = func(judgeID int, eventID int) (int64, error) {
		if judgeID == 1 {
			return 0, nil
		}
		return 1, nil
	}
	countSubmittedReviewsForJudgeEvent = func(judgeID int, eventID int) (int64, error) {
		if judgeID == 1 {
			return 0, nil
		}
		return 1, nil
	}
	getJudgeByIDDBFn = func(judgeID int) (model.Judge, error) {
		return model.Judge{JudgeID: judgeID, JudgeName: "judge-" + strconv.Itoa(judgeID)}, nil
	}

	progress, err := getReviewEventProgressSrc(1)
	if err != nil {
		t.Fatalf("getReviewEventProgressSrc failed: %v", err)
	}
	if progress.TotalWorks != 1 || progress.CompletedWorks != 1 {
		t.Fatalf("unexpected progress totals: %+v", progress)
	}
	if len(progress.JudgeProgress) != 2 {
		t.Fatalf("expected 2 judge progress rows, got %+v", progress.JudgeProgress)
	}
}

func TestGetWorkReviewStatusSrcSkipsStatusMismatchedEvents(t *testing.T) {
	backupReviewSrcHooks(t)

	getWorkByIDFn = func(workID int) (model.Work, error) {
		return model.Work{WorkID: workID, TrackID: 1, WorkStatus: "submission_success"}, nil
	}
	listReviewEventsByTrackIDDBFn = func(trackID int) ([]model.ReviewEvent, error) {
		now := time.Now()
		return []model.ReviewEvent{
			{EventID: 1, TrackID: trackID, EventName: "e1", WorkStatus: "submission_success", StartTime: now, EndTime: now},
			{EventID: 2, TrackID: trackID, EventName: "e2", WorkStatus: "reviewing", StartTime: now, EndTime: now},
		}, nil
	}
	getAssignableJudgeIDsForWorkInEvent = func(eventID int, workID int) ([]int, error) {
		if eventID == 2 {
			t.Fatal("status mismatched event should be skipped")
		}
		return []int{1}, nil
	}
	listReviewsByWorkAndEventDBFn = func(workID int, eventID int) ([]model.Review, error) {
		return []model.Review{{ReviewID: 1, WorkID: workID, ReviewEventID: eventID, JudgeID: 1}}, nil
	}

	status, err := getWorkReviewStatusSrc(1001)
	if err != nil {
		t.Fatalf("getWorkReviewStatusSrc failed: %v", err)
	}
	if status.Summary["eventCount"] != 1 {
		t.Fatalf("expected only one matched event, got %+v", status.Summary)
	}
	if len(status.Events) != 1 || !status.Events[0].Completed {
		t.Fatalf("unexpected work review status events: %+v", status.Events)
	}
}
