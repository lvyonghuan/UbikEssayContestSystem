package submission

import (
	"context"
	"errors"
	"fmt"
	"main/database/pgsql"
	"main/database/redis"
	"main/model"
	_const "main/util/const"
	"main/util/log"
	"main/util/password"
	"main/util/scriptflow"
	"main/util/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/lvyonghuan/Ubik-Util/uerr"
)

const (
	defaultSubmissionWorkStatus = "submission_success"
)

var (
	getAuthorByAuthorNameFn        = pgsql.GetAuthorByAuthorName
	createAuthorFn                 = pgsql.CreateAuthor
	getAuthorByAuthorIDFn          = pgsql.GetAuthorByAuthorID
	updateAuthorFn                 = pgsql.UpdateAuthor
	submissionWorkFn               = pgsql.SubmissionWork
	updateWorkFn                   = pgsql.UpdateWork
	deleteWorkFn                   = pgsql.DeleteWork
	findWorksByAuthorIDFn          = pgsql.GetWorksByAuthorID
	countWorksByAuthorAndTrackFn   = pgsql.CountWorksByAuthorAndTrack
	countWorksByAuthorAndContestFn = pgsql.CountWorksByAuthorAndContest
	getTrackByIDFn                 = pgsql.GetTrackByID
	setUploadFilePermissionFn      = redis.SetUploadFilePermission
	getStartAndEndDateFn           = redis.GetStartAndEndDate
	resolveFlowForExecutionFn      = pgsql.ResolveFlowForExecution

	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		executor := scriptflow.NewExecutor(".", 5*time.Second, []string{"python3", "python", "bash", "sh", "node"})
		return executor.ExecuteChain(context.Background(), chain, input)
	}

	readDirFn = os.ReadDir
	removeFn  = os.Remove
)

func submissionSrcWarn(message string) {
	if log.Logger != nil {
		log.Logger.Warn(message)
	}
}

func newSubmissionSrcError(message string) error {
	err := errors.New(message)
	submissionSrcWarn("Submission src error: " + err.Error())
	return err
}

func submissionSrcExtractError(err error) error {
	parsedErr := uerr.ExtractError(err)
	submissionSrcWarn("Submission src error: " + parsedErr.Error())
	return parsedErr
}

func registerAuthorSrc(author *model.Author) error {
	tmpAuthor := model.Author{
		AuthorName: author.AuthorName,
	}

	err := getAuthorByAuthorNameFn(&tmpAuthor)
	if err != nil {
		if errors.Is(err, _const.UsernameNotExist) {
			hashedPassword, hashErr := password.HashPassword(author.Password)
			if hashErr != nil {
				return submissionSrcExtractError(hashErr)
			}
			author.Password = hashedPassword

			err = createAuthorFn(author)
			if err != nil {
				return submissionSrcExtractError(err)
			}
		} else {
			return submissionSrcExtractError(err)
		}
	} else {
		return newSubmissionSrcError("username already exists")
	}

	return nil
}

func authorLoginSrc(author *model.Author) (token.ResponseToken, error) {
	tempAuthor := model.Author{AuthorName: author.AuthorName}
	err := getAuthorByAuthorNameFn(&tempAuthor)
	if err != nil {
		return token.ResponseToken{}, submissionSrcExtractError(err)
	}

	if !password.CheckPasswordHash(author.Password, tempAuthor.Password) {
		return token.ResponseToken{}, newSubmissionSrcError("bad request")
	}

	tokens, err := token.GenTokenAndRefreshToken(int64(tempAuthor.AuthorID), _const.RoleAuthor)
	if err != nil {
		return token.ResponseToken{}, submissionSrcExtractError(err)
	}

	return tokens, nil
}

func refreshTokenSrc(authorID int64) (token.ResponseToken, error) {
	tokens, err := token.GenTokenAndRefreshToken(authorID, _const.RoleAuthor)
	if err != nil {
		return token.ResponseToken{}, submissionSrcExtractError(err)
	}

	return tokens, nil
}

func updateAuthorSrc(author *model.Author) error {
	err := updateAuthorFn(author)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	return nil
}

func submissionWorkSrc(work *model.Work) error {
	if err := checkSubmissionTimeValid(work.TrackID); err != nil {
		return submissionSrcExtractError(err)
	}

	track, err := getTrackByIDFn(work.TrackID)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	if track.ContestID <= 0 {
		return newSubmissionSrcError("track is not bound to a contest")
	}

	count, err := countWorksByAuthorAndContestFn(work.AuthorID, track.ContestID)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		work.TrackID,
		work,
		map[string]any{
			"phase":         "create",
			"authorID":      work.AuthorID,
			"contestID":     track.ContestID,
			"trackID":       work.TrackID,
			"existingCount": count,
			"work":          toWorkMap(*work),
		},
		"",
	)
	if err != nil {
		return submissionSrcExtractError(err)
	}
	if !allowed {
		return newSubmissionSrcError("submission blocked by script flow")
	}

	ensureDefaultWorkStatus(work)

	return performWorkOperationWithPermission(work, submissionWorkFn, "Submission work")
}

