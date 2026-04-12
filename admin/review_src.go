package admin

import (
	"errors"
	"fmt"
	"main/database/pgsql"
	"main/model"
	_const "main/util/const"
	"main/util/password"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
	"github.com/xuri/excelize/v2"
)

var (
	createJudgeDBFn                    = pgsql.CreateJudge
	updateJudgeByIDDBFn                = pgsql.UpdateJudgeByID
	deleteJudgeByIDDBFn                = pgsql.DeleteJudgeByID
	getJudgeByIDDBFn                   = pgsql.GetJudgeByID
	listJudgesDBFn                     = pgsql.ListJudges
	createReviewEventDBFn              = pgsql.CreateReviewEvent
	updateReviewEventDBFn              = pgsql.UpdateReviewEvent
	deleteReviewEventDBFn              = pgsql.DeleteReviewEvent
	getReviewEventByIDDBFn             = pgsql.GetReviewEventByID
	listReviewEventsByTrackIDDBFn      = pgsql.ListReviewEventsByTrackID
	listReviewEventsByContestIDDBFn    = pgsql.ListReviewEventsByContestID
	replaceReviewEventJudgesDBFn       = pgsql.ReplaceReviewEventJudges
	listJudgeIDsByReviewEventDBFn      = pgsql.ListJudgeIDsByReviewEvent
	updateReviewEventJudgeDeadlineDBFn = pgsql.UpdateReviewEventJudgeDeadline
	listReviewsByWorkAndEventDBFn      = pgsql.ListReviewsByWorkAndEvent
	listReviewResultsByWorkIDDBFn      = pgsql.ListReviewResultsByWorkID
	upsertReviewResultDBFn             = pgsql.UpsertReviewResult
	deleteReviewResultsByEventIDDBFn   = pgsql.DeleteReviewResultsByEventID
	listReviewResultsByTrackIDDBFn     = pgsql.ListReviewResultsByTrackID
	getReviewWorksByEventDBFn          = pgsql.GetReviewWorksByEvent
	countAssignedWorksForJudgeInEvent  = pgsql.CountAssignedWorksForJudgeInEvent
	countSubmittedReviewsForJudgeEvent = pgsql.CountSubmittedReviewsForJudgeInEvent
	getContestListDBFn                 = pgsql.GetContestList
	getTrackListDBFn                   = pgsql.GetTrackList
	getDistinctTrackStatusesDBFn       = pgsql.GetDistinctWorkStatusesByTrackID

	newExcelFileFn = excelize.NewFile
)

type dashboardOverview struct {
	TrackSubmissionCount   map[string]int `json:"trackSubmissionCount"`
	ParticipatingAuthors   int            `json:"participatingAuthors"`
	CompletedJudgeTasks    int            `json:"completedJudgeTasks"`
	TotalTrackJudges       int            `json:"totalTrackJudges"`
	CompletedReviewedWorks int            `json:"completedReviewedWorks"`
}

type contestTrackStatusStat struct {
	TrackID       int            `json:"trackID"`
	TrackName     string         `json:"trackName"`
	TotalWorks    int            `json:"totalWorks"`
	StatusCounts  map[string]int `json:"statusCounts"`
	TotalAuthors  int            `json:"totalAuthors"`
	DistinctState []string       `json:"distinctStates"`
}

type contestDailySubmissionStat struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type judgeProgressStat struct {
	JudgeID        int     `json:"judgeID"`
	JudgeName      string  `json:"judgeName"`
	AssignedCount  int     `json:"assignedCount"`
	SubmittedCount int     `json:"submittedCount"`
	CompletionRate float64 `json:"completionRate"`
}

type reviewEventProgress struct {
	EventID          int             `json:"eventID"`
	EventName        string          `json:"eventName"`
	TrackID          int             `json:"trackID"`
	AssignedJudgeIDs []int           `json:"assignedJudgeIDs"`
	TotalWorks       int             `json:"totalWorks"`
	CompletedWorks   int             `json:"completedWorks"`
	JudgeProgress    []judgeProgress `json:"judgeProgress"`
}

type judgeProgress struct {
	JudgeID        int     `json:"judgeID"`
	JudgeName      string  `json:"judgeName"`
	AssignedCount  int     `json:"assignedCount"`
	SubmittedCount int     `json:"submittedCount"`
	CompletionRate float64 `json:"completionRate"`
}

