package system

import (
	"errors"
	"fmt"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"strconv"
	"sync"
	"time"
)

var (
	contestEndScheduleMu sync.Mutex
	contestEndTimers     = map[int]*time.Timer{}

	getContestListForScheduleFn         = pgsql.GetContestList
	getTracksByContestForScheduleFn     = pgsql.GetTracksByContestID
	getTrackByIDForScheduleFn           = pgsql.GetTrackByID
	getContestEndExecutionForScheduleFn = func(contestID int, trackID int) (contestEndExecutionState, error) {
		track, err := getTrackByIDForScheduleFn(trackID)
		if err != nil {
			return contestEndExecutionState{}, err
		}
		if contestID > 0 && track.ContestID > 0 && track.ContestID != contestID {
			return contestEndExecutionState{}, errors.New("track " + strconv.Itoa(trackID) + " is not under contest " + strconv.Itoa(contestID))
		}
		return contestEndStateFromTrack(track), nil
	}
	markTrackContestEndReplayRequestedForScheduleFn = pgsql.MarkTrackContestEndReplayRequested
	markContestEndReplayRequestedFn                 = func(contestID int, trackID int, triggerSource string) error {
		return markTrackContestEndReplayRequestedForScheduleFn(trackID, triggerSource)
	}
	runContestEndForContestFn = executeContestEndForContestWithSource
	afterFuncFn               = time.AfterFunc

	initContestEndSchedulesFn    = initContestEndSchedules
	registerContestEndScheduleFn = registerContestEndSchedule
	cancelContestEndScheduleFn   = cancelContestEndSchedule
)

func RegisterContestEndSchedule(contest model.Contest) {
	registerContestEndScheduleFn(contest)
}

func CancelContestEndSchedule(contestID int) {
	cancelContestEndScheduleFn(contestID)
}

func RequestContestEndReplay(contestID int, trackID int) error {
	if contestID <= 0 {
		return errors.New("invalid contest id")
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_replay_requested: contestID=%d requestedTrackID=%d",
			contestID,
			trackID,
		))
	}

	tracks, err := getTracksByContestForScheduleFn(contestID)
	if err != nil {
		return err
	}

	targetTrackIDs := make([]int, 0, len(tracks))
	if trackID > 0 {
		matched := false
		for _, track := range tracks {
			if track.TrackID == trackID {
				targetTrackIDs = append(targetTrackIDs, track.TrackID)
				matched = true
				break
			}
		}
		if !matched {
			return errors.New("track " + strconv.Itoa(trackID) + " is not under contest " + strconv.Itoa(contestID))
		}
	} else {
		for _, track := range tracks {
			if track.TrackID <= 0 {
				continue
			}
			targetTrackIDs = append(targetTrackIDs, track.TrackID)
		}
	}

	for _, targetTrackID := range targetTrackIDs {
		err = markContestEndReplayRequestedFn(contestID, targetTrackID, contestEndTriggerSourceManual)
		if err != nil {
			return err
		}
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_replay_marked: contestID=%d trackID=%d source=%s",
				contestID,
				targetTrackID,
				contestEndTriggerSourceManual,
			))
		}
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_replay_dispatch: contestID=%d targets=%v source=%s",
			contestID,
			targetTrackIDs,
			contestEndTriggerSourceManual,
		))
	}

	go triggerContestEndWithSource(contestID, contestEndTriggerSourceManual)

	return nil
}

func initContestEndSchedules() error {
	contests, err := getContestListForScheduleFn()
	if err != nil {
		return err
	}

	for _, contest := range contests {
		registerContestEndSchedule(contest)
	}

	return nil
}

