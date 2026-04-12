package admin

import (
	"strconv"
	"strings"
	"time"

	"main/model"
	"main/util/response"

	"github.com/gin-gonic/gin"
)

var (
	createJudgeAccountSrcFn         = createJudgeAccountSrc
	batchCreateJudgeAccountsSrcFn   = batchCreateJudgeAccountsSrc
	updateJudgeAccountSrcFn         = updateJudgeAccountSrc
	deleteJudgeAccountSrcFn         = deleteJudgeAccountSrc
	createReviewEventSrcFn          = createReviewEventSrc
	updateReviewEventSrcFn          = updateReviewEventSrc
	assignReviewEventJudgesSrcFn    = assignReviewEventJudgesSrc
	deleteReviewEventSrcFn          = deleteReviewEventSrc
	getReviewEventProgressSrcFn     = getReviewEventProgressSrc
	listTrackStatusesSrcFn          = listTrackStatusesSrc
	getWorkReviewStatusSrcFn        = getWorkReviewStatusSrc
	getWorkReviewResultsSrcFn       = getWorkReviewResultsSrc
	regenerateWorkReviewResultsFn   = regenerateWorkReviewResultsSrc
	rankTrackWorksSrcFn             = rankTrackWorksSrc
	exportTrackReviewExcelSrcFn     = exportTrackReviewExcelSrc
	getDashboardOverviewSrcFn       = getDashboardOverviewSrc
	getContestTrackStatusStatsSrcFn = getContestTrackStatusStatsSrc
	getContestDailyStatsSrcFn       = getContestDailySubmissionsStatsSrc
	getContestJudgeStatsSrcFn       = getContestJudgeProgressStatsSrc
	regenerateContestResultsSrcFn   = regenerateContestReviewResultsSrc
	updateJudgeDeadlineSrcFn        = updateJudgeDeadlineReminderSrc
)

type batchJudgeAccountRequest struct {
	Judges []judgeAccountInput `json:"judges"`
}

type assignJudgesRequest struct {
	JudgeIDs []int `json:"judgeIDs"`
}

type judgeDeadlineRequest struct {
	DeadlineAt time.Time `json:"deadlineAt"`
}

func registerJudgeReviewRoutes(admin *gin.RouterGroup) {
	judge := admin.Group("/judge", checkAccessToken)
	{
		judge.POST("/account", createJudgeAccount)
		judge.POST("/accounts", batchCreateJudgeAccounts)
		judge.PUT("/:judge_id", updateJudgeAccount)
		judge.DELETE("/:judge_id", deleteJudgeAccount)

		review := judge.Group("/review")
		{
			review.POST("/event", createReviewEvent)
			review.PUT("/:event_id", updateReviewEvent)
			review.PUT("/:event_id/assign", assignReviewEventJudges)
			review.DELETE("/:event_id", deleteReviewEvent)

			review.GET("/:event_id", getReviewEventProgress)
			review.GET("/track/:track_id/status", listTrackStatuses)
			review.GET("/status/:work_id", getWorkReviewStatus)
			review.GET("/result/:work_id", getWorkReviewResults)
			review.POST("/result/:work_id/gen", regenerateWorkReviewResults)
			review.GET("/rank/:track_id", getTrackReviewRanking)
			review.GET("/export/:track_id", exportTrackReviewExcel)
		}
	}

	stats := admin.Group("", checkAccessToken)
	{
		stats.GET("/dashboard/overview", getDashboardOverview)
		stats.GET("/contests/:contest_id/stats/tracks-status", getContestTrackStatusStats)
		stats.GET("/contests/:contest_id/stats/daily-submissions", getContestDailySubmissionStats)
		stats.GET("/contests/:contest_id/stats/judges-progress", getContestJudgeProgressStats)
		stats.POST("/review-results/generate/:contest_id", regenerateContestReviewResults)
		stats.POST("/review-events/:event_id/judges/:judge_id/deadline", updateJudgeDeadline)
	}
}

func createJudgeAccount(c *gin.Context) {
	var req judgeAccountInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	judge, err := createJudgeAccountSrcFn(c.GetInt("admin_token_id"), req)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, judge)
}

func batchCreateJudgeAccounts(c *gin.Context) {
	var req batchJudgeAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	judges, err := batchCreateJudgeAccountsSrcFn(c.GetInt("admin_token_id"), req.Judges)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, judges)
}

