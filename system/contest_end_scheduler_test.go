package system

import (
	"errors"
	"main/model"
	"testing"
	"time"
)

func backupContestEndSchedulerHooks(t *testing.T) {
	origGetContestListForScheduleFn := getContestListForScheduleFn
	origGetTracksByContestForScheduleFn := getTracksByContestForScheduleFn
	origGetTrackByIDForScheduleFn := getTrackByIDForScheduleFn
	origGetContestEndExecutionForScheduleFn := getContestEndExecutionForScheduleFn
	origMarkTrackContestEndReplayRequestedForScheduleFn := markTrackContestEndReplayRequestedForScheduleFn
	origMarkContestEndReplayRequestedFn := markContestEndReplayRequestedFn
	origRunContestEndForContestFn := runContestEndForContestFn
	origAfterFuncFn := afterFuncFn
	origRegisterContestEndScheduleFn := registerContestEndScheduleFn
	origCancelContestEndScheduleFn := cancelContestEndScheduleFn
	origRunningStale := contestEndRunningStaleDuration

	t.Cleanup(func() {
		getContestListForScheduleFn = origGetContestListForScheduleFn
		getTracksByContestForScheduleFn = origGetTracksByContestForScheduleFn
		getTrackByIDForScheduleFn = origGetTrackByIDForScheduleFn
		getContestEndExecutionForScheduleFn = origGetContestEndExecutionForScheduleFn
		markTrackContestEndReplayRequestedForScheduleFn = origMarkTrackContestEndReplayRequestedForScheduleFn
		markContestEndReplayRequestedFn = origMarkContestEndReplayRequestedFn
		runContestEndForContestFn = origRunContestEndForContestFn
		afterFuncFn = origAfterFuncFn
		registerContestEndScheduleFn = origRegisterContestEndScheduleFn
		cancelContestEndScheduleFn = origCancelContestEndScheduleFn
		contestEndRunningStaleDuration = origRunningStale

		contestEndScheduleMu.Lock()
		for contestID, timer := range contestEndTimers {
			if timer != nil {
				timer.Stop()
			}
			delete(contestEndTimers, contestID)
		}
		contestEndScheduleMu.Unlock()
	})
}

func TestShouldTriggerContestEndForContest(t *testing.T) {
	backupContestEndSchedulerHooks(t)

	getTracksByContestForScheduleFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 11}, {TrackID: 12}}, nil
	}
	getContestEndExecutionForScheduleFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		if trackID == 11 {
			return contestEndExecutionState{Status: contestEndExecutionStatusSuccess}, nil
		}
		return contestEndExecutionState{Status: "failed"}, nil
	}

	should, err := shouldTriggerContestEndForContest(3)
	if err != nil {
		t.Fatalf("shouldTriggerContestEndForContest failed: %v", err)
	}
	if !should {
		t.Fatal("contest should trigger when at least one track is failed")
	}

	getContestEndExecutionForScheduleFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		return contestEndExecutionState{Status: contestEndExecutionStatusSuccess}, nil
	}

	should, err = shouldTriggerContestEndForContest(3)
	if err != nil {
		t.Fatalf("shouldTriggerContestEndForContest failed: %v", err)
	}
	if should {
		t.Fatal("contest should not trigger when all tracks are success")
	}
}

func TestRegisterContestEndScheduleForExpiredContest(t *testing.T) {
	backupContestEndSchedulerHooks(t)

	called := make(chan string, 1)
	runContestEndForContestFn = func(contestID int, triggerSource string) error {
		called <- triggerSource
		return nil
	}
	getTracksByContestForScheduleFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 21}}, nil
	}

	getContestEndExecutionForScheduleFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		return contestEndExecutionState{Status: contestEndExecutionStatusSuccess}, nil
	}

	registerContestEndSchedule(model.Contest{ContestID: 1, ContestEndDate: time.Now().Add(-2 * time.Minute)})
	select {
	case source := <-called:
		t.Fatalf("should not trigger for success-only contest, got source %s", source)
	case <-time.After(200 * time.Millisecond):
	}

	getContestEndExecutionForScheduleFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		return contestEndExecutionState{Status: "failed"}, nil
	}

	registerContestEndSchedule(model.Contest{ContestID: 1, ContestEndDate: time.Now().Add(-2 * time.Minute)})
	select {
	case source := <-called:
		if source != contestEndTriggerSourceStartup {
			t.Fatalf("unexpected trigger source: %s", source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("expected expired contest to trigger when failed track exists")
	}
}

func TestRequestContestEndReplay(t *testing.T) {
	backupContestEndSchedulerHooks(t)

	getTracksByContestForScheduleFn = func(contestID int) ([]model.Track, error) {
		return []model.Track{{TrackID: 31}, {TrackID: 32}}, nil
	}

	marked := map[int]bool{}
	markContestEndReplayRequestedFn = func(contestID int, trackID int, triggerSource string) error {
		if triggerSource != contestEndTriggerSourceManual {
			t.Fatalf("unexpected replay trigger source: %s", triggerSource)
		}
		marked[trackID] = true
		return nil
	}

	runCalled := make(chan string, 1)
	runContestEndForContestFn = func(contestID int, triggerSource string) error {
		runCalled <- triggerSource
		return nil
	}

	if err := RequestContestEndReplay(5, 0); err != nil {
		t.Fatalf("RequestContestEndReplay failed: %v", err)
	}
	if !marked[31] || !marked[32] {
		t.Fatalf("expected all tracks marked for replay, got %+v", marked)
	}

	select {
	case source := <-runCalled:
		if source != contestEndTriggerSourceManual {
			t.Fatalf("unexpected trigger source: %s", source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("manual replay should trigger contest execution")
	}

	if err := RequestContestEndReplay(5, 99); err == nil {
		t.Fatal("RequestContestEndReplay should reject track outside contest")
	}

	markContestEndReplayRequestedFn = func(contestID int, trackID int, triggerSource string) error {
		return errors.New("mark failed")
	}
	if err := RequestContestEndReplay(5, 31); err == nil {
		t.Fatal("RequestContestEndReplay should return mark error")
	}
}
