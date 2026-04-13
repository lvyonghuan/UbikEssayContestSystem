package system

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	"main/util/log"
	"sync"
	"time"
)

var (
	contestEndScheduleMu sync.Mutex
	contestEndTimers     = map[int]*time.Timer{}

	getContestListForScheduleFn = pgsql.GetContestList
	runContestEndForContestFn   = executeContestEndForContest
	afterFuncFn                 = time.AfterFunc

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
		return
	}

	cancelContestEndSchedule(contest.ContestID)

	delay := time.Until(contest.ContestEndDate)
	if delay <= 0 {
		go triggerContestEnd(contest.ContestID)
		return
	}

	timer := afterFuncFn(delay, func() {
		triggerContestEnd(contest.ContestID)
	})

	contestEndScheduleMu.Lock()
	contestEndTimers[contest.ContestID] = timer
	contestEndScheduleMu.Unlock()
}

func cancelContestEndSchedule(contestID int) {
	contestEndScheduleMu.Lock()
	timer, exists := contestEndTimers[contestID]
	if exists {
		delete(contestEndTimers, contestID)
	}
	contestEndScheduleMu.Unlock()

	if exists {
		timer.Stop()
	}
}

func triggerContestEnd(contestID int) {
	cancelContestEndSchedule(contestID)

	if err := runContestEndForContestFn(contestID); err != nil {
		log.Logger.Error(errors.New("Run contest_end hook failed: " + err.Error()))
	}
}