type workReviewStatus struct {
	WorkID  int                  `json:"workID"`
	Events  []workEventReview    `json:"events"`
	Summary map[string]int       `json:"summary"`
	Meta    map[string]time.Time `json:"meta,omitempty"`
}

type workEventReview struct {
	EventID          int    `json:"eventID"`
	EventName        string `json:"eventName"`
	AssignedJudges   int    `json:"assignedJudges"`
	SubmittedReviews int    `json:"submittedReviews"`
	Completed        bool   `json:"completed"`
}

type trackRankItem struct {
	WorkID      int     `json:"workID"`
	WorkTitle   string  `json:"workTitle"`
	AuthorID    int     `json:"authorID"`
	AuthorName  string  `json:"authorName"`
	FinalScore  float64 `json:"finalScore"`
	ReviewCount int     `json:"reviewCount"`
}

type judgeAccountInput struct {
	JudgeName string `json:"judgeName"`
	Password  string `json:"password"`
}

type reviewEventInput struct {
	TrackID    int       `json:"trackID"`
	EventName  string    `json:"eventName"`
	WorkStatus string    `json:"workStatus"`
	StartTime  time.Time `json:"startTime"`
	EndTime    time.Time `json:"endTime"`
}

func createJudgeAccountSrc(adminID int, input judgeAccountInput) (model.Judge, error) {
	input.JudgeName = strings.TrimSpace(input.JudgeName)
	input.Password = strings.TrimSpace(input.Password)
	if input.JudgeName == "" || input.Password == "" {
		return model.Judge{}, errors.New("judgeName and password are required")
	}

	hashed, err := password.HashPassword(input.Password)
	if err != nil {
		return model.Judge{}, uerr.ExtractError(err)
	}

	judge := model.Judge{JudgeName: input.JudgeName, Password: hashed}
	if err = createJudgeDBFn(&judge); err != nil {
		return model.Judge{}, uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Judges, _const.Create,
		genDetails([]string{"judge_id", "judge_name"}, []string{strconv.Itoa(judge.JudgeID), judge.JudgeName}))

	return judge, nil
}

func batchCreateJudgeAccountsSrc(adminID int, inputs []judgeAccountInput) ([]model.Judge, error) {
	created := make([]model.Judge, 0, len(inputs))
	for _, input := range inputs {
		judge, err := createJudgeAccountSrc(adminID, input)
		if err != nil {
			return created, err
		}
		created = append(created, judge)
	}

	return created, nil
}

func updateJudgeAccountSrc(adminID int, judgeID int, input judgeAccountInput) error {
	if judgeID <= 0 {
		return errors.New("invalid judge_id")
	}

	updates := model.Judge{}
	if name := strings.TrimSpace(input.JudgeName); name != "" {
		updates.JudgeName = name
	}
	if pwd := strings.TrimSpace(input.Password); pwd != "" {
		hashed, err := password.HashPassword(pwd)
		if err != nil {
			return uerr.ExtractError(err)
		}
		updates.Password = hashed
	}
	if strings.TrimSpace(updates.JudgeName) == "" && strings.TrimSpace(updates.Password) == "" {
		return errors.New("nothing to update")
	}

	if err := updateJudgeByIDDBFn(judgeID, &updates); err != nil {
		return uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Judges, _const.Update,
		genDetails([]string{"judge_id"}, []string{strconv.Itoa(judgeID)}))

	return nil
}

func deleteJudgeAccountSrc(adminID int, judgeID int) error {
	if judgeID <= 0 {
		return errors.New("invalid judge_id")
	}

	if err := deleteJudgeByIDDBFn(judgeID); err != nil {
		return uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Judges, _const.Delete,
		genDetails([]string{"judge_id"}, []string{strconv.Itoa(judgeID)}))

	return nil
}

func createReviewEventSrc(adminID int, input reviewEventInput) (model.ReviewEvent, error) {
	if input.TrackID <= 0 {
		return model.ReviewEvent{}, errors.New("invalid trackID")
	}
	input.EventName = strings.TrimSpace(input.EventName)
	if input.EventName == "" {
		return model.ReviewEvent{}, errors.New("eventName is required")
	}
	if !input.EndTime.IsZero() && !input.StartTime.IsZero() && input.EndTime.Before(input.StartTime) {
		return model.ReviewEvent{}, errors.New("endTime must be after startTime")
	}

	event := model.ReviewEvent{
		TrackID:    input.TrackID,
		EventName:  input.EventName,
		WorkStatus: strings.TrimSpace(input.WorkStatus),
		StartTime:  input.StartTime,
		EndTime:    input.EndTime,
	}
	if err := createReviewEventDBFn(&event); err != nil {
		return model.ReviewEvent{}, uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Reviews, _const.Create,
		genDetails([]string{"event_id", "event_name"}, []string{strconv.Itoa(event.EventID), event.EventName}))

	return event, nil
}

