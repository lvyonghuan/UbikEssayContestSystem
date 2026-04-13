package judge

import (
	"errors"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/password"
	"main/util/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

var (
	getJudgeByIDFn                    = pgsql.GetJudgeByID
	getJudgeByNameFn                  = pgsql.GetJudgeByName
	listReviewEventsByJudgeIDFn       = pgsql.ListReviewEventsByJudgeID
	getReviewEventByIDFn              = pgsql.GetReviewEventByID
	isJudgeAssignedToEventFn          = pgsql.IsJudgeAssignedToEvent
	getReviewWorksByEventFn           = pgsql.GetReviewWorksByEventForJudge
	upsertReviewFn                    = pgsql.UpsertReview
	listReviewsByJudgeAndEventFn      = pgsql.ListReviewsByJudgeAndEvent
	getReviewByIDFn                   = pgsql.GetReviewByID
	updateReviewByIDFn                = pgsql.UpdateReviewByID
	getWorkByIDFn                     = pgsql.GetWorkByID
	hasReviewedWorkInOtherEventsFn    = pgsql.HasJudgeReviewedWorkInOtherEventsByWork
	genTokenAndRefreshTokenFn         = token.GenTokenAndRefreshToken
	countAssignedWorksForJudgeInEvent = pgsql.CountAssignedWorksForJudgeInEvent
	countSubmittedReviewsForJudgeFn   = pgsql.CountSubmittedReviewsForJudgeInEvent

	readDirFn = os.ReadDir
)

var (
	errJudgeNotFound       = errors.New("judge not found")
	errReviewEventNotFound = errors.New("review event not found")
	errReviewNotFound      = errors.New("review not found")
	errEventAccessDenied   = errors.New("forbidden: judge is not assigned to this event")
	errWorkFileNotFound    = errors.New("work file not found")
)

type ReviewSubmitInput struct {
	WorkID          int            `json:"workID"`
	EventID         int            `json:"eventID"`
	JudgeScore      float64        `json:"judgeScore"`
	JudgeComment    string         `json:"judgeComment"`
	DimensionScores map[string]any `json:"dimensionScores"`
}

func judgeLoginSrc(judgeID int, judgeName string, plainPassword string) (token.ResponseToken, error) {
	judgeName = strings.TrimSpace(judgeName)
	plainPassword = strings.TrimSpace(plainPassword)
	if plainPassword == "" {
		return token.ResponseToken{}, errors.New("password is required")
	}

	var (
		judge model.Judge
		err   error
	)

	if judgeID > 0 {
		judge, err = getJudgeByIDFn(judgeID)
	} else if judgeName != "" {
		judge, err = getJudgeByNameFn(judgeName)
	} else {
		return token.ResponseToken{}, errors.New("judgeID or judgeName is required")
	}
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return token.ResponseToken{}, errJudgeNotFound
		}
		return token.ResponseToken{}, parsedErr
	}

	if plainPassword != judge.Password && !password.CheckPasswordHash(plainPassword, judge.Password) {
		return token.ResponseToken{}, errors.New("login error")
	}

	tokens, tokenErr := genTokenAndRefreshTokenFn(int64(judge.JudgeID), _const.RoleJudge)
	if tokenErr != nil {
		return token.ResponseToken{}, uerr.ExtractError(tokenErr)
	}

	return tokens, nil
}

func listJudgeEventsSrc(judgeID int, offset int, limit int) ([]model.ReviewEvent, error) {
	events, err := listReviewEventsByJudgeIDFn(judgeID, offset, limit)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	return events, nil
}

func getJudgeEventByIDSrc(judgeID int, eventID int) (model.ReviewEvent, error) {
	if err := ensureJudgeAssignedToEvent(judgeID, eventID); err != nil {
		return model.ReviewEvent{}, err
	}

	event, err := getReviewEventByIDFn(eventID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return model.ReviewEvent{}, errReviewEventNotFound
		}
		return model.ReviewEvent{}, parsedErr
	}

	_, _ = countAssignedWorksForJudgeInEvent(judgeID, eventID)
	_, _ = countSubmittedReviewsForJudgeFn(judgeID, eventID)

	return event, nil
}

func listJudgeEventWorksSrc(judgeID int, eventID int, offset int, limit int) ([]model.Work, error) {
	if err := ensureJudgeAssignedToEvent(judgeID, eventID); err != nil {
		return nil, err
	}

	works, err := getReviewWorksByEventFn(eventID, judgeID, offset, limit)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	return works, nil
}

