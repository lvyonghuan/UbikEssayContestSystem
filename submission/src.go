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
	resolveFlowChainForExecutionFn = pgsql.ResolveFlowChainForExecution

	executeScriptChainFn = func(chain scriptflow.ChainConfig, input scriptflow.ExecuteInput) (scriptflow.ChainResult, error) {
		executor := scriptflow.NewExecutor(".", 5*time.Second, []string{"python3", "python", "bash", "sh", "node"})
		return executor.ExecuteChain(context.Background(), chain, input)
	}

	readDirFn = os.ReadDir
	removeFn  = os.Remove
)

func registerAuthorSrc(author *model.Author) error {
	tmpAuthor := model.Author{
		AuthorName: author.AuthorName,
	}

	err := getAuthorByAuthorNameFn(&tmpAuthor)
	if err != nil {
		if errors.Is(err, _const.UsernameNotExist) {
			hashedPassword, hashErr := password.HashPassword(author.Password)
			if hashErr != nil {
				log.Logger.Warn("Register author hash password failed: " + hashErr.Error())
				return uerr.ExtractError(hashErr)
			}
			author.Password = hashedPassword

			err = createAuthorFn(author)
			if err != nil {
				log.Logger.Warn("Register author failed: " + err.Error())
				return uerr.ExtractError(err)
			}
		} else {
			log.Logger.Warn("Failed to check if username exists: " + err.Error())
			return uerr.ExtractError(err)
		}
	} else {
		return errors.New("username already exists")
	}

	return nil
}