func updateReviewEventSrc(adminID int, eventID int, input reviewEventInput) error {
	if eventID <= 0 {
		return errors.New("invalid event_id")
	}

	updates := model.ReviewEvent{}
	if input.TrackID > 0 {
		updates.TrackID = input.TrackID
	}
	if name := strings.TrimSpace(input.EventName); name != "" {
		updates.EventName = name
	}
	if status := strings.TrimSpace(input.WorkStatus); status != "" {
		updates.WorkStatus = status
	}
	if !input.StartTime.IsZero() {
		updates.StartTime = input.StartTime
	}
	if !input.EndTime.IsZero() {
		updates.EndTime = input.EndTime
	}

	if err := updateReviewEventDBFn(eventID, &updates); err != nil {
		return uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Reviews, _const.Update,
		genDetails([]string{"event_id"}, []string{strconv.Itoa(eventID)}))

	return nil
}

func assignReviewEventJudgesSrc(adminID int, eventID int, judgeIDs []int) error {
	if eventID <= 0 {
		return errors.New("invalid event_id")
	}

	cleanIDs := make([]int, 0, len(judgeIDs))
	seen := map[int]struct{}{}
	for _, judgeID := range judgeIDs {
		if judgeID <= 0 {
			continue
		}
		if _, ok := seen[judgeID]; ok {
			continue
		}
		if _, err := getJudgeByIDDBFn(judgeID); err != nil {
			return uerr.ExtractError(err)
		}
		seen[judgeID] = struct{}{}
		cleanIDs = append(cleanIDs, judgeID)
	}

	if err := replaceReviewEventJudgesDBFn(eventID, cleanIDs); err != nil {
		return uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Reviews, _const.Update,
		genDetails([]string{"event_id", "judge_count"}, []string{strconv.Itoa(eventID), strconv.Itoa(len(cleanIDs))}))

	return nil
}

func deleteReviewEventSrc(adminID int, eventID int) error {
	if eventID <= 0 {
		return errors.New("invalid event_id")
	}

	if err := deleteReviewEventDBFn(eventID); err != nil {
		return uerr.ExtractError(err)
	}

	createActionLogFn(adminID, _const.Reviews, _const.Delete,
		genDetails([]string{"event_id"}, []string{strconv.Itoa(eventID)}))

	return nil
}

func getReviewEventProgressSrc(eventID int) (reviewEventProgress, error) {
	event, err := getReviewEventByIDDBFn(eventID)
	if err != nil {
		return reviewEventProgress{}, uerr.ExtractError(err)
	}

	judgeIDs, err := listJudgeIDsByReviewEventDBFn(eventID)
	if err != nil {
		return reviewEventProgress{}, uerr.ExtractError(err)
	}

	works, err := getReviewWorksByEventDBFn(eventID, 0, 1000000)
	if err != nil {
		return reviewEventProgress{}, uerr.ExtractError(err)
	}

	completedWorks := 0
	for _, work := range works {
		reviews, reviewErr := listReviewsByWorkAndEventDBFn(work.WorkID, eventID)
		if reviewErr != nil {
			return reviewEventProgress{}, uerr.ExtractError(reviewErr)
		}
		submittedJudgeSet := map[int]struct{}{}
		for _, review := range reviews {
			submittedJudgeSet[review.JudgeID] = struct{}{}
		}
		if len(judgeIDs) > 0 && len(submittedJudgeSet) >= len(judgeIDs) {
			completedWorks++
		}
	}

	judgeProgresses := make([]judgeProgress, 0, len(judgeIDs))
	for _, judgeID := range judgeIDs {
		assignedCount, _ := countAssignedWorksForJudgeInEvent(judgeID, eventID)
		submittedCount, _ := countSubmittedReviewsForJudgeEvent(judgeID, eventID)

		judgeName := ""
		if judge, judgeErr := getJudgeByIDDBFn(judgeID); judgeErr == nil {
			judgeName = judge.JudgeName
		}

		rate := 0.0
		if assignedCount > 0 {
			rate = float64(submittedCount) / float64(assignedCount)
		}
		judgeProgresses = append(judgeProgresses, judgeProgress{
			JudgeID:        judgeID,
			JudgeName:      judgeName,
			AssignedCount:  int(assignedCount),
			SubmittedCount: int(submittedCount),
			CompletionRate: rate,
		})
	}

	return reviewEventProgress{
		EventID:          event.EventID,
		EventName:        event.EventName,
		TrackID:          event.TrackID,
		AssignedJudgeIDs: judgeIDs,
		TotalWorks:       len(works),
		CompletedWorks:   completedWorks,
		JudgeProgress:    judgeProgresses,
	}, nil
}