func registerContestEndSchedule(contest model.Contest) {
	if contest.ContestID <= 0 || contest.ContestEndDate.IsZero() {
		if log.Logger != nil {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_schedule_skip_invalid_contest: contestID=%d endDate=%s",
				contest.ContestID,
				formatContestEndTimeForLog(contest.ContestEndDate),
			))
		}
		return
	}

	cancelContestEndSchedule(contest.ContestID)

	delay := time.Until(contest.ContestEndDate)
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_schedule_register: contestID=%d endDate=%s now=%s delay_ms=%d",
			contest.ContestID,
			formatContestEndTimeForLog(contest.ContestEndDate),
			time.Now().UTC().Format(time.RFC3339),
			delay.Milliseconds(),
		))
	}
	if delay <= 0 {
		shouldExecute, err := shouldTriggerContestEndForContest(contest.ContestID)
		if err != nil {
			if log.Logger != nil {
				log.Logger.Warn("Check contest_end replay status failed, fallback to execute: contestID=" + strconv.Itoa(contest.ContestID) + " err=" + err.Error())
			}
			go triggerContestEndWithSource(contest.ContestID, contestEndTriggerSourceStartup)
			return
		}

		if shouldExecute {
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_schedule_immediate_trigger: contestID=%d source=%s",
					contest.ContestID,
					contestEndTriggerSourceStartup,
				))
			}
			go triggerContestEndWithSource(contest.ContestID, contestEndTriggerSourceStartup)
		} else if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_schedule_immediate_skip: contestID=%d reason=no-track-needs-execution",
				contest.ContestID,
			))
		}

		return
	}

	timer := afterFuncFn(delay, func() {
		triggerContestEndWithSource(contest.ContestID, contestEndTriggerSourceTimer)
	})

	contestEndScheduleMu.Lock()
	contestEndTimers[contest.ContestID] = timer
	contestEndScheduleMu.Unlock()
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_schedule_timer_set: contestID=%d fireAt=%s delay_ms=%d",
			contest.ContestID,
			contest.ContestEndDate.UTC().Format(time.RFC3339),
			delay.Milliseconds(),
		))
	}
}

func cancelContestEndSchedule(contestID int) {
	contestEndScheduleMu.Lock()
	timer, exists := contestEndTimers[contestID]
	if exists {
		delete(contestEndTimers, contestID)
	}
	contestEndScheduleMu.Unlock()

	if exists {
		stopped := timer.Stop()
		if log.Logger != nil {
			log.Logger.Debug(fmt.Sprintf(
				"contest_end_schedule_timer_canceled: contestID=%d stopped=%t",
				contestID,
				stopped,
			))
		}
	}
}

func triggerContestEnd(contestID int) {
	triggerContestEndWithSource(contestID, contestEndTriggerSourceSystem)
}

func triggerContestEndWithSource(contestID int, triggerSource string) {
	cancelContestEndSchedule(contestID)
	startedAt := time.Now().UTC()
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_trigger_start: contestID=%d source=%s at=%s",
			contestID,
			triggerSource,
			startedAt.Format(time.RFC3339),
		))
	}

	if err := runContestEndForContestFn(contestID, triggerSource); err != nil {
		if log.Logger != nil {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_trigger_fail: contestID=%d source=%s duration_ms=%d err=%s",
				contestID,
				triggerSource,
				time.Since(startedAt).Milliseconds(),
				err.Error(),
			))
		}
		log.Logger.Error(errors.New("Run contest_end hook failed: " + err.Error()))
		return
	}

	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_trigger_success: contestID=%d source=%s duration_ms=%d",
			contestID,
			triggerSource,
			time.Since(startedAt).Milliseconds(),
		))
	}
}

func shouldTriggerContestEndForContest(contestID int) (bool, error) {
	tracks, err := getTracksByContestForScheduleFn(contestID)
	if err != nil {
		return false, err
	}

	for _, track := range tracks {
		if track.TrackID <= 0 {
			continue
		}

		shouldExecute, err := shouldTriggerContestEndForTrack(contestID, track.TrackID)
		if err != nil {
			return false, err
		}

		if shouldExecute {
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_schedule_contest_decision: contestID=%d trackID=%d shouldExecute=%t",
					contestID,
					track.TrackID,
					shouldExecute,
				))
			}
			return true, nil
		}
	}
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_schedule_contest_decision: contestID=%d shouldExecute=false reason=all-tracks-skipped",
			contestID,
		))
	}

	return false, nil
}

func shouldTriggerContestEndForTrack(contestID int, trackID int) (bool, error) {
	state, err := getContestEndExecutionForScheduleFn(contestID, trackID)
	if err != nil {
		if isContestEndExecutionNotFound(err) {
			if log.Logger != nil {
				log.Logger.Debug(fmt.Sprintf(
					"contest_end_schedule_track_state: contestID=%d trackID=%d state=not-found shouldExecute=true",
					contestID,
					trackID,
				))
			}
			return true, nil
		}
		if log.Logger != nil {
			log.Logger.Warn(fmt.Sprintf(
				"contest_end_schedule_track_state_error: contestID=%d trackID=%d err=%s",
				contestID,
				trackID,
				err.Error(),
			))
		}

		return false, err
	}

	shouldExecute := shouldExecuteContestEndByState(state, time.Now().UTC())
	if log.Logger != nil {
		log.Logger.Debug(fmt.Sprintf(
			"contest_end_schedule_track_state: contestID=%d trackID=%d shouldExecute=%t %s",
			contestID,
			trackID,
			shouldExecute,
			formatContestEndStateForLog(state),
		))
	}

	return shouldExecute, nil
}
