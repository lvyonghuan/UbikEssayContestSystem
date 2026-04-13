package pgsql

import (
	"errors"
	"main/model"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"gorm.io/gorm"
)

func GetJudgeByID(judgeID int) (model.Judge, error) {
	var judge model.Judge
	err := postgresDB.Where("judge_id = ?", judgeID).First(&judge).Error
	if err != nil {
		return model.Judge{}, uerr.NewError(err)
	}

	return judge, nil
}

func GetJudgeByName(judgeName string) (model.Judge, error) {
	var judge model.Judge
	err := postgresDB.Where("judge_name = ?", strings.TrimSpace(judgeName)).First(&judge).Error
	if err != nil {
		return model.Judge{}, uerr.NewError(err)
	}

	return judge, nil
}

func CreateJudge(judge *model.Judge) error {
	err := postgresDB.Create(judge).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateJudgeByID(judgeID int, updated *model.Judge) error {
	err := postgresDB.Model(&model.Judge{}).Where("judge_id = ?", judgeID).Updates(updated).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteJudgeByID(judgeID int) error {
	err := postgresDB.Where("judge_id = ?", judgeID).Delete(&model.Judge{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListJudges(offset int, limit int) ([]model.Judge, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	var judges []model.Judge
	err := postgresDB.Order("judge_id ASC").Offset(offset).Limit(limit).Find(&judges).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return judges, nil
}

func CreateReviewEvent(event *model.ReviewEvent) error {
	err := postgresDB.Create(event).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func UpdateReviewEvent(eventID int, updated *model.ReviewEvent) error {
	err := postgresDB.Model(&model.ReviewEvent{}).Where("event_id = ?", eventID).Updates(updated).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func DeleteReviewEvent(eventID int) error {
	err := postgresDB.Where("event_id = ?", eventID).Delete(&model.ReviewEvent{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetReviewEventByID(eventID int) (model.ReviewEvent, error) {
	var event model.ReviewEvent
	err := postgresDB.Where("event_id = ?", eventID).First(&event).Error
	if err != nil {
		return model.ReviewEvent{}, uerr.NewError(err)
	}

	return event, nil
}

func ListReviewEventsByJudgeID(judgeID int, offset int, limit int) ([]model.ReviewEvent, error) {
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	var events []model.ReviewEvent
	err := postgresDB.Table("review_events").
		Select("review_events.event_id", "review_events.track_id", "review_events.event_name", "review_events.work_status", "review_events.start_time", "review_events.end_time").
		Joins("JOIN review_event_judges ON review_event_judges.event_id = review_events.event_id").
		Where("review_event_judges.judge_id = ?", judgeID).
		Order("review_events.start_time DESC").
		Offset(offset).
		Limit(limit).
		Scan(&events).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	for i := range events {
		judgeIDs, listErr := ListJudgeIDsByReviewEvent(events[i].EventID)
		if listErr == nil {
			events[i].JudgeIDs = judgeIDs
		}
	}

	return events, nil
}

func ListReviewEventsByContestID(contestID int) ([]model.ReviewEvent, error) {
	var events []model.ReviewEvent
	err := postgresDB.Table("review_events").
		Select("review_events.event_id", "review_events.track_id", "review_events.event_name", "review_events.work_status", "review_events.start_time", "review_events.end_time").
		Joins("JOIN tracks ON tracks.track_id = review_events.track_id").
		Where("tracks.contest_id = ?", contestID).
		Order("review_events.event_id ASC").
		Scan(&events).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	for i := range events {
		judgeIDs, listErr := ListJudgeIDsByReviewEvent(events[i].EventID)
		if listErr == nil {
			events[i].JudgeIDs = judgeIDs
		}
	}

	return events, nil
}

func ListReviewEventsByTrackID(trackID int) ([]model.ReviewEvent, error) {
	var events []model.ReviewEvent
	err := postgresDB.Where("track_id = ?", trackID).
		Order("event_id ASC").
		Find(&events).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	for i := range events {
		judgeIDs, listErr := ListJudgeIDsByReviewEvent(events[i].EventID)
		if listErr == nil {
			events[i].JudgeIDs = judgeIDs
		}
	}

	return events, nil
}

func ReplaceReviewEventJudges(eventID int, judgeIDs []int) error {
	tx := postgresDB.Begin()
	if tx.Error != nil {
		return uerr.NewError(tx.Error)
	}

	if err := tx.Where("event_id = ?", eventID).Delete(&model.ReviewEventJudge{}).Error; err != nil {
		tx.Rollback()
		return uerr.NewError(err)
	}

	for _, judgeID := range judgeIDs {
		if judgeID <= 0 {
			continue
		}
		row := model.ReviewEventJudge{EventID: eventID, JudgeID: judgeID}
		if err := tx.Create(&row).Error; err != nil {
			tx.Rollback()
			return uerr.NewError(err)
		}
	}

	if err := tx.Commit().Error; err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListJudgeIDsByReviewEvent(eventID int) ([]int, error) {
	type row struct {
		JudgeID int `gorm:"column:judge_id"`
	}
	var rows []row
	err := postgresDB.Table("review_event_judges").
		Select("judge_id").
		Where("event_id = ?", eventID).
		Order("judge_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	judgeIDs := make([]int, 0, len(rows))
	for _, r := range rows {
		judgeIDs = append(judgeIDs, r.JudgeID)
	}

	return judgeIDs, nil
}

func IsJudgeAssignedToEvent(eventID int, judgeID int) (bool, error) {
	var count int64
	err := postgresDB.Model(&model.ReviewEventJudge{}).
		Where("event_id = ? AND judge_id = ?", eventID, judgeID).
		Count(&count).Error
	if err != nil {
		return false, uerr.NewError(err)
	}

	return count > 0, nil
}

func UpdateReviewEventJudgeDeadline(eventID int, judgeID int, deadlineAt *time.Time) error {
	err := postgresDB.Model(&model.ReviewEventJudge{}).
		Where("event_id = ? AND judge_id = ?", eventID, judgeID).
		Update("deadline_at", deadlineAt).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func GetReviewWorksByEvent(eventID int, offset int, limit int) ([]model.Work, error) {
	event, err := GetReviewEventByID(eventID)
	if err != nil {
		return nil, err
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	var works []model.Work
	query := postgresDB.Table("works").
		Select("works.work_id", "works.work_title", "works.track_id", "works.author_id", "works.work_status", "works.work_infos", "authors.author_name", "tracks.track_name").
		Joins("LEFT JOIN authors ON authors.author_id = works.author_id").
		Joins("LEFT JOIN tracks ON tracks.track_id = works.track_id").
		Where("works.track_id = ?", event.TrackID)

	if status := strings.TrimSpace(event.WorkStatus); status != "" {
		query = query.Where("works.work_status = ?", status)
	}

	err = query.Order("works.work_id ASC").Offset(offset).Limit(limit).Scan(&works).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return works, nil
}

func GetReviewWorksByEventForJudge(eventID int, judgeID int, offset int, limit int) ([]model.Work, error) {
	event, err := GetReviewEventByID(eventID)
	if err != nil {
		return nil, err
	}

	if offset < 0 {
		offset = 0
	}
	if limit <= 0 {
		limit = 20
	}

	var works []model.Work
	query := postgresDB.Table("works").
		Select("works.work_id", "works.work_title", "works.track_id", "works.author_id", "works.work_status", "works.work_infos", "authors.author_name", "tracks.track_name").
		Joins("LEFT JOIN authors ON authors.author_id = works.author_id").
		Joins("LEFT JOIN tracks ON tracks.track_id = works.track_id").
		Where("works.track_id = ?", event.TrackID)

	if status := strings.TrimSpace(event.WorkStatus); status != "" {
		query = query.Where("works.work_status = ?", status)
	}

	if judgeID > 0 {
		reviewedWorkIDs, reviewedErr := listJudgeReviewedWorkIDsInTrackExcludingEvent(judgeID, event.TrackID, eventID)
		if reviewedErr != nil {
			return nil, reviewedErr
		}
		if len(reviewedWorkIDs) > 0 {
			query = query.Where("works.work_id NOT IN ?", reviewedWorkIDs)
		}
	}

	err = query.Order("works.work_id ASC").Offset(offset).Limit(limit).Scan(&works).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return works, nil
}

func listJudgeReviewedWorkIDsInTrackExcludingEvent(judgeID int, trackID int, excludeEventID int) ([]int, error) {
	if judgeID <= 0 || trackID <= 0 {
		return []int{}, nil
	}

	type row struct {
		WorkID int `gorm:"column:work_id"`
	}
	rows := make([]row, 0)
	query := postgresDB.Table("reviews").
		Select("DISTINCT reviews.work_id").
		Joins("JOIN works ON works.work_id = reviews.work_id").
		Where("reviews.judge_id = ? AND works.track_id = ?", judgeID, trackID)

	if excludeEventID > 0 {
		query = query.Where("reviews.review_event_id <> ?", excludeEventID)
	}

	err := query.Order("reviews.work_id ASC").Scan(&rows).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	workIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		workIDs = append(workIDs, row.WorkID)
	}

	return workIDs, nil
}

func CountAssignableWorksForJudgeInEvent(judgeID int, eventID int) (int64, error) {
	event, err := GetReviewEventByID(eventID)
	if err != nil {
		return 0, err
	}

	query := postgresDB.Model(&model.Work{}).Where("track_id = ?", event.TrackID)
	if status := strings.TrimSpace(event.WorkStatus); status != "" {
		query = query.Where("work_status = ?", status)
	}

	reviewedWorkIDs, reviewedErr := listJudgeReviewedWorkIDsInTrackExcludingEvent(judgeID, event.TrackID, eventID)
	if reviewedErr != nil {
		return 0, reviewedErr
	}
	if len(reviewedWorkIDs) > 0 {
		query = query.Where("work_id NOT IN ?", reviewedWorkIDs)
	}

	var count int64
	if countErr := query.Count(&count).Error; countErr != nil {
		return 0, uerr.NewError(countErr)
	}

	return count, nil
}

func HasJudgeReviewedWorkInOtherEventsByWork(judgeID int, workID int, eventID int) (bool, error) {
	if judgeID <= 0 || workID <= 0 {
		return false, nil
	}

	query := postgresDB.Model(&model.Review{}).
		Where("judge_id = ? AND work_id = ?", judgeID, workID)
	if eventID > 0 {
		query = query.Where("review_event_id <> ?", eventID)
	}

	var count int64
	err := query.Count(&count).Error
	if err != nil {
		return false, uerr.NewError(err)
	}

	return count > 0, nil
}

func GetAssignableJudgeIDsForWorkInEvent(eventID int, workID int) ([]int, error) {
	event, err := GetReviewEventByID(eventID)
	if err != nil {
		return nil, err
	}

	work, workErr := GetWorkByID(workID)
	if workErr != nil {
		return nil, workErr
	}
	if work.TrackID != event.TrackID {
		return []int{}, nil
	}
	if status := strings.TrimSpace(event.WorkStatus); status != "" && work.WorkStatus != status {
		return []int{}, nil
	}

	type row struct {
		JudgeID int `gorm:"column:judge_id"`
	}
	rows := make([]row, 0)
	err = postgresDB.Table("review_event_judges AS rej").
		Select("rej.judge_id").
		Where("rej.event_id = ?", eventID).
		Where("NOT EXISTS (SELECT 1 FROM reviews r WHERE r.judge_id = rej.judge_id AND r.work_id = ? AND r.review_event_id <> ?)", workID, eventID).
		Order("rej.judge_id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	judgeIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		judgeIDs = append(judgeIDs, row.JudgeID)
	}

	return judgeIDs, nil
}

func UpsertReview(workID int, reviewEventID int, judgeID int, workReviews map[string]any) (model.Review, error) {
	var review model.Review
	err := postgresDB.Where("work_id = ? AND review_event_id = ? AND judge_id = ?", workID, reviewEventID, judgeID).First(&review).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.Review{}, uerr.NewError(err)
		}

		review = model.Review{
			WorkID:        workID,
			ReviewEventID: reviewEventID,
			JudgeID:       judgeID,
			WorkReviews:   workReviews,
		}
		if createErr := postgresDB.Create(&review).Error; createErr != nil {
			return model.Review{}, uerr.NewError(createErr)
		}

		return review, nil
	}

	review.WorkReviews = workReviews
	if saveErr := postgresDB.Model(&review).Update("work_reviews", review.WorkReviews).Error; saveErr != nil {
		return model.Review{}, uerr.NewError(saveErr)
	}

	return review, nil
}

func GetReviewByID(reviewID int) (model.Review, error) {
	var review model.Review
	err := postgresDB.Where("review_id = ?", reviewID).First(&review).Error
	if err != nil {
		return model.Review{}, uerr.NewError(err)
	}

	return review, nil
}

func UpdateReviewByID(reviewID int, judgeID int, workReviews map[string]any) error {
	err := postgresDB.Model(&model.Review{}).
		Where("review_id = ? AND judge_id = ?", reviewID, judgeID).
		Update("work_reviews", workReviews).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListReviewsByJudgeAndEvent(judgeID int, eventID int) ([]model.Review, error) {
	var reviews []model.Review
	err := postgresDB.Where("judge_id = ? AND review_event_id = ?", judgeID, eventID).
		Order("review_id ASC").
		Find(&reviews).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return reviews, nil
}

func ListReviewsByWorkAndEvent(workID int, eventID int) ([]model.Review, error) {
	var reviews []model.Review
	err := postgresDB.Where("work_id = ? AND review_event_id = ?", workID, eventID).
		Order("review_id ASC").
		Find(&reviews).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return reviews, nil
}

func DeleteReviewsByEventID(eventID int) error {
	err := postgresDB.Where("review_event_id = ?", eventID).Delete(&model.Review{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func CountSubmittedReviewsForJudgeInEvent(judgeID int, eventID int) (int64, error) {
	var count int64
	err := postgresDB.Model(&model.Review{}).
		Where("judge_id = ? AND review_event_id = ?", judgeID, eventID).
		Count(&count).Error
	if err != nil {
		return 0, uerr.NewError(err)
	}

	return count, nil
}

func CountAssignedWorksForJudgeInEvent(judgeID int, eventID int) (int64, error) {
	assigned, assignErr := IsJudgeAssignedToEvent(eventID, judgeID)
	if assignErr != nil {
		return 0, assignErr
	}
	if !assigned {
		return 0, nil
	}

	return CountAssignableWorksForJudgeInEvent(judgeID, eventID)
}

func UpsertReviewResult(workID int, reviewEventID int, reviews map[string]any) (model.ReviewResult, error) {
	var result model.ReviewResult
	err := postgresDB.Where("work_id = ? AND review_event_id = ?", workID, reviewEventID).First(&result).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return model.ReviewResult{}, uerr.NewError(err)
		}

		result = model.ReviewResult{
			WorkID:        workID,
			ReviewEventID: reviewEventID,
			Reviews:       reviews,
		}
		if createErr := postgresDB.Create(&result).Error; createErr != nil {
			return model.ReviewResult{}, uerr.NewError(createErr)
		}

		return result, nil
	}

	result.Reviews = reviews
	if saveErr := postgresDB.Model(&result).Update("reviews", result.Reviews).Error; saveErr != nil {
		return model.ReviewResult{}, uerr.NewError(saveErr)
	}

	return result, nil
}

func ListReviewResultsByEventID(eventID int) ([]model.ReviewResult, error) {
	var results []model.ReviewResult
	err := postgresDB.Where("review_event_id = ?", eventID).
		Order("result_id ASC").
		Find(&results).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return results, nil
}

func ListReviewResultsByWorkID(workID int) ([]model.ReviewResult, error) {
	var results []model.ReviewResult
	err := postgresDB.Where("work_id = ?", workID).
		Order("result_id ASC").
		Find(&results).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return results, nil
}

func DeleteReviewResultsByEventID(eventID int) error {
	err := postgresDB.Where("review_event_id = ?", eventID).Delete(&model.ReviewResult{}).Error
	if err != nil {
		return uerr.NewError(err)
	}

	return nil
}

func ListReviewResultsByTrackID(trackID int) ([]model.ReviewResult, error) {
	var results []model.ReviewResult
	err := postgresDB.Table("review_results").
		Select("review_results.result_id", "review_results.work_id", "review_results.review_event_id", "review_results.reviews").
		Joins("JOIN review_events ON review_events.event_id = review_results.review_event_id").
		Where("review_events.track_id = ?", trackID).
		Order("review_results.result_id ASC").
		Scan(&results).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	return results, nil
}

func GetReviewResultByID(resultID int) (model.ReviewResult, error) {
	var result model.ReviewResult
	err := postgresDB.Where("result_id = ?", resultID).First(&result).Error
	if err != nil {
		return model.ReviewResult{}, uerr.NewError(err)
	}

	return result, nil
}

func GetReviewResultByWorkAndEvent(workID int, eventID int) (model.ReviewResult, error) {
	var result model.ReviewResult
	err := postgresDB.Where("work_id = ? AND review_event_id = ?", workID, eventID).First(&result).Error
	if err != nil {
		return model.ReviewResult{}, uerr.NewError(err)
	}

	return result, nil
}

func GetDistinctWorkStatusesByTrackID(trackID int) ([]string, error) {
	type statusRow struct {
		Status string `gorm:"column:work_status"`
	}
	rows := make([]statusRow, 0)
	err := postgresDB.Table("works").
		Select("DISTINCT work_status").
		Where("track_id = ?", trackID).
		Order("work_status ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, uerr.NewError(err)
	}

	statuses := make([]string, 0, len(rows))
	for _, row := range rows {
		if strings.TrimSpace(row.Status) == "" {
			continue
		}
		statuses = append(statuses, row.Status)
	}

	return statuses, nil
}

func GetJudgesByJudgeID(judge model.Judge) error {
	loaded, err := GetJudgeByID(judge.JudgeID)
	if err != nil {
		return err
	}
	judge = loaded
	return nil
}

func CreateJudges(judge model.Judge) error {
	return CreateJudge(&judge)
}

func UpdateJudges(judge model.Judge) error {
	return UpdateJudgeByID(judge.JudgeID, &judge)
}

func DeleteJudges(judge model.Judge) error {
	return DeleteJudgeByID(judge.JudgeID)
}

func GetAllJudges() ([]model.Judge, error) {
	return ListJudges(0, 0)
}
