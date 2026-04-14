package system

import (
	"errors"
	"fmt"
	"main/model"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

const (
	contestEndExecutionStatusPending         = "pending"
	contestEndExecutionStatusSuccess         = "success"
	contestEndExecutionStatusRunning         = "running"
	contestEndExecutionStatusReplayRequested = "replay_requested"

	contestEndTriggerSourceSystem  = "system"
	contestEndTriggerSourceStartup = "startup"
	contestEndTriggerSourceTimer   = "timer"
	contestEndTriggerSourceManual  = "manual"
)

var contestEndRunningStaleDuration = 30 * time.Minute

type contestEndExecutionState struct {
	Status         string
	AttemptCount   int
	LastError      string
	LastStartedAt  *time.Time
	LastFinishedAt *time.Time
	TriggerSource  string
	UpdatedAt      time.Time
}

func isContestEndExecutionNotFound(err error) bool {
	if err == nil {
		return false
	}

	parsedErr := uerr.ExtractError(err)
	if errors.Is(parsedErr, gorm.ErrRecordNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
		return true
	}

	return strings.Contains(strings.ToLower(parsedErr.Error()), "record not found")
}

func contestEndStateFromTrack(track model.Track) contestEndExecutionState {
	status := strings.ToLower(strings.TrimSpace(track.ContestEndStatus))
	if status == "" {
		status = contestEndExecutionStatusPending
	}

	source := strings.TrimSpace(track.ContestEndTriggerSource)
	if source == "" {
		source = contestEndTriggerSourceSystem
	}

	return contestEndExecutionState{
		Status:         status,
		AttemptCount:   track.ContestEndAttemptCount,
		LastError:      track.ContestEndLastError,
		LastStartedAt:  track.ContestEndLastStartedAt,
		LastFinishedAt: track.ContestEndLastFinishedAt,
		TriggerSource:  source,
		UpdatedAt:      track.ContestEndUpdatedAt,
	}
}

func shouldExecuteContestEndByState(state contestEndExecutionState, now time.Time) bool {
	status := strings.ToLower(strings.TrimSpace(state.Status))
	switch status {
	case contestEndExecutionStatusSuccess:
		return false
	case contestEndExecutionStatusRunning:
		if state.LastStartedAt == nil {
			return true
		}
		return state.LastStartedAt.Before(now.Add(-contestEndRunningStaleDuration))
	default:
		return true
	}
}

func formatContestEndOptionalTimeForLog(ts *time.Time) string {
	if ts == nil || ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func formatContestEndTimeForLog(ts time.Time) string {
	if ts.IsZero() {
		return "-"
	}
	return ts.UTC().Format(time.RFC3339)
}

func formatContestEndStateForLog(state contestEndExecutionState) string {
	return fmt.Sprintf(
		"status=%s attempt=%d trigger=%s started=%s finished=%s updated=%s last_error=%q",
		strings.TrimSpace(state.Status),
		state.AttemptCount,
		strings.TrimSpace(state.TriggerSource),
		formatContestEndOptionalTimeForLog(state.LastStartedAt),
		formatContestEndOptionalTimeForLog(state.LastFinishedAt),
		formatContestEndTimeForLog(state.UpdatedAt),
		strings.TrimSpace(state.LastError),
	)
}
