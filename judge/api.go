package judge

import (
	"errors"
	_const "main/util/const"
	"main/util/log"
	"main/util/response"
	"main/util/token"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

var (
	checkTokenFn = token.CheckToken
	runServerFn  = func(r *gin.Engine, port string) error {
		return r.Run(":" + port)
	}
)

type judgeLoginRequest struct {
	JudgeID   int    `json:"judgeID"`
	JudgeName string `json:"judgeName"`
	Password  string `json:"password"`
}

type judgeSubmitReviewRequest struct {
	WorkID          int            `json:"workID"`
	EventID         int            `json:"eventID"`
	JudgeScore      float64        `json:"judgeScore"`
	JudgeComment    string         `json:"judgeComment"`
	DimensionScores map[string]any `json:"dimensionScores"`
}

type judgeUpdateReviewRequest struct {
	JudgeScore      float64        `json:"judgeScore"`
	JudgeComment    string         `json:"judgeComment"`
	DimensionScores map[string]any `json:"dimensionScores"`
}

func checkJudgeAccessToken(c *gin.Context) {
	bearerToken := c.GetHeader("Authorization")

	id, role, err := checkTokenFn(bearerToken)
	if err != nil {
		response.RespError(c, 401, err.Error())
		c.Abort()
		return
	}

	if role != _const.RoleJudge {
		response.RespError(c, 403, "forbidden: insufficient permissions")
		c.Abort()
		return
	}

	c.Set("judge_token_id", int(id))
	c.Next()
}

func judgeLogin(c *gin.Context) {
	var req judgeLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	tokens, err := judgeLoginSrc(req.JudgeID, req.JudgeName, req.Password)
	if err != nil {
		if errors.Is(err, errJudgeNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "required") || strings.Contains(strings.ToLower(err.Error()), "login error") {
			response.RespError(c, 400, err.Error())
			return
		}
		log.Logger.Warn("Judge login error: " + err.Error())
		response.RespError(c, 500, "error: Judge login error")
		return
	}

	response.RespSuccess(c, tokens)
}

func getJudgeEvents(c *gin.Context) {
	offset := 0
	if offsetStr := strings.TrimSpace(c.DefaultQuery("offset", "0")); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			response.RespError(c, 400, "error: Invalid offset")
			return
		}
		offset = parsedOffset
	}

	limit := 20
	if limitStr := strings.TrimSpace(c.DefaultQuery("limit", "20")); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			response.RespError(c, 400, "error: Invalid limit")
			return
		}
		limit = parsedLimit
	}

	events, err := listJudgeEventsSrc(c.GetInt("judge_token_id"), offset, limit)
	if err != nil {
		response.RespError(c, 500, "error: List judge review events error")
		return
	}

	response.RespSuccess(c, events)
}

func getReviewEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "error: Invalid event_id")
		return
	}

	event, err := getJudgeEventByIDSrc(c.GetInt("judge_token_id"), eventID)
	if err != nil {
		if errors.Is(err, errReviewEventNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		if errors.Is(err, errEventAccessDenied) {
			response.RespError(c, 403, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get review event error")
		return
	}

	response.RespSuccess(c, event)
}

func getEventWorks(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "error: Invalid event_id")
		return
	}

	offset := 0
	if offsetStr := strings.TrimSpace(c.DefaultQuery("offset", "0")); offsetStr != "" {
		parsedOffset, err := strconv.Atoi(offsetStr)
		if err != nil || parsedOffset < 0 {
			response.RespError(c, 400, "error: Invalid offset")
			return
		}
		offset = parsedOffset
	}

	limit := 20
	if limitStr := strings.TrimSpace(c.DefaultQuery("limit", "20")); limitStr != "" {
		parsedLimit, err := strconv.Atoi(limitStr)
		if err != nil || parsedLimit <= 0 || parsedLimit > 100 {
			response.RespError(c, 400, "error: Invalid limit")
			return
		}
		limit = parsedLimit
	}

	works, err := listJudgeEventWorksSrc(c.GetInt("judge_token_id"), eventID, offset, limit)
	if err != nil {
		if errors.Is(err, errEventAccessDenied) {
			response.RespError(c, 403, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get review event works error")
		return
	}

	response.RespSuccess(c, works)
}

func getReviewWorkFile(c *gin.Context) {
	eventID, err := strconv.Atoi(strings.TrimSpace(c.Query("event_id")))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "error: Invalid event_id")
		return
	}

	workID, err := strconv.Atoi(strings.TrimSpace(c.Query("work_id")))
	if err != nil || workID <= 0 {
		response.RespError(c, 400, "error: Invalid work_id")
		return
	}

	filePath, err := getJudgeReviewWorkFilePathSrc(c.GetInt("judge_token_id"), eventID, workID)
	if err != nil {
		if errors.Is(err, errEventAccessDenied) {
			response.RespError(c, 403, err.Error())
			return
		}
		if errors.Is(err, errWorkFileNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get review work file error")
		return
	}

	c.FileAttachment(filePath, filepath.Base(filePath))
}

func submitReviewResult(c *gin.Context) {
	var req judgeSubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	review, err := submitJudgeReviewSrc(c.GetInt("judge_token_id"), ReviewSubmitInput{
		WorkID:          req.WorkID,
		EventID:         req.EventID,
		JudgeScore:      req.JudgeScore,
		JudgeComment:    req.JudgeComment,
		DimensionScores: req.DimensionScores,
	})
	if err != nil {
		if errors.Is(err, errEventAccessDenied) {
			response.RespError(c, 403, err.Error())
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "invalid") || strings.Contains(strings.ToLower(err.Error()), "does not belong") || strings.Contains(strings.ToLower(err.Error()), "status") {
			response.RespError(c, 400, err.Error())
			return
		}
		response.RespError(c, 500, "error: Submit review result error")
		return
	}

	response.RespSuccess(c, review)
}

func getReviewResultsByEvent(c *gin.Context) {
	eventID, err := strconv.Atoi(c.Param("event_id"))
	if err != nil || eventID <= 0 {
		response.RespError(c, 400, "error: Invalid event_id")
		return
	}

	reviews, err := listJudgeEventReviewsSrc(c.GetInt("judge_token_id"), eventID)
	if err != nil {
		if errors.Is(err, errEventAccessDenied) {
			response.RespError(c, 403, err.Error())
			return
		}
		response.RespError(c, 500, "error: Get review results error")
		return
	}

	response.RespSuccess(c, reviews)
}

func updateReviewResult(c *gin.Context) {
	reviewID, err := strconv.Atoi(c.Param("result_id"))
	if err != nil || reviewID <= 0 {
		response.RespError(c, 400, "error: Invalid result_id")
		return
	}

	var req judgeUpdateReviewRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response.RespError(c, 400, "bad request")
		return
	}

	review, err := updateJudgeReviewSrc(c.GetInt("judge_token_id"), reviewID, req.JudgeScore, req.JudgeComment, req.DimensionScores)
	if err != nil {
		if errors.Is(err, errReviewNotFound) {
			response.RespError(c, 404, err.Error())
			return
		}
		if strings.Contains(strings.ToLower(err.Error()), "forbidden") || strings.Contains(strings.ToLower(err.Error()), "invalid") {
			response.RespError(c, 403, err.Error())
			return
		}
		response.RespError(c, 500, "error: Update review result error")
		return
	}

	response.RespSuccess(c, review)
}

func health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