func updateSubmissionSrc(work *model.Work) error {
	if err := checkSubmissionTimeValid(work.TrackID); err != nil {
		return submissionSrcExtractError(err)
	}

	if err := hydrateWorkStatusForUpdate(work); err != nil {
		return submissionSrcExtractError(err)
	}

	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionUpdatePre,
		work.TrackID,
		work,
		map[string]any{
			"phase":    "update",
			"authorID": work.AuthorID,
			"trackID":  work.TrackID,
			"workID":   work.WorkID,
			"work":     toWorkMap(*work),
		},
		"update",
	)
	if err != nil {
		return submissionSrcExtractError(err)
	}
	if !allowed {
		return newSubmissionSrcError("submission update blocked by script flow")
	}

	if strings.TrimSpace(work.WorkStatus) == "" {
		ensureDefaultWorkStatus(work)
	}

	return performWorkOperationWithPermission(work, updateWorkFn, "Update submission")
}

func deleteSubmissionSrc(work *model.Work) error {
	allowed, err := checkHookAllowedAndApplyPatch(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionDeletePre,
		work.TrackID,
		work,
		map[string]any{
			"phase":    "delete",
			"authorID": work.AuthorID,
			"trackID":  work.TrackID,
			"workID":   work.WorkID,
			"work":     toWorkMap(*work),
		},
		"delete",
	)
	if err != nil {
		return submissionSrcExtractError(err)
	}
	if !allowed {
		return newSubmissionSrcError("submission delete blocked by script flow")
	}

	err = deleteWorkFn(work)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	if err := removeSubmissionFiles(*work); err != nil {
		log.Logger.Warn("Delete submission files failed: " + err.Error())
	}

	return nil
}

func findSubmissionsByAuthorIDSrc(authorID int) ([]model.Work, error) {
	works, err := findWorksByAuthorIDFn(authorID)
	if err != nil {
		return nil, submissionSrcExtractError(err)
	}
	return works, nil
}

func checkSubmissionTimeValid(trackID int) error {
	start, end, err := getStartAndEndDateFn(trackID)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	nowTimeUnix := time.Now().Unix()
	switch {
	case nowTimeUnix < start:
		return newSubmissionSrcError("contest has not started yet")
	case nowTimeUnix > end:
		return newSubmissionSrcError("contest has already ended")
	default:
		return nil
	}
}

// checkHookAllowedAndApplyPatch 执行Hook检查，如果允许则应用Patch到work对象
// 返回true表示通过检查，false表示被Hook拒绝
func checkHookAllowedAndApplyPatch(scope string, eventKey string, trackID int, work *model.Work, payload map[string]any, blockedMessageKey string) (bool, error) {
	hookResult, err := runTrackHook(scope, eventKey, trackID, payload)
	if err != nil {
		return false, submissionSrcExtractError(err)
	}
	if !hookResult.Allowed {
		if hookResult.Reason != "" {
			return false, newSubmissionSrcError(hookResult.Reason)
		}
		return false, newSubmissionSrcError("submission " + blockedMessageKey + " blocked by script flow")
	}
	mergeWorkInfos(work, hookResult.Patch)
	if patchedStatus, ok := extractWorkStatusFromPatch(hookResult.Patch); ok {
		setWorkStatus(work, patchedStatus)
	}
	return true, nil
}

// performWorkOperationWithPermission 执行工作操作（如提交、更新）并设置文件上传权限
func performWorkOperationWithPermission(work *model.Work, operation func(*model.Work) error, opName string) error {
	err := operation(work)
	if err != nil {
		log.Logger.Warn(opName + " failed: " + err.Error())
		return submissionSrcExtractError(err)
	}

	err = setUploadFilePermissionFn(work.AuthorID, work.TrackID, work.WorkID)
	if err != nil {
		return submissionSrcExtractError(err)
	}

	return nil
}