func listTrackStatusesSrc(trackID int) ([]string, error) {
	if trackID <= 0 {
		return nil, errors.New("invalid track_id")
	}

	statuses, err := getDistinctTrackStatusesDBFn(trackID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	return statuses, nil
}

func getWorkReviewStatusSrc(workID int) (workReviewStatus, error) {
	work, err := getWorkByIDFn(workID)
	if err != nil {
		return workReviewStatus{}, uerr.ExtractError(err)
	}

	events, err := listReviewEventsByTrackIDDBFn(work.TrackID)
	if err != nil {
		return workReviewStatus{}, uerr.ExtractError(err)
	}

	items := make([]workEventReview, 0, len(events))
	summary := map[string]int{
		"eventCount":     0,
		"completedEvents": 0,
	}
	for _, event := range events {
		judgeIDs, judgeErr := listJudgeIDsByReviewEventDBFn(event.EventID)
		if judgeErr != nil {
			return workReviewStatus{}, uerr.ExtractError(judgeErr)
		}
		reviews, reviewErr := listReviewsByWorkAndEventDBFn(workID, event.EventID)
		if reviewErr != nil {
			return workReviewStatus{}, uerr.ExtractError(reviewErr)
		}
		submittedJudgeSet := map[int]struct{}{}
		for _, review := range reviews {
			submittedJudgeSet[review.JudgeID] = struct{}{}
		}
		completed := len(judgeIDs) > 0 && len(submittedJudgeSet) >= len(judgeIDs)
		if completed {
			summary["completedEvents"]++
		}
		summary["eventCount"]++

		items = append(items, workEventReview{
			EventID:          event.EventID,
			EventName:        event.EventName,
			AssignedJudges:   len(judgeIDs),
			SubmittedReviews: len(submittedJudgeSet),
			Completed:        completed,
		})
	}

	return workReviewStatus{WorkID: workID, Events: items, Summary: summary}, nil
}

func getWorkReviewResultsSrc(workID int) ([]model.ReviewResult, error) {
	results, err := listReviewResultsByWorkIDDBFn(workID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	return results, nil
}

func regenerateWorkReviewResultsSrc(workID int) ([]model.ReviewResult, error) {
	work, err := getWorkByIDFn(workID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	events, err := listReviewEventsByTrackIDDBFn(work.TrackID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	regenerated := make([]model.ReviewResult, 0, len(events))
	for _, event := range events {
		if status := strings.TrimSpace(event.WorkStatus); status != "" && work.WorkStatus != status {
			continue
		}
		result, genErr := generateReviewResultForWorkAndEvent(workID, event.EventID)
		if genErr != nil {
			return nil, genErr
		}
		regenerated = append(regenerated, result)
	}

	return regenerated, nil
}

func rankTrackWorksSrc(trackID int) ([]trackRankItem, error) {
	results, err := listReviewResultsByTrackIDDBFn(trackID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	ranks := make([]trackRankItem, 0, len(results))
	for _, result := range results {
		work, workErr := getWorkByIDFn(result.WorkID)
		if workErr != nil {
			continue
		}
		author := model.Author{AuthorID: work.AuthorID}
		_ = getAuthorByIDFn(&author)

		ranks = append(ranks, trackRankItem{
			WorkID:      work.WorkID,
			WorkTitle:   work.WorkTitle,
			AuthorID:    work.AuthorID,
			AuthorName:  author.AuthorName,
			FinalScore:  getFloatValue(result.Reviews, "finalScore"),
			ReviewCount: int(getFloatValue(result.Reviews, "reviewCount")),
		})
	}

	sort.SliceStable(ranks, func(i, j int) bool {
		if ranks[i].FinalScore == ranks[j].FinalScore {
			return ranks[i].WorkID < ranks[j].WorkID
		}
		return ranks[i].FinalScore > ranks[j].FinalScore
	})

	return ranks, nil
}

func exportTrackReviewExcelSrc(trackID int) (string, error) {
	ranks, err := rankTrackWorksSrc(trackID)
	if err != nil {
		return "", err
	}

	excel := newExcelFileFn()
	sheet := excel.GetSheetName(0)
	headers := []string{"作品", "作者", "作者邮箱", "最终分数", "评语", "评委评分"}
	for idx, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		_ = excel.SetCellValue(sheet, cell, header)
	}

	for i, item := range ranks {
		row := i + 2
		work, workErr := getWorkByIDFn(item.WorkID)
		if workErr != nil {
			continue
		}
		author := model.Author{AuthorID: work.AuthorID}
		_ = getAuthorByIDFn(&author)

		results, _ := listReviewResultsByWorkIDDBFn(work.WorkID)
		comments := ""
		judgeScores := ""
		if len(results) > 0 {
			last := results[len(results)-1]
			comments = strings.TrimSpace(getStringValue(last.Reviews, "comments"))
			judgeScores = formatJudgeScores(last.Reviews["judgeScores"])
		}

		vals := []any{item.WorkTitle, author.AuthorName, author.AuthorEmail, item.FinalScore, comments, judgeScores}
		for col, val := range vals {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = excel.SetCellValue(sheet, cell, val)
		}
	}

	tmpDir := filepath.Join(os.TempDir(), "ubik_exports")
	if err = os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return "", err
	}

	outputPath := filepath.Join(tmpDir, fmt.Sprintf("track_%d_review_%d.xlsx", trackID, time.Now().UnixNano()))
	if err = excel.SaveAs(outputPath); err != nil {
		return "", err
	}

	return outputPath, nil
}

func getDashboardOverviewSrc() (dashboardOverview, error) {
	contests, err := getContestListDBFn()
	if err != nil {
		return dashboardOverview{}, uerr.ExtractError(err)
	}

	trackSubmissionCount := map[string]int{}
	authorSet := map[int]struct{}{}
	totalTrackJudgesSet := map[int]struct{}{}
	completedJudgeTasksSet := map[int]struct{}{}
	completedReviewedWorks := 0

	for _, contest := range contests {
		tracks, trackErr := getTrackListDBFn(contest.ContestID)
		if trackErr != nil {
			return dashboardOverview{}, uerr.ExtractError(trackErr)
		}
		for _, track := range tracks {
			works, worksErr := getWorksByTrackFn(track.TrackID)
			if worksErr != nil {
				return dashboardOverview{}, worksErr
			}
			trackSubmissionCount[strconv.Itoa(track.TrackID)] = len(works)
			for _, work := range works {
				authorSet[work.AuthorID] = struct{}{}
			}

			events, eventsErr := listReviewEventsByTrackIDDBFn(track.TrackID)
			if eventsErr != nil {
				return dashboardOverview{}, uerr.ExtractError(eventsErr)
			}
			for _, event := range events {
				judgeIDs, judgeErr := listJudgeIDsByReviewEventDBFn(event.EventID)
				if judgeErr != nil {
					return dashboardOverview{}, uerr.ExtractError(judgeErr)
				}
				for _, judgeID := range judgeIDs {
					totalTrackJudgesSet[judgeID] = struct{}{}
					assigned, _ := countAssignedWorksForJudgeInEvent(judgeID, event.EventID)
					submitted, _ := countSubmittedReviewsForJudgeEvent(judgeID, event.EventID)
					if assigned > 0 && submitted >= assigned {
						completedJudgeTasksSet[judgeID] = struct{}{}
					}
				}

				eventWorks, ewErr := getReviewWorksByEventDBFn(event.EventID, 0, 1000000)
				if ewErr != nil {
					return dashboardOverview{}, uerr.ExtractError(ewErr)
				}
				for _, work := range eventWorks {
					reviews, reviewErr := listReviewsByWorkAndEventDBFn(work.WorkID, event.EventID)
					if reviewErr != nil {
						return dashboardOverview{}, uerr.ExtractError(reviewErr)
					}
					submittedJudges := map[int]struct{}{}
					for _, review := range reviews {
						submittedJudges[review.JudgeID] = struct{}{}
					}
					if len(judgeIDs) > 0 && len(submittedJudges) >= len(judgeIDs) {
						completedReviewedWorks++
					}
				}
			}
		}
	}

	return dashboardOverview{
		TrackSubmissionCount:   trackSubmissionCount,
		ParticipatingAuthors:   len(authorSet),
		CompletedJudgeTasks:    len(completedJudgeTasksSet),
		TotalTrackJudges:       len(totalTrackJudgesSet),
		CompletedReviewedWorks: completedReviewedWorks,
	}, nil
}

func getContestTrackStatusStatsSrc(contestID int) ([]contestTrackStatusStat, error) {
	tracks, err := getTrackListDBFn(contestID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	stats := make([]contestTrackStatusStat, 0, len(tracks))
	for _, track := range tracks {
		works, worksErr := getWorksByTrackFn(track.TrackID)
		if worksErr != nil {
			return nil, worksErr
		}
		statusCounts := map[string]int{}
		authors := map[int]struct{}{}
		for _, work := range works {
			status := strings.TrimSpace(work.WorkStatus)
			if status == "" {
				status = "unknown"
			}
			statusCounts[status]++
			authors[work.AuthorID] = struct{}{}
		}

		distinctStates := make([]string, 0, len(statusCounts))
		for status := range statusCounts {
			distinctStates = append(distinctStates, status)
		}
		sort.Strings(distinctStates)

		stats = append(stats, contestTrackStatusStat{
			TrackID:       track.TrackID,
			TrackName:     track.TrackName,
			TotalWorks:    len(works),
			StatusCounts:  statusCounts,
			TotalAuthors:  len(authors),
			DistinctState: distinctStates,
		})
	}

	return stats, nil
}

func getContestDailySubmissionsStatsSrc(contestID int) ([]contestDailySubmissionStat, error) {
	tracks, err := getTrackListDBFn(contestID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	loc, locErr := time.LoadLocation("Asia/Shanghai")
	if locErr != nil {
		loc = time.FixedZone("CST", 8*3600)
	}

	daily := map[string]int{}
	for _, track := range tracks {
		works, worksErr := getWorksByTrackFn(track.TrackID)
		if worksErr != nil {
			return nil, worksErr
		}
		for _, work := range works {
			timeValue := ""
			if work.WorkInfos != nil {
				if uploaded, ok := work.WorkInfos["file_uploaded_at"].(string); ok {
					timeValue = strings.TrimSpace(uploaded)
				} else if submitted, ok := work.WorkInfos["submitted_at"].(string); ok {
					timeValue = strings.TrimSpace(submitted)
				}
			}
			if timeValue == "" {
				continue
			}
			parsed, parseErr := time.Parse(time.RFC3339, timeValue)
			if parseErr != nil {
				continue
			}
			day := parsed.In(loc).Format("2006-01-02")
			daily[day]++
		}
	}

	dates := make([]string, 0, len(daily))
	for d := range daily {
		dates = append(dates, d)
	}
	sort.Strings(dates)

	result := make([]contestDailySubmissionStat, 0, len(dates))
	for _, d := range dates {
		result = append(result, contestDailySubmissionStat{Date: d, Count: daily[d]})
	}

	return result, nil
}

func getContestJudgeProgressStatsSrc(contestID int) ([]judgeProgressStat, error) {
	events, err := listReviewEventsByContestIDDBFn(contestID)
	if err != nil {
		return nil, uerr.ExtractError(err)
	}

	agg := map[int]*judgeProgressStat{}
	for _, event := range events {
		judgeIDs, judgeErr := listJudgeIDsByReviewEventDBFn(event.EventID)
		if judgeErr != nil {
			return nil, uerr.ExtractError(judgeErr)
		}
		for _, judgeID := range judgeIDs {
			assigned, _ := countAssignedWorksForJudgeInEvent(judgeID, event.EventID)
			submitted, _ := countSubmittedReviewsForJudgeEvent(judgeID, event.EventID)
			entry, ok := agg[judgeID]
			if !ok {
				judgeName := ""
				if judge, geErr := getJudgeByIDDBFn(judgeID); geErr == nil {
					judgeName = judge.JudgeName
				}
				entry = &judgeProgressStat{JudgeID: judgeID, JudgeName: judgeName}
				agg[judgeID] = entry
			}
			entry.AssignedCount += int(assigned)
			entry.SubmittedCount += int(submitted)
		}
	}

	result := make([]judgeProgressStat, 0, len(agg))
	for _, v := range agg {
		if v.AssignedCount > 0 {
			v.CompletionRate = float64(v.SubmittedCount) / float64(v.AssignedCount)
		}
		result = append(result, *v)
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].CompletionRate == result[j].CompletionRate {
			return result[i].JudgeID < result[j].JudgeID
		}
		return result[i].CompletionRate > result[j].CompletionRate
	})

	return result, nil
}

func regenerateContestReviewResultsSrc(contestID int) (int, error) {
	events, err := listReviewEventsByContestIDDBFn(contestID)
	if err != nil {
		return 0, uerr.ExtractError(err)
	}

	generated := 0
	for _, event := range events {
		if err := deleteReviewResultsByEventIDDBFn(event.EventID); err != nil {
			return generated, uerr.ExtractError(err)
		}
		works, workErr := getReviewWorksByEventDBFn(event.EventID, 0, 1000000)
		if workErr != nil {
			return generated, uerr.ExtractError(workErr)
		}
		for _, work := range works {
			if _, genErr := generateReviewResultForWorkAndEvent(work.WorkID, event.EventID); genErr != nil {
				return generated, genErr
			}
			generated++
		}
	}

	return generated, nil
}

func updateJudgeDeadlineReminderSrc(eventID int, judgeID int, deadlineAt time.Time) error {
	if eventID <= 0 || judgeID <= 0 {
		return errors.New("invalid event_id or judge_id")
	}
	if deadlineAt.IsZero() {
		return errors.New("deadlineAt is required")
	}

	if err := updateReviewEventJudgeDeadlineDBFn(eventID, judgeID, &deadlineAt); err != nil {
		return uerr.ExtractError(err)
	}

	return nil
}

func generateReviewResultForWorkAndEvent(workID int, eventID int) (model.ReviewResult, error) {
	reviews, err := listReviewsByWorkAndEventDBFn(workID, eventID)
	if err != nil {
		return model.ReviewResult{}, uerr.ExtractError(err)
	}
	judgeIDs, judgeErr := listJudgeIDsByReviewEventDBFn(eventID)
	if judgeErr != nil {
		return model.ReviewResult{}, uerr.ExtractError(judgeErr)
	}

	totalScore := 0.0
	scoreCount := 0
	comments := make([]string, 0, len(reviews))
	judgeScores := map[string]float64{}
	for _, review := range reviews {
		score := getFloatValue(review.WorkReviews, "judgeScore")
		if score > 0 {
			totalScore += score
			scoreCount++
		}
		comment := strings.TrimSpace(getStringValue(review.WorkReviews, "judgeComment"))
		if comment != "" {
			comments = append(comments, comment)
		}
		judgeScores[strconv.Itoa(review.JudgeID)] = score
	}

	finalScore := 0.0
	if scoreCount > 0 {
		finalScore = totalScore / float64(scoreCount)
	}

	payload := map[string]any{
		"finalScore":         finalScore,
		"reviewCount":        len(reviews),
		"assignedJudgeCount": len(judgeIDs),
		"comments":           strings.Join(comments, "\n\n"),
		"judgeScores":        judgeScores,
		"generatedAt":        time.Now().UTC().Format(time.RFC3339),
	}

	result, upsertErr := upsertReviewResultDBFn(workID, eventID, payload)
	if upsertErr != nil {
		return model.ReviewResult{}, uerr.ExtractError(upsertErr)
	}

	return result, nil
}

func getFloatValue(source map[string]any, key string) float64 {
	if source == nil {
		return 0
	}
	value, ok := source[key]
	if !ok {
		return 0
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func getStringValue(source map[string]any, key string) string {
	if source == nil {
		return ""
	}
	value, ok := source[key]
	if !ok || value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return fmt.Sprintf("%v", value)
}

func formatJudgeScores(raw any) string {
	if raw == nil {
		return ""
	}
	if scoreMap, ok := raw.(map[string]any); ok {
		keys := make([]string, 0, len(scoreMap))
		for k := range scoreMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+":"+fmt.Sprintf("%.2f", getFloatValue(scoreMap, k)))
		}
		return strings.Join(parts, ",")
	}
	if scoreMap, ok := raw.(map[string]float64); ok {
		keys := make([]string, 0, len(scoreMap))
		for k := range scoreMap {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, k := range keys {
			parts = append(parts, k+":"+fmt.Sprintf("%.2f", scoreMap[k]))
		}
		return strings.Join(parts, ",")
	}
	return fmt.Sprintf("%v", raw)
}