func updateJudgeAccount(c *gin.Context) {
	judgeID, err := strconv.Atoi(c.Param("judge_id"))
	if err != nil || judgeID <= 0 {
		response.RespError(c, 400, "invalid judge_id")
		return
	}

	var req judgeAccountInput
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = updateJudgeAccountSrcFn(c.GetInt("admin_token_id"), judgeID, req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func deleteJudgeAccount(c *gin.Context) {
	judgeID, err := strconv.Atoi(c.Param("judge_id"))
	if err != nil || judgeID <= 0 {
		response.RespError(c, 400, "invalid judge_id")
		return
	}

	if err = deleteJudgeAccountSrcFn(c.GetInt("admin_token_id"), judgeID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func createReviewEvent(c *gin.Context) {
	var req reviewEventInput
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	event, err := createReviewEventSrcFn(c.GetInt("admin_token_id"), req)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, event)
}

func updateReviewEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "invalid event_id")
		return
	}

	var req reviewEventInput
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = updateReviewEventSrcFn(c.GetInt("admin_token_id"), eventID, req); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func assignReviewEventJudges(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "invalid event_id")
		return
	}

	var req assignJudgesRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = assignReviewEventJudgesSrcFn(c.GetInt("admin_token_id"), eventID, req.JudgeIDs); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func deleteReviewEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "invalid event_id")
		return
	}

	if err = deleteReviewEventSrcFn(c.GetInt("admin_token_id"), eventID); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}

	response.RespSuccess(c, nil)
}

func getReviewEventProgress(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "invalid event_id")
		return
	}

	progress, err := getReviewEventProgressSrcFn(eventID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, progress)
}

func listTrackStatuses(c *gin.Context) {
	trackID, err := strconv.Atoi(c.Param("track_id"))
	if err != nil || trackID <= 0 {
		response.RespError(c, 400, "invalid track_id")
		return
	}

	statuses, err := listTrackStatusesSrcFn(trackID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, statuses)
}

func getWorkReviewStatus(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil || workID <= 0 {
		response.RespError(c, 400, "invalid work_id")
		return
	}

	status, err := getWorkReviewStatusSrcFn(workID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, status)
}

func getWorkReviewResults(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil || workID <= 0 {
		response.RespError(c, 400, "invalid work_id")
		return
	}

	results, err := getWorkReviewResultsSrcFn(workID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, results)
}

func regenerateWorkReviewResults(c *gin.Context) {
	workID, err := strconv.Atoi(c.Param("work_id"))
	if err != nil || workID <= 0 {
		response.RespError(c, 400, "invalid work_id")
		return
	}

	results, err := regenerateWorkReviewResultsFn(workID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, results)
}

func getTrackReviewRanking(c *gin.Context) {
	trackID, err := strconv.Atoi(c.Param("track_id"))
	if err != nil || trackID <= 0 {
		response.RespError(c, 400, "invalid track_id")
		return
	}

	ranking, err := rankTrackWorksSrcFn(trackID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, ranking)
}

func exportTrackReviewExcel(c *gin.Context) {
	trackID, err := strconv.Atoi(c.Param("track_id"))
	if err != nil || trackID <= 0 {
		response.RespError(c, 400, "invalid track_id")
		return
	}

	format := strings.TrimSpace(c.DefaultQuery("format", "xlsx"))
	if format != "xlsx" {
		response.RespError(c, 400, "only xlsx format is supported")
		return
	}

	filePath, err := exportTrackReviewExcelSrcFn(trackID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	c.FileAttachment(filePath, filepathBase(filePath))
}

func getDashboardOverview(c *gin.Context) {
	overview, err := getDashboardOverviewSrcFn()
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, overview)
}

func getContestTrackStatusStats(c *gin.Context) {
	contestID, err := strconv.Atoi(c.Param("contest_id"))
	if err != nil || contestID <= 0 {
		response.RespError(c, 400, "invalid contest_id")
		return
	}
	stats, err := getContestTrackStatusStatsSrcFn(contestID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, stats)
}

func getContestDailySubmissionStats(c *gin.Context) {
	contestID, err := strconv.Atoi(c.Param("contest_id"))
	if err != nil || contestID <= 0 {
		response.RespError(c, 400, "invalid contest_id")
		return
	}
	stats, err := getContestDailyStatsSrcFn(contestID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, stats)
}

func getContestJudgeProgressStats(c *gin.Context) {
	contestID, err := strconv.Atoi(c.Param("contest_id"))
	if err != nil || contestID <= 0 {
		response.RespError(c, 400, "invalid contest_id")
		return
	}
	stats, err := getContestJudgeStatsSrcFn(contestID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, stats)
}

func regenerateContestReviewResults(c *gin.Context) {
	contestID, err := strconv.Atoi(c.Param("contest_id"))
	if err != nil || contestID <= 0 {
		response.RespError(c, 400, "invalid contest_id")
		return
	}
	generated, err := regenerateContestResultsSrcFn(contestID)
	if err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, gin.H{"generated": generated})
}

func updateJudgeDeadline(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "invalid event_id")
		return
	}
	judgeID, err := strconv.Atoi(c.Param("judge_id"))
	if err != nil || judgeID <= 0 {
		response.RespError(c, 400, "invalid judge_id")
		return
	}
	var req judgeDeadlineRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	if err = updateJudgeDeadlineSrcFn(eventID, judgeID, req.DeadlineAt); err != nil {
		response.RespError(c, 500, err.Error())
		return
	}
	response.RespSuccess(c, nil)
}

func filepathBase(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		idx = strings.LastIndex(path, "\\")
	}
	if idx == -1 {
		return path
	}
	return path[idx+1:]
}

// keep go compiler aware of model package in this file for swagger-friendly future expansion
var _ = model.Response{}