func submitJudgeReviewSrc(judgeID int, input ReviewSubmitInput) (model.Review, error) {
	if input.WorkID <= 0 || input.EventID <= 0 {
		return model.Review{}, errors.New("invalid workID or eventID")
	}

	event, err := getJudgeEventByIDSrc(judgeID, input.EventID)
	if err != nil {
		return model.Review{}, err
	}

	work, err := getWorkByIDFn(input.WorkID)
	if err != nil {
		return model.Review{}, uerr.ExtractError(err)
	}
	if work.TrackID != event.TrackID {
		return model.Review{}, errors.New("work does not belong to review event track")
	}
	if status := strings.TrimSpace(event.WorkStatus); status != "" && work.WorkStatus != status {
		return model.Review{}, errors.New("work status is not in review scope")
	}

	alreadyReviewed, reviewedErr := hasReviewedWorkInOtherEventsFn(judgeID, input.WorkID, input.EventID)
	if reviewedErr != nil {
		return model.Review{}, uerr.ExtractError(reviewedErr)
	}
	if alreadyReviewed {
		return model.Review{}, errors.New("work already reviewed by judge in another event")
	}

	judgeComment := strings.TrimSpace(input.JudgeComment)
	workReviews := map[string]any{
		"judgeScore":      input.JudgeScore,
		"judgeComment":    judgeComment,
		"dimensionScores": input.DimensionScores,
		"reviewStatus":    "submitted",
		"submittedAt":     time.Now().UTC().Format(time.RFC3339),
	}

	review, err := upsertReviewFn(input.WorkID, input.EventID, judgeID, workReviews)
	if err != nil {
		return model.Review{}, uerr.ExtractError(err)
	}

	return review, nil
}

func listJudgeEventReviewsSrc(judgeID int, eventID int) ([]model.Review, error) {
	if err := ensureJudgeAssignedToEvent(judgeID, eventID); err != nil {
		return nil, err
	}

	reviews, err := listReviewsByJudgeAndEventFn(judgeID, eventID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	return reviews, nil
}

func updateJudgeReviewSrc(judgeID int, reviewID int, judgeScore float64, judgeComment string, dimensionScores map[string]any) (model.Review, error) {
	if reviewID <= 0 {
		return model.Review{}, errors.New("invalid reviewID")
	}

	review, err := getReviewByIDFn(reviewID)
	if err != nil {
		parsedErr := uerr.ExtractError(err)
		if errors.Is(parsedErr, gorm.ErrRecordNotFound) || strings.Contains(strings.ToLower(parsedErr.Error()), "record not found") {
			return model.Review{}, errReviewNotFound
		}
		return model.Review{}, parsedErr
	}
	if review.JudgeID != judgeID {
		return model.Review{}, errors.New("forbidden: can only update own review")
	}

	workReviews := map[string]any{
		"judgeScore":      judgeScore,
		"judgeComment":    strings.TrimSpace(judgeComment),
		"dimensionScores": dimensionScores,
		"reviewStatus":    "updated",
		"updatedAt":       time.Now().UTC().Format(time.RFC3339),
	}

	if err = updateReviewByIDFn(reviewID, judgeID, workReviews); err != nil {
		return model.Review{}, uerr.ExtractError(err)
	}

	review.WorkReviews = workReviews
	return review, nil
}

func getJudgeReviewWorkFilePathSrc(judgeID int, eventID int, workID int) (string, error) {
	if err := ensureJudgeAssignedToEvent(judgeID, eventID); err != nil {
		return "", err
	}

	event, err := getReviewEventByIDFn(eventID)
	if err != nil {
		return "", uerr.ExtractError(err)
	}

	work, err := getWorkByIDFn(workID)
	if err != nil {
		return "", uerr.ExtractError(err)
	}
	if work.TrackID != event.TrackID {
		return "", errors.New("work does not belong to review event track")
	}

	return resolveWorkFilePath(work)
}

func ensureJudgeAssignedToEvent(judgeID int, eventID int) error {
	if judgeID <= 0 || eventID <= 0 {
		return errors.New("invalid judgeID or eventID")
	}

	assigned, err := isJudgeAssignedToEventFn(eventID, judgeID)
	if err != nil {
		return uerr.ExtractError(err)
	}
	if !assigned {
		return errEventAccessDenied
	}

	return nil
}

func resolveWorkFilePath(work model.Work) (string, error) {
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", errWorkFileNotFound
		}
		return "", uerr.ExtractError(err)
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	selectedName := ""
	selectedTime := time.Time{}
	hasDocx := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}

		info, infoErr := entry.Info()
		if infoErr != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(name))
		isDocx := ext == ".docx"

		if isDocx {
			if !hasDocx || selectedName == "" || info.ModTime().After(selectedTime) {
				hasDocx = true
				selectedName = name
				selectedTime = info.ModTime()
			}
			continue
		}

		if hasDocx {
			continue
		}

		if selectedName == "" || info.ModTime().After(selectedTime) {
			selectedName = name
			selectedTime = info.ModTime()
		}
	}

	if selectedName == "" {
		return "", errWorkFileNotFound
	}

	return filepath.Join(dstDir, selectedName), nil
}