func authorLoginSrc(author *model.Author) (token.ResponseToken, error) {
	tempAuthor := model.Author{AuthorName: author.AuthorName}
	err := getAuthorByAuthorNameFn(&tempAuthor)
	if err != nil {
		log.Logger.Warn("Author login failed: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	if !password.CheckPasswordHash(author.Password, tempAuthor.Password) {
		return token.ResponseToken{}, errors.New("bad request")
	}

	tokens, err := token.GenTokenAndRefreshToken(int64(tempAuthor.AuthorID), _const.RoleAuthor)
	if err != nil {
		log.Logger.Warn("Generate token failed: " + err.Error())
		return token.ResponseToken{}, uerr.ExtractError(err)
	}

	return tokens, nil
}

func refreshTokenSrc(authorID int64) (token.ResponseToken, error) {
	return token.GenTokenAndRefreshToken(authorID, _const.RoleAuthor)
}

func updateAuthorSrc(author *model.Author) error {
	err := updateAuthorFn(author)
	if err != nil {
		log.Logger.Warn("Update author failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

func submissionWorkSrc(work *model.Work) error {
	if err := checkSubmissionTimeValid(work.TrackID); err != nil {
		return err
	}

	track, err := getTrackByIDFn(work.TrackID)
	if err != nil {
		log.Logger.Warn("Get track by id failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	if track.ContestID <= 0 {
		return errors.New("track is not bound to a contest")
	}

	count, err := countWorksByAuthorAndContestFn(work.AuthorID, track.ContestID)
	if err != nil {
		log.Logger.Warn("Count submissions failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	hookResult, err := runTrackHookWithContest(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionPre,
		track.ContestID,
		work.TrackID,
		map[string]any{
			"phase":         "create",
			"authorID":      work.AuthorID,
			"contestID":     track.ContestID,
			"trackID":       work.TrackID,
			"existingCount": count,
			"work":          toWorkMap(*work),
		},
	)
	if err != nil {
		return err
	}
	if !hookResult.Allowed {
		if hookResult.Reason != "" {
			return errors.New(hookResult.Reason)
		}
		return errors.New("submission blocked by script flow")
	}
	mergeWorkInfos(work, hookResult.Patch)

	err = submissionWorkFn(work)
	if err != nil {
		log.Logger.Warn("Submission work failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = setUploadFilePermissionFn(work.AuthorID, work.TrackID, work.WorkID)
	if err != nil {
		log.Logger.Warn("Set upload file permission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

func updateSubmissionSrc(work *model.Work) error {
	if err := checkSubmissionTimeValid(work.TrackID); err != nil {
		return err
	}

	hookResult, err := runTrackHook(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionUpdatePre,
		work.TrackID,
		map[string]any{
			"phase":    "update",
			"authorID": work.AuthorID,
			"trackID":  work.TrackID,
			"workID":   work.WorkID,
			"work":     toWorkMap(*work),
		},
	)
	if err != nil {
		return err
	}
	if !hookResult.Allowed {
		if hookResult.Reason != "" {
			return errors.New(hookResult.Reason)
		}
		return errors.New("submission update blocked by script flow")
	}
	mergeWorkInfos(work, hookResult.Patch)

	err = updateWorkFn(work)
	if err != nil {
		log.Logger.Warn("Update submission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	err = setUploadFilePermissionFn(work.AuthorID, work.TrackID, work.WorkID)
	if err != nil {
		log.Logger.Warn("Set upload file permission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	return nil
}

func deleteSubmissionSrc(work *model.Work) error {
	hookResult, err := runTrackHook(
		scriptflow.ScopeSubmission,
		scriptflow.EventSubmissionDeletePre,
		work.TrackID,
		map[string]any{
			"phase":    "delete",
			"authorID": work.AuthorID,
			"trackID":  work.TrackID,
			"workID":   work.WorkID,
			"work":     toWorkMap(*work),
		},
	)
	if err != nil {
		return err
	}
	if !hookResult.Allowed {
		if hookResult.Reason != "" {
			return errors.New(hookResult.Reason)
		}
		return errors.New("submission delete blocked by script flow")
	}

	err = deleteWorkFn(work)
	if err != nil {
		log.Logger.Warn("Delete submission failed: " + err.Error())
		return uerr.ExtractError(err)
	}

	if err := removeSubmissionFiles(*work); err != nil {
		log.Logger.Warn("Delete submission files failed: " + err.Error())
	}

	return nil
}

func findSubmissionsByAuthorIDSrc(authorID int) ([]model.Work, error) {
	works, err := findWorksByAuthorIDFn(authorID)
	if err != nil {
		log.Logger.Warn("Failed to get works by author id: " + err.Error())
		return nil, uerr.ExtractError(err)
	}
	return works, nil
}

func checkSubmissionTimeValid(trackID int) error {
	start, end, err := getStartAndEndDateFn(trackID)
	if err != nil {
		return uerr.ExtractError(err)
	}

	nowTimeUnix := time.Now().Unix()
	switch {
	case nowTimeUnix < start:
		return errors.New("contest has not started yet")
	case nowTimeUnix > end:
		return errors.New("contest has already ended")
	default:
		return nil
	}
}

func runTrackHook(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
	track, err := getTrackByIDFn(trackID)
	if err != nil {
		log.Logger.Warn("Get track by id failed in runTrackHook: " + err.Error())
		return scriptflow.ChainResult{Allowed: false}, uerr.ExtractError(err)
	}

	return runTrackHookWithContest(scope, eventKey, track.ContestID, trackID, payload)
}

func runTrackHookWithContest(scope string, eventKey string, contestID int, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
	if resolveFlowChainForExecutionFn != nil {
		chains, err := resolveFlowChainForExecutionFn(scope, eventKey, contestID, trackID)
		if err != nil {
			if errors.Is(err, pgsql.ErrFlowNotMounted) {
				return scriptflow.ChainResult{Allowed: true}, nil
			}
			log.Logger.Warn("Resolve script flow chain failed: " + err.Error())
			return scriptflow.ChainResult{Allowed: false}, uerr.ExtractError(err)
		}

		merged := scriptflow.ChainResult{Allowed: true, Patch: map[string]any{}}
		for _, chain := range chains {
			result, err := executeResolvedFlow(scope, eventKey, trackID, chain.Flow, chain.Steps, payload)
			if err != nil {
				if errors.Is(err, scriptflow.ErrExecutionBlocked) {
					merged.Allowed = false
					merged.Reason = result.Reason
					mergeWorkInfosResult(merged.Patch, result.Patch)
					merged.Steps = append(merged.Steps, result.Steps...)
					if len(merged.Patch) == 0 {
						merged.Patch = nil
					}
					return merged, nil
				}
				return result, err
			}

			mergeWorkInfosResult(merged.Patch, result.Patch)
			merged.Steps = append(merged.Steps, result.Steps...)
			if !result.Allowed {
				merged.Allowed = false
				merged.Reason = result.Reason
				if len(merged.Patch) == 0 {
					merged.Patch = nil
				}
				return merged, nil
			}
		}

		if len(merged.Patch) == 0 {
			merged.Patch = nil
		}
		return merged, nil
	}

	return runTrackHookLegacy(scope, eventKey, trackID, payload)
}

func runTrackHookLegacy(scope string, eventKey string, trackID int, payload map[string]any) (scriptflow.ChainResult, error) {
	flow, steps, err := resolveFlowForExecutionFn(scope, eventKey, "track", trackID)
	if err != nil {
		if errors.Is(err, pgsql.ErrFlowNotMounted) {
			return scriptflow.ChainResult{Allowed: true}, nil
		}
		log.Logger.Warn("Resolve script flow failed: " + err.Error())
		return scriptflow.ChainResult{Allowed: false}, uerr.ExtractError(err)
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
		return result, err
	}

	return result, nil
}

func executeResolvedFlow(scope string, eventKey string, trackID int, flow model.ScriptFlow, steps []pgsql.ResolvedFlowStep, payload map[string]any) (scriptflow.ChainResult, error) {
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
			return result, err
		}
		return result, err
	}

	return result, nil
}

func mergeWorkInfosResult(dst map[string]any, src map[string]any) {
	if dst == nil || len(src) == 0 {
		return
	}
	for k, v := range src {
		dst[k] = v
	}
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
		"workID":    work.WorkID,
		"workTitle": work.WorkTitle,
		"trackID":   work.TrackID,
		"authorID":  work.AuthorID,
		"workInfos": infos,
	}
}

func removeSubmissionFiles(work model.Work) error {
	dstDir := filepath.Join(_const.FileRootPath, strconv.Itoa(work.TrackID), strconv.Itoa(work.AuthorID))
	entries, err := readDirFn(dstDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
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
			return rmErr
		}
	}

	return nil
}