func runTrackHook(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
	flow, steps, err := resolveFlowForExecutionFn(scope, eventKey, "track", trackID)
	if err != nil {
		if errors.Is(err, pgsql.ErrFlowNotMounted) {
			return scriptflow.ChainResult{Allowed: true}, nil
		}
		return scriptflow.ChainResult{Allowed: false}, submissionSrcExtractError(err)
	}

	chain := scriptflow.ChainConfig{
		Scope:    scope,
		EventKey: eventKey,
		FlowKey:  flow.FlowKey,
		Steps:    make([]scriptflow.StepConfig, 0, len(steps)),
	}
	for _, step := range steps {
		timeout := 5 * time.Second
		if step.Step.TimeoutMs > 0 {
			timeout = time.Duration(step.Step.TimeoutMs) * time.Millisecond
		}
		strategy := strings.TrimSpace(step.Step.FailureStrategy)
		if strategy == "" {
			strategy = "fail_close"
		}
		chain.Steps = append(chain.Steps, scriptflow.StepConfig{
			StepName:        step.Step.StepName,
			Interpreter:     step.Script.Interpreter,
			ScriptPath:      filepath.ToSlash(step.Version.RelativePath),
			Timeout:         timeout,
			FailureStrategy: strategy,
			InputTemplate:   step.Step.InputTemplate,
		})
	}

	result, err := executeScriptChainFn(chain, scriptflow.ExecuteInput{
		Scope:    scope,
		EventKey: eventKey,
		FlowKey:  flow.FlowKey,
		TraceID:  fmt.Sprintf("%d", time.Now().UnixNano()),
		NowUnix:  time.Now().Unix(),
		Context: map[string]any{
			"trackID": trackID,
		},
		Payload: payload,
	})
	if err != nil {
		if errors.Is(err, scriptflow.ErrExecutionBlocked) {
			return result, nil
		}
		return result, submissionSrcExtractError(err)
	}

	return result, nil
}

func mergeWorkInfos(work *model.Work, patch map[string]any) {
	if len(patch) == 0 {
		return
	}
	if work.WorkInfos == nil {
		work.WorkInfos = map[string]any{}
	}
	for k, v := range patch {
		work.WorkInfos[k] = v
	}
}

func toWorkMap(work model.Work) map[string]any {
	infos := map[string]any{}
	for k, v := range work.WorkInfos {
		infos[k] = v
	}
	return map[string]any{
		"workID":     work.WorkID,
		"workTitle":  work.WorkTitle,
		"trackID":    work.TrackID,
		"authorID":   work.AuthorID,
		"workStatus": work.WorkStatus,
		"workInfos":  infos,
	}
}

func hydrateWorkStatusForUpdate(work *model.Work) error {
	if strings.TrimSpace(work.WorkStatus) != "" || work.WorkID <= 0 {
		return nil
	}

	existing := model.Work{WorkID: work.WorkID}
	if err := getSubmissionByWorkIDFn(&existing); err != nil {
		return submissionSrcExtractError(err)
	}

	if strings.TrimSpace(existing.WorkStatus) == "" {
		ensureDefaultWorkStatus(work)
		return nil
	}

	setWorkStatus(work, existing.WorkStatus)
	return nil
}

func ensureDefaultWorkStatus(work *model.Work) {
	if strings.TrimSpace(work.WorkStatus) == "" {
		setWorkStatus(work, defaultSubmissionWorkStatus)
		return
	}

	setWorkStatus(work, work.WorkStatus)
}

func setWorkStatus(work *model.Work, status string) {
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return
	}
	work.WorkStatus = normalized
	if work.WorkInfos == nil {
		work.WorkInfos = map[string]any{}
	}
	work.WorkInfos["workStatus"] = normalized
	work.WorkInfos["status"] = normalized
}

func extractWorkStatusFromPatch(patch map[string]any) (string, bool) {
	if len(patch) == 0 {
		return "", false
	}

	for _, key := range []string{"workStatus", "work_status", "status"} {
		if value, ok := patch[key]; ok {
			if status, ok := normalizeStatusValue(value); ok {
				return status, true
			}
		}
	}

	return "", false
}

func normalizeStatusValue(value any) (string, bool) {
	status, ok := value.(string)
	if !ok {
		return "", false
	}
	normalized := strings.TrimSpace(status)
	if normalized == "" {
		return "", false
	}

	return normalized, true
}

func removeSubmissionFiles(work model.Work) error {
	dstDir := filepath.Join(_const.SubmissionFileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		wrappedErr := uerr.NewError(err)
		submissionSrcWarn("Submission src error: " + wrappedErr.Error())
		return wrappedErr
	}

	prefix := strconv.Itoa(work.WorkID) + "."
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if rmErr := removeFn(filepath.Join(dstDir, name)); rmErr != nil {
			wrappedErr := uerr.NewError(rmErr)
			submissionSrcWarn("Submission src error: " + wrappedErr.Error())
			return wrappedErr
		}
	}

	return nil
}
